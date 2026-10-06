"""Checks the agent can run on what it wrote: README structure, compile, import/collect, and a read-only test run."""
import os, py_compile, subprocess, sys

import structure_check

HERE = os.path.dirname(os.path.abspath(__file__))


def _pytest(repo, args, base_url, timeout):
    env = dict(os.environ, PYTHONPATH=os.pathsep.join([HERE, repo]), BASE_URL=base_url or os.environ.get("BASE_URL", ""),
               HEADLESS="true", PYTEST_DISABLE_PLUGIN_AUTOLOAD="")
    r = subprocess.run([sys.executable, "-m", "pytest", "-p", "hydra_guard", "-q", "--no-header", "--tb=short", *args],
                       cwd=repo, env=env, capture_output=True, text=True, timeout=timeout)
    out = (r.stdout + r.stderr).strip().splitlines()
    return r.returncode, "\n".join(out[-40:])


def run_checks(repo, app, base_url=""):
    problems = structure_check.check(repo, app)
    for root in (os.path.join(repo, "apps", app), os.path.join(repo, "tests", app)):
        for dirpath, _, names in os.walk(root):
            for n in names:
                if n.endswith(".py"):
                    try:
                        py_compile.compile(os.path.join(dirpath, n), doraise=True)
                    except py_compile.PyCompileError as e:
                        problems.append(f"{os.path.relpath(os.path.join(dirpath, n), repo)}: error de sintaxis: {e.msg.strip()[:200]}")
    if not problems and os.path.isdir(os.path.join(repo, "tests", app)):
        code, out = _pytest(repo, ["--collect-only", f"tests/{app}"], base_url, 90)
        if code != 0:  # imports/fixtures do not resolve
            problems.append("pytest no pudo importar/recolectar los tests:\n" + out)
    return problems


def run_test(repo, app, test_file, base_url):
    code, out = _pytest(repo, [f"tests/{app}/{os.path.basename(test_file)}", "-x"], base_url, 240)
    verdict = "PASÓ" if code == 0 else "FALLÓ"
    return f"{verdict} (código {code})\n{out}"


def verify_locators(repo, app, page_stem, url, class_file=None):
    """Which static locators of apps/<App>/locators/<page>_locators.py resolve on the real page. GET only; nothing is clicked.
    Locators flagged '# dinámico' (they only exist after an action), API_* and CSV_* constants are skipped."""
    import render
    from playwright.sync_api import sync_playwright
    path = class_file or os.path.join(repo, "apps", app, "locators", f"{page_stem}_locators.py")
    locs = {n: v for n, (v, dyn) in render.read_locators(path).items()
            if not dyn and not n.startswith(("API_", "CSV_")) and isinstance(v, str)}
    if not locs:
        return "no hay localizadores estáticos que verificar"
    rows = []
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        pg = browser.new_page()
        pg.route("**/*", lambda route, req: route.continue_() if req.method in ("GET", "HEAD") else route.abort())
        pg.goto(url, wait_until="networkidle", timeout=60000)
        for name, sel in locs.items():
            try:
                n = pg.locator(sel).count()
            except Exception as e:  # invalid selector syntax
                rows.append(f"{name:28} SELECTOR INVÁLIDO: {str(e)[:80]}")
                continue
            rows.append(f"{name:28} {n} coincidencias" + ("" if n == 1 else "   <-- " + ("NO ENCONTRADO" if n == 0 else "AMBIGUO (usa uno más específico)")))
        browser.close()
    return "\n".join(rows)
