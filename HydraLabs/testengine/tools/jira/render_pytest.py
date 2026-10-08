#!/usr/bin/env python3
"""Normalized cases (*.case.json) -> README-format pytest files. Deterministic, stdlib only.

Writes, under --out:
  apps/<App>/data/<case>_data.csv        values found in the steps (one row per case)
  apps/<App>/data/expected_result.csv    test_id,expected_result for the report
  apps/<App>/locators/<app>_locators.py  selector constants (candidates, flagged TODO when guessed)
  apps/<App>/modules/<app>_module.py     one *_flujo per distinct action, built on DriverWrapper / driver.page
  tests/<App>/test_<case>.py             the scenario: business-mode steps + module calls + assertions
  utils/test_files.py, utils/network_mock.py   generic helpers (new files; nothing existing is modified)

Steps and expected results are mapped by a small grammar (ACTIONS / EXPECTED below). Whatever it cannot map is
never guessed: a pending step makes the test skip with the list; a pending expected result is reported in the
run but not asserted. Some expected results are anchored to the step they follow (ANCHORS): "stays disabled"
is true right after the bad upload, not at the end of the test. The shared locators/module files are rebuilt
from ALL given cases each run, so reruns are idempotent.

Facts confirmed against the real app survive regeneration: apps/<App>/locators.confirmed.json
  {"locators": {"INPUT_ORDER_NUMBER": "#orderNumber", ...}, "widgets": {"store": "combobox"}}
overrides the guessed values (and drops their TODO) and switches a select to a custom-combobox flow.

    python3 render_pytest.py --cases out/jira [--include-review] --app CobroOrdenes --out automation
"""
import argparse, csv, glob, json, os, re, shutil, unicodedata

# ---- grammar -------------------------------------------------------------------------------
ACTIONS = [  # (regex, op) - first match wins; groups become the op's arguments
    (r'^Navigate to (/\S*?)\.?$', "nav"),
    (r'^Verify "(.+?)" is (enabled|disabled)\.?$', "state"),
    (r'^Click "(.+?)" and capture the downloaded file\.?$', "download"),
    (r'^(?:Click|Press) "(.+?)"\.?$', "click"),
    (r'^Double-click "(.+?)"\.?$', "dblclick"),
    (r'^Upload (\S+) \(binary content, (.+?)\)\.?$', "upload_binary"),
    (r'^Upload (\S+) containing: (.+?)\.?$', "upload_csv"),
    (r'^Upload that file via the drop zone\.?$', "upload_prev"),
    (r'^Wait for the response\.?$', "wait_response"),
    (r'^Enter (.+?) (\S+?)\.?$', "fill"),
    (r'^Select (.+?) (\S+?)\.?$', "select"),
]
PRECONDITIONS = [  # searched anywhere in the precondition text
    (r'mocked to return HTTP (\d{3}) after (?:a )?([\d.]+) ?s', "mock_post"),
]
EXPECTED = [
    (r'^The download succeeds\.?$', "download_ok"),
    (r'^.*non-CSV file is rejected client-side and "(.+?)" stays (enabled|disabled)\.?$', "rejected_client"),
    (r'^.*\bat most (one|\d+) POST request.*$', "max_posts"),
    (r'^An error (?:toast|message|alert).* shown to the user\.?$', "error_shown"),
    (r'^The button returns to the enabled state with no stuck spinner\.?$', "button_ready"),
    (r'^Orders? .*\bappears?\b.*$', "numbers_visible"),
    (r'^No .*"(.+?)".* appears\.?$', "text_hidden"),
    (r'^The "(.+?)" .*disappears\.?$', "text_hidden"),
    (r'^.*\bappears in "(.+?)"\.?$', "text_visible"),
]
ANCHORS = {"rejected_client": "upload_binary"}  # expected op -> the step op it is checked right after

# items of an "Upload <file> containing: ..." step -> CSV row (order number, store)
CSV_ITEMS = re.compile(r"(?P<empty>an empty order number)"
                       r"|an unknown store \((?P<store>[^)]+)\)"
                       r"|a (?P<len>\d[\d,]*)-character order number"
                       r"|an SQL-injection string \((?P<sql>.+)\)$")


def plain(s):
    return re.sub(r"\s+", " ", s.replace("`", "")).strip()


def slug(s):
    s = unicodedata.normalize("NFKD", s).encode("ascii", "ignore").decode().lower()
    return re.sub(r"[^a-z0-9]+", "_", s).strip("_")


def app_slug(name):  # CobroOrdenes -> cobro_ordenes
    return re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()


def parse(rules, text, search=False):
    for pat, op in rules:
        m = (re.search if search else re.match)(pat, text)
        if m:
            return op, m.groups()
    return None, ()


# ---- per-case analysis -----------------------------------------------------------------------
def analyse(case):
    token = slug(case["id"])
    steps, pending, data = [], [], {}
    for s in case["steps"]:
        text = plain(s["action"])
        op, g = parse(ACTIONS, text)
        if op is None:
            pending.append(f"{s['n']}: {text}")
        if op in ("fill", "select"):
            data[slug(g[0])] = g[1]
        steps.append({"n": s["n"], "text": text, "op": op, "g": g})
    expected = []
    for e in case.get("expected", []):
        text = plain(e["text"])
        op, g = parse(EXPECTED, text)
        expected.append({"text": text, "op": op, "g": g})
    pre_op, pre_g = parse(PRECONDITIONS, plain(case.get("precondition", "")), search=True)
    return {"token": token, "steps": steps, "pending": pending, "expected": expected, "data": data,
            "pre": {"op": pre_op, "g": pre_g}}


# ---- shared module / locators (union over all cases) -----------------------------------------
class App:
    def __init__(self, confirmed=None):
        self.locators = {}  # NAME -> (value, todo comment or "")
        self.flujos = {}    # function name -> source
        self.confirmed = (confirmed or {}).get("locators", {})  # NAME -> value verified on the real app
        self.widgets = (confirmed or {}).get("widgets", {})     # field slug -> "combobox"

    def loc(self, name, value, todo=""):
        self.locators.setdefault(name, (value, todo))
        return name

    def flujo(self, name, src):
        self.flujos.setdefault(name, src)
        return name


def q(s):  # selector value safe inside a double-quoted role selector
    return s.replace('"', '\\"')


def button(app, label):
    return app.loc(f"BOTON_{slug(label).upper()}", f"role=button[name=\"{q(label)}\"]")


def state_flujo(app, label, state):
    L, base = button(app, label), slug(label)
    return app.flujo(f"verificar_{base}_{state}_flujo", f'''def verificar_{base}_{state}_flujo(driver):
    """Verifica que el botón {label} esté {'habilitado' if state == 'enabled' else 'deshabilitado'}."""
    assert driver.page.locator(L.{L}).is_{state}(), "El botón '{label}' debería estar {state}."
    add_step("El botón {label} está {state}.", "Exitoso")''')


def csv_rows(items_text, defaults):
    """'an empty order number; an unknown store (XXXX); ...' -> list of row expressions, or None if any part is unknown."""
    rows, last = [], 0
    for m in CSV_ITEMS.finditer(items_text):
        if items_text[last:m.start()].strip(" ;"):
            return None  # text between items that no rule explains
        last = m.end()
        if m.group("empty") is not None:
            rows.append('("", valid_store)')
        elif m.group("store") is not None:
            rows.append(f'(valid_order, {m.group("store")!r})')
        elif m.group("len") is not None:
            rows.append(f'("9" * {int(m.group("len").replace(",", ""))}, valid_store)')
        else:
            rows.append(f'({m.group("sql")!r}, valid_store)')
    if items_text[last:].strip(" ;.") or not rows:
        return None
    return rows if defaults.get("order_number") and defaults.get("store") else None


def build_calls(app, a, defaults):
    """-> (calls, ctx). calls = [(step dict, python call line or None)]. Registers locators/functions on the way."""
    ctx = {"has_download": False, "has_mock": False, "last_button": None, "pre_line": None, "anchor_index": {}}
    pre = a["pre"]
    if pre["op"] == "mock_post":
        app.loc("API_POST_PATRON", "**/*", "por defecto se simulan TODOS los POST (seguro); acotar al endpoint real")
        n = app.flujo("mockear_peticion_post_flujo", '''def mockear_peticion_post_flujo(driver, status, delay_ms):
    """Simula la respuesta HTTP de las peticiones POST y las cuenta (ver utils/network_mock.py)."""
    mock = RequestMock(driver.page, L.API_POST_PATRON, "POST", status, delay_ms).install()
    add_step(f"Peticiones POST simuladas: HTTP {status} tras {delay_ms} ms.", "Exitoso")
    return mock''')
        ctx["pre_line"] = f"mock = co.{n}(driver, {int(pre['g'][0])}, {int(float(pre['g'][1]) * 1000)})"
        ctx["has_mock"] = True

    calls = []
    for i, st in enumerate(a["steps"]):
        op, g, line = st["op"], st["g"], None
        if op == "nav":
            n = app.flujo(f"navegar_a_{slug(g[0])}_flujo", f'''def navegar_a_{slug(g[0])}_flujo(driver):
    """Navega a {g[0]}."""
    driver.get(Config.BASE_URL.rstrip("/") + {g[0]!r})
    add_step("El sistema muestra la pantalla {g[0]}.", "Exitoso")''')
            line = f"co.{n}(driver)"
        elif op in ("click", "dblclick", "state", "download"):
            L, base = button(app, g[0]), slug(g[0])
            if op in ("click", "dblclick"):
                ctx["last_button"] = g[0]
            if op == "click":
                n = app.flujo(f"hacer_clic_{base}_flujo", f'''def hacer_clic_{base}_flujo(driver):
    """Clic en el botón {g[0]}."""
    DriverWrapper(driver).click(L.{L}, description="el botón {g[0]}")''')
                line = f"co.{n}(driver)"
            elif op == "dblclick":
                n = app.flujo(f"doble_clic_{base}_flujo", f'''def doble_clic_{base}_flujo(driver):
    """Doble clic en el botón {g[0]}."""
    add_step("Doble clic en el botón {g[0]}.")
    driver.page.dblclick(L.{L})''')
                line = f"co.{n}(driver)"
            elif op == "state":
                line = f"co.{state_flujo(app, g[0], g[1])}(driver)"
            else:
                ctx["has_download"] = True
                n = app.flujo(f"descargar_con_{base}_flujo", f'''def descargar_con_{base}_flujo(driver, downloads_dir, file_base_name):
    """Clic en {g[0]} y guarda el archivo descargado; devuelve su ruta."""
    os.makedirs(downloads_dir, exist_ok=True)
    with driver.page.expect_download() as info:
        DriverWrapper(driver).click(L.{L}, description="el botón {g[0]}")
    download = info.value
    path = os.path.join(downloads_dir, f"{{file_base_name}}_{{download.suggested_filename}}")
    download.save_as(path)
    add_step(f"Archivo descargado: {{os.path.basename(path)}}", "Exitoso")
    return path''')
                line = f"archivo_descargado = co.{n}(driver, DOWNLOADS_DIR, CASO_ID)"
        elif op == "fill":
            f = slug(g[0])
            L = app.loc(f"INPUT_{f.upper()}", f"role=textbox[name=\"{q(g[0])}\" i]", f"confirmar el selector del campo '{g[0]}'")
            n = app.flujo(f"capturar_{f}_flujo", f'''def capturar_{f}_flujo(driver, {f}):
    """Captura el campo {g[0]}."""
    DriverWrapper(driver).send_keys(L.{L}, {f}, description="el campo {g[0]}")''')
            line = f"co.{n}(driver, {f})"
        elif op == "select":
            f = slug(g[0])
            L = app.loc(f"SELECT_{f.upper()}", f"role=combobox[name=\"{q(g[0])}\" i]", f"confirmar el selector de '{g[0]}' y si la opción va por value o por label")
            if app.widgets.get(f) == "combobox":  # custom (non-<select>) widget: open the list, pick the option
                n = app.flujo(f"seleccionar_{f}_flujo", f'''def seleccionar_{f}_flujo(driver, {f}):
    """Selecciona {g[0]} (combobox personalizado: abre la lista y elige la opción)."""
    add_step("Seleccionando {g[0]}.")
    driver.page.click(L.{L})
    driver.page.get_by_role("option", name={f}, exact=True).click()''')
            else:
                n = app.flujo(f"seleccionar_{f}_flujo", f'''def seleccionar_{f}_flujo(driver, {f}):
    """Selecciona {g[0]}."""
    add_step("Seleccionando {g[0]}.")
    driver.page.select_option(L.{L}, value={f})''')
            line = f"co.{n}(driver, {f})"
        elif op in ("upload_prev", "upload_binary", "upload_csv"):
            app.loc("INPUT_ARCHIVO", "input[type='file']", "confirmar el selector de la zona de carga (drop zone)")
            if op == "upload_prev":
                n = app.flujo("subir_archivo_flujo", '''def subir_archivo_flujo(driver, ruta_archivo):
    """Sube un archivo mediante la zona de carga."""
    add_step(f"Subiendo el archivo {os.path.basename(ruta_archivo)}.")
    driver.page.set_input_files(L.INPUT_ARCHIVO, ruta_archivo)''')
                line = f"co.{n}(driver, archivo_descargado)" if ctx["has_download"] else None
            elif op == "upload_binary":
                n = app.flujo("subir_archivo_binario_flujo", '''def subir_archivo_binario_flujo(driver, nombre, mime):
    """Sube un archivo binario (no CSV) mediante la zona de carga."""
    add_step(f"Subiendo el archivo binario {nombre} ({mime}).")
    driver.page.set_input_files(L.INPUT_ARCHIVO, binary_payload(nombre, mime))''')
                line = f"co.{n}(driver, {g[0]!r}, {g[1]!r})"
            else:
                rows = csv_rows(g[1], defaults)
                if rows:
                    header = [k for k in ("order_number", "store")]
                    app.loc("CSV_ENCABEZADO", header, "confirmar el encabezado con el CSV de muestra del portal")
                    n = app.flujo("subir_csv_flujo", '''def subir_csv_flujo(driver, nombre, filas):
    """Arma un CSV con las filas dadas y lo sube mediante la zona de carga."""
    add_step(f"Subiendo el CSV {nombre} con {len(filas)} filas de prueba.")
    driver.page.set_input_files(L.INPUT_ARCHIVO, csv_payload(nombre, L.CSV_ENCABEZADO, filas))''')
                    a["data"].setdefault("valid_order", defaults["order_number"])
                    a["data"].setdefault("valid_store", defaults["store"])
                    line = f"filas_{slug(g[0])} = [{', '.join(rows)}]\n    co.{n}(driver, {g[0]!r}, filas_{slug(g[0])})"
        elif op == "wait_response":
            n = app.flujo("esperar_respuesta_flujo", '''def esperar_respuesta_flujo(driver):
    """Espera a que termine la actividad de red."""
    driver.page.wait_for_load_state("networkidle")
    add_step("La respuesta del servidor fue recibida.", "Exitoso")''')
            line = f"co.{n}(driver)"
        if line is None and op is not None:  # understood, but depends on something this case does not provide
            st["op"] = None
            a["pending"].append(f"{st['n']}: {st['text']}")
        elif line is not None:
            ctx["anchor_index"][op] = i
        calls.append((st, line))
    return calls, ctx


TEXT_CHECKS = {  # shared by every case: identical source whichever case registers it first
    "visible": '''def verificar_texto_visible_flujo(driver, texto):
    """Verifica que el texto esté visible."""
    expect(driver.page.get_by_text(texto).first).to_be_visible()
    add_step(f"El texto '{texto}' está visible.", "Exitoso")''',
    "hidden": '''def verificar_texto_oculto_flujo(driver, texto):
    """Verifica que el texto esté oculto."""
    expect(driver.page.get_by_text(texto).first).to_be_hidden()
    add_step(f"El texto '{texto}' está oculto.", "Exitoso")''',
}


def build_expected(app, a, ctx):
    """-> [(expected dict, python lines or None, index of the step it follows or None for 'at the end')]"""
    out = []
    for e in a["expected"]:
        op, g, line = e["op"], e["g"], None
        if op in ("text_visible", "text_hidden"):
            kind = "visible" if op == "text_visible" else "hidden"
            n = app.flujo(f"verificar_texto_{'visible' if kind == 'visible' else 'oculto'}_flujo", TEXT_CHECKS[kind])
            line = f"co.{n}(driver, {g[0]!r})"
        elif op == "numbers_visible":
            nums = re.findall(r"\b\d{6,}\b", e["text"])
            n = app.flujo("verificar_texto_visible_flujo", TEXT_CHECKS["visible"])
            line = "\n    ".join(f"co.{n}(driver, {x!r})" for x in nums) if nums else None
        elif op == "download_ok" and ctx["has_download"]:
            line = 'assert archivo_descargado and os.path.exists(archivo_descargado), "La descarga no generó un archivo."'
        elif op == "rejected_client":
            line = f"co.{state_flujo(app, g[0], g[1])}(driver)"
            if ctx["has_mock"]:
                line += '\n    assert mock.count == 0, "Se envió una petición aunque el archivo debía rechazarse en el cliente."'
        elif op == "max_posts" and ctx["has_mock"]:
            n = 1 if g[0] == "one" else int(g[0])
            line = (f'assert mock.count <= {n}, f"Se enviaron {{mock.count}} peticiones POST (máximo {n}): '
                    f'posible doble cobro."')
        elif op == "error_shown":
            app.loc("MENSAJE_ERROR", "role=alert", "confirmar el selector del aviso o toast de error")
            n = app.flujo("verificar_mensaje_de_error_flujo", '''def verificar_mensaje_de_error_flujo(driver):
    """Verifica que se muestre un aviso de error al usuario."""
    expect(driver.page.locator(L.MENSAJE_ERROR).first).to_be_visible()
    add_step("Se muestra un aviso de error al usuario.", "Exitoso")''')
            line = f"co.{n}(driver)"
        elif op == "button_ready" and ctx["last_button"]:
            app.loc("SPINNER", "role=progressbar", "confirmar el selector del indicador de carga")
            m = app.flujo("verificar_sin_spinner_flujo", '''def verificar_sin_spinner_flujo(driver):
    """Verifica que no quede un indicador de carga atascado."""
    expect(driver.page.locator(L.SPINNER).first).to_be_hidden()
    add_step("No hay indicador de carga atascado.", "Exitoso")''')
            line = f"co.{state_flujo(app, ctx['last_button'], 'enabled')}(driver)\n    co.{m}(driver)"
        anchor = ctx["anchor_index"].get(ANCHORS.get(op)) if op in ANCHORS else None
        out.append((e, line, anchor))
    return out


# ---- rendering ---------------------------------------------------------------------------------
def render_locators(app_name, app):
    rows = [f"# apps/{app_name}/locators/{app_slug(app_name)}_locators.py",
            "#", "# Generado por HydraLabs. Los selectores son CANDIDATOS derivados de los textos del caso",
            "# (role + nombre accesible); los marcados TODO son suposiciones: confirmar con Codegen o el DOM real.", "",
            "", f"class {app_name}Locators:"]
    for name, (value, todo) in app.locators.items():
        if name in app.confirmed:  # verified on the real app: replaces the guess and drops its TODO
            rows.append(f"    {name} = {app.confirmed[name]!r}  # confirmado en el portal real")
        else:
            rows.append(f"    {name} = {value!r}" + (f"  # TODO: {todo}" if todo else ""))
    return "\n".join(rows) + "\n"


def render_module(app_name, app):
    head = f'''# apps/{app_name}/modules/{app_slug(app_name)}_module.py
#
# Generado por HydraLabs: un flujo reutilizable por acción distinta de los casos.

import os
from playwright.sync_api import expect
from utils.driver_wrapper import DriverWrapper
from utils.network_mock import RequestMock
from utils.test_files import binary_payload, csv_payload
from apps.{app_name}.locators.{app_slug(app_name)}_locators import {app_name}Locators as L
from config.settings import Config
from utils.templete_report import add_step
'''
    return head + "\n\n" + "\n\n\n".join(app.flujos.values()) + "\n"


def render_test(case, a, calls, ctx, expected, app_name):
    token, key = a["token"], case["id"]
    title = re.sub(r"^TC-\d+\s*[–-]\s*", "", plain(case["name"]))
    fn = f"test_{token}_{slug(title)[:48]}"
    cols = list(a["data"])
    L = ["# tests/%s/test_%s.py" % (app_name, token), "#", f"# {key} - {plain(case['name'])}",
         "# Generado por HydraLabs desde Jira. Pasos pendientes: " + (str(len(a["pending"])) if a["pending"] else "ninguno"), "",
         "import os", "import pytest", "",
         f"from apps.{app_name}.modules import {app_slug(app_name)}_module as co",
         "from utils.csv_reader import cargar_datos_csv", "from config.settings import Config",
         "from utils.templete_report import set_business_mode, add_step", "",
         "ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))"]
    if cols:
        L += [f'CASO_CSV = os.path.join(ROOT_DIR, "apps", "{app_name}", "data", "{token}_data.csv")',
              "DATA_CASO = cargar_datos_csv(CASO_CSV)"]
    L += ["", f"CASO_ID = {key!r}"]
    if cols:
        L += ["datos_caso = next((fila for fila in DATA_CASO if fila[0] == CASO_ID), None)"]
    if ctx["has_download"]:
        L += ['DOWNLOADS_DIR = os.path.join(ROOT_DIR, "reports", "downloads")']
    L += ["", "", f"def {fn}(driver):"]
    if a["pending"]:
        L.append(f"    pytest.skip({('Pasos sin automatizar: ' + ' | '.join(a['pending']))!r})")
    if cols:
        L += ["    if not datos_caso:", '        pytest.skip(f"No se encontraron datos para el caso {CASO_ID} en el CSV.")',
              f"    _caso_id, {', '.join(cols)} = datos_caso" if len(cols) > 1 else f"    _caso_id, {cols[0]} = datos_caso"]
    L += ["", "    set_business_mode(True)", ""]
    if plain(case.get("precondition", "")):
        L += ["    # ----- Precondición -----", f"    add_step({('Precondición: ' + plain(case['precondition']))!r}, level=\"business\")"]
    if ctx["pre_line"]:
        L.append(f"    {ctx['pre_line']}")
    by_anchor = {}
    for e, line, anchor in expected:
        if anchor is not None:
            by_anchor.setdefault(anchor, []).append((e, line))
    for i, (st, line) in enumerate(calls):
        L += ["", f"    # ----- Paso {st['n']}: {st['text']} -----", f"    add_step({st['text']!r}, level=\"business\")"]
        L.append(f"    {line}" if line else "    # PENDIENTE: sin regla para este paso")
        for e, eline in by_anchor.get(i, []):  # an expected result that is true right after this step
            L.append(f"    add_step({('Esperado: ' + e['text'])!r}, level=\"business\")")
            L.append(f"    {eline}")
    rest = [(e, line) for e, line, anchor in expected if anchor is None]
    if rest:
        L += ["", "    # ----- Resultados esperados -----"]
    for e, line in rest:
        L.append(f"    add_step({('Esperado: ' + e['text'])!r}, level=\"business\")")
        L.append(f"    {line}" if line else "    # PENDIENTE: resultado esperado sin aserción automática")
    L += ["", "    driver.delete_all_cookies()", ""]
    return "\n".join(L)


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write(text)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--cases", required=True, help="directory with *.case.json (cmd output of the Jira workflow)")
    ap.add_argument("--include-review", action="store_true", help="also read <cases>/review")
    ap.add_argument("--only", default="", help="render just this case id; the other cases in --cases still supply the default values")
    ap.add_argument("--app", required=True, help="app folder name, e.g. CobroOrdenes")
    ap.add_argument("--out", required=True)
    a = ap.parse_args()

    files = sorted(glob.glob(os.path.join(a.cases, "*.case.json")))
    if a.include_review:
        files += sorted(glob.glob(os.path.join(a.cases, "review", "*.case.json")))
    cases = sorted((json.load(open(f, encoding="utf-8")) for f in files), key=lambda c: c["id"])

    analyses = [analyse(c) for c in cases]
    defaults = {}  # first value seen for each captured field: a valid row for steps that need one (e.g. junk CSVs)
    for an in analyses:
        for k, v in an["data"].items():
            defaults.setdefault(k, v)

    if a.only:  # defaults above came from every case; from here on only the requested one is written
        keep = [i for i, c in enumerate(cases) if c["id"] == a.only]
        cases, analyses = [cases[i] for i in keep], [analyses[i] for i in keep]
    app_dir, tests_dir = os.path.join(a.out, "apps", a.app), os.path.join(a.out, "tests", a.app)
    confirmed_path = os.path.join(app_dir, "locators.confirmed.json")
    confirmed = json.load(open(confirmed_path, encoding="utf-8")) if os.path.exists(confirmed_path) else None
    app, report, expected_rows = App(confirmed), [], []
    for case, an in zip(cases, analyses):
        calls, ctx = build_calls(app, an, defaults)
        exp = build_expected(app, an, ctx)
        write(os.path.join(tests_dir, f"test_{an['token']}.py"), render_test(case, an, calls, ctx, exp, a.app))
        if an["data"]:
            write(os.path.join(app_dir, "data", f"{an['token']}_data.csv"),
                  "caso_id," + ",".join(an["data"]) + "\n" + ",".join([case["id"]] + [v for v in an["data"].values()]) + "\n")
        expected_rows.append((case["id"], " ".join(e["text"] for e in an["expected"])))
        report.append({"case": case["id"], "steps_mapped": sum(1 for s, _ in calls if s["op"]), "steps": len(calls),
                       "expected_asserted": sum(1 for _, l, _ in exp if l), "expected": len(exp), "pending": an["pending"]})

    write(os.path.join(app_dir, "locators", f"{app_slug(a.app)}_locators.py"), render_locators(a.app, app))
    write(os.path.join(app_dir, "modules", f"{app_slug(a.app)}_module.py"), render_module(a.app, app))
    for sub in ("locators", "modules"):
        init = os.path.join(app_dir, sub, "__init__.py")
        if not os.path.exists(init):
            write(init, "")
    support = os.path.join(os.path.dirname(os.path.abspath(__file__)), "support")
    for name in ("test_files.py", "network_mock.py"):  # new generic utils; never overwrites anything else
        os.makedirs(os.path.join(a.out, "utils"), exist_ok=True)
        shutil.copyfile(os.path.join(support, name), os.path.join(a.out, "utils", name))
    os.makedirs(os.path.join(app_dir, "data"), exist_ok=True)  # a case with no CSV fields never created it
    with open(os.path.join(app_dir, "data", "expected_result.csv"), "w", encoding="utf-8", newline="") as f:
        w = csv.writer(f)
        w.writerow(["test_id", "expected_result"])
        w.writerows(expected_rows)
    print(json.dumps(report, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
