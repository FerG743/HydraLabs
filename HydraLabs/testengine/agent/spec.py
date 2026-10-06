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
import re

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
def _check_errors(i, ch, known, mock, steps):
    errs = []
    kind = ch.get("kind")
    if kind not in CHECKS:
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
    if "locator" in ch and ch["locator"] not in known:
        errs.append(f"checks[{i}]: el localizador {ch['locator']} no está declarado en locators")
    if kind in ("max_requests", "no_requests") and not mock:
        errs.append(f"checks[{i}]: {kind} requiere 'mock' en la spec")
    if kind == "download_ok" and not any(isinstance(s.get("do"), dict) and s["do"].get("action") == "download" for s in steps.values()):
        errs.append(f"checks[{i}]: download_ok requiere un paso download")
    return errs


NAME = re.compile(r"^[A-Z][A-Z0-9_]*$")
KEY = re.compile(r"^[a-z][a-z0-9_]*$")


def as_list(x):
    return x if isinstance(x, list) else [x]


def locator_selector(v):
    return v if isinstance(v, str) else v.get("selector", "")


def validate(spec, case, existing_locators=()):
    """-> list of error strings the agent must fix (empty = valid)."""
    errs = []
    if not isinstance(spec, dict):
        return ["la spec debe ser un objeto JSON"]
    if not re.match(r"^[a-z][a-z0-9_]*$", str(spec.get("page", ""))):
        errs.append("page: usa minúsculas_con_guiones_bajos (ej. cobro_ordenes)")
    locs = spec.get("locators") or {}
    known = set(locs) | set(existing_locators)
    for name, v in locs.items():
        if not NAME.match(name):
            errs.append(f"locators.{name}: el nombre va en MAYÚSCULAS_CON_GUIONES_BAJOS")
        sel = locator_selector(v)
        if not sel or re.match(r"^(/html|//)", sel):
            errs.append(f"locators.{name}: selector vacío o XPath absoluto (usa data-testid, id, CSS o role=...)")
    data = spec.get("data") or {}
    for k in data:
        if not KEY.match(k):
            errs.append(f"data.{k}: usa minúsculas_con_guiones_bajos")
    mock = spec.get("mock")
    if mock and not {"pattern", "status", "delay_ms"} <= set(mock):
        errs.append("mock: requiere pattern, status y delay_ms")

    steps = {s.get("n"): s for s in spec.get("steps") or []}
    for i, cs in enumerate(case["steps"], 1):
        s = steps.get(i)
        if s is None:
            errs.append(f"steps: falta el paso {i} ({cs['action'][:50]!r}); da 'do' o 'skip' con el motivo")
            continue
        if "skip" in s:
            continue
        do = s.get("do") or {}
        action = do.get("action")
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
        if "locator" in do and do["locator"] not in known:
            errs.append(f"steps[{i}]: el localizador {do['locator']} no está declarado en locators")
        if action in ("fill", "select"):
            key = do.get("data")
            if key not in data:
                errs.append(f"steps[{i}]: data.{key} no existe en data")
            elif str(data[key]) not in cs["action"].replace("`", ""):
                errs.append(f"steps[{i}]: el valor data.{key}={data[key]!r} no aparece en el paso (no inventes valores)")
        if action == "upload_csv":
            for r in do.get("rows") or []:
                for cell in r:
                    if isinstance(cell, dict) and "data" in cell and cell["data"] not in data:
                        errs.append(f"steps[{i}]: la celda usa data.{cell['data']} que no existe")
        if action == "upload_file":
            if not any(isinstance(x.get("do"), dict) and x["do"].get("save_as") == do.get("from_download") for x in steps.values()):
                errs.append(f"steps[{i}]: from_download '{do.get('from_download')}' no coincide con ningún download.save_as")
        if action == "verify":
            if do.get("kind") not in VERIFY_KINDS:
                errs.append(f"steps[{i}]: verify.kind '{do.get('kind')}' inválido; usa uno de {sorted(VERIFY_KINDS)}")
            elif do["kind"] in ("enabled", "disabled", "text_contains") and "locator" not in do:
                errs.append(f"steps[{i}]: verify {do['kind']} requiere 'locator'")
            elif do["kind"] in ("visible", "hidden") and not ({"locator", "text"} & set(do)):
                errs.append(f"steps[{i}]: verify {do['kind']} necesita 'locator' o 'text'")
        if action == "goto" and not str(do.get("path", "")).startswith("/"):
            errs.append(f"steps[{i}]: goto.path debe empezar con /")

    checks = {c.get("n"): c for c in spec.get("checks") or []}
    for i, ce in enumerate(case.get("expected", []), 1):
        c = checks.get(i)
        if c is None:
            errs.append(f"checks: falta el resultado esperado {i} ({ce['text'][:50]!r}); da 'check' o 'skip' con el motivo")
            continue
        if "skip" in c:
            continue
        for ch in as_list(c.get("check") or {}):
            errs += _check_errors(i, ch, known, mock, steps)
        after = c.get("after_step")
        if after is not None and after not in steps:
            errs.append(f"checks[{i}]: after_step {after} no existe")
    return errs
