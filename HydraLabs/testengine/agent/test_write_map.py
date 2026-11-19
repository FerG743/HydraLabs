import json, os, sys, tempfile, unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import agent  # noqa: E402

CRAWL = {"pages": [{"url": "http://h/cobro_ordenes", "elements": [
    {"role": "button", "tag": "button", "name": "Subir y Procesar", "locator": "role=button[name=\"Subir y Procesar\"]", "stable": True}],
    "after": "Click \"Subir y Procesar\".", "writes": ["blocked POST http://h/uploadFile"]}]}


def crawl_file():
    fd, path = tempfile.mkstemp(suffix=".json")
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        json.dump(CRAWL, f)
    return path


class WriteActionsAreMarked(unittest.TestCase):
    def test_the_page_summary_says_the_control_writes_and_its_result_is_unknown(self):
        out = agent.page_summary(crawl_file(), "http://h/cobro_ordenes")
        self.assertIn("ESCRIBE", out)
        self.assertIn("POST http://h/uploadFile", out)
        self.assertIn("no supongas", out)
        self.assertIn("blocked POST http://h/uploadFile", out)

    def test_an_observed_result_replaces_the_unknown_one(self):
        page = dict(CRAWL["pages"][0], observed=["Completado 0 / 2 Proceso completado"], writes=["sent POST http://h/uploadFile"])
        fd, path = tempfile.mkstemp(suffix=".json")
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump({"pages": [page]}, f)
        out = agent.page_summary(path, "http://h/cobro_ordenes")
        self.assertIn("resultado observado en la app: ", out)
        self.assertNotIn("no observado", out)

    def test_the_map_summary_marks_it_too(self):
        self.assertIn("ESCRIBE", agent.map_summary(crawl_file()))

    def test_a_page_without_writes_gets_no_mark(self):
        plain = {"pages": [{"url": "http://h/x", "elements": CRAWL["pages"][0]["elements"]}]}
        fd, path = tempfile.mkstemp(suffix=".json")
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump(plain, f)
        self.assertNotIn("ESCRIBE", agent.page_summary(path, "http://h/x"))


if __name__ == "__main__":
    unittest.main()
