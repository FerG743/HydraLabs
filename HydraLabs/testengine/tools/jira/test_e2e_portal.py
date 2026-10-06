"""Drives the GENERATED TC-3 test in a real Chromium against a tiny fake portal (not your real one).
Proves the mechanics end to end: mocked 500 after a delay, request counter, binary/junk uploads, placed
assertions - and that the double-charge check really fails on a portal that lacks the double-submit guard."""
import http.server, importlib.util, json, os, socketserver, sys, tempfile, threading, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import jira2case  # noqa: E402

try:
    from playwright.sync_api import sync_playwright
except ImportError:  # pragma: no cover
    sync_playwright = None

PORTAL = """<!doctype html><title>Cobro</title><h1>Cobro de ordenes</h1>
<input type=file id=f><button id=go disabled>Subir y Procesar</button>
<div id=spin role=progressbar hidden>...</div><div id=msg role=alert hidden></div>
<script>
const f = document.getElementById('f'), go = document.getElementById('go');
let pending = false;
f.addEventListener('change', () => { const x = f.files[0]; go.disabled = !(x && x.name.toLowerCase().endsWith('.csv')); });
go.addEventListener('click', async () => {
  if (%(guard)s && pending) return;           // the double-submit guard under test
  pending = true; if (%(guard)s) go.disabled = true; spin.hidden = false;
  try { const r = await fetch('/api/ordenes', {method: 'POST', body: f.files[0]}); if (!r.ok) throw new Error(r.status); }
  catch (e) { msg.textContent = 'Error al procesar'; msg.hidden = false; }
  finally { pending = false; spin.hidden = true; go.disabled = false; }
});
</script>"""

STUBS = {
    "config/__init__.py": "", "utils/__init__.py": "",
    "config/settings.py": "class Config:\n    BASE_URL = ''\n    PLAYWRIGHT_TIMEOUT = 8\n",
    "utils/templete_report.py": "def set_business_mode(v): pass\ndef add_step(msg, status=None, level=None): pass\n",
    "utils/csv_reader.py": ("import csv\ndef cargar_datos_csv(path):\n    with open(path, newline='', encoding='utf-8') as f:\n"
                            "        return [tuple(r) for r in list(csv.reader(f))[1:]]\n"),
    "utils/driver_wrapper.py": ("class DriverWrapper:\n    def __init__(self, driver): self.driver, self.page = driver, driver.page\n"
                                "    def click(self, selector, description=None): self.page.click(selector)\n"
                                "    def send_keys(self, selector, text, description=None, is_secret=False): self.page.fill(selector, text)\n"),
}


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        guarded = not self.path.startswith("/buggy")
        body = (PORTAL % {"guard": "true" if guarded else "false"}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass


class Driver:  # the same surface the framework's fixture exposes
    def __init__(self, page):
        self.page = page

    def get(self, url):
        self.page.goto(url)

    def delete_all_cookies(self):
        self.page.context.clear_cookies()


@unittest.skipUnless(sync_playwright, "playwright not installed")
class PortalE2E(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        cases = os.path.join(cls.tmp.name, "cases")
        os.makedirs(cases)
        names = {"DWQ-130": "TC-1 – Manual", "DWQ-131": "TC-2 – Bulk", "DWQ-132": "TC-3 – Invalid input (Negative)"}
        for key, name in names.items():
            with open(os.path.join(HERE, "fixtures", "rendered", key + ".html"), encoding="utf-8") as f:
                case = jira2case.convert(f.read(), key, name, "en", "DWQ-129", "Medium")
            with open(os.path.join(cases, key + ".case.json"), "w", encoding="utf-8") as f:
                json.dump(case, f)
        cls.out = os.path.join(cls.tmp.name, "out")
        import subprocess
        subprocess.check_call([sys.executable, os.path.join(HERE, "render_pytest.py"), "--cases", cases, "--app", "CobroOrdenes",
                               "--out", cls.out], stdout=subprocess.DEVNULL)
        for rel, src in STUBS.items():
            p = os.path.join(cls.out, rel)
            os.makedirs(os.path.dirname(p), exist_ok=True)
            with open(p, "w") as f:
                f.write(src)
        sys.path.insert(0, cls.out)
        cls.server = socketserver.TCPServer(("127.0.0.1", 0), Handler)
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()
        cls.pw = sync_playwright().start()
        try:
            cls.browser = cls.pw.chromium.launch(headless=True)
        except Exception as e:  # no browser installed
            cls.pw.stop()
            raise unittest.SkipTest(f"chromium unavailable: {e}")

    @classmethod
    def tearDownClass(cls):
        cls.browser.close()
        cls.pw.stop()
        cls.server.shutdown()
        cls.server.server_close()
        sys.path.remove(cls.out)
        for m in [m for m in sys.modules if m.split(".")[0] in ("apps", "utils", "config")]:
            del sys.modules[m]
        cls.tmp.cleanup()

    def run_tc3(self, base_path):
        import importlib
        settings = importlib.import_module("config.settings")
        settings.Config.BASE_URL = f"http://127.0.0.1:{self.server.server_address[1]}{base_path}"
        spec = importlib.util.spec_from_file_location("gen_tc3", os.path.join(self.out, "tests", "CobroOrdenes", "test_dwq_132.py"))
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        fn = next(getattr(mod, n) for n in dir(mod) if n.startswith("test_"))
        page = self.browser.new_page()
        try:
            return fn(Driver(page))
        finally:
            page.close()

    def test_guarded_portal_passes_the_generated_tc3(self):
        self.assertIsNone(self.run_tc3(""))

    def test_portal_without_the_guard_fails_the_double_charge_check(self):
        with self.assertRaises(AssertionError) as cm:
            self.run_tc3("/buggy")
        self.assertIn("posible doble cobro", str(cm.exception))
        self.assertIn("2 peticiones POST", str(cm.exception))


if __name__ == "__main__":
    unittest.main()
