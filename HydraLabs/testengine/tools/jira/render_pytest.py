#!/usr/bin/env python3
"""Normalized cases (*.case.json) -> README-format pytest files. Deterministic, stdlib only.

Writes, under --out:
  apps/<App>/data/<case>_data.csv        values found in the steps (one row per case)
  apps/<App>/data/expected_result.csv    test_id,expected_result for the report
  apps/<App>/locators/<app>_locators.py  selector constants (candidates, flagged TODO when guessed)
  apps/<App>/modules/<app>_module.py     one *_flujo per distinct action, built on DriverWrapper / driver.page
  tests/<App>/test_<case>.py             the scenario: business-mode steps + module calls + assertions

Steps and expected results are mapped by a small grammar (RULES below). Whatever it cannot map is
never guessed: a pending step makes the test skip with the list; a pending expected result is
reported in the run but not asserted. The shared locators/module files are rebuilt from ALL given
cases each run, so reruns are idempotent.

    python3 render_pytest.py --cases out/jira [--include-review] --app CobroOrdenes --out automation
"""
import argparse, csv, glob, json, os, re, unicodedata

# ---- grammar -------------------------------------------------------------------------------
ACTIONS = [  # (regex, op) - first match wins; groups become the op's arguments
    (r'^Navigate to (/\S*?)\.?$', "nav"),
    (r'^Verify "(.+?)" is (enabled|disabled)\.?$', "state"),
    (r'^Click "(.+?)" and capture the downloaded file\.?$', "download"),
    (r'^(?:Click|Press) "(.+?)"\.?$', "click"),
    (r'^Double-click "(.+?)"\.?$', "dblclick"),
    (r'^Upload that file via the drop zone\.?$', "upload_prev"),
    (r'^Wait for the response\.?$', "wait_response"),
    (r'^Enter (.+?) (\S+?)\.?$', "fill"),
    (r'^Select (.+?) (\S+?)\.?$', "select"),
]
EXPECTED = [
    (r'^The download succeeds\.?$', "download_ok"),
    (r'^Orders? .*\bappears?\b.*$', "numbers_visible"),
    (r'^No .*"(.+?)".* appears\.?$', "text_hidden"),
    (r'^The "(.+?)" .*disappears\.?$', "text_hidden"),
    (r'^.*\bappears in "(.+?)"\.?$', "text_visible"),
]


def plain(s):
    return re.sub(r"\s+", " ", s.replace("`", "")).strip()


def slug(s):
    s = unicodedata.normalize("NFKD", s).encode("ascii", "ignore").decode().lower()
    return re.sub(r"[^a-z0-9]+", "_", s).strip("_")


def app_slug(name):  # CobroOrdenes -> cobro_ordenes
    return re.sub(r"(?<!^)(?=[A-Z])", "_", name).lower()


def parse(rules, text):
    for pat, op in rules:
        m = re.match(pat, text)
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
    return {"token": token, "steps": steps, "pending": pending, "expected": expected, "data": data}


# ---- shared module / locators (union over all cases) -----------------------------------------
class App:
    def __init__(self):
        self.locators = {}  # NAME -> (selector, todo comment or "")
        self.flujos = {}    # function name -> source

    def loc(self, name, selector, todo=""):
        self.locators.setdefault(name, (selector, todo))
        return name

    def flujo(self, name, src):
        self.flujos.setdefault(name, src)
        return name


def q(s):  # selector value safe inside a double-quoted role selector
    return s.replace('"', '\\"')


def build_calls(app, a):
    """-> list of (step dict, python call line or None). Registers locators/functions on the way."""
    calls, has_download = [], False
    for st in a["steps"]:
        op, g, line = st["op"], st["g"], None
        if op == "nav":
            n = app.flujo(f"navegar_a_{slug(g[0])}_flujo", f'''def navegar_a_{slug(g[0])}_flujo(driver):
    """Navega a {g[0]}."""
    driver.get(Config.BASE_URL.rstrip("/") + {g[0]!r})
    add_step("El sistema muestra la pantalla {g[0]}.", "Exitoso")''')
            line = f"co.{n}(driver)"
        elif op in ("click", "dblclick", "state", "download"):
            L = app.loc(f"BOTON_{slug(g[0]).upper()}", f"role=button[name=\"{q(g[0])}\"]")
            base = slug(g[0])
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
                n = app.flujo(f"verificar_{base}_{g[1]}_flujo", f'''def verificar_{base}_{g[1]}_flujo(driver):
    """Verifica que el botón {g[0]} esté {'habilitado' if g[1] == 'enabled' else 'deshabilitado'}."""
    assert driver.page.locator(L.{L}).is_{g[1]}(), "El botón '{g[0]}' debería estar {g[1]}."
    add_step("El botón {g[0]} está {g[1]}.", "Exitoso")''')
                line = f"co.{n}(driver)"
            else:
                has_download = True
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
            n = app.flujo(f"seleccionar_{f}_flujo", f'''def seleccionar_{f}_flujo(driver, {f}):
    """Selecciona {g[0]}."""
    add_step("Seleccionando {g[0]}.")
    driver.page.select_option(L.{L}, value={f})''')
            line = f"co.{n}(driver, {f})"
        elif op == "upload_prev":
            L = app.loc("INPUT_ARCHIVO", "input[type='file']", "confirmar el selector de la zona de carga (drop zone)")
            n = app.flujo("subir_archivo_flujo", '''def subir_archivo_flujo(driver, ruta_archivo):
    """Sube un archivo mediante la zona de carga."""
    add_step(f"Subiendo el archivo {os.path.basename(ruta_archivo)}.")
    driver.page.set_input_files(L.INPUT_ARCHIVO, ruta_archivo)''')
            line = f"co.{n}(driver, archivo_descargado)" if has_download else None
        elif op == "wait_response":
            n = app.flujo("esperar_respuesta_flujo", '''def esperar_respuesta_flujo(driver):
    """Espera a que termine la actividad de red."""
    driver.page.wait_for_load_state("networkidle")
    add_step("La respuesta del servidor fue recibida.", "Exitoso")''')
            line = f"co.{n}(driver)"
        if line is None and op is not None:  # understood, but depends on something this case does not provide
            st["op"] = None
            a["pending"].append(f"{st['n']}: {st['text']}")
        calls.append((st, line))
    return calls, has_download


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


def build_expected(app, a, has_download):
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
        elif op == "download_ok" and has_download:
            line = 'assert archivo_descargado and os.path.exists(archivo_descargado), "La descarga no generó un archivo."'
        out.append((e, line))
    return out


# ---- rendering ---------------------------------------------------------------------------------
def render_locators(app_name, app):
    rows = [f"# apps/{app_name}/locators/{app_slug(app_name)}_locators.py",
            "#", "# Generado por HydraLabs. Los selectores son CANDIDATOS derivados de los textos del caso",
            "# (role + nombre accesible); los marcados TODO son suposiciones: confirmar con Codegen o el DOM real.", "",
            "", f"class {app_name}Locators:"]
    for name, (sel, todo) in app.locators.items():
        rows.append(f"    {name} = {sel!r}" + (f"  # TODO: {todo}" if todo else ""))
    return "\n".join(rows) + "\n"


def render_module(app_name, app):
    head = f'''# apps/{app_name}/modules/{app_slug(app_name)}_module.py
#
# Generado por HydraLabs: un flujo reutilizable por acción distinta de los casos.

import os
from playwright.sync_api import expect
from utils.driver_wrapper import DriverWrapper
from apps.{app_name}.locators.{app_slug(app_name)}_locators import {app_name}Locators as L
from config.settings import Config
from utils.templete_report import add_step
'''
    return head + "\n\n" + "\n\n\n".join(app.flujos.values()) + "\n"


def render_test(case, a, calls, expected, app_name):
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
    if any("archivo_descargado" in (c or "") for _, c in calls):
        L += ['DOWNLOADS_DIR = os.path.join(ROOT_DIR, "reports", "downloads")']
    L += ["", "", f"def {fn}(driver):"]
    if a["pending"]:
        L.append(f"    pytest.skip({('Pasos sin automatizar: ' + ' | '.join(a['pending']))!r})")
    if cols:
        L += ["    if not datos_caso:", '        pytest.skip(f"No se encontraron datos para el caso {CASO_ID} en el CSV.")',
              f"    _caso_id, {', '.join(cols)} = datos_caso" if len(cols) > 1 else f"    _caso_id, {cols[0]} = datos_caso"]
    L += ["", "    set_business_mode(True)", ""]
    if plain(case.get("precondition", "")):
        L += [f"    # ----- Precondición -----", f"    add_step({('Precondición: ' + plain(case['precondition']))!r}, level=\"business\")"]
    for st, line in calls:
        L += ["", f"    # ----- Paso {st['n']}: {st['text']} -----", f"    add_step({st['text']!r}, level=\"business\")"]
        L.append(f"    {line}" if line else "    # PENDIENTE: sin regla para este paso")
    if expected:
        L += ["", "    # ----- Resultados esperados -----"]
    for e, line in expected:
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
    ap.add_argument("--app", required=True, help="app folder name, e.g. CobroOrdenes")
    ap.add_argument("--out", required=True)
    a = ap.parse_args()

    files = sorted(glob.glob(os.path.join(a.cases, "*.case.json")))
    if a.include_review:
        files += sorted(glob.glob(os.path.join(a.cases, "review", "*.case.json")))
    cases = sorted((json.load(open(f, encoding="utf-8")) for f in files), key=lambda c: c["id"])

    app, report, expected_rows = App(), [], []
    app_dir, tests_dir = os.path.join(a.out, "apps", a.app), os.path.join(a.out, "tests", a.app)
    for case in cases:
        an = analyse(case)
        calls, has_dl = build_calls(app, an)
        exp = build_expected(app, an, has_dl)
        write(os.path.join(tests_dir, f"test_{an['token']}.py"), render_test(case, an, calls, exp, a.app))
        if an["data"]:
            write(os.path.join(app_dir, "data", f"{an['token']}_data.csv"),
                  "caso_id," + ",".join(an["data"]) + "\n" + ",".join([case["id"]] + [v for v in an["data"].values()]) + "\n")
        expected_rows.append((case["id"], " ".join(e["text"] for e in an["expected"])))
        report.append({"case": case["id"], "steps_mapped": sum(1 for s, _ in calls if s["op"]), "steps": len(calls),
                       "expected_asserted": sum(1 for _, l in exp if l), "expected": len(exp), "pending": an["pending"]})

    write(os.path.join(app_dir, "locators", f"{app_slug(a.app)}_locators.py"), render_locators(a.app, app))
    write(os.path.join(app_dir, "modules", f"{app_slug(a.app)}_module.py"), render_module(a.app, app))
    for sub in ("locators", "modules"):
        init = os.path.join(app_dir, sub, "__init__.py")
        if not os.path.exists(init):
            write(init, "")
    with open(os.path.join(app_dir, "data", "expected_result.csv"), "w", encoding="utf-8", newline="") as f:
        w = csv.writer(f)
        w.writerow(["test_id", "expected_result"])
        w.writerows(expected_rows)
    print(json.dumps(report, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
