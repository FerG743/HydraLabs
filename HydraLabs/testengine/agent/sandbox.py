"""Read-only file tools for the agent, confined to one repo. The model cannot write files: render.py does."""
import os

MAX_BYTES = 60_000


class Sandbox:
    def __init__(self, repo, app):
        self.repo, self.app = os.path.realpath(repo), app

    def _resolve(self, path):
        full = os.path.realpath(os.path.join(self.repo, path))
        if full != self.repo and not full.startswith(self.repo + os.sep):
            raise PermissionError(f"'{path}' está fuera del repositorio")
        return full

    def list_files(self, path="."):
        full = self._resolve(path)
        if not os.path.isdir(full):
            return f"'{path}' no es una carpeta"
        skip = {".git", "__pycache__", "reports", "logs", ".venv", "node_modules"}
        out = []
        for root, dirs, names in os.walk(full):
            dirs[:] = sorted(d for d in dirs if d not in skip)
            for n in sorted(names):
                if not n.endswith(".pyc"):
                    out.append(os.path.relpath(os.path.join(root, n), self.repo))
            if len(out) > 200:
                break
        return "\n".join(out[:200]) or "(vacío)"

    def read_file(self, path):
        full = self._resolve(path)
        if not os.path.isfile(full):
            return f"'{path}' no existe"
        with open(full, encoding="utf-8", errors="replace") as f:
            text = f.read(MAX_BYTES + 1)
        return text[:MAX_BYTES] + ("\n...[truncado]" if len(text) > MAX_BYTES else "")
