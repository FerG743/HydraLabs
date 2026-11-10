"""Runs a GENERATED test against a real URL without needing the framework repo: stubs only the framework's
config/utils (same call shapes as the real DriverWrapper/add_step/cargar_datos_csv) and hands the test a real
Playwright page. By default a safety net aborts every non-GET request the test did not deliberately mock
(registered first = lowest priority, so the test's own RequestMock wins when its pattern matches)."""
import importlib.util, os, shutil, sys, tempfile, traceback
from urllib.parse import urlparse

STUBS = {
    "config/__init__.py": "", "utils/__init__.py": "",
    "config/settings.py": "class Config:\n    BASE_URL = ''\n    PLAYWRIGHT_TIMEOUT = 8\n",
    "utils/templete_report.py": ("STEPS = []\ndef set_business_mode(v): pass\n"
                                 "def add_step(msg, status=None, level=None): STEPS.append((status, msg))\n"),
    "utils/driver_wrapper.py": ("class DriverWrapper:\n    def __init__(self, driver): self.driver, self.page = driver, driver.page\n"
                                "    def click(self, selector, description=None): self.page.click(selector)\n"
                                "    def send_keys(self, selector, text, description=None, is_secret=False): self.page.fill(selector, text)\n"),
}


LOOPBACK = {"127.0.0.1", "localhost", "::1"}


def write_allowed(method, host, allow_writes, extra_hosts=()):
    """Reads always pass. A write passes only with allow_writes AND only to a loopback host (or one named in
    HYDRA_WRITE_HOSTS): a local frontend started in PROD mode would otherwise send real writes to production."""
    return method in ("GET", "HEAD") or (allow_writes and (host in LOOPBACK or host in extra_hosts))


class Driver:  # the surface the framework's `driver` fixture exposes
    def __init__(self, page):
        self.page = page

    def get(self, url):
        self.page.goto(url)

    def delete_all_cookies(self):
        self.page.context.clear_cookies()


# the framework's own sign-in, so the verify run signs in exactly as the delivered tests will
_FRAMEWORK_UTILS = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "..", "templates", "framework", "utils")
_LOGIN_PY = os.path.join(_FRAMEWORK_UTILS, "login.py")
# the framework's own CSV reader (ENV:NAME values from the environment), not a copy that could read data differently
with open(os.path.join(_FRAMEWORK_UTILS, "csv_reader.py"), encoding="utf-8") as _f:
    STUBS["utils/csv_reader.py"] = _f.read()


def _sign_in(page, base_url, login):
    spec = importlib.util.spec_from_file_location("hydra_login", _LOGIN_PY)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    mod.sign_in(page, base_url, login)


def run(out_dir, token, base_url, allow_writes=False, headed=False, shots_dir=None, slow_ms=0, login=None,
        overrides=None, captures=None):
    """overrides: {column: value} written into this case's data row (a playlist's `with`).
    captures: {name: css selector} read from the page after the test passes; result["captured"] holds them."""
    from playwright.sync_api import sync_playwright
    work = tempfile.mkdtemp(prefix="hydra_run_")
    shutil.copytree(out_dir, work, dirs_exist_ok=True)
    if overrides:
        _override_row(os.path.join(work, "apps"), token, overrides)
    for rel, src in STUBS.items():
        path = os.path.join(work, rel)
        if not os.path.exists(path):  # never clobber generated utils (test_files, network_mock)
            os.makedirs(os.path.dirname(path), exist_ok=True)
            with open(path, "w") as f:
                f.write(src)
    for m in [m for m in sys.modules if m.split(".")[0] in ("apps", "utils", "config")]:
        del sys.modules[m]
    sys.path.insert(0, work)
    result = {"test": token, "status": None, "error": None, "aborted_writes": [], "steps": [], "captured": {}}
    try:
        import config.settings as settings
        settings.Config.BASE_URL = base_url
        try:  # generated code that does not even import is a failed run, reported like any other
            spec = importlib.util.spec_from_file_location(f"gen_{token}", os.path.join(work, "tests", _tests_dir(work), f"test_{token}.py"))
            mod = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(mod)
            fn = next(getattr(mod, n) for n in dir(mod) if n.startswith("test_"))
        except Exception as e:
            result["status"], result["error"] = "failed", f"the generated test does not load: {type(e).__name__}: {str(e).strip()[:400]}"
            return result
        from _pytest.outcomes import Skipped
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=not headed, slow_mo=slow_ms)
            page = browser.new_context(viewport={"width": 1280, "height": 900}).new_page()
            extra = [h for h in os.environ.get("HYDRA_WRITE_HOSTS", "").split(",") if h]

            def net(route, request):  # registered first = lowest priority, so a test's own RequestMock wins
                if write_allowed(request.method, urlparse(request.url).hostname, allow_writes, extra):
                    route.continue_()
                else:
                    result["aborted_writes"].append(f"{request.method} {request.url}")
                    route.abort()
            try:
                if login and not any(m.name == "signed_out" for m in getattr(fn, "pytestmark", [])):
                    _sign_in(page, base_url, login)  # before the guard: signing in is the one write a read-only run makes
                page.route("**/*", net)
                fn(Driver(page))
                for name, selector in (captures or {}).items():  # the end state of this case, read before the browser closes
                    try:
                        result["captured"][name] = page.locator(selector).first.evaluate(
                            "e => 'value' in e ? e.value : e.innerText", timeout=8000)  # the stub Config's PLAYWRIGHT_TIMEOUT
                    except Exception as e:
                        raise RuntimeError(f"capture {name}: could not read {selector}: {str(e).strip()[:200]}")
                result["status"] = "passed"
            except Skipped as e:
                result["status"], result["error"] = "skipped", str(e)
            except Exception as e:  # AssertionError, playwright timeouts, ...
                result["status"], result["error"] = "failed", f"{type(e).__name__}: {str(e).strip()[:600]}"
                result["trace"] = "".join(traceback.format_exception_only(type(e), e))[:300]
            finally:
                if shots_dir:  # the final state, pass or fail: evidence for a person and for the live view
                    try:
                        os.makedirs(shots_dir, exist_ok=True)
                        page.screenshot(path=os.path.join(shots_dir, f"{token}.png"), full_page=True)
                    except Exception:
                        pass
                import utils.templete_report as tr
                result["steps"] = list(getattr(tr, "STEPS", []))
                browser.close()
    finally:
        sys.path.remove(work)
        shutil.rmtree(work, ignore_errors=True)
    return result


def _override_row(apps_dir, token, overrides):
    """Rewrite the one data row of this case's CSV (in the work copy only). Columns come from the generated header."""
    import csv, glob
    path = glob.glob(os.path.join(apps_dir, "*", "data", f"{token}_data.csv"))[0]
    with open(path, newline="", encoding="utf-8") as f:
        rows = list(csv.reader(f))
    header = rows[0]
    for col, value in overrides.items():
        if col not in header:
            raise KeyError(f"{token}: no data column {col!r} (columns: {', '.join(header[1:])})")
        rows[1][header.index(col)] = value
    with open(path, "w", newline="", encoding="utf-8") as f:
        csv.writer(f, lineterminator="\n").writerows(rows)


def _tests_dir(work):
    base = os.path.join(work, "tests")
    return os.listdir(base)[0]
