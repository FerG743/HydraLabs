import json, os, subprocess, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
DECIDE = os.path.join(HERE, "decide.py")
sys.path.insert(0, HERE)
import crawl  # noqa: E402


def files():
    d = tempfile.mkdtemp()
    m, w = os.path.join(d, "map.json"), os.path.join(d, "writes.json")
    json.dump({"pages": [], "decisions": [{"method": "POST", "host": "127.0.0.1", "path": "/uploadFile", "status": "blocked",
                                           "choice": "keep_blocked", "example": "x", "options": ["allow", "stub", "keep_blocked"]}]},
              open(m, "w", encoding="utf-8"))
    return m, w


def run(*args):
    return subprocess.run([sys.executable, DECIDE, *args], capture_output=True, text=True)


class Decide(unittest.TestCase):
    def test_listing_shows_each_endpoint_and_its_choice(self):
        m, _ = files()
        self.assertIn("POST   /uploadFile", run(m).stdout)

    def test_allow_records_the_choice_and_grants_only_that_endpoint(self):
        m, w = files()
        r = run(m, "--writes", w, "--set", "POST /uploadFile", "allow")
        self.assertEqual(r.returncode, 0, r.stderr)
        self.assertEqual(json.load(open(m))["decisions"][0]["choice"], "allow")
        cfg = crawl.load_writes(w)
        self.assertTrue(cfg["allow_writes"])
        self.assertEqual((cfg["hosts"], cfg["endpoints"]), (["127.0.0.1"], ["POST /uploadFile"]))
        self.assertTrue(crawl.may_write("http://127.0.0.1:8000/uploadFile", False, True, cfg["hosts"], cfg["endpoints"], "POST"))
        self.assertFalse(crawl.may_write("http://127.0.0.1:8000/other", False, True, cfg["hosts"], cfg["endpoints"], "POST"))

    def test_stub_and_keep_blocked_grant_nothing(self):
        m, w = files()
        self.assertEqual(run(m, "--set", "POST /uploadFile", "stub").returncode, 0)
        self.assertFalse(os.path.exists(w))
        self.assertEqual(json.load(open(m))["decisions"][0]["choice"], "stub")

    def test_an_unknown_endpoint_or_choice_is_refused(self):
        m, w = files()
        self.assertNotEqual(run(m, "--writes", w, "--set", "POST /nope", "allow").returncode, 0)
        self.assertNotEqual(run(m, "--set", "POST /uploadFile", "always").returncode, 0)


if __name__ == "__main__":
    unittest.main()
