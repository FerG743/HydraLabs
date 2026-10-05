import json, os, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import gherkin_gen, jira2case, resolve_refs  # noqa: E402

LINT = os.path.join(HERE, "..", "..", "bin", "lint")


def mk(id, name, pre, ref=None):
    c = {"id": id, "name": name, "language": "en", "meta": {}, "precondition": pre,
         "steps": [{"n": "1", "action": "Do it."}], "expected": [{"text": "It works."}], "warnings": []}
    if ref:
        c["precondition_ref"] = ref
        c["warnings"].append(f"precondition refers to {ref}: resolve it from that case")
    return c


class Resolve(unittest.TestCase):
    def test_title_prefix_resolves_and_keeps_original(self):
        a, b = mk("A-1", "TC-1 – First", "Portal is up."), mk("A-2", "TC-2 – Second", "Same as TC-1.", "TC-1")
        self.assertEqual(resolve_refs.resolve([a, b]), {"A-2": "resolved from A-1"})
        self.assertEqual(b["precondition"], "Same as TC-1: Portal is up.")
        self.assertEqual(b["precondition_original"], "Same as TC-1.")
        self.assertEqual(b["warnings"], [])

    def test_does_not_confuse_tc1_with_tc10(self):
        cases = [mk("A-10", "TC-10 – Tenth", "Wrong one."), mk("A-2", "TC-2 – Second", "Same as TC-1.", "TC-1")]
        self.assertEqual(resolve_refs.resolve(cases)["A-2"], "not found")
        self.assertIn("precondition refers to TC-1: not found", cases[1]["warnings"][0])

    def test_ambiguous_reference_is_not_guessed(self):
        cases = [mk("A-1", "TC-1 – One", "x"), mk("B-1", "TC-1 – Also one", "y"), mk("A-2", "TC-2", "Same as TC-1.", "TC-1")]
        self.assertTrue(resolve_refs.resolve(cases)["A-2"].startswith("ambiguous: "))
        self.assertEqual(cases[2]["precondition"], "Same as TC-1.")

    def test_chain_resolves_transitively(self):
        cases = [mk("A-1", "TC-1 – One", "Base."), mk("A-2", "TC-2 – Two", "Same as TC-1.", "TC-1"),
                 mk("A-3", "TC-3 – Three", "Same as TC-2.", "TC-2")]
        resolve_refs.resolve(cases)
        self.assertEqual(cases[2]["precondition"], "Same as TC-2: Same as TC-1: Base.")

    def test_cycle_is_reported(self):
        cases = [mk("A-1", "TC-1 – One", "Same as TC-2.", "TC-2"), mk("A-2", "TC-2 – Two", "Same as TC-1.", "TC-1")]
        res = resolve_refs.resolve(cases)
        self.assertTrue(all(v.startswith("cycle") for v in res.values()), res)

    def test_rerun_is_stable_and_follows_target_edits(self):
        a, b = mk("A-1", "TC-1 – First", "Portal is up."), mk("A-2", "TC-2 – Second", "Same as TC-1.", "TC-1")
        resolve_refs.resolve([a, b])
        resolve_refs.resolve([a, b])
        self.assertEqual(b["precondition"], "Same as TC-1: Portal is up.")  # not "Same as TC-1: Same as TC-1: ..."
        a["precondition"] = "Portal is up. VPN on."
        resolve_refs.resolve([a, b])
        self.assertEqual(b["precondition"], "Same as TC-1: Portal is up. VPN on.")

    def test_generator_traces_resolution(self):
        a, b = mk("A-1", "TC-1 – First", "Portal is up."), mk("A-2", "TC-2 – Second", "Same as TC-1.", "TC-1")
        resolve_refs.resolve([a, b])
        f = gherkin_gen.generate(b)
        self.assertIn('# Precondition resolved from A-1 ("Same as TC-1.")', f)
        self.assertNotIn("NEEDS-INPUT", f)
        self.assertIn("Given Same as TC-1: Portal is up.", f)


class DirectoryPass(unittest.TestCase):
    def test_real_fixtures_move_dwq_131_to_clean(self):
        names = {"DWQ-130": "TC-1 – Manual order entry and processing (Happy Path)",
                 "DWQ-131": "TC-2 – Bulk CSV upload using the portal's sample file (Happy Path)",
                 "DWQ-132": "TC-3 – Invalid input and backend failure under duplicate submission (Negative / Resilience)"}
        with tempfile.TemporaryDirectory() as d:
            os.makedirs(os.path.join(d, "review"))
            for key, name in names.items():  # the layout the n8n workflow leaves behind
                with open(os.path.join(HERE, "fixtures", "rendered", key + ".html"), encoding="utf-8") as f:
                    c = jira2case.convert(f.read(), key, name, "en", "DWQ-129", "Medium")
                sub = os.path.join(d, "review") if key == "DWQ-131" else d
                stem = os.path.join(sub, key + "_case")
                with open(stem + ".case.json", "w", encoding="utf-8") as f:
                    json.dump(c, f, ensure_ascii=False)
                with open(stem + ".feature", "w", encoding="utf-8") as f:
                    f.write(gherkin_gen.generate(c))
                with open(stem + ".notes.txt", "w") as f:
                    f.write("old notes\n")
            untouched = open(os.path.join(d, "DWQ-130_case.feature"), encoding="utf-8").read()

            res = resolve_refs.run(d, LINT, "en")

            self.assertEqual(res["resolved"], {"DWQ-131": "resolved from DWQ-130"})
            self.assertEqual([m["clean"] for m in res["rewritten"]], [True])
            self.assertTrue(os.path.exists(os.path.join(d, "DWQ-131_case.feature")))      # moved out of review/
            self.assertFalse(os.path.exists(os.path.join(d, "review", "DWQ-131_case.feature")))
            feature = open(os.path.join(d, "DWQ-131_case.feature"), encoding="utf-8").read()
            self.assertIn("Given Same as TC-1: Portal is deployed and reachable.", feature)
            self.assertEqual(open(os.path.join(d, "DWQ-130_case.feature"), encoding="utf-8").read(), untouched)
            notes = open(os.path.join(d, "DWQ-131_case.notes.txt")).read()
            self.assertIn("resolved from DWQ-130", notes)
            self.assertNotIn("NEEDS-INPUT", notes)


if __name__ == "__main__":
    unittest.main()
