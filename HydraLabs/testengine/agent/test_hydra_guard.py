import os, sys, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from hydra_guard import write_allowed  # noqa: E402


class PytestPluginGuard(unittest.TestCase):
    def test_reads_always_pass(self):
        self.assertTrue(write_allowed("GET", "172.22.64.228", False))

    def test_writes_blocked_unless_allowed(self):
        self.assertFalse(write_allowed("POST", "127.0.0.1", False))

    def test_loopback_is_not_writable_by_default_any_more(self):
        self.assertFalse(write_allowed("POST", "127.0.0.1", True))

    def test_allowed_writes_reach_only_the_listed_hosts(self):
        self.assertTrue(write_allowed("POST", "127.0.0.1", True, ["127.0.0.1"]))
        self.assertFalse(write_allowed("POST", "172.22.64.228", True, ["127.0.0.1"]))  # PROD, not listed
        self.assertTrue(write_allowed("POST", "qa.example", True, ["qa.example"]))

    def test_with_endpoints_named_only_those_writes_go_through(self):
        self.assertTrue(write_allowed("POST", "127.0.0.1", True, ["127.0.0.1"], "/uploadFile", ["POST /uploadFile"]))
        self.assertFalse(write_allowed("POST", "127.0.0.1", True, ["127.0.0.1"], "/other", ["POST /uploadFile"]))


if __name__ == "__main__":
    unittest.main()
