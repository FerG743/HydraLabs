import os, sys, tempfile, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import modindex  # noqa: E402


def write(path, body):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(body)


class ModIndexTest(unittest.TestCase):
    def setUp(self):
        self.repo = tempfile.mkdtemp()
        write(os.path.join(self.repo, "apps", "Portal", "locators", "portal_locators.py"),
              "class PortalLocators:\n    BOTON_AGREGAR = \"role=button[name='Agregar']\"\n    INPUT_CUENTA = '#cuenta'\n")
        write(os.path.join(self.repo, "apps", "Portal", "modules", "tarjetas_module.py"),
              "from apps.Portal.locators.portal_locators import PortalLocators as L\n\n"
              "def agregar_flujo(driver):\n    driver.get(Config.BASE_URL.rstrip('/') + '/Tarjetas')\n"
              "    driver.page.locator(L.INPUT_CUENTA).fill('1')\n    driver.page.locator(L.BOTON_AGREGAR).click()\n")

    def test_a_page_lists_the_selectors_its_functions_use(self):
        pages = modindex.page_locators(self.repo, "Portal")
        self.assertEqual(pages["/Tarjetas"], {"BOTON_AGREGAR": "role=button[name='Agregar']", "INPUT_CUENTA": "#cuenta"})

    def test_only_the_case_pages_are_sent(self):
        self.assertEqual(modindex.locator_lines(self.repo, "Portal", ["/Tarjetas"]),
                         {"/Tarjetas": ["BOTON_AGREGAR = role=button[name='Agregar']", "INPUT_CUENTA = #cuenta"]})
        self.assertEqual(modindex.locator_lines(self.repo, "Portal", ["/otra"]), {})

    def test_an_empty_repo_gives_nothing(self):
        self.assertEqual(modindex.page_locators(tempfile.mkdtemp(), "Portal"), {})


if __name__ == "__main__":
    unittest.main()
