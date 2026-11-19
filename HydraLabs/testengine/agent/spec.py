"""The declarative spec the agent fills in (instead of writing code), and its deterministic validator.

spec = {
  "page": "cobro_ordenes",                       # file stem: <page>_locators.py / <page>_module.py
  "locators": {"BOTON_PROCESAR": "role=button[name=\"Procesar\"]",           # NAME -> selector (str)
               "MENSAJE_ERROR": {"selector": "li[role=\"status\"]", "dynamic": true}},   # dynamic: only exists after an action
  "data": {"order_number": "9800074297"},        # values taken from the steps; they go to the CSV
  "mock": {"pattern": "**/uploadFile", "status": 500, "delay_ms": 1500},      # optional: simulate POST responses
  "steps":  [{"n": 1, "do": {"action": "goto", "path": "/cobro_ordenes"}}, {"n": 2, "skip": "why this cannot be automated"}],
  "checks": [{"n": 1, "check": {"kind": "visible", "locator": "TABLA"}, "after_step": 3}]   # n = expected result number
}
Step texts and expected texts are NOT in the spec: they are copied from the Jira case by index, so they cannot drift."""
import difflib, keyword, re

ACTIONS = {  # action -> (required args, optional args)
    "goto": ({"path"}, set()),
    "click": ({"locator"}, set()),
    "dblclick": ({"locator"}, set()),
    "fill": ({"locator", "data"}, set()),
    "select": ({"locator", "data"}, {"widget"}),
    "upload_csv": ({"locator", "filename", "header", "rows"}, set()),
    "upload_binary": ({"locator", "filename", "mime"}, {"size"}),
    "download": ({"locator", "save_as"}, set()),
    "upload_file": ({"locator", "from_download"}, set()),
    "wait_response": (set(), set()),
    "verify": ({"kind"}, {"locator", "text"}),  # a step that is an assertion ("Verify X is disabled"); kind: see VERIFY_KINDS
}
VERIFY_KINDS = {"visible", "hidden", "enabled", "disabled", "text_contains"}
CHECKS = {  # kind -> (required, optional); a check targets a locator OR visible text
    "visible": (set(), {"locator", "text"}),
    "hidden": (set(), {"locator", "text"}),
    "enabled": ({"locator"}, set()),
    "disabled": ({"locator"}, set()),
    "text_contains": ({"locator", "text"}, set()),
    "max_requests": ({"n"}, set()),
    "no_requests": (set(), set()),
    "download_ok": (set(), set()),
}
SPEC_KEYS = {"page", "locators", "data", "mock", "steps", "checks"}
NAME = re.compile(r"^[A-Z][A-Z0-9_]*$")
KEY = re.compile(r"^[a-z][a-z0-9_]*$")


def _known(x, known):
    """A locator name is a string that is declared; anything else (a list, a number) is not, and must not crash the check."""
    return isinstance(x, str) and x in known


def _hint(name, known):
    """'¿quisiste decir X?' for a name the spec used but did not declare: the model often typed a near-miss of a real name."""
    close = difflib.get_close_matches(name, sorted(k for k in known if isinstance(k, str)), n=1, cutoff=0.6)
    return f" (¿quisiste decir {close[0]}?)" if close else ""


def used_locators(spec):
    """Every locator NAME the steps and checks refer to."""
    names = []
    for s in _objects(spec.get("steps")).values():
        do = s.get("do")
        if isinstance(do, dict) and isinstance(do.get("locator"), str):
            names.append(do["locator"])
    for c in _objects(spec.get("checks")).values():
        for ch in as_list(c.get("check") or {}):
            if isinstance(ch, dict) and isinstance(ch.get("locator"), str):
                names.append(ch["locator"])
    return names


def resolve(spec, existing):
    """Declares, from the repo's own locators file, the names the spec uses but did not declare (exact name, or the same
    name without its PAGE__ prefix). Only names the repo already has: nothing is invented. -> the names declared."""
    locs = spec.get("locators") or {}
    if not isinstance(locs, dict):
        return []
    locs = dict(locs)
    added = []
    for name in used_locators(spec):
        if name in locs:
            continue
        for cand in (name, name.split("__", 1)[-1]):
            if cand in existing:
                value, dynamic = existing[cand]
                locs[name] = {"selector": value, "dynamic": True} if dynamic else value
                added.append(name)
                break
    if added:
        spec["locators"] = locs
    return added


def normalize(spec):
    """Fixes two habits that are clear in intent, and says what it changed: unknown root keys are dropped, and a select's
    Playwright-style 'values' becomes 'data' when it names a data key (or holds a value a data key already has)."""
    notes = []
    if not isinstance(spec, dict) or "pages" in spec:
        return notes
    for k in sorted(set(spec) - SPEC_KEYS):
        del spec[k]
        notes.append(f"ignoré la clave '{k}' de la raíz")
    data = spec.get("data") if isinstance(spec.get("data"), dict) else {}
    for s in _objects(spec.get("steps")).values():
        do = s.get("do")
        if not (isinstance(do, dict) and do.get("action") == "select" and "data" not in do and "values" in do):
            continue
        v = do["values"][0] if isinstance(do["values"], list) and len(do["values"]) == 1 else do["values"]
        if not isinstance(v, str):
            continue
        key = v if v in data else next((k for k, x in data.items() if str(x) == v), None)
        if key:
            del do["values"]
            do["data"] = key
            notes.append(f"paso {s['n']}: 'values' → data.{key}")
    return notes


def _objects(v):
    """The steps or checks of a spec as dicts keyed by their 'n'; items that are not objects with an integer 'n' are left out."""
    if not isinstance(v, list):
        return {}
    return {x.get("n"): x for x in v if isinstance(x, dict) and isinstance(x.get("n"), int)}


def _check_errors(i, ch, known, mock, steps):
    errs = []
    kind = ch.get("kind")
    if not isinstance(kind, str) or kind not in CHECKS:
        hint = f" (recibí las claves {sorted(ch)}; formato exacto: " + '{"kind": "visible", "locator": "NOMBRE"})' if kind is None else ""
        return [f"checks[{i}]: kind '{kind}' inválido; usa uno de {sorted(CHECKS)}" + hint]
    req, opt = CHECKS[kind]
    args = set(ch) - {"kind"}
    if req - args:
        errs.append(f"checks[{i}]: {kind} requiere {sorted(req - args)}")
    if args - req - opt:
        errs.append(f"checks[{i}]: argumentos no permitidos en {kind}: {sorted(args - req - opt)}")
    if kind in ("visible", "hidden") and not ({"locator", "text"} & args):
        errs.append(f"checks[{i}]: {kind} necesita 'locator' o 'text'")
    if "locator" in ch and not _known(ch["locator"], known):
        errs.append(f"checks[{i}]: el localizador {ch['locator']} no está declarado en locators (agrégalo a 'locators' en la raíz){_hint(ch['locator'], known)}")
    if kind in ("max_requests", "no_requests") and not mock:
        errs.append(f"checks[{i}]: {kind} requiere 'mock' en la spec")
    if kind == "download_ok" and not any(isinstance(s.get("do"), dict) and s["do"].get("action") == "download" for s in steps.values()):
        errs.append(f"checks[{i}]: download_ok requiere un paso download")
    return errs


def as_list(x):
    return x if isinstance(x, list) else [x]


def locator_selector(v):
    if isinstance(v, str):
        return v
    sel = v.get("selector", "") if isinstance(v, dict) else ""
    return sel if isinstance(sel, str) else ""


def validate(spec, case, existing_locators=(), kinds=None):
    """-> list of error strings the agent must fix (empty = valid). kinds: selector -> the crawled role of that element."""
    errs = []
    if not isinstance(spec, dict):
        return ["la spec debe ser un objeto JSON"]
    if "pages" in spec:  # the model wrapped page and locators in 'pages': the validator reads them at the root
        errs.append("la spec no lleva 'pages': 'page' y 'locators' van en la raíz del objeto, no dentro de 'pages'")
    elif set(spec) - SPEC_KEYS:
        errs.append(f"claves no permitidas en la raíz de la spec: {sorted(set(spec) - SPEC_KEYS)}")
    if not re.match(r"^[a-z][a-z0-9_]*$", str(spec.get("page", ""))):
        errs.append("page: usa minúsculas_con_guiones_bajos (ej. cobro_ordenes)")
    locs = spec.get("locators") or {}
    if not isinstance(locs, dict):
        errs.append("locators: debe ser un objeto NOMBRE -> selector")
        locs = {}
    known = set(locs) | set(existing_locators)
    for name, v in locs.items():
        if not NAME.match(name):
            errs.append(f"locators.{name}: el nombre va en MAYÚSCULAS_CON_GUIONES_BAJOS")
        sel = locator_selector(v)
        if not sel or re.match(r"^(/html|//)", sel):
            errs.append(f"locators.{name}: selector vacío o XPath absoluto (usa data-testid, id, CSS o role=...)")
        elif re.search(r"(^|[\[\s])ref=", sel):  # the browser's snapshot id is not a selector: it dies with the page
            errs.append(f"locators.{name}: {sel!r} usa una referencia del snapshot (ref=...), no un selector; usa role=, #id o data-testid del mapa")
    data = spec.get("data") or {}
    if not isinstance(data, dict):
        errs.append("data: debe ser un objeto clave -> valor")
        data = {}
    for k in data:
        if not KEY.match(k):
            errs.append(f"data.{k}: usa minúsculas_con_guiones_bajos")
    mock = spec.get("mock")
    if mock and (not isinstance(mock, dict) or not {"pattern", "status", "delay_ms"} <= set(mock)):
        errs.append("mock: requiere pattern, status y delay_ms")
    for key in ("steps", "checks"):
        items = spec.get(key) or []
        if not isinstance(items, list) or any(not isinstance(x, dict) for x in items):
            errs.append(f"{key}: debe ser una lista de objetos JSON, cada uno con \"n\" (un entero) y su contenido")

    steps = _objects(spec.get("steps"))
    for i, cs in enumerate(case["steps"], 1):
        s = steps.get(i)
        if s is None:
            errs.append(f"steps: falta el paso {i} ({cs['action'][:50]!r}); da 'do' o 'skip' con el motivo")
            continue
        if "skip" in s:
            continue
        do = s.get("do")
        if not isinstance(do, dict):
            errs.append(f"steps[{i}]: 'do' debe ser un objeto, p. ej. {{\"action\": \"goto\", \"path\": \"/ruta\"}}")
            continue
        action = do.get("action") if isinstance(do.get("action"), str) else None
        if action not in ACTIONS:
            if action is None:  # the model wrote {"goto": ...} or {"type": ...}: say what was found and the exact shape
                errs.append(f"steps[{i}]: a 'do' le falta la clave \"action\" (recibí las claves {sorted(do)}). Formato exacto: "
                            f'{{"n": {i + 1}, "do": {{"action": "goto", "path": "/ruta"}}}}; los argumentos van junto a "action", no anidados')
            else:
                errs.append(f"steps[{i}]: acción '{action}' inválida; usa una de {sorted(ACTIONS)}")
            continue
        req, opt = ACTIONS[action]
        args = set(do) - {"action"}
        if req - args:
            errs.append(f"steps[{i}]: la acción {action} requiere {sorted(req - args)}")
        if args - req - opt:
            errs.append(f"steps[{i}]: argumentos no permitidos en {action}: {sorted(args - req - opt)}")
        if "locator" in do and not _known(do["locator"], known):
            errs.append(f"steps[{i}]: el localizador {do['locator']} no está declarado en locators (agrégalo a 'locators' en la raíz){_hint(do['locator'], known)}")
        if action == "select" and kinds and isinstance(do.get("locator"), str):  # the app says what the control is
            role = kinds.get(locator_selector(locs.get(do["locator"])))
            if role in ("button", "link", "textbox") or (role == "combobox" and do.get("widget") != "combobox"):
                errs.append(f"steps[{i}]: {do['locator']} es un {role} en la app, no un <select>; usa click"
                            + (' y widget: "combobox" en select' if role == "combobox" else ""))
        if action in ("fill", "select"):
            key = do.get("data")
            if not isinstance(key, str):
                errs.append(f"steps[{i}]: \"data\" debe ser el NOMBRE de una clave de data (texto), p. ej. \"order_number\"; recibí {key!r}")
            elif not _known(key, data):
                errs.append(f"steps[{i}]: data.{key} no existe en data")
            elif str(data[key]) not in cs["action"].replace("`", ""):
                errs.append(f"steps[{i}]: el valor data.{key}={data[key]!r} no aparece en el paso (no inventes valores)")
        if action == "upload_csv":
            for r in do.get("rows") or []:
                for cell in r if isinstance(r, list) else []:
                    if isinstance(cell, dict) and "data" in cell and not _known(cell["data"], data):
                        errs.append(f"steps[{i}]: la celda usa data.{cell['data']} que no existe")
        if action == "download":  # save_as becomes a Python variable in the generated module: "ejemplo-ordenes.csv" broke it
            sa = do.get("save_as")
            if not (isinstance(sa, str) and sa.isidentifier() and not keyword.iskeyword(sa)):
                errs.append(f"steps[{i}]: save_as debe ser un nombre de variable (letras, números y _, sin guiones ni puntos, p. ej. \"archivo_descargado\"); recibí {sa!r}")
        if action == "upload_file":
            if not isinstance(do.get("from_download"), str):
                errs.append(f"steps[{i}]: \"from_download\" debe ser el nombre del archivo de un download.save_as (texto, p. ej. \"sample.csv\"); recibí {do.get('from_download')!r}")
            elif not any(isinstance(x.get("do"), dict) and x["do"].get("save_as") == do.get("from_download") for x in steps.values()):
                errs.append(f"steps[{i}]: from_download '{do.get('from_download')}' no coincide con ningún download.save_as")
        if action == "verify":
            kind = do.get("kind")
            if not (isinstance(kind, str) and kind in VERIFY_KINDS):
                errs.append(f"steps[{i}]: verify.kind '{kind}' inválido; usa uno de {sorted(VERIFY_KINDS)}")
            elif kind in ("enabled", "disabled", "text_contains") and "locator" not in do:
                errs.append(f"steps[{i}]: verify {kind} requiere 'locator'")
            elif kind in ("visible", "hidden") and not ({"locator", "text"} & set(do)):
                errs.append(f"steps[{i}]: verify {kind} necesita 'locator' o 'text'")
        if action == "goto" and not str(do.get("path", "")).startswith("/"):
            errs.append(f"steps[{i}]: goto.path debe empezar con /")

    checks = _objects(spec.get("checks"))
    for i, ce in enumerate(case.get("expected", []), 1):
        c = checks.get(i)
        if c is None:
            errs.append(f"checks: falta el resultado esperado {i} ({ce['text'][:50]!r}); da 'check' o 'skip' con el motivo")
            continue
        if "skip" in c:
            continue
        for ch in as_list(c.get("check") or {}):
            if not isinstance(ch, dict):
                errs.append(f"checks[{i}]: 'check' debe ser un objeto con 'kind', p. ej. {{\"kind\": \"visible\", \"locator\": \"NOMBRE\"}}")
                continue
            errs += _check_errors(i, ch, known, mock, steps)
        after = c.get("after_step")
        if after is not None and (not isinstance(after, int) or after not in steps):
            errs.append(f"checks[{i}]: after_step {after} no existe")
    return errs
