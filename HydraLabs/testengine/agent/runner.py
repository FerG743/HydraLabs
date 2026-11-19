"""Checks the agent can run on what it wrote: README structure, compile, import/collect, and a read-only test run."""
import json, os, py_compile, re, subprocess, sys

import structure_check

HERE = os.path.dirname(os.path.abspath(__file__))


def load_writes(path):
    """The owning team's writes config (same format as tools/crawl/crawl.py load_writes). No file: nothing is writable."""
    if not path:
        return {"allow_writes": False, "hosts": [], "endpoints": []}
    cfg = json.load(open(path, encoding="utf-8"))
    return {"allow_writes": bool(cfg.get("allow_writes")),
            "hosts": [h for h in cfg.get("hosts", []) if isinstance(h, str)],
            "endpoints": [e for e in cfg.get("endpoints", []) if isinstance(e, str)]}


def _pytest(repo, args, base_url, timeout, writes=None):
    w = writes or load_writes("")
    env = dict(os.environ, PYTHONPATH=os.pathsep.join([HERE, repo]), BASE_URL=base_url or os.environ.get("BASE_URL", ""),
               HEADLESS="true", PYTEST_DISABLE_PLUGIN_AUTOLOAD="",
               HYDRA_ALLOW_WRITES="1" if w["allow_writes"] and w["hosts"] else "0",
               HYDRA_WRITE_HOSTS=",".join(w["hosts"]), HYDRA_WRITE_ENDPOINTS=",".join(w["endpoints"]))
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


def verdict_of(code, out):
    """PASÓ only when pytest exits 0, at least one test passed, and nothing was skipped."""
    if code != 0:
        return "FALLÓ"
    if not re.search(r"\b\d+ passed\b", out) or re.search(r"\b\d+ skipped\b", out):
        return "FALLÓ"  # exit 0 with nothing checked, or with a skipped test
    return "PASÓ"


def pending_count(test_file):
    """Expected results or steps the renderer could not automate: written as '# PENDIENTE' comments, not skips."""
    with open(test_file, encoding="utf-8") as f:
        return sum(1 for line in f if "# PENDIENTE" in line)


def run_test(repo, app, test_file, base_url, writes=None):
    code, out = _pytest(repo, [f"tests/{app}/{os.path.basename(test_file)}", "-x"], base_url, 240, writes=writes)
    verdict = verdict_of(code, out)
    pending = pending_count(os.path.join(repo, "tests", app, os.path.basename(test_file)))
    if verdict == "PASÓ" and pending:
        verdict = "FALLÓ"
    note = ""
    if verdict == "FALLÓ" and code == 0:
        note = f"\nNo se verificó todo: {pending} resultado(s) o paso(s) sin automatizar (PENDIENTE) o sin ejecutar." if pending \
            else "\nNingún test pasó (todo se saltó o no se ejecutó): no se verificó nada."
    return f"{verdict} (código {code}){note}\n{out}"


def probe(url, selectors):
    """{selector: (matches, role)} on the real page: the role is the aria role, or the tag when there is none.
    GET only; nothing is clicked. A selector that fails gives (-1, None)."""
    from playwright.sync_api import sync_playwright
    out = {}
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        try:
            pg = browser.new_page()
            pg.route("**/*", lambda route, req: route.continue_() if req.method in ("GET", "HEAD") else route.abort())
            pg.goto(url, wait_until="networkidle", timeout=60000)
            for sel in selectors:
                try:
                    loc = pg.locator(sel)
                    n = loc.count()
                    role = loc.first.evaluate("e => e.getAttribute('role') || e.tagName.toLowerCase()") if n else None
                    out[sel] = (n, role)
                except Exception:
                    out[sel] = (-1, None)
        except Exception:
            pass
        finally:
            browser.close()
    return out


def count_selector(url, selector):
    """How many elements the selector matches on the real page. GET only; nothing is clicked. -1 if the page or selector fails."""
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        try:
            pg = browser.new_page()
            pg.route("**/*", lambda route, req: route.continue_() if req.method in ("GET", "HEAD") else route.abort())
            pg.goto(url, wait_until="networkidle", timeout=60000)
            return pg.locator(selector).count()
        except Exception:
            return -1
        finally:
            browser.close()


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
