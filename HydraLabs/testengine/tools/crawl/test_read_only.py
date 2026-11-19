import json, os, sys, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import crawl  # noqa: E402


class FakeRoute:
    def __init__(self):
        self.calls = []

    def continue_(self):
        self.calls.append("continue")

    def abort(self):
        self.calls.append("abort")


class FakeRequest:
    def __init__(self, method, url):
        self.method, self.url = method, url


class Decisions(unittest.TestCase):
    def test_each_write_endpoint_is_listed_once_with_its_status_and_default_choice(self):
        d = crawl.decisions_from(["blocked POST http://127.0.0.1:8000/uploadFile?x=1", "blocked POST http://127.0.0.1:8000/uploadFile"])
        self.assertEqual(len(d), 1)
        self.assertEqual((d[0]["method"], d[0]["host"], d[0]["path"], d[0]["status"], d[0]["choice"]),
                         ("POST", "127.0.0.1", "/uploadFile", "blocked", "keep_blocked"))
        self.assertEqual(d[0]["options"], ["allow", "stub", "keep_blocked"])

    def test_a_choice_the_team_already_made_is_kept_and_endpoints_from_earlier_runs_stay(self):
        old = [{"method": "POST", "host": "127.0.0.1", "path": "/uploadFile", "status": "blocked", "choice": "stub"},
               {"method": "POST", "host": "127.0.0.1", "path": "/login", "status": "sent", "choice": "allow"}]
        d = {(x["method"], x["path"]): x for x in crawl.decisions_from(["sent POST http://127.0.0.1:8000/uploadFile"], old)}
        self.assertEqual(d[("POST", "/uploadFile")]["choice"], "stub")
        self.assertEqual(d[("POST", "/uploadFile")]["status"], "sent")
        self.assertEqual(d[("POST", "/login")]["choice"], "allow")


class MayWrite(unittest.TestCase):
    def test_a_walked_write_to_a_host_not_listed_is_not_allowed(self):
        self.assertFalse(crawl.may_write("https://portal.example.com/x", False, True, ["127.0.0.1"]))

    def test_a_walked_write_to_a_listed_host_is_allowed_only_when_switched_on(self):
        self.assertTrue(crawl.may_write("http://127.0.0.1:8000/uploadFile", False, True, ["127.0.0.1"]))
        self.assertFalse(crawl.may_write("http://127.0.0.1:8000/uploadFile", False, False, ["127.0.0.1"]))

    def test_with_endpoints_named_only_those_writes_go_through(self):
        ok = crawl.may_write("http://127.0.0.1:8000/uploadFile", False, True, ["127.0.0.1"], ["POST /uploadFile"], "POST")
        other = crawl.may_write("http://127.0.0.1:8000/deleteAll", False, True, ["127.0.0.1"], ["POST /uploadFile"], "POST")
        self.assertEqual((ok, other), (True, False))

    def test_no_config_means_no_walked_writes_anywhere(self):
        self.assertFalse(crawl.may_write("http://127.0.0.1:8000/uploadFile", False, True, []))

    def test_no_config_file_is_read_only(self):
        self.assertEqual(crawl.load_writes(""), {"allow_writes": False, "hosts": [], "endpoints": []})

    def test_a_config_file_is_read(self):
        import tempfile
        fd, path = tempfile.mkstemp(suffix=".json")
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump({"allow_writes": True, "hosts": ["127.0.0.1", "qa.example"]}, f)
        self.assertEqual(crawl.load_writes(path), {"allow_writes": True, "hosts": ["127.0.0.1", "qa.example"], "endpoints": []})


class ReadOnlyRoute(unittest.TestCase):
    def run_one(self, method, url, allow=False):
        seen = []
        route = FakeRoute()
        crawl.read_only_route(seen, lambda url, method: allow)(route, FakeRequest(method, url))
        return route.calls, seen

    def test_reads_pass(self):
        self.assertEqual(self.run_one("GET", "http://h/x")[0], ["continue"])

    def test_a_post_is_aborted_and_recorded_with_its_url(self):
        calls, seen = self.run_one("POST", "http://h/uploadFile")
        self.assertEqual(calls, ["abort"])
        self.assertEqual(seen, ["blocked POST http://h/uploadFile"])

    def test_an_allowed_write_goes_through_and_is_recorded_as_sent(self):
        calls, seen = self.run_one("POST", "http://h/login", allow=True)
        self.assertEqual((calls, seen), (["continue"], ["sent POST http://h/login"]))


if __name__ == "__main__":
    unittest.main()
