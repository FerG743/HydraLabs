#!/usr/bin/env python3
"""Reads a framework repo (README layout: apps/<App>/{modules,locators}, tests/<App>) and prints
JSON: every reusable *_flujo function with its signature and the matrix steps it covers, the locator
constants per class, and which cases already have a test. Read-only; stdlib only.

    python3 tools/catalog.py /path/to/repo [App]
"""
import ast, json, os, re, sys

STEPS = re.compile(r"[Pp]asos?\s+(\d+)(?:\s*[-–y,]\s*(\d+))?")  # "Paso 03", "Pasos 04-07"


def parse(path):
    with open(path, encoding="utf-8") as f:
        return ast.parse(f.read())


def functions(path, app):
    out = []
    for node in parse(path).body:
        if not isinstance(node, ast.FunctionDef) or node.name.startswith("_"):
            continue
        doc = ast.get_docstring(node) or ""
        m = STEPS.search(doc)
        params = [a.arg for a in node.args.args]
        out.append({
            "name": node.name,
            "module": f"apps.{app}.modules.{os.path.basename(path)[:-3]}",
            "params": [p for p in params if p != "driver"],
            "takes_driver": "driver" in params,
            "steps": [int(m.group(1)), int(m.group(2) or m.group(1))] if m else None,
            "doc": " ".join(doc.split())[:200],
        })
    return out


def locators(path):
    out = {}
    for node in parse(path).body:
        if isinstance(node, ast.ClassDef):
            out[node.name] = [t.id for s in node.body if isinstance(s, ast.Assign)
                              for t in s.targets if isinstance(t, ast.Name)]
    return out


def main(repo, only=None):
    apps_dir = os.path.join(repo, "apps")
    result = {}
    for app in sorted(os.listdir(apps_dir)):
        if only and app != only or not os.path.isdir(os.path.join(apps_dir, app, "modules")):
            continue
        mods, locs = [], {}
        for kind, fn in (("modules", mods), ("locators", None)):
            d = os.path.join(apps_dir, app, kind)
            for f in sorted(os.listdir(d)):
                if f.endswith(".py") and f != "__init__.py":
                    if kind == "modules":
                        mods += functions(os.path.join(d, f), app)
                    else:
                        locs[f[:-3]] = locators(os.path.join(d, f))
        tdir = os.path.join(repo, "tests", app)
        tests = sorted(f for f in os.listdir(tdir) if f.startswith("test_")) if os.path.isdir(tdir) else []
        data = sorted(os.listdir(os.path.join(apps_dir, app, "data")))
        result[app] = {"functions": mods, "locators": locs, "tests": tests, "data_files": data}
    print(json.dumps(result, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2] if len(sys.argv) > 2 else None)
