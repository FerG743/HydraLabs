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
    "utils/csv_reader.py": ("import csv\ndef cargar_datos_csv(path):\n    with open(path, newline='', encoding='utf-8') as f:\n"
                            "        return [tuple(r) for r in list(csv.reader(f))[1:]]\n"),
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


def run(out_dir, token, base_url, allow_writes=False, headed=False, shots_dir=None):
    from playwright.sync_api import sync_playwright
    work = tempfile.mkdtemp(prefix="hydra_run_")
    shutil.copytree(out_dir, work, dirs_exist_ok=True)
    for rel, src in STUBS.items():
        path = os.path.join(work, rel)
        if not os.path.exists(path):  # never clobber generated utils (test_files, network_mock)
            os.makedirs(os.path.dirname(path), exist_ok=True)
            with open(path, "w") as f:
                f.write(src)
    for m in [m for m in sys.modules if m.split(".")[0] in ("apps", "utils", "config")]:
        del sys.modules[m]
    sys.path.insert(0, work)
    result = {"test": token, "status": None, "error": None, "aborted_writes": [], "steps": []}
    try:
        import config.settings as settings
        settings.Config.BASE_URL = base_url
        spec = importlib.util.spec_from_file_location(f"gen_{token}", os.path.join(work, "tests", _tests_dir(work), f"test_{token}.py"))
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        fn = next(getattr(mod, n) for n in dir(mod) if n.startswith("test_"))
        from _pytest.outcomes import Skipped
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=not headed)
            page = browser.new_context(viewport={"width": 1280, "height": 900}).new_page()
            extra = [h for h in os.environ.get("HYDRA_WRITE_HOSTS", "").split(",") if h]

            def net(route, request):  # registered first = lowest priority, so a test's own RequestMock wins
                if write_allowed(request.method, urlparse(request.url).hostname, allow_writes, extra):
                    route.continue_()
                else:
                    result["aborted_writes"].append(f"{request.method} {request.url}")
                    route.abort()
            page.route("**/*", net)
            try:
                fn(Driver(page))
                result["status"] = "passed"
            except Skipped as e:
                result["status"], result["error"] = "skipped", str(e)
            except Exception as e:  # AssertionError, playwright timeouts, ...
                result["status"], result["error"] = "failed", f"{type(e).__name__}: {str(e).strip()[:600]}"
                result["trace"] = "".join(traceback.format_exception_only(type(e), e))[:300]
                if shots_dir:
                    os.makedirs(shots_dir, exist_ok=True)
                    page.screenshot(path=os.path.join(shots_dir, f"{token}.png"), full_page=True)
            finally:
                import utils.templete_report as tr
                result["steps"] = list(getattr(tr, "STEPS", []))
                browser.close()
    finally:
        sys.path.remove(work)
        shutil.rmtree(work, ignore_errors=True)
    return result


def _tests_dir(work):
    base = os.path.join(work, "tests")
    return os.listdir(base)[0]
