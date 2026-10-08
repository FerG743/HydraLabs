import os, sys, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from hydra_guard import write_allowed  # noqa: E402


class PytestPluginGuard(unittest.TestCase):
    def test_reads_always_pass(self):
        self.assertTrue(write_allowed("GET", "172.22.64.228", False))

    def test_writes_blocked_unless_allowed(self):
        self.assertFalse(write_allowed("POST", "127.0.0.1", False))

    def test_allowed_writes_only_reach_loopback(self):
        self.assertTrue(write_allowed("POST", "127.0.0.1", True))
        self.assertFalse(write_allowed("POST", "172.22.64.228", True))  # PROD, refused even with writes on
        self.assertTrue(write_allowed("POST", "qa.example", True, ["qa.example"]))


if __name__ == "__main__":
    unittest.main()
