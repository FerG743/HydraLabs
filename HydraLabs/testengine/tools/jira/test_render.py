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
        self.assertEqual(len(by["DWQ-132"]["pending"]), 2)  # the two uploads have no rule: reported, not faked
        for rel in ("apps/CobroOrdenes/data/dwq_130_data.csv", "apps/CobroOrdenes/data/expected_result.csv",
                    "apps/CobroOrdenes/locators/cobro_ordenes_locators.py", "apps/CobroOrdenes/modules/cobro_ordenes_module.py",
                    "tests/CobroOrdenes/test_dwq_130.py"):
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

    def test_dwq_132_skips_when_steps_are_pending(self):
        from _pytest.outcomes import Skipped
        with self.assertRaises(Skipped) as cm:
            self.run_test("dwq_132", mock.MagicMock())
        self.assertIn("evil.csv.exe", str(cm.exception))


if __name__ == "__main__":
    unittest.main()
