"""spec + Jira case -> README-layout files, deterministically. The agent never writes Python: this does.

    apps/<App>/locators/<page>_locators.py   constants only (merged by name across cases; conflicts are refused)
    apps/<App>/modules/<page>_module.py      one *_flujo per distinct action (merged by name)
    apps/<App>/data/<case>_data.csv          the values from spec.data (one row)
    apps/<App>/data/expected_result.csv      test_id,expected_result (row replaced/added for this case)
    tests/<App>/test_<case>.py               Option A scenario: business steps (texts copied from the case) + module calls
    utils/test_files.py, utils/network_mock.py   generic helpers, copied only when the generated code needs them
"""
import ast, csv, os, re, shutil, unicodedata

HERE = os.path.dirname(os.path.abspath(__file__))
SUPPORT = os.path.join(HERE, "..", "tools", "jira", "support")


def plain(s):
    return re.sub(r"\s+", " ", s.replace("`", "")).strip()


def slug(s):
    s = unicodedata.normalize("NFKD", s).encode("ascii", "ignore").decode().lower()
    return re.sub(r"[^a-z0-9]+", "_", s).strip("_")


def camel(page):
    return "".join(p.capitalize() for p in page.split("_"))


# ---- existing files (merge by name) -----------------------------------------------------------------
def read_locators(path):
    """{NAME: (value, dynamic)} from an existing locators file."""
    out = {}
    if not os.path.exists(path):
        return out
    with open(path, encoding="utf-8") as f:
        for line in f:
            m = re.match(r"^\s+([A-Z][A-Z0-9_]*) = (.+?)(\s+# .*)?$", line.rstrip("\n"))
            if m:
                try:
                    out[m.group(1)] = (ast.literal_eval(m.group(2)), "dinámico" in (m.group(3) or ""))
                except (ValueError, SyntaxError):
                    pass
    return out


def read_functions(path):
    out = {}
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            src = f.read()
        for n in ast.parse(src).body:
            if isinstance(n, ast.FunctionDef):
                out[n.name] = ast.get_source_segment(src, n)
    return out


# ---- code generation ----------------------------------------------------------------------------------
class Build:
    def __init__(self, spec, case):
        self.spec, self.case = spec, case
        self.funcs = {}
        self.locs = {n: (v if isinstance(v, str) else v["selector"], isinstance(v, dict) and bool(v.get("dynamic")))
                     for n, v in (spec.get("locators") or {}).items()}
        self.data = {k: str(v) for k, v in (spec.get("data") or {}).items()}
        self.mock = spec.get("mock")
        if self.mock:
            self.locs.setdefault("API_POST_PATRON", (self.mock["pattern"], False))
        self.downloads = False

    def f(self, name, src):
        self.funcs.setdefault(name, src)
        return f"co.{name}"

    def desc(self, loc):
        return loc.lower().replace("_", " ")

    def step(self, do):
        a = do["action"]
        L = do.get("locator")
        if a == "goto":
            p = do["path"]
            return self.f(f"navegar_a_{slug(p)}_flujo", f'''def navegar_a_{slug(p)}_flujo(driver):
    """Navega a {p}."""
    driver.get(Config.BASE_URL.rstrip("/") + {p!r})
    add_step("El sistema muestra la pantalla {p}.", "Exitoso")''') + "(driver)"
        if a == "click":
            return self.f(f"hacer_clic_{L.lower()}_flujo", f'''def hacer_clic_{L.lower()}_flujo(driver):
    """Clic en {self.desc(L)}."""
    DriverWrapper(driver).click(L.{L}, description="{self.desc(L)}")''') + "(driver)"
        if a == "dblclick":
            return self.f(f"doble_clic_{L.lower()}_flujo", f'''def doble_clic_{L.lower()}_flujo(driver):
    """Doble clic en {self.desc(L)}."""
    add_step("Doble clic en {self.desc(L)}.")
    driver.page.dblclick(L.{L})''') + "(driver)"
        if a == "fill":
            return self.f(f"capturar_{L.lower()}_flujo", f'''def capturar_{L.lower()}_flujo(driver, valor):
    """Captura {self.desc(L)}."""
    DriverWrapper(driver).send_keys(L.{L}, valor, description="{self.desc(L)}")''') + f"(driver, {do['data']})"
        if a == "select":
            if do.get("widget") == "combobox":
                src = f'''def seleccionar_{L.lower()}_flujo(driver, valor):
    """Selecciona {self.desc(L)} (combobox personalizado: abre la lista y elige la opción)."""
    add_step("Seleccionando {self.desc(L)}.")
    DriverWrapper(driver).click(L.{L}, description="{self.desc(L)}")
    driver.page.get_by_role("option", name=valor, exact=True).click()'''
            else:
                src = f'''def seleccionar_{L.lower()}_flujo(driver, valor):
    """Selecciona {self.desc(L)}."""
    add_step("Seleccionando {self.desc(L)}.")
    driver.page.select_option(L.{L}, value=valor)'''
            return self.f(f"seleccionar_{L.lower()}_flujo", src) + f"(driver, {do['data']})"
        if a == "upload_csv":
            return self.f(f"subir_csv_en_{L.lower()}_flujo", f'''def subir_csv_en_{L.lower()}_flujo(driver, nombre, encabezado, filas):
    """Arma un CSV con las filas dadas y lo sube en {self.desc(L)}."""
    add_step(f"Subiendo el CSV {{nombre}} con {{len(filas)}} filas de prueba.")
    driver.page.set_input_files(L.{L}, csv_payload(nombre, encabezado, filas))''') + f"(driver, {do['filename']!r}, {do['header']!r}, [{', '.join(self.row(r) for r in do['rows'])}])"
        if a == "upload_binary":
            return self.f(f"subir_binario_en_{L.lower()}_flujo", f'''def subir_binario_en_{L.lower()}_flujo(driver, nombre, mime, tamano):
    """Sube un archivo binario (no CSV) en {self.desc(L)}."""
    add_step(f"Subiendo el archivo binario {{nombre}} ({{mime}}).")
    driver.page.set_input_files(L.{L}, binary_payload(nombre, mime, tamano))''') + f"(driver, {do['filename']!r}, {do['mime']!r}, {int(do.get('size', 2048))})"
        if a == "download":
            self.downloads = True
            return f"{do['save_as']} = " + self.f(f"descargar_con_{L.lower()}_flujo", f'''def descargar_con_{L.lower()}_flujo(driver, downloads_dir, file_base_name):
    """Clic en {self.desc(L)} y guarda el archivo descargado; devuelve su ruta."""
    os.makedirs(downloads_dir, exist_ok=True)
    with driver.page.expect_download() as info:
        DriverWrapper(driver).click(L.{L}, description="{self.desc(L)}")
    download = info.value
    path = os.path.join(downloads_dir, f"{{file_base_name}}_{{download.suggested_filename}}")
    download.save_as(path)
    add_step(f"Archivo descargado: {{os.path.basename(path)}}", "Exitoso")
    return path''') + "(driver, DOWNLOADS_DIR, CASO_ID)"
        if a == "upload_file":
            return self.f(f"subir_archivo_en_{L.lower()}_flujo", f'''def subir_archivo_en_{L.lower()}_flujo(driver, ruta_archivo):
    """Sube un archivo existente en {self.desc(L)}."""
    add_step(f"Subiendo el archivo {{os.path.basename(ruta_archivo)}}.")
    driver.page.set_input_files(L.{L}, ruta_archivo)''') + f"(driver, {do['from_download']})"
        if a == "verify":
            return self.check({k: v for k, v in do.items() if k != "action"})
        if a == "wait_response":
            return self.f("esperar_respuesta_flujo", '''def esperar_respuesta_flujo(driver):
    """Espera a que termine la actividad de red."""
    driver.page.wait_for_load_state("networkidle")
    add_step("La respuesta del servidor fue recibida.", "Exitoso")''') + "(driver)"
        raise ValueError(a)

    def row(self, cells):
        def cell(c):
            if isinstance(c, dict) and "data" in c:
                return c["data"]
            if isinstance(c, dict) and "repeat" in c:
                return f"{c['repeat']['char']!r} * {int(c['repeat']['times'])}"
            return repr("" if c is None else str(c))
        return "(" + ", ".join(cell(c) for c in cells) + ("," if len(cells) == 1 else "") + ")"

    def check(self, ch):
        k, L = ch["kind"], ch.get("locator")
        text_fn = {"visible": "visible", "hidden": "oculto"}
        if k in ("visible", "hidden"):
            if L:
                word = text_fn[k]
                return self.f(f"verificar_{L.lower()}_{word}_flujo", f'''def verificar_{L.lower()}_{word}_flujo(driver):
    """Verifica que {self.desc(L)} esté {word}."""
    expect(driver.page.locator(L.{L}).first).to_be_{'visible' if k == 'visible' else 'hidden'}()
    add_step("{self.desc(L).capitalize()} está {word}.", "Exitoso")''') + "(driver)"
            word = text_fn[k]
            return self.f(f"verificar_texto_{word}_flujo", f'''def verificar_texto_{word}_flujo(driver, texto):
    """Verifica que el texto esté {word}."""
    expect(driver.page.get_by_text(texto).first).to_be_{'visible' if k == 'visible' else 'hidden'}()
    add_step(f"El texto '{{texto}}' está {word}.", "Exitoso")''') + f"(driver, {ch['text']!r})"
        if k in ("enabled", "disabled"):
            return self.f(f"verificar_{L.lower()}_{k}_flujo", f'''def verificar_{L.lower()}_{k}_flujo(driver):
    """Verifica que {self.desc(L)} esté {'habilitado' if k == 'enabled' else 'deshabilitado'}."""
    assert driver.page.locator(L.{L}).is_{k}(), "{self.desc(L).capitalize()} debería estar {k}."
    add_step("{self.desc(L).capitalize()} está {k}.", "Exitoso")''') + "(driver)"
        if k == "text_contains":
            return self.f(f"verificar_texto_en_{L.lower()}_flujo", f'''def verificar_texto_en_{L.lower()}_flujo(driver, texto):
    """Verifica que {self.desc(L)} contenga el texto."""
    expect(driver.page.locator(L.{L}).first).to_contain_text(texto)
    add_step(f"{self.desc(L).capitalize()} contiene '{{texto}}'.", "Exitoso")''') + f"(driver, {ch['text']!r})"
        if k == "max_requests":
            n = int(ch["n"])
            return f'assert mock.count <= {n}, f"Se enviaron {{mock.count}} peticiones POST (máximo {n}): posible doble cobro."'
        if k == "no_requests":
            return 'assert mock.count == 0, f"Se enviaron {mock.count} peticiones POST y no debía enviarse ninguna."'
        if k == "download_ok":
            dl = next(s["do"]["save_as"] for s in self.spec["steps"] if s.get("do", {}).get("action") == "download")
            return f'assert {dl} and os.path.exists({dl}), "La descarga no generó un archivo."'
        raise ValueError(k)


def module_header(app, page, body):
    imports = ["import os"] if "os." in body else []
    imports += ["from playwright.sync_api import expect"] if "expect(" in body else []
    imports += ["from utils.driver_wrapper import DriverWrapper"]
    imports += ["from utils.network_mock import RequestMock"] if "RequestMock" in body else []
    helpers = [h for h in ("binary_payload", "csv_payload") if h in body]
    imports += [f"from utils.test_files import {', '.join(helpers)}"] if helpers else []
    imports += [f"from apps.{app}.locators.{page}_locators import {camel(page)}Locators as L", "from config.settings import Config",
                "from utils.templete_report import add_step"]
    return f"# apps/{app}/modules/{page}_module.py\n#\n# Generado por HydraLabs: un flujo reutilizable por acción distinta.\n\n" + "\n".join(imports) + "\n"


def render(repo, app, case, spec):
    """Writes the files. Returns {"files": [...], "warnings": [...], "errors": [...]} (errors: nothing is written)."""
    page, token = spec["page"], slug(case["id"])
    apps, tests = os.path.join(repo, "apps", app), os.path.join(repo, "tests", app)
    b = Build(spec, case)
    warnings, errors = [], []

    # steps / checks -> calls
    lines_steps, pending = [], []
    for i, cs in enumerate(case["steps"], 1):
        s = next(x for x in spec["steps"] if x["n"] == i)
        text = plain(cs["action"])
        call = None
        if "skip" in s:
            pending.append(f"{i}: {text} ({s['skip']})")
        else:
            call = b.step(s["do"])
        lines_steps.append((i, text, call, s.get("skip")))
    checks = {c["n"]: c for c in spec.get("checks") or []}
    placed, final = {}, []
    for j, ce in enumerate(case.get("expected", []), 1):
        c = checks[j]
        text = plain(ce["text"])
        from spec import as_list
        line = None if "skip" in c else "\n".join(b.check(ch) for ch in as_list(c["check"]))
        entry = (text, line, c.get("skip"))
        (placed.setdefault(c["after_step"], []) if c.get("after_step") else final).append(entry)

    # merge locators / functions with what already exists
    loc_path = os.path.join(apps, "locators", f"{page}_locators.py")
    mod_path = os.path.join(apps, "modules", f"{page}_module.py")
    locs = read_locators(loc_path)
    for name, (sel, dyn) in b.locs.items():
        if name in locs and locs[name][0] != sel:
            errors.append(f"el localizador {name} ya existe con otro valor ({locs[name][0]!r}); reutilízalo o usa otro nombre")
        locs.setdefault(name, (sel, dyn))
    funcs = read_functions(mod_path)
    for name, src in b.funcs.items():
        if name in funcs and funcs[name].strip() != src.strip():
            errors.append(f"la función {name} ya existe con otro cuerpo: usa otro localizador o nombre")
        funcs.setdefault(name, src)
    if b.mock:
        funcs.setdefault("mockear_peticion_post_flujo", '''def mockear_peticion_post_flujo(driver, status, delay_ms):
    """Simula la respuesta HTTP de las peticiones POST y las cuenta (ver utils/network_mock.py)."""
    mock = RequestMock(driver.page, L.API_POST_PATRON, "POST", status, delay_ms).install()
    add_step(f"Peticiones POST simuladas: HTTP {status} tras {delay_ms} ms.", "Exitoso")
    return mock''')
    if errors:
        return {"files": [], "warnings": warnings, "errors": errors}

    files = []

    def write(path, text):
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "w", encoding="utf-8", newline="\n") as f:
            f.write(text)
        files.append(os.path.relpath(path, repo))

    # locators
    rows = [f"# apps/{app}/locators/{page}_locators.py", "#", "# Generado por HydraLabs. Prioridad: data-testid > CSS > role/texto > XPath.", "",
            "", f"class {camel(page)}Locators:"]
    for name, (sel, dyn) in locs.items():
        rows.append(f"    {name} = {sel!r}" + ("  # dinámico: solo existe tras una acción" if dyn else ""))
    write(loc_path, "\n".join(rows) + "\n")
    # module (+ helpers it needs)
    body = "\n\n\n".join(funcs.values())
    write(mod_path, module_header(app, page, body) + "\n\n" + body + "\n")
    for sub in ("locators", "modules"):
        init = os.path.join(apps, sub, "__init__.py")
        if not os.path.exists(init):
            write(init, "")
    for helper, needle in (("test_files.py", "_payload"), ("network_mock.py", "RequestMock")):
        if needle in body and not os.path.exists(os.path.join(repo, "utils", helper)):
            os.makedirs(os.path.join(repo, "utils"), exist_ok=True)
            shutil.copyfile(os.path.join(SUPPORT, helper), os.path.join(repo, "utils", helper))
            files.append(f"utils/{helper}")
            warnings.append(f"utils/{helper} es un archivo nuevo (herramienta genérica); no modifica nada existente")

    # data + expected_result
    cols = list(b.data)
    if cols:
        write(os.path.join(apps, "data", f"{token}_data.csv"), "caso_id," + ",".join(cols) + "\n" + ",".join([case["id"]] + [b.data[c] for c in cols]) + "\n")
    er = os.path.join(apps, "data", "expected_result.csv")
    rows_er = []
    if os.path.exists(er):
        with open(er, encoding="utf-8", newline="") as f:
            rows_er = [r for r in list(csv.reader(f))[1:] if r and r[0] != case["id"]]
    rows_er.append([case["id"], " ".join(plain(e["text"]) for e in case.get("expected", []))])
    os.makedirs(os.path.dirname(er), exist_ok=True)
    with open(er, "w", encoding="utf-8", newline="") as f:
        w = csv.writer(f)
        w.writerow(["test_id", "expected_result"])
        w.writerows(rows_er)
    files.append(os.path.relpath(er, repo))

    # test
    title = re.sub(r"^TC-\d+\s*[–-]\s*", "", plain(case["name"]))
    L = [f"# tests/{app}/test_{token}.py", "#", f"# {case['id']} - {plain(case['name'])}",
         "# Generado por HydraLabs. Pasos pendientes: " + (str(len(pending)) if pending else "ninguno"), "",
         "import os", "import pytest", "", f"from apps.{app}.modules import {page}_module as co",
         "from utils.csv_reader import cargar_datos_csv", "from config.settings import Config",
         "from utils.templete_report import set_business_mode, add_step", "",
         "ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))"]
    if cols:
        L += [f'CASO_CSV = os.path.join(ROOT_DIR, "apps", "{app}", "data", "{token}_data.csv")', "DATA_CASO = cargar_datos_csv(CASO_CSV)"]
    L += ["", f"CASO_ID = {case['id']!r}"]
    if cols:
        L.append("datos_caso = next((fila for fila in DATA_CASO if fila[0] == CASO_ID), None)")
    if b.downloads:
        L.append('DOWNLOADS_DIR = os.path.join(ROOT_DIR, "reports", "downloads")')
    L += ["", "", f"def test_{token}_{slug(title)[:48]}(driver):"]
    if pending:
        L.append(f"    pytest.skip({('Pasos sin automatizar: ' + ' | '.join(pending))!r})")
    if cols:
        L += ["    if not datos_caso:", '        pytest.skip(f"No se encontraron datos para el caso {CASO_ID} en el CSV.")',
              f"    _caso_id, {', '.join(cols)} = datos_caso" if len(cols) > 1 else f"    _caso_id, {cols[0]} = datos_caso"]
    L += ["", "    set_business_mode(True)", ""]
    if plain(case.get("precondition", "")):
        L += ["    # ----- Precondición -----", f"    add_step({('Precondición: ' + plain(case['precondition']))!r}, level=\"business\")"]
    if b.mock:
        m = b.mock
        L.append(f"    mock = co.mockear_peticion_post_flujo(driver, {int(m['status'])}, {int(m['delay_ms'])})")

    def emit_checks(entries):
        for text, line, skip in entries:
            L.append(f"    add_step({('Esperado: ' + text)!r}, level=\"business\")")
            L.append(("    " + line.replace("\n", "\n    ")) if line else f"    # PENDIENTE: resultado esperado sin aserción ({skip})")

    for i, text, call, skip in lines_steps:
        L += ["", f"    # ----- Paso {i}: {text} -----", f"    add_step({text!r}, level=\"business\")"]
        L.append(f"    {call}" if call else f"    # PENDIENTE: {skip}")
        emit_checks(placed.get(i, []))
    if final:
        L += ["", "    # ----- Resultados esperados -----"]
        emit_checks(final)
    L += ["", "    driver.delete_all_cookies()", ""]
    write(os.path.join(tests, f"test_{token}.py"), "\n".join(L))
    return {"files": files, "warnings": warnings, "errors": []}


def scaffold(repo, app):
    """Layout the README requires, in advance and idempotently. Returns warnings (e.g. suite not registered)."""
    for sub in ("data", "locators", "modules"):
        os.makedirs(os.path.join(repo, "apps", app, sub), exist_ok=True)
    os.makedirs(os.path.join(repo, "tests", app), exist_ok=True)
    er = os.path.join(repo, "apps", app, "data", "expected_result.csv")
    if not os.path.exists(er):
        with open(er, "w", encoding="utf-8", newline="") as f:
            f.write("test_id,expected_result\n")
    for sub in ("locators", "modules"):
        init = os.path.join(repo, "apps", app, sub, "__init__.py")
        if not os.path.exists(init):
            open(init, "w").close()
    runner, warnings = os.path.join(repo, "runner_central.py"), []
    if os.path.exists(runner):
        with open(runner, encoding="utf-8") as f:
            if f"'{app}'" not in f.read() and f'"{app}"' not in f.read():
                warnings.append(f"registra la suite '{app}' en runner_central.py (choices y rutas), como pide el README")
    return warnings
