import copy, json, os, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "..", "tools", "jira"))
sys.path.insert(0, os.path.join(HERE, "..", "tools", "jira", "support"))
import agent, spec as specmod, test_spec_render as T  # noqa: E402

CRAWL = {"pages": [{"url": "http://h/cobro_ordenes", "elements": [
    {"role": "combobox", "tag": "button", "name": "Ingresa la tienda", "locator": "#name", "stable": True},
    {"role": "button", "tag": "button", "name": "Agregar", "locator": "role=button[name=\"Agregar\"]", "stable": True}]}]}


def crawl_file():
    fd, path = tempfile.mkstemp(suffix=".json")
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        json.dump(CRAWL, f)
    return path


class Lookup(unittest.TestCase):
    def test_a_known_element_comes_back_with_its_role_and_actions(self):
        out = agent.lookup_element(crawl_file(), "Ingresa la tienda")
        self.assertIn("rol: combobox", out)
        self.assertIn("select (widget: combobox)", out)

    def test_a_selector_finds_the_same_element(self):
        self.assertIn("Agregar", agent.lookup_element(crawl_file(), "role=button[name=\"Agregar\"]"))

    def test_an_unknown_element_says_so_instead_of_inventing_one(self):
        self.assertIn("no está en el mapa", agent.lookup_element(crawl_file(), "Boton fantasma"))


class SelectOnTheRightControl(unittest.TestCase):
    def setUp(self):
        self.case = T.case("DWQ-130")
        self.kinds = {"#name": "combobox", "role=button[name=\"Agregar\"]": "button"}

    def spec_with_select(self, locator_selector, widget=None):
        s = copy.deepcopy(T.golden("DWQ-130"))
        s["locators"]["STORE_COMBO"] = locator_selector
        do = {"action": "select", "locator": "STORE_COMBO", "data": "order_number"}
        if widget:
            do["widget"] = widget
        s["steps"][2]["do"] = do
        return s

    def test_a_select_on_a_button_is_refused_with_the_fix(self):
        out = "\n".join(specmod.validate(self.spec_with_select("#name"), self.case, kinds={"#name": "button"}))
        self.assertIn("es un button en la app, no un <select>", out)

    def test_a_select_on_a_combobox_without_the_widget_is_refused(self):
        out = "\n".join(specmod.validate(self.spec_with_select("#name"), self.case, kinds=self.kinds))
        self.assertIn("widget: \"combobox\"", out)

    def test_a_select_on_a_combobox_with_the_widget_passes_that_check(self):
        out = "\n".join(specmod.validate(self.spec_with_select("#name", widget="combobox"), self.case, kinds=self.kinds))
        self.assertNotIn("en la app, no un <select>", out)


if __name__ == "__main__":
    unittest.main()


class LiveRoleWhenTheCrawlHasNone(unittest.TestCase):
    def test_a_select_on_a_control_the_page_calls_a_button_is_refused_even_without_a_crawl(self):
        import runner
        s = copy.deepcopy(T.golden("DWQ-130"))
        s["locators"]["STORE_COMBO"] = "#name"
        s["steps"][2]["do"] = {"action": "select", "locator": "STORE_COMBO", "data": "order_number"}
        state = {"case": T.case("DWQ-130"), "crawl": "", "base_url": "http://x"}
        real = runner.probe
        runner.probe = lambda url, sels: {sel: (1, "button") for sel in sels}
        try:
            out = agent.submit_spec(s, tempfile.mkdtemp(), "PortalQA", state)
        finally:
            runner.probe = real
        self.assertIn("es un button en la app, no un <select>", out)
