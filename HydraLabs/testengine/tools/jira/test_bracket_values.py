"""A value written as [NAME] is read from .env: the case's data CSV holds ENV:NAME, never the value itself."""
import os, sys, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import render_pytest as r  # noqa: E402


class BracketValues(unittest.TestCase):
    def test_brackets_become_an_env_pointer(self):
        case = {"id": "PQA-T9", "name": "Add", "precondition": "", "steps": [
            {"n": "1", "action": "Enter cuenta `[QA_CARD]`."}], "expected": []}
        self.assertEqual(r.analyse(case)["data"], {"cuenta": "ENV:QA_CARD"})

    def test_a_literal_value_is_still_written_as_it_is(self):
        case = {"id": "PQA-T8", "name": "Add", "precondition": "", "steps": [
            {"n": "1", "action": "Enter cuenta `12345`."}], "expected": []}
        self.assertEqual(r.analyse(case)["data"], {"cuenta": "12345"})


class FieldValue(unittest.TestCase):
    def test_the_field_shows_the_value_entered(self):
        import json, subprocess, tempfile
        with tempfile.TemporaryDirectory() as d:
            cases, out = os.path.join(d, "cases"), os.path.join(d, "out")
            os.makedirs(cases)
            with open(os.path.join(cases, "PQA-T3.case.json"), "w", encoding="utf-8") as f:
                json.dump({"id": "PQA-T3", "name": "Account is shown", "precondition": "",
                           "steps": [{"n": "1", "action": "Enter cuenta `[QA_CARD]`."}],
                           "expected": [{"text": 'The "Ingresa la cuenta" field shows the value entered.'}]}, f)
            subprocess.run([sys.executable, os.path.join(HERE, "render_pytest.py"), "--cases", cases,
                            "--app", "PortalQAdev", "--out", out], check=True, capture_output=True)
            module = open(os.path.join(out, "apps", "PortalQAdev", "modules", "portal_qadev_module.py"), encoding="utf-8").read()
            test = open(os.path.join(out, "tests", "PortalQAdev", "test_pqa_t3.py"), encoding="utf-8").read()
            self.assertIn("to_have_value(valor)", module)
            self.assertIn("verificar_valor_", test)
            self.assertIn("(driver, cuenta)", test)


if __name__ == "__main__":
    unittest.main()


class DropdownValue(unittest.TestCase):
    def test_the_dropdown_shows_the_option_chosen(self):
        import json, subprocess, tempfile
        with tempfile.TemporaryDirectory() as d:
            cases, out = os.path.join(d, "cases"), os.path.join(d, "out")
            os.makedirs(cases)
            with open(os.path.join(cases, "PQA-T4.case.json"), "w", encoding="utf-8") as f:
                json.dump({"id": "PQA-T4", "name": "Action is shown", "precondition": "",
                           "steps": [{"n": "1", "action": "Select Selecciona la acción `DESBLOQUEO`."}],
                           "expected": [{"text": 'The "Selecciona la acción" dropdown shows the option chosen.'}]}, f)
            subprocess.run([sys.executable, os.path.join(HERE, "render_pytest.py"), "--cases", cases,
                            "--app", "PortalQAdev", "--out", out], check=True, capture_output=True)
            module = open(os.path.join(out, "apps", "PortalQAdev", "modules", "portal_qadev_module.py"), encoding="utf-8").read()
            test = open(os.path.join(out, "tests", "PortalQAdev", "test_pqa_t4.py"), encoding="utf-8").read()
            self.assertIn("e.options[e.selectedIndex].text", module)
            self.assertIn("(driver, selecciona_la_accion)", test)

