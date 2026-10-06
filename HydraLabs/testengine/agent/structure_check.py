"""Deterministic README-compliance check for what the agent wrote under apps/<App>/ and tests/<App>/.
Returns a list of violations ("path:line: message") the agent must fix. Rules come from the framework README:
layout, locators = constants only, modules via DriverWrapper, tests = business steps + module calls + data from CSV,
absolute imports, no secrets in code, and only DriverWrapper methods that really exist."""
import ast, csv, os, re

WRAPPER_METHODS = {"click", "send_keys", "wait", "switch_to_frame", "get_performance_metrics"}  # what DriverWrapper really has
NATIVE_BASICS = {"click", "fill", "type"}  # README: clicking/typing must go through DriverWrapper, never page.* directly
SECRET = re.compile(r"(?i)(password|contrase[ñn]a|token|secret)\w*\s*=\s*['\"][^'\"]{3,}['\"]")


def _read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


def _py(path):
    with open(path, encoding="utf-8") as f:
        src = f.read()
    return src, ast.parse(src)


def _calls_on(tree, name):
    """(attr, line) for every <name>.<attr>(...) call, e.g. wrapper.is_visible()."""
    for n in ast.walk(tree):
        if isinstance(n, ast.Call) and isinstance(n.func, ast.Attribute) and isinstance(n.func.value, ast.Name) and n.func.value.id == name:
            yield n.func.attr, n.lineno


def _page_calls(tree):
    """page.<attr>() reached as driver.page.<attr>() or page.<attr>()."""
    for n in ast.walk(tree):
        if isinstance(n, ast.Call) and isinstance(n.func, ast.Attribute):
            v = n.func.value
            if (isinstance(v, ast.Attribute) and v.attr == "page") or (isinstance(v, ast.Name) and v.id == "page"):
                yield n.func.attr, n.lineno


def check(repo, app):
    out = []
    apps, tests = os.path.join(repo, "apps", app), os.path.join(repo, "tests", app)
    rel = lambda p: os.path.relpath(p, repo)

    def files(base, sub, ext):
        d = os.path.join(base, sub) if sub else base
        return sorted(os.path.join(d, f) for f in (os.listdir(d) if os.path.isdir(d) else []) if f.endswith(ext) and f != "__init__.py")

    # --- locators: constants only
    for p in files(apps, "locators", ".py"):
        src, tree = _py(p)
        for node in tree.body:
            if isinstance(node, ast.ClassDef):
                for st in node.body:
                    if isinstance(st, (ast.FunctionDef, ast.AsyncFunctionDef)):
                        out.append(f"{rel(p)}:{st.lineno}: los localizadores solo declaran selectores, sin lógica ni interacciones")
                    if isinstance(st, ast.Assign):
                        for t in st.targets:
                            if isinstance(t, ast.Name) and not t.id.isupper():
                                out.append(f"{rel(p)}:{st.lineno}: la constante '{t.id}' debe ir en MAYÚSCULAS")
                        v = st.value
                        if isinstance(v, ast.Constant) and isinstance(v.value, str) and re.match(r"^(/html|//)", v.value):
                            out.append(f"{rel(p)}:{st.lineno}: evita XPath absoluto/frágil; usa data-testid, id o CSS")
            elif not isinstance(node, (ast.Import, ast.ImportFrom, ast.Expr)):
                out.append(f"{rel(p)}:{node.lineno}: el archivo de localizadores solo contiene clases con constantes")

    # --- modules: through DriverWrapper, absolute imports, existing wrapper methods
    for p in files(apps, "modules", ".py"):
        src, tree = _py(p)
        out += _imports(rel(p), tree)
        if "DriverWrapper" not in src and re.search(r"\.(click|send_keys)\(", src):
            out.append(f"{rel(p)}:1: usa DriverWrapper para click/send_keys (from utils.driver_wrapper import DriverWrapper)")
        for attr, line in _calls_on(tree, "wrapper"):
            if attr not in WRAPPER_METHODS:
                out.append(f"{rel(p)}:{line}: DriverWrapper no tiene '{attr}()'; usa driver.page.locator(...) para eso")
        for attr, line in _page_calls(tree):
            if attr in NATIVE_BASICS:
                out.append(f"{rel(p)}:{line}: no uses page.{attr}() directo; usa DriverWrapper.{'send_keys' if attr != 'click' else 'click'}()")
        if SECRET.search(src):
            out.append(f"{rel(p)}:1: no pongas credenciales en el código; usa ENV: en el CSV")

    # --- tests: business steps, module calls, CSV data, no DOM logic
    test_files = files(tests, "", ".py")
    test_files = [p for p in test_files if os.path.basename(p).startswith("test_")]
    if not test_files:
        out.append(f"tests/{app}/: falta al menos un test_*.py")
    for p in test_files:
        src, tree = _py(p)
        out += _imports(rel(p), tree)
        for must, why in (("set_business_mode(True)", "activa el modo de negocio"), ('level="business"', "registra pasos con add_step(..., level=\"business\")"),
                          ("delete_all_cookies()", "limpia el estado al final con driver.delete_all_cookies()")):
            if must not in src:
                out.append(f"{rel(p)}:1: {why}")
        for attr, line in _page_calls(tree):
            out.append(f"{rel(p)}:{line}: los tests no llevan lógica DOM (page.{attr}); muévela a un módulo en apps/{app}/modules/")
        if not any(isinstance(n, ast.FunctionDef) and n.name.startswith("test_") and n.args.args and n.args.args[0].arg == "driver" for n in ast.walk(tree)):
            out.append(f"{rel(p)}:1: falta una función test_* con el parámetro 'driver'")

    # --- data + expected results
    er = os.path.join(apps, "data", "expected_result.csv")
    if not os.path.exists(er):
        out.append(f"apps/{app}/data/expected_result.csv: falta (cabecera test_id,expected_result)")
    else:
        with open(er, encoding="utf-8", newline="") as f:
            rows = list(csv.reader(f))
        if not rows or [c.strip() for c in rows[0]] != ["test_id", "expected_result"]:
            out.append(f"{rel(er)}:1: la cabecera debe ser test_id,expected_result")
        listed = {r[0] for r in rows[1:] if r}
        for p in test_files:
            m = re.search(r"CASO_ID\s*=\s*['\"]([^'\"]+)['\"]", _read(p))
            if m and m.group(1) not in listed:
                out.append(f"{rel(er)}: falta una fila para {m.group(1)}")
    for p in files(apps, "data", ".csv"):
        text = _read(p)
        if re.search(r"(?i)(password|contrase[ñn]a)", text.splitlines()[0] if text else "") and "ENV:" not in text:
            out.append(f"{rel(p)}:1: las contraseñas van como ENV:NOMBRE, nunca en claro")
    return out


def _imports(path, tree):
    return [f"{path}:{n.lineno}: usa importaciones absolutas desde la raíz (from apps... / from utils...)"
            for n in ast.walk(tree) if isinstance(n, ast.ImportFrom) and n.level]
