"""Agent loop mechanics with a scripted fake LM and a fake MCP. No real model, browser or network."""
import http.server, json, os, socketserver, sys, tempfile, threading, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import agent  # noqa: E402
import test_spec_render as fixtures  # noqa: E402

FRAMEWORK = {  # just enough of the framework for `pytest --collect-only` to import the generated test
    "config/__init__.py": "", "utils/__init__.py": "",
    "config/settings.py": "class Config:\n    BASE_URL = ''\n",
    "utils/templete_report.py": "def set_business_mode(v): pass\ndef add_step(msg, status=None, level=None): pass\n",
    "utils/csv_reader.py": "import csv\ndef cargar_datos_csv(p):\n    with open(p, newline='', encoding='utf-8') as f:\n        return [tuple(r) for r in list(csv.reader(f))[1:]]\n",
    "utils/driver_wrapper.py": "class DriverWrapper:\n    def __init__(self, d): self.page = d.page\n    def click(self, s, description=None): self.page.click(s)\n    def send_keys(self, s, t, description=None, is_secret=False): self.page.fill(s, t)\n",
    "conftest.py": "import pytest\n@pytest.fixture\ndef driver():\n    raise RuntimeError('not used in collect-only')\n",
}
CASE = fixtures.case("DWQ-130")


def call(name, **args):
    return {"role": "assistant", "content": "", "tool_calls": [{"id": f"c-{name}", "type": "function",
                                                                "function": {"name": name, "arguments": json.dumps(args)}}]}


class FakeMCP:
    def __init__(self):
        self.calls = []

    def tools(self):
        return [{"name": n, "description": f"{n}. extra", "inputSchema": {"type": "object", "properties": {}}}
                for n in ("browser_navigate", "browser_snapshot", "browser_run_code_unsafe", "browser_take_screenshot")]

    def call(self, name, args):
        self.calls.append((name, args))
        return "### Snapshot\n- button \"Agregar\" [ref=e3]", False

    def close(self):
        pass


def fake_lm(script):
    seen = []

    class H(http.server.BaseHTTPRequestHandler):
        def do_POST(self):
            body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
            seen.append(body)
            msg = dict(script[min(len(seen) - 1, len(script) - 1)])
            usage = msg.pop("usage_tokens", 0)  # lets a test make a turn "cost" tokens
            out = json.dumps({"choices": [{"message": msg}], "usage": {"total_tokens": usage}}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(out)

        def log_message(self, *a):
            pass

    srv = socketserver.TCPServer(("127.0.0.1", 0), H)
    threading.Thread(target=srv.serve_forever, daemon=True).start()
    return srv, f"http://127.0.0.1:{srv.server_address[1]}/v1", seen


class AgentLoop(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.repo = self.tmp.name
        for rel, text in FRAMEWORK.items():
            p = os.path.join(self.repo, rel)
            os.makedirs(os.path.dirname(p), exist_ok=True)
            with open(p, "w") as f:
                f.write(text)

    def tearDown(self):
        self.tmp.cleanup()

    def drive(self, script, **kw):
        srv, url, seen = fake_lm(script)
        try:
            res = agent.run(CASE, self.repo, "Portal", "http://x", "fake", url, mcp=FakeMCP(), **kw)
        finally:
            srv.shutdown()
            srv.server_close()
        return res, seen

    def test_happy_path_submits_a_spec_and_files_are_generated(self):
        script = [call("list_files", path="."), call("browser_navigate", url="http://x/p"),
                  call("submit_spec", spec=fixtures.golden("DWQ-130")), call("finish", summary="listo; nada sin automatizar")]
        res, seen = self.drive(script)
        self.assertTrue(res["finished"])
        self.assertEqual(res["problems"], [])
        self.assertTrue(os.path.exists(os.path.join(self.repo, "tests/Portal/test_dwq_130.py")))
        self.assertTrue(os.path.exists(os.path.join(self.repo, "apps/Portal/modules/cobro_ordenes_module.py")))
        result = next(m["content"] for m in seen[-1]["messages"] if m["role"] == "tool" and m["content"].startswith("Generado"))
        self.assertIn("apps/Portal/locators/cobro_ordenes_locators.py", result)

    def test_token_budget_stops_a_stuck_run_with_a_reason(self):
        turn = {**call("browser_snapshot"), "usage_tokens": 40000}
        res, seen = self.drive([turn], max_tokens=100000, max_turns=20)
        self.assertFalse(res["finished"])
        self.assertIn("token budget", res["summary"])
        self.assertLessEqual(len(seen), 4)  # 3 turns x 40k crosses 100k; it must not keep going to 20 turns

    def test_identical_browser_call_is_not_run_twice(self):
        mcp = FakeMCP()
        srv, url, seen = fake_lm([call("browser_snapshot"), call("browser_snapshot"), call("finish", summary="x")])
        try:
            agent.run(CASE, self.repo, "Portal", "http://x", "fake", url, mcp=mcp, max_turns=3)
        finally:
            srv.shutdown()
            srv.server_close()
        self.assertEqual([n for n, _ in mcp.calls].count("browser_snapshot"), 1)
        msgs = [m["content"] for m in seen[-1]["messages"] if m["role"] == "tool"]
        self.assertTrue(any("no aporta nada nuevo" in t for t in msgs), msgs)

    def test_app_map_reaches_the_model_and_evaluate_is_not_offered(self):
        crawl = os.path.join(self.repo, "c.json")
        json.dump({"pages": [{"url": "http://h/p", "elements": [
            {"role": "textbox", "name": "Orden", "locator": "#orderNumber", "stable": True},
            {"role": "button", "name": "Carousel", "locator": "[data-testid=x]", "stable": False}]}]}, open(crawl, "w"))
        res, seen = self.drive([call("finish", summary="x")], max_turns=1, app_map=agent.map_summary(crawl))
        first = seen[0]
        self.assertIn("#orderNumber", first["messages"][1]["content"])
        self.assertNotIn("data-testid=x", first["messages"][1]["content"])  # unstable locators are not offered
        self.assertNotIn("browser_evaluate", agent.BROWSER_TOOLS)  # the real allow-list, not the fake browser's

    def test_malformed_spec_string_tells_the_model_where_it_breaks(self):
        broken = '{"page": "p",\n "locators": {"A": "x" "B": "y"}}'  # missing comma, like the real run
        res, seen = self.drive([call("submit_spec", spec=broken), call("submit_spec", spec=broken), call("finish", summary="x")], max_turns=3)
        msgs = [m["content"] for m in seen[-1]["messages"] if m["role"] == "tool"]
        self.assertTrue(any("línea 2" in t and "Cerca de" in t for t in msgs), msgs)
        self.assertTrue(any("misma spec" in t for t in msgs), msgs)  # the identical resend is not processed again

    def test_invalid_spec_is_bounced_with_actionable_errors_and_writes_nothing(self):
        bad = fixtures.golden("DWQ-130")
        bad["steps"][2]["do"]["data"] = "no_existe"
        res, seen = self.drive([call("submit_spec", spec=bad), call("finish", summary="x")] * 4, max_turns=2)
        msgs = [m["content"] for m in seen[-1]["messages"] if m["role"] == "tool"]
        self.assertTrue(any("La spec tiene errores" in t and "no_existe" in t for t in msgs), msgs)
        self.assertFalse(os.path.exists(os.path.join(self.repo, "tests/Portal/test_dwq_130.py")))

    def test_the_model_has_no_way_to_write_files(self):
        res, seen = self.drive([call("write_file", path="utils/driver_wrapper.py", content="evil = 1\n"), call("finish", summary="x")], max_turns=1)
        with open(os.path.join(self.repo, "utils/driver_wrapper.py")) as f:
            self.assertIn("class DriverWrapper", f.read())                      # untouched
        self.assertNotIn("write_file", {t["function"]["name"] for t in seen[0]["tools"]})   # not even offered
        out = agent.dispatch("write_file", {"path": "x.py", "content": "y"}, None, FakeMCP(), self.repo, "Portal", "http://x", {}, {"case": CASE, "spec": None, "rendered": False})
        self.assertTrue(out.startswith("herramienta desconocida"), out)   # and calling it anyway does nothing

    def test_finish_is_bounced_until_a_spec_has_been_rendered(self):
        res, seen = self.drive([call("finish", summary="listo")] * 5)
        bounced = [m["content"] for m in seen[-1]["messages"] if m["role"] == "tool" and m["content"].startswith("No puedes terminar")]
        self.assertEqual(len(bounced), agent.FINISH_REFUSALS)  # refused twice, then let through with the problems reported
        self.assertIn("submit_spec", bounced[0])
        self.assertTrue(res["problems"])

    def test_dangerous_and_useless_tools_are_not_offered(self):
        _, seen = self.drive([call("finish", summary="x")], max_turns=1)
        offered = {t["function"]["name"] for t in seen[0]["tools"]}
        self.assertIn("browser_navigate", offered)
        self.assertNotIn("browser_run_code_unsafe", offered)   # could unroute the read-only guard
        self.assertNotIn("browser_take_screenshot", offered)   # a text model cannot use images
        self.assertTrue({"submit_spec", "verify_locators", "finish"} <= offered)

    def test_old_tool_results_are_compacted_but_recent_ones_kept(self):
        msgs = [{"role": "system", "content": "s"}] + [{"role": "tool", "tool_call_id": str(i), "content": "x" * 4000} for i in range(8)]
        agent.compact(msgs, budget_chars=12000)
        self.assertTrue(msgs[1]["content"].startswith("[omitido"))
        self.assertEqual(msgs[-1]["content"], "x" * 4000)  # the newest results survive


if __name__ == "__main__":
    unittest.main()
