"""Playlist checks and value passing, with a fake runner (no browser). Run: python3 -m unittest test_playlist"""
import csv, os, shutil, sys, tempfile, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "support"))
import playlist  # noqa: E402
import harness  # noqa: E402

CASES = {
    "DWQ-101": {"id": "DWQ-101", "status": "ready"},
    "DWQ-102": {"id": "DWQ-102", "status": "ready", "depends_on": ["DWQ-101"]},
    "DWQ-103": {"id": "DWQ-103", "status": "ready"},
    "DWQ-900": {"id": "DWQ-900", "status": "no-data"},
}


def make_out(d):
    """A minimal generated folder: one CSV per case and the app's locators class."""
    os.makedirs(os.path.join(d, "apps", "CobroOrdenes", "data"))
    os.makedirs(os.path.join(d, "apps", "CobroOrdenes", "locators"))
    for token, cols in [("dwq_101", ["order_number"]), ("dwq_102", ["order_number", "reason"]), ("dwq_103", ["order_number", "api_key"])]:
        with open(os.path.join(d, "apps", "CobroOrdenes", "data", f"{token}_data.csv"), "w", newline="", encoding="utf-8") as f:
            csv.writer(f, lineterminator="\n").writerows([["caso_id", *cols], [token.upper().replace("_", "-"), *["x"] * len(cols)]])
    with open(os.path.join(d, "apps", "CobroOrdenes", "locators", "cobro_ordenes_locators.py"), "w", encoding="utf-8") as f:
        f.write('class CobroOrdenesLocators:\n    ORDER_NUMBER = "#orderNumber"\n')


class Base(unittest.TestCase):
    def setUp(self):
        self.out = tempfile.mkdtemp(prefix="pl_out_")
        make_out(self.out)

    def tearDown(self):
        shutil.rmtree(self.out, ignore_errors=True)

    def check(self, steps):
        return playlist.validate({"id": "p", "steps": steps}, CASES, self.out, playlist.load_selectors(self.out))


class Validate(Base):
    def test_a_valid_flow_has_no_errors(self):
        errs, warns = self.check([
            {"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}},
            {"case": "DWQ-102", "with": {"order_number": "$order_id", "reason": "Cliente duplicado"}},
        ])
        self.assertEqual(errs, [])
        self.assertEqual(warns, [])

    def test_unknown_or_unready_case_is_rejected(self):
        errs, _ = self.check([{"case": "DWQ-404"}, {"case": "DWQ-900"}])
        self.assertTrue(any("'DWQ-404' is not in the cases file" in e for e in errs), errs)
        self.assertTrue(any("DWQ-900 is no-data" in e for e in errs), errs)

    def test_consumer_before_producer_is_rejected(self):
        errs, _ = self.check([
            {"case": "DWQ-102", "with": {"order_number": "$order_id"}},
            {"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}},
        ])
        self.assertTrue(any("must be captured by an earlier step" in e for e in errs), errs)

    def test_same_step_cannot_consume_its_own_capture(self):
        errs, _ = self.check([{"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}, "with": {"order_number": "$order_id"}}])
        self.assertTrue(any("must be captured by an earlier step" in e for e in errs), errs)

    def test_with_column_must_exist_in_the_generated_data(self):
        errs, _ = self.check([{"case": "DWQ-101", "with": {"no_such_field": "x"}}])
        self.assertTrue(any("no data column 'no_such_field'" in e for e in errs), errs)

    def test_duplicate_capture_and_unknown_locator_are_rejected(self):
        errs, _ = self.check([
            {"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}},
            {"case": "DWQ-103", "captures": {"order_id": "NOPE"}},
        ])
        self.assertTrue(any("already produced in step 1" in e for e in errs), errs)
        self.assertTrue(any("no locator 'NOPE'" in e for e in errs), errs)

    def test_depends_on_must_run_earlier(self):
        errs, _ = self.check([{"case": "DWQ-102", "with": {"order_number": "x"}}, {"case": "DWQ-101"}])
        self.assertTrue(any("depends_on DWQ-101, which must run in an earlier step" in e for e in errs), errs)

    def test_unused_capture_is_only_a_warning(self):
        errs, warns = self.check([{"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}}])
        self.assertEqual(errs, [])
        self.assertEqual(warns, ["capture order_id is never used"])


class Run(Base):
    def fake(self, results, calls):
        """results: {token: result dict}. Records every call's overrides and captures."""
        def runner(out_dir, token, base_url, allow_writes=False, headed=False, login=None, overrides=None, captures=None, **kw):
            calls.append({"token": token, "overrides": overrides, "captures": captures})
            return dict({"status": "passed", "error": None, "captured": {}}, **results.get(token, {}))
        return runner

    def pl(self, steps, **extra):
        return dict({"id": "order-lifecycle", "steps": steps}, **extra)

    def test_a_captured_value_reaches_the_next_case(self):
        calls = []
        results = {"dwq_101": {"captured": {"order_id": "A-42"}}}
        r = playlist.run(self.pl([
            {"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}},
            {"case": "DWQ-102", "with": {"order_number": "$order_id", "reason": "Cliente duplicado"}},
        ]), CASES, self.out, "http://x", runner=self.fake(results, calls))
        self.assertEqual(r["status"], "passed", r)
        self.assertEqual(calls[0]["captures"], {"order_id": "#orderNumber"})  # the locator's selector, not its name
        self.assertEqual(calls[1]["overrides"], {"order_number": "A-42", "reason": "Cliente duplicado"})
        self.assertEqual(r["steps"][0]["captures"], {"order_id": "<redacted>"})

    def test_logged_capture_is_shown_and_env_values_are_read(self):
        calls = []
        os.environ["PLAYLIST_TEST_KEY"] = "k-123"
        try:
            r = playlist.run(self.pl([
                {"case": "DWQ-101", "captures": {"order_id": {"source": "ORDER_NUMBER", "log": True}}},
                {"case": "DWQ-103", "with": {"order_number": "$order_id", "api_key": "ENV:PLAYLIST_TEST_KEY"}},
            ]), CASES, self.out, "http://x", runner=self.fake({"dwq_101": {"captured": {"order_id": "A-42"}}}, calls))
        finally:
            del os.environ["PLAYLIST_TEST_KEY"]
        self.assertEqual(r["steps"][0]["captures"], {"order_id": "A-42"})
        self.assertEqual(calls[1]["overrides"], {"order_number": "A-42", "api_key": "k-123"})

    def test_a_failed_step_stops_the_flow(self):
        calls = []
        r = playlist.run(self.pl([{"case": "DWQ-101"}, {"case": "DWQ-103"}]), CASES, self.out, "http://x",
                         runner=self.fake({"dwq_101": {"status": "failed", "error": "AssertionError"}}, calls))
        self.assertEqual(r["status"], "failed")
        self.assertEqual([s["status"] for s in r["steps"]], ["failed", "skipped"])
        self.assertEqual([c["token"] for c in calls], ["dwq_101"])

    def test_without_stop_on_failure_a_missing_value_skips_only_its_step(self):
        calls = []
        r = playlist.run(self.pl([
            {"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}},
            {"case": "DWQ-102", "with": {"order_number": "$order_id"}},
            {"case": "DWQ-103"},
        ], stopOnFailure=False), CASES, self.out, "http://x",
            runner=self.fake({"dwq_101": {"status": "failed", "error": "boom"}}, calls))
        self.assertEqual([s["status"] for s in r["steps"]], ["failed", "skipped", "passed"])
        self.assertIn("$order_id was not captured", r["steps"][1]["error"])
        self.assertEqual([c["token"] for c in calls], ["dwq_101", "dwq_103"])

    def test_an_invalid_playlist_never_runs(self):
        calls = []
        r = playlist.run(self.pl([{"case": "DWQ-102", "with": {"order_number": "$x"}}]), CASES, self.out, "http://x",
                         runner=self.fake({}, calls))
        self.assertEqual(r["status"], "invalid")
        self.assertEqual(calls, [])


class OverrideRow(unittest.TestCase):
    def test_only_the_named_column_of_the_case_row_changes(self):
        d = tempfile.mkdtemp()
        try:
            make_out(d)
            harness._override_row(os.path.join(d, "apps"), "dwq_102", {"reason": "Cliente duplicado"})
            with open(os.path.join(d, "apps", "CobroOrdenes", "data", "dwq_102_data.csv"), encoding="utf-8") as f:
                rows = list(csv.reader(f))
            self.assertEqual(rows, [["caso_id", "order_number", "reason"], ["DWQ-102", "x", "Cliente duplicado"]])
        finally:
            shutil.rmtree(d, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
