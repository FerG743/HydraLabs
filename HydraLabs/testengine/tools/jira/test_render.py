"""Renders the three DWQ cases to README-format pytest files, then EXECUTES the generated tests against a fake
browser with just enough of the real framework stubbed (same call shapes as utils/driver_wrapper.py)."""
import csv, importlib, importlib.util, json, os, subprocess, sys, tempfile, textwrap, unittest
from unittest import mock

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import jira2case  # noqa: E402

STUBS = {
    "config/__init__.py": "",
    "config/settings.py": "class Config:\n    BASE_URL = 'https://portal.test/'\n    PLAYWRIGHT_TIMEOUT = 5\n",
    "utils/__init__.py": "",
    "utils/templete_report.py": "STEPS = []\ndef set_business_mode(v): pass\ndef add_step(msg, status=None, level=None): STEPS.append(msg)\n",
    "utils/csv_reader.py": ("import csv\ndef cargar_datos_csv(path):\n"
                           "    with open(path, newline='', encoding='utf-8') as f:\n        return [tuple(r) for r in list(csv.reader(f))[1:]]\n"),
    "utils/driver_wrapper.py": textwrap.dedent("""
        class DriverWrapper:
            def __init__(self, driver): self.driver, self.page = driver, driver.page
            def click(self, selector, description=None): self.page.click(selector)
            def send_keys(self, selector, text, description=None, is_secret=False): self.page.fill(selector, text)
        """),
    "playwright/__init__.py": "",
    "playwright/sync_api.py": "from unittest.mock import MagicMock\nexpect = MagicMock()\n",
}


def case_from_fixture(key):
    with open(os.path.join(HERE, "fixtures", "rendered", key + ".html"), encoding="utf-8") as f:
        html = f.read()
    return jira2case.convert(html, key, f"{key} case", "en", "DWQ-129", "Medium")


class OnlyTest(unittest.TestCase):
    """DWQ-132's junk.csv step can only be mapped with default values that other cases (DWQ-130) supply. Rendering one
    case at a time (the router does) must still see them, or a deterministic step gets sent to the model."""

    def render(self, cases, only=None):
        with tempfile.TemporaryDirectory() as d:
            for k in cases:
                with open(os.path.join(d, k + ".case.json"), "w", encoding="utf-8") as f:
                    json.dump(case_from_fixture(k), f)
            out = os.path.join(d, "out")
            cmd = [sys.executable, os.path.join(HERE, "render_pytest.py"), "--cases", d, "--app", "App", "--out", out]
            r = subprocess.run(cmd + (["--only", only] if only else []), capture_output=True, text=True)
            self.assertEqual(r.returncode, 0, r.stderr)
            return json.loads(r.stdout), sorted(os.listdir(os.path.join(out, "tests", "App")))

    def test_only_renders_one_case_but_keeps_the_defaults_from_the_others(self):
        report, files = self.render(["DWQ-130", "DWQ-132"], only="DWQ-132")
        self.assertEqual(files, ["test_dwq_132.py"])
        self.assertEqual([r["case"] for r in report], ["DWQ-132"])
        self.assertEqual(report[0]["pending"], [])

    def test_alone_without_siblings_the_step_stays_pending(self):  # documents WHY the router passes every stored case
        report, _ = self.render(["DWQ-132"])
        self.assertTrue(report[0]["pending"])


class RenderTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.tmp = tempfile.TemporaryDirectory()
        root = cls.tmp.name
        cases = os.path.join(root, "cases")
        os.makedirs(cases)
        for key in ("DWQ-130", "DWQ-131", "DWQ-132"):
            c = case_from_fixture(key)
            c["name"] = {"DWQ-130": "TC-1 – Manual order entry (Happy Path)", "DWQ-131": "TC-2 – Bulk CSV upload (Happy Path)",
                         "DWQ-132": "TC-3 – Invalid input (Negative)"}[key]
            with open(os.path.join(cases, key + ".case.json"), "w", encoding="utf-8") as f:
                json.dump(c, f)
        cls.out = os.path.join(root, "out")
        cls.report = json.loads(subprocess.check_output(
            [sys.executable, os.path.join(HERE, "render_pytest.py"), "--cases", cases, "--app", "CobroOrdenes", "--out", cls.out]))
        for rel, src in STUBS.items():  # stub framework next to the generated tree
            p = os.path.join(cls.out, rel)
            os.makedirs(os.path.dirname(p), exist_ok=True)
            with open(p, "w") as f:
                f.write(src)
        sys.path.insert(0, cls.out)
        importlib.invalidate_caches()

    @classmethod
    def tearDownClass(cls):
        sys.path.remove(cls.out)
        for m in [m for m in sys.modules if m.split(".")[0] in ("apps", "utils", "config", "playwright", "tests")]:
            del sys.modules[m]
        cls.tmp.cleanup()

    def run_test(self, token, driver):
        mod = self._load(token)
        fn = next(getattr(mod, n) for n in dir(mod) if n.startswith("test_"))
        return fn(driver)

    def _load(self, token):
        path = os.path.join(self.out, "tests", "CobroOrdenes", f"test_{token}.py")
        spec = importlib.util.spec_from_file_location(f"gen_{token}", path)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod

    def test_report_and_layout(self):
        by = {r["case"]: r for r in self.report}
        self.assertEqual((by["DWQ-130"]["steps_mapped"], by["DWQ-130"]["pending"]), (6, []))
        self.assertEqual(by["DWQ-131"]["steps_mapped"], 5)
        self.assertEqual((by["DWQ-132"]["steps_mapped"], by["DWQ-132"]["pending"]), (5, []))
        self.assertEqual((by["DWQ-132"]["expected_asserted"], by["DWQ-132"]["expected"]), (5, 5))
        for rel in ("apps/CobroOrdenes/data/dwq_130_data.csv", "apps/CobroOrdenes/data/expected_result.csv",
                    "apps/CobroOrdenes/locators/cobro_ordenes_locators.py", "apps/CobroOrdenes/modules/cobro_ordenes_module.py",
                    "tests/CobroOrdenes/test_dwq_130.py", "utils/test_files.py", "utils/network_mock.py"):
            self.assertTrue(os.path.exists(os.path.join(self.out, rel)), rel)
        with open(os.path.join(self.out, "apps/CobroOrdenes/data/dwq_130_data.csv")) as f:
            rows = list(csv.reader(f))
        self.assertEqual(rows, [["caso_id", "order_number", "store"], ["DWQ-130", "9800074297", "LIVR"]])

    def test_dwq_130_executes_the_steps_in_order(self):
        driver = mock.MagicMock()
        self.run_test("dwq_130", driver)
        driver.get.assert_called_once_with("https://portal.test/cobro_ordenes")
        page = driver.page
        page.fill.assert_called_once_with('role=textbox[name="order number" i]', "9800074297")
        page.select_option.assert_called_once_with('role=combobox[name="store" i]', value="LIVR")
        self.assertEqual([c.args[0] for c in page.click.call_args_list],
                         ['role=button[name="Agregar"]', 'role=button[name="Procesar"]'])
        page.locator.assert_any_call('role=button[name="Procesar"]')  # the is_disabled check on step 2
        driver.delete_all_cookies.assert_called_once()

    def test_dwq_131_downloads_then_uploads_that_file(self):
        driver = mock.MagicMock()
        dl = driver.page.expect_download.return_value.__enter__.return_value.value
        dl.suggested_filename = "sample.csv"
        dl.save_as.side_effect = lambda p: open(p, "w").close()
        self.run_test("dwq_131", driver)
        uploaded = driver.page.set_input_files.call_args.args
        self.assertEqual(uploaded[0], "input[type='file']")
        self.assertTrue(uploaded[1].endswith("sample.csv"))
        self.assertEqual(driver.page.click.call_args_list[-1].args[0], 'role=button[name="Subir y Procesar"]')

    def test_dwq_132_installs_the_mock_and_builds_the_bad_uploads(self):
        driver = mock.MagicMock()
        self.run_test("dwq_132", driver)
        page = driver.page
        self.assertEqual(page.route.call_args.args[0], "**/*")  # safe default: every POST is simulated until narrowed
        first, second = (c.args[1] for c in page.set_input_files.call_args_list)
        self.assertEqual((first["name"], first["mimeType"]), ("evil.csv.exe", "application/octet-stream"))
        self.assertIn(b"\x00", first["buffer"])  # binary: can never be a valid CSV
        self.assertEqual((second["name"], second["mimeType"]), ("junk.csv", "text/csv"))
        text = second["buffer"].decode()
        self.assertTrue(text.startswith("order_number,store\n,LIVR\n"))      # empty order number
        self.assertIn("9800074297,XXXX\n", text)                              # unknown store
        self.assertIn("9" * 10000 + ",LIVR", text)                             # 10,000-character order number
        self.assertIn("'; DROP TABLE x;--,LIVR", text)                         # SQL injection string
        page.dblclick.assert_called_once_with('role=button[name="Subir y Procesar"]')

    def test_confirmed_file_overrides_guesses_and_switches_the_combobox_flow(self):
        with tempfile.TemporaryDirectory() as out:
            os.makedirs(os.path.join(out, "apps", "CobroOrdenes"))
            with open(os.path.join(out, "apps", "CobroOrdenes", "locators.confirmed.json"), "w") as f:
                json.dump({"locators": {"INPUT_ORDER_NUMBER": "#orderNumber", "API_POST_PATRON": "**/uploadFile"},
                           "widgets": {"store": "combobox"}}, f)
            subprocess.check_call([sys.executable, os.path.join(HERE, "render_pytest.py"), "--cases", os.path.join(self.tmp.name, "cases"),
                                   "--app", "CobroOrdenes", "--out", out], stdout=subprocess.DEVNULL)
            loc = open(os.path.join(out, "apps/CobroOrdenes/locators/cobro_ordenes_locators.py"), encoding="utf-8").read()
            self.assertIn("INPUT_ORDER_NUMBER = '#orderNumber'  # confirmado en el portal real", loc)
            self.assertIn("API_POST_PATRON = '**/uploadFile'  # confirmado", loc)
            self.assertIn("SELECT_STORE = 'role=combobox[name=\"store\" i]'  # TODO", loc)  # not confirmed: still a flagged guess
            mod = open(os.path.join(out, "apps/CobroOrdenes/modules/cobro_ordenes_module.py"), encoding="utf-8").read()
            self.assertIn('get_by_role("option", name=store, exact=True).click()', mod)
            self.assertNotIn("select_option", mod)


if __name__ == "__main__":
    unittest.main()
