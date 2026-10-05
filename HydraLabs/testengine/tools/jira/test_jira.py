import glob, json, os, subprocess, sys, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(os.path.dirname(HERE), "scorecard"))
import gherkin_gen, jira2case, scorecard  # noqa: E402

META = {  # what the Jira API reports outside the description
    "DWQ-130": ("TC-1 – Manual order entry and processing (Happy Path)", "Medium"),
    "DWQ-131": ("TC-2 – Bulk CSV upload using the portal's sample file (Happy Path)", "Medium"),
    "DWQ-132": ("TC-3 – Invalid input and backend failure under duplicate submission (Negative / Resilience)", "Medium"),
}


def load(key):
    html = open(os.path.join(HERE, "fixtures", key + ".html"), encoding="utf-8").read()
    name, prio = META[key]
    return jira2case.convert(html, key, name, "en", "DWQ-129", prio)


class JiraGen(unittest.TestCase):
    def test_matches_hand_written_cases(self):  # the adapter reproduces what a human read from the issue
        for key in META:
            want = json.load(open(os.path.join(HERE, "..", "scorecard", "cases", key + ".json"), encoding="utf-8"))
            got = load(key)
            self.assertEqual([s["action"] for s in got["steps"]], [s["action"] for s in want["steps"]], key)
            self.assertEqual(got["expected"], want["expected"], key)
            self.assertEqual(got["meta"].get("gate", "").split(".")[0], "Blocker", key)
            if key != "DWQ-131":  # TC-2's "Same as TC-1" was expanded by hand in the fixture
                self.assertEqual(got["precondition"], want["precondition"], key)

    def test_real_jira_rendering_matches_connector_html(self):  # <tt> code spans, auto-linked "TC-1" with a status lozenge
        for key in META:
            real = jira2case.convert(open(os.path.join(HERE, "fixtures", "rendered", key + ".html"), encoding="utf-8").read(),
                                     key, META[key][0], "en", "DWQ-129", "Medium")
            want = load(key)
            for f in ("steps", "expected", "precondition"):
                self.assertEqual(real[f], want[f], (key, f))
            self.assertEqual(real["meta"], want["meta"], key)

    def test_labels_priority_and_conflict(self):
        c = load("DWQ-132")
        self.assertEqual(c["meta"]["labels"], ["e2e", "negative", "resilience", "idempotency", "cobro-ordenes"])
        self.assertEqual(c["meta"]["priority"], "Highest")
        self.assertTrue(any("priority conflict" in w for w in c["warnings"]))  # description Highest vs Jira Medium

    def test_reference_precondition_is_flagged(self):
        c = load("DWQ-131")
        self.assertEqual(c["precondition_ref"], "TC-1")
        self.assertIn("NEEDS-INPUT: precondition refers to TC-1", gherkin_gen.generate(c))

    def test_generated_feature_is_strictly_faithful(self):  # same bar as the model scorecard
        for key in META:
            case = load(key)
            feature = gherkin_gen.generate(case)
            m = scorecard.score(case, feature)
            self.assertTrue(m["lint"], (key, m["notes"]["lint"]))
            self.assertEqual((m["steps"], m["expected"], m["order"], m["extra"], m["invented"], m["clean"]),
                             (1.0, 1.0, True, 0, 0, True), (key, m["notes"]))

    def test_missing_sections_become_needs_input(self):
        c = jira2case.convert("<ul><li><p><strong>Steps:</strong> Open the page</p></li></ul>", "X-1", "t")
        self.assertIn("no expected results found", c["warnings"])
        self.assertIn("NEEDS-INPUT: the case has no expected results", gherkin_gen.generate(c))

    def test_spanish_keywords(self):
        c = load("DWQ-130")
        c["language"] = "es"
        f = gherkin_gen.generate(c)
        for kw in ("Característica:", "Antecedentes:", "Escenario:", "Dado ", "Cuando ", "Entonces "):
            self.assertIn(kw, f)


if __name__ == "__main__":
    unittest.main()
