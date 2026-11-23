"""The 3-passes gate: a state change is a candidate only if every run saw it."""
import os, sys, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import discover as d  # noqa: E402


class Transitions(unittest.TestCase):
    def test_a_change_is_caused_by_the_step_before_it(self):
        steps = [
            {"step": "Navigate", "controls": {"Procesar": {"enabled": False, "visible": True}}},
            {"step": "Click Agregar", "controls": {"Procesar": {"enabled": True, "visible": True}}},
        ]
        self.assertEqual(d.transitions(steps), [("Navigate", "Procesar", "enabled", False, True)])

    def test_only_changes_seen_in_every_run_are_kept(self):
        always = ("Click Agregar", "Procesar", "enabled", False, True)
        once = ("Click Procesar", "Agregar", "enabled", True, False)
        runs = [[always, once], [always], [always]]
        self.assertEqual(d.gate(runs, need=3), [always])
        self.assertEqual(d.gate([[always, once], [always, once], [always, once]], need=3), [always, once])


if __name__ == "__main__":
    unittest.main()


class ExplorationPlan(unittest.TestCase):
    def test_fills_fields_and_selects_options_and_never_clicks_a_button(self):
        import explore
        elements = [
            {"role": "textbox", "name": "Ingresa la cuenta"},
            {"role": "combobox", "name": "Selecciona la acción", "options": ["BLOQUEO", "DESBLOQUEO"]},
            {"role": "button", "name": "Agregar"},
            {"role": "button", "name": "Procesar"},
        ]
        acts = explore.plan(elements, {"Ingresa la cuenta": "4111"})
        self.assertEqual(acts, [("fill", "Ingresa la cuenta", "4111"),
                                ("select", "Selecciona la acción", "BLOQUEO"),
                                ("select", "Selecciona la acción", "DESBLOQUEO")])
        self.assertFalse(any(a[1] in ("Agregar", "Procesar") for a in acts))


class WriteGate(unittest.TestCase):
    def test_writes_only_on_local_or_labelled_test_hosts(self):
        import explore
        self.assertTrue(explore.writes_allowed("http://127.0.0.1:3000"))
        self.assertTrue(explore.writes_allowed("https://omsweb-qa-front.example.com/app"))
        self.assertTrue(explore.writes_allowed("https://portal.dev.example.com"))
        self.assertFalse(explore.writes_allowed("https://portal.example.com"))
        self.assertFalse(explore.writes_allowed("https://bank.example.com"))
