import os, sys, tempfile, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import sandbox, structure_check  # noqa: E402

LOCATORS = '''class PortalLocators:
    BOTON_AGREGAR = "role=button[name='Agregar']"
    INPUT_ORDEN = "#orderNumber"
'''
MODULE = '''from utils.driver_wrapper import DriverWrapper
from apps.Portal.locators.portal_locators import PortalLocators
from utils.templete_report import add_step


def agregar_orden_flujo(driver, orden):
    wrapper = DriverWrapper(driver)
    wrapper.send_keys(PortalLocators.INPUT_ORDEN, orden, description="el campo de orden")
    add_step("Se ingresó la orden.", "Exitoso")
    wrapper.click(PortalLocators.BOTON_AGREGAR, description="el botón Agregar")
    assert driver.page.locator(PortalLocators.BOTON_AGREGAR).is_visible()
'''
TEST = '''import os
import pytest
from apps.Portal.modules.portal_module import agregar_orden_flujo
from utils.csv_reader import cargar_datos_csv
from config.settings import Config
from utils.templete_report import set_business_mode, add_step

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
DATA = cargar_datos_csv(os.path.join(ROOT_DIR, "apps", "Portal", "data", "dwq_130_data.csv"))
CASO_ID = "DWQ-130"
datos_caso = next((f for f in DATA if f[0] == CASO_ID), None)


def test_dwq_130_captura_manual(driver):
    if not datos_caso:
        pytest.skip("sin datos")
    _id, orden = datos_caso
    set_business_mode(True)
    add_step("Navegando al portal", level="business")
    driver.get(Config.BASE_URL)
    add_step("Agregando la orden", level="business")
    agregar_orden_flujo(driver, orden)
    driver.delete_all_cookies()
'''


def build(root, **override):
    files = {"apps/Portal/locators/portal_locators.py": LOCATORS, "apps/Portal/modules/portal_module.py": MODULE,
             "apps/Portal/data/dwq_130_data.csv": "caso_id,orden\nDWQ-130,9800074297\n",
             "apps/Portal/data/expected_result.csv": "test_id,expected_result\nDWQ-130,La orden aparece en la tabla.\n",
             "tests/Portal/test_dwq_130.py": TEST}
    files.update(override)
    for rel, text in files.items():
        if text is None:
            continue
        p = os.path.join(root, rel)
        os.makedirs(os.path.dirname(p), exist_ok=True)
        with open(p, "w", encoding="utf-8") as f:
            f.write(text)


class StructureCheck(unittest.TestCase):
    def violations(self, **override):
        with tempfile.TemporaryDirectory() as d:
            build(d, **override)
            return structure_check.check(d, "Portal")

    def test_readme_conformant_example_is_clean(self):
        self.assertEqual(self.violations(), [])

    def test_each_readme_rule_is_enforced(self):
        cases = {
            "lógica": ("apps/Portal/locators/portal_locators.py", LOCATORS + "    def x(self): pass\n"),
            "XPath": ("apps/Portal/locators/portal_locators.py", LOCATORS + '    RUTA = "/html/body/div[2]/button"\n'),
            "MAYÚSCULAS": ("apps/Portal/locators/portal_locators.py", LOCATORS + '    boton = "#b"\n'),
            "no tiene 'is_visible()'": ("apps/Portal/modules/portal_module.py", MODULE + "    DriverWrapper(driver).is_visible('#x')\n".replace("DriverWrapper(driver)", "wrapper")),
            "no uses page.click": ("apps/Portal/modules/portal_module.py", MODULE + "    driver.page.click('#x')\n"),
            "modo de negocio": ("tests/Portal/test_dwq_130.py", TEST.replace("set_business_mode(True)", "pass")),
            "lógica DOM": ("tests/Portal/test_dwq_130.py", TEST.replace("driver.get(Config.BASE_URL)", "driver.page.goto(Config.BASE_URL)")),
            "importaciones absolutas": ("apps/Portal/modules/portal_module.py", "from ..locators import portal_locators\n" + MODULE),
            "falta una fila para DWQ-130": ("apps/Portal/data/expected_result.csv", "test_id,expected_result\n"),
            "cabecera": ("apps/Portal/data/expected_result.csv", "id,resultado\nDWQ-130,x\n"),
            "credenciales": ("apps/Portal/modules/portal_module.py", MODULE + '    password = "Liverpool2026!"\n'),
        }
        for needle, (rel, text) in cases.items():
            msgs = "\n".join(self.violations(**{rel: text}))
            self.assertIn(needle, msgs, f"rule not enforced: {needle}\n{msgs}")

    def test_missing_files_are_reported(self):
        with tempfile.TemporaryDirectory() as d:
            msgs = "\n".join(structure_check.check(d, "Portal"))
        self.assertIn("falta al menos un test_*.py", msgs)
        self.assertIn("expected_result.csv: falta", msgs)


class Sandbox(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.sb = sandbox.Sandbox(self.tmp.name, "Portal")
        os.makedirs(os.path.join(self.tmp.name, "utils"))
        with open(os.path.join(self.tmp.name, "utils", "driver_wrapper.py"), "w") as f:
            f.write("class DriverWrapper: pass\n")

    def tearDown(self):
        self.tmp.cleanup()

    def test_reads_anywhere_inside_the_repo(self):
        self.assertIn("DriverWrapper", self.sb.read_file("utils/driver_wrapper.py"))
        self.assertIn("utils/driver_wrapper.py", self.sb.list_files("."))

    def test_has_no_write_capability_at_all(self):  # the model cannot write files: render.py does
        self.assertFalse(hasattr(self.sb, "write_file"))

    def test_path_traversal_and_symlinks_out_of_the_repo_are_blocked(self):
        with self.assertRaises(PermissionError):
            self.sb.read_file("../../etc/passwd")
        outside = tempfile.mkdtemp()
        os.symlink(outside, os.path.join(self.tmp.name, "escape"))
        with self.assertRaises(PermissionError):
            self.sb.list_files("escape")


if __name__ == "__main__":
    unittest.main()
