import os, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "..", "tools", "jira"))
sys.path.insert(0, os.path.join(HERE, "..", "tools", "jira", "support"))
import agent, test_spec_render as T  # noqa: E402


def state():
    return {"case": T.case("DWQ-131"), "crawl": "", "base_url": "http://127.0.0.1:3000"}


class MissingSpec(unittest.TestCase):
    def test_a_call_without_the_spec_argument_says_what_is_missing(self):
        out = agent.submit_spec(None, tempfile.mkdtemp(), "PortalQA", state())
        self.assertIn("Falta el objeto spec", out)
        self.assertNotIn("forma esperada", out)

    def test_a_spec_that_is_not_an_object_says_so(self):
        out = agent.submit_spec("[1, 2]", tempfile.mkdtemp(), "PortalQA", state())
        self.assertIn("debe ser un objeto JSON", out)


if __name__ == "__main__":
    unittest.main()
