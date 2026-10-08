"""Proves the read-only guard: through the real Playwright MCP, a click that makes the page POST never reaches the
server, while GETs still work. Needs node (npx) and network access to fetch @playwright/mcp the first time."""
import http.server, os, re, socketserver, sys, threading, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from mcp_client import MCPClient, playwright_command  # noqa: E402

PAGE = b"""<title>guard</title><h1>Cobro</h1><button id=go>Procesar</button><div id=out></div>
<script>fetch('/data').then(r => r.text()).then(t => document.getElementById('out').textContent = 'loaded ' + t);
document.getElementById('go').onclick = () => fetch('/submit', {method: 'POST', body: 'x'})
  .then(() => document.getElementById('out').textContent = 'posted').catch(() => document.getElementById('out').textContent = 'blocked');</script>"""


class Handler(http.server.BaseHTTPRequestHandler):
    seen = []

    def do_GET(self):
        Handler.seen.append(("GET", self.path))
        self.send_response(200)
        self.send_header("Content-Type", "text/html" if self.path == "/" else "text/plain")
        self.end_headers()
        self.wfile.write(PAGE if self.path == "/" else b"data")

    def do_POST(self):
        Handler.seen.append(("POST", self.path))
        self.send_response(200)
        self.end_headers()

    def log_message(self, *a):
        pass


class GuardTest(unittest.TestCase):
    def test_post_from_a_click_never_reaches_the_server(self):
        srv = socketserver.TCPServer(("127.0.0.1", 0), Handler)
        threading.Thread(target=srv.serve_forever, daemon=True).start()
        url = f"http://127.0.0.1:{srv.server_address[1]}/"
        mcp = MCPClient(playwright_command(), node_setup="source ~/.nvm/nvm.sh && nvm use 24")
        try:
            names = {t["name"] for t in mcp.tools()}
            self.assertTrue({"browser_navigate", "browser_snapshot", "browser_click"} <= names, names)
            mcp.call("browser_navigate", {"url": url})
            snap, _ = mcp.call("browser_snapshot")
            ref = re.search(r'button "Procesar" \[ref=(e\d+)\]', snap).group(1)
            out, is_error = mcp.call("browser_click", {"target": ref})
            self.assertFalse(is_error, out)                          # the click really happened
            mcp.call("browser_wait_for", {"time": 1})
            after, _ = mcp.call("browser_snapshot")
        finally:
            mcp.close()
            srv.shutdown()
            srv.server_close()
        self.assertIn(("GET", "/data"), Handler.seen)                 # reads still work
        self.assertNotIn("POST", [m for m, _ in Handler.seen])        # the write never left the browser
        self.assertIn("blocked", after)                               # the page saw the abort
        self.assertTrue(any("hydra-guard] blocked POST" in l for l in mcp.stderr_lines), mcp.stderr_lines[-5:])


if __name__ == "__main__":
    unittest.main()
