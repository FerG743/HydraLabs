import os, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import runner  # noqa: E402


def run_with(pytest_out, test_text, code=0):
    import shutil
    repo = tempfile.mkdtemp()
    os.makedirs(os.path.join(repo, "tests", "PortalQA"))
    with open(os.path.join(repo, "tests", "PortalQA", "test_x.py"), "w", encoding="utf-8") as f:
        f.write(test_text)
    real = runner._pytest
    runner._pytest = lambda *a, **k: (code, pytest_out)
    try:
        return runner.run_test(repo, "PortalQA", "tests/PortalQA/test_x.py", "http://127.0.0.1:3000")
    finally:
        runner._pytest = real
        shutil.rmtree(repo)


class Verdict(unittest.TestCase):
    def test_a_passing_test_is_a_pass(self):
        self.assertEqual(runner.verdict_of(0, "1 passed in 4.28s"), "PASÓ")

    def test_a_test_that_only_skipped_is_not_a_pass(self):
        self.assertEqual(runner.verdict_of(0, "s\n1 skipped in 0.82s"), "FALLÓ")

    def test_a_partial_skip_is_not_a_pass(self):
        self.assertEqual(runner.verdict_of(0, "1 passed, 1 skipped in 2s"), "FALLÓ")

    def test_a_failure_is_a_failure(self):
        self.assertEqual(runner.verdict_of(1, "1 failed, 1 passed in 2s"), "FALLÓ")

    def test_no_summary_at_all_is_not_a_pass(self):
        self.assertEqual(runner.verdict_of(0, ""), "FALLÓ")

    def test_a_pending_expected_result_in_the_test_is_not_a_pass(self):
        out = run_with("1 passed in 1s", "def test_x():\n    pass\n    # PENDIENTE: resultado esperado 3 (sin automatizar)\n")
        self.assertTrue(out.startswith("FALLÓ"))
        self.assertIn("1 resultado(s)", out)

    def test_a_fully_automated_test_passes(self):
        self.assertTrue(run_with("1 passed in 1s", "def test_x():\n    assert True\n").startswith("PASÓ"))

    def test_a_repo_relative_test_path_is_read_from_the_repo(self):  # agent.py passes tests/<App>/test_x.py
        import shutil
        repo = tempfile.mkdtemp()
        os.makedirs(os.path.join(repo, "tests", "PortalQA"))
        with open(os.path.join(repo, "tests", "PortalQA", "test_x.py"), "w", encoding="utf-8") as f:
            f.write("# PENDIENTE: resultado esperado 2 (sin automatizar)\n")
        real = runner._pytest
        runner._pytest = lambda *a, **k: (0, "1 passed in 1s")
        try:
            out = runner.run_test(repo, "PortalQA", "tests/PortalQA/test_x.py", "http://127.0.0.1:3000")
        finally:
            runner._pytest = real
            shutil.rmtree(repo)
        self.assertTrue(out.startswith("FALLÓ"))
        self.assertIn("1 resultado(s)", out)

    def test_an_all_skipped_run_says_so(self):
        out = run_with("1 skipped in 0.82s", "def test_x():\n    assert True\n")
        self.assertTrue(out.startswith("FALLÓ"))
        self.assertIn("no se verificó nada", out)


if __name__ == "__main__":
    unittest.main()
