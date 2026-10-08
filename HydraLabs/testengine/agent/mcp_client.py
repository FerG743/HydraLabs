"""Minimal MCP client over stdio (newline-delimited JSON-RPC): initialize, tools/list, tools/call. No dependencies."""
import json, os, queue, subprocess, tempfile, threading

GUARD = os.path.join(os.path.dirname(os.path.abspath(__file__)), "guard.cjs")


def playwright_command(output_dir=None, headless=True, extra=()):
    """Playwright MCP with the read-only guard injected. Pinned: option names change between releases.
    Its snapshots/screenshots go to a temp dir, never into the repo it is working next to."""
    out = output_dir or tempfile.mkdtemp(prefix="hydra_mcp_")
    cmd = ["npx", "-y", "@playwright/mcp@0.0.83", "--isolated", "--init-page", GUARD, "--output-dir", out, *extra]
    return cmd + (["--headless"] if headless else [])


class MCPClient:
    def __init__(self, cmd, env=None, node_setup=None):
        full = os.environ.copy()
        full.update(env or {})
        shell_cmd = " ".join(subprocess.list2cmdline([c]) if " " in c else c for c in cmd)
        if node_setup:  # e.g. "source ~/.nvm/nvm.sh && nvm use 24"
            shell_cmd = f"{node_setup} >/dev/null && {shell_cmd}"
        self.proc = subprocess.Popen(["bash", "-c", shell_cmd], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                     stderr=subprocess.PIPE, text=True, env=full, cwd=tempfile.gettempdir())
        self._id, self._replies, self.stderr_lines = 0, {}, []
        self._cv = threading.Condition()
        threading.Thread(target=self._read_stdout, daemon=True).start()
        threading.Thread(target=self._read_stderr, daemon=True).start()
        self.request("initialize", {"protocolVersion": "2025-06-18", "capabilities": {},
                                    "clientInfo": {"name": "hydra-agent", "version": "0.1"}}, timeout=120)
        self._send({"jsonrpc": "2.0", "method": "notifications/initialized"})

    def _read_stdout(self):
        for line in self.proc.stdout:
            try:
                msg = json.loads(line)
            except ValueError:
                continue
            if "id" in msg and ("result" in msg or "error" in msg):
                with self._cv:
                    self._replies[msg["id"]] = msg
                    self._cv.notify_all()

    def _read_stderr(self):
        for line in self.proc.stderr:
            self.stderr_lines.append(line.rstrip())

    def _send(self, msg):
        self.proc.stdin.write(json.dumps(msg) + "\n")
        self.proc.stdin.flush()

    def request(self, method, params=None, timeout=60):
        self._id += 1
        rid = self._id
        self._send({"jsonrpc": "2.0", "id": rid, "method": method, "params": params or {}})
        with self._cv:
            if not self._cv.wait_for(lambda: rid in self._replies, timeout=timeout):
                raise TimeoutError(f"MCP {method} timed out after {timeout}s")
            reply = self._replies.pop(rid)
        if "error" in reply:
            raise RuntimeError(f"MCP {method} failed: {reply['error']}")
        return reply["result"]

    def tools(self):
        return self.request("tools/list")["tools"]

    def call(self, name, arguments=None, timeout=90):
        """-> (text, is_error)"""
        res = self.request("tools/call", {"name": name, "arguments": arguments or {}}, timeout=timeout)
        text = "\n".join(c.get("text", "") for c in res.get("content", []) if c.get("type") == "text")
        return text, bool(res.get("isError"))

    def close(self):
        try:
            self.proc.stdin.close()
            self.proc.terminate()
            self.proc.wait(timeout=10)
        except Exception:
            self.proc.kill()
