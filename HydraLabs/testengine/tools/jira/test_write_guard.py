"""The write guard of the runner: with writes enabled, still never to a non-loopback host."""
import os, sys, unittest

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "support"))
from harness import write_allowed  # noqa: E402


class WriteGuard(unittest.TestCase):
    def test_reads_always_pass(self):
        self.assertTrue(write_allowed("GET", "172.22.64.228", False))

    def test_writes_blocked_by_default(self):
        self.assertFalse(write_allowed("POST", "127.0.0.1", False))

    def test_writes_allowed_to_loopback_only(self):
        self.assertTrue(write_allowed("POST", "127.0.0.1", True))
        self.assertTrue(write_allowed("POST", "localhost", True))
        self.assertFalse(write_allowed("POST", "172.22.64.228", True))  # PROD: refused even with allowWrites

    def test_explicit_extra_host(self):
        self.assertTrue(write_allowed("POST", "qa.example", True, ["qa.example"]))


if __name__ == "__main__":
    unittest.main()
