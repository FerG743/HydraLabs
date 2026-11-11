#!/usr/bin/env python3
"""Coverage gaps: which pages and elements of the app map no generated test reaches.

Static and read-only (stdlib only, like catalog.py). It follows each test in tests/<App>/ to the module functions it
calls, and collects what those functions use: the locator constants (L.NAME, valued in apps/<App>/locators) and the
paths they open ('/cobro_ordenes'). An element of the crawl is covered when its locator is one of those values; a page
is covered when its path is one of those paths, or when one of its elements is covered.

    python3 tools/coverage.py --suite out/bundles/DWQ-131 --suite out/bundles/DWQ-140 --app PortalQA \
        --crawl out/appmap/PortalQA.crawl.json --out out/coverage.json

It judges what the code refers to, not what runs: a test that is skipped or fails still counts as covering.
"""
import argparse, ast, json, os, re, sys
from datetime import datetime
from urllib.parse import urlparse

PATH = re.compile(r"^/[\w\-./%]*$")


def tree(path):
    with open(path, encoding="utf-8") as f:
        return ast.parse(f.read())


def test_calls(tests_dir, app):
    """(module file, function) for every call a test makes into apps/<app>/modules."""
    calls = set()
    for name in sorted(os.listdir(tests_dir)) if os.path.isdir(tests_dir) else []:
        if not (name.startswith("test_") and name.endswith(".py")):
            continue
        t = tree(os.path.join(tests_dir, name))
        mods, direct = {}, {}  # local name -> module file ; local name -> (module file, function)
        for node in ast.walk(t):
            if not isinstance(node, ast.ImportFrom) or not node.module:
                continue
            if node.module == f"apps.{app}.modules":  # from apps.X.modules import portal_qa_module as co
                for a in node.names:
                    mods[a.asname or a.name] = a.name
            elif node.module.startswith(f"apps.{app}.modules."):  # from apps.X.modules.m import fn
                mod = node.module.rsplit(".", 1)[1]
                for a in node.names:
                    direct[a.asname or a.name] = (mod, a.name)
        for node in ast.walk(t):
            if not isinstance(node, ast.Call):
                continue
            f = node.func
            if isinstance(f, ast.Attribute) and isinstance(f.value, ast.Name) and f.value.id in mods:
                calls.add((mods[f.value.id], f.attr))
            elif isinstance(f, ast.Name) and f.id in direct:
                calls.add(direct[f.id])
    return calls


def module_uses(path, funcs):
    """Locator constants (class, NAME) and path strings used by the given functions, and the functions they call."""
    t = tree(path)
    locs = {}  # local name -> (locators module file, class)
    for node in t.body:
        if isinstance(node, ast.ImportFrom) and node.module and ".locators." in node.module:
            for a in node.names:
                locs[a.asname or a.name] = (node.module.rsplit(".", 1)[1], a.name)
    defs = {n.name: n for n in t.body if isinstance(n, ast.FunctionDef)}
    todo, seen, consts, paths = list(funcs), set(), set(), set()
    while todo:  # a function may call another function of the same module
        name = todo.pop()
        if name in seen or name not in defs:
            continue
        seen.add(name)
        for node in ast.walk(defs[name]):
            if isinstance(node, ast.Attribute) and isinstance(node.value, ast.Name) and node.value.id in locs:
                consts.add((locs[node.value.id], node.attr))
            if isinstance(node, ast.Name) and node.id in defs:
                todo.append(node.id)
            if isinstance(node, ast.Constant) and isinstance(node.value, str) and node.value != "/" and PATH.match(node.value):  # a bare "/" is in rstrip("/") too
                paths.add(node.value)
    return consts, paths


def locator_values(app_repo, app, loc_file, cls):
    """NAME -> the locator string, for one locators class."""
    path = os.path.join(app_repo, "apps", app, "locators", loc_file + ".py")
    if not os.path.exists(path):
        return {}
    out = {}
    for node in tree(path).body:
        if isinstance(node, ast.ClassDef) and node.name == cls:
            for s in node.body:
                if isinstance(s, ast.Assign) and isinstance(s.value, ast.Constant) and isinstance(s.value.value, str):
                    for t in s.targets:
                        if isinstance(t, ast.Name):
                            out[t.id] = s.value.value
    return out


def norm_path(p):
    return (p.rstrip("/") or "/")


def suite_reach(root, app):
    """What one suite root (a repo, or one bundle: apps/ and tests/ side by side) reaches: locator values, paths, tests."""
    tests_dir = os.path.join(root, "tests", app)
    mod_dir = os.path.join(root, "apps", app, "modules")
    by_mod = {}
    for mod, fn in test_calls(tests_dir, app):
        by_mod.setdefault(mod, set()).add(fn)
    values, paths = set(), set()
    for mod, fns in by_mod.items():
        mpath = os.path.join(mod_dir, mod + ".py")
        if not os.path.exists(mpath):
            continue
        consts, ps = module_uses(mpath, fns)
        paths |= ps
        for (loc_file, cls), name in consts:
            v = locator_values(root, app, loc_file, cls).get(name)
            if v:
                values.add(v)
    n = len([x for x in os.listdir(tests_dir) if x.startswith("test_")]) if os.path.isdir(tests_dir) else 0
    return values, paths, n


def reach(roots, app):
    """Locator values, normalised paths and test count that the suite roots reach together."""
    values, paths, tests = set(), set(), 0
    for root in roots:
        v, p, n = suite_reach(root, app)
        values |= v; paths |= {norm_path(x) for x in p}; tests += n
    return values, paths, tests


def judge(page, values, paths):
    """Whether a crawled page is covered, how many of its elements, and the elements no test reaches."""
    covered_path = norm_path(urlparse(page["url"]).path) in paths
    els = page.get("elements", [])
    gaps = [e.get("locator") for e in els if e.get("locator") not in values]
    covered_els = len(els) - len(gaps)
    return covered_path or covered_els > 0, covered_els, gaps


def analyse(roots, app, crawl, flows=()):
    values, paths, tests = reach(roots, app)
    crawled = json.load(open(crawl, encoding="utf-8")).get("pages", [])
    pages, n_el, c_el, c_pg = [], 0, 0, 0
    for p in crawled:
        covered, covered_els, gaps = judge(p, values, paths)
        els = p.get("elements", [])
        n_el += len(els); c_el += covered_els; c_pg += covered
        pages.append({"url": p["url"], "covered": covered, "elements": len(els), "covered_elements": covered_els,
                      "gaps": gaps})
    return {"app": app, "generatedAt": datetime.now().isoformat(timespec="seconds"), "tests": tests,
            "summary": {"pages": len(pages), "coveredPages": c_pg, "elements": n_el, "coveredElements": c_el},
            "pages": pages, "flows": [flow_report(name, root, app, crawled) for name, root in flows]}


def flow_report(name, root, app, crawled):
    """A playlist's own suite root: the pages it reaches, and how many of each page's elements it exercises."""
    values, paths, tests = reach([root], app)
    reached = []
    for p in crawled:
        covered, covered_els, _ = judge(p, values, paths)
        if covered:
            reached.append({"url": p["url"], "elements": len(p.get("elements", [])), "covered_elements": covered_els})
    return {"name": name, "tests": tests, "pages": reached}


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--suite", action="append", default=[],
                    help="a root with apps/<App> and tests/<App> (a bundle, or the app repo); repeat for each. None: every element is a gap")
    ap.add_argument("--flow", action="append", default=[], metavar="NAME=ROOT",
                    help="a playlist's suite root (out/playlists/<NAME>); reported under flows, not in the gaps")
    ap.add_argument("--app", required=True)
    ap.add_argument("--crawl", required=True, help="the app map (<App>.crawl.json)")
    ap.add_argument("--out", help="write here instead of stdout")
    a = ap.parse_args(argv)
    flows = [tuple(f.split("=", 1)) for f in a.flow]
    result = analyse(a.suite, a.app, a.crawl, flows)
    text = json.dumps(result, ensure_ascii=False, indent=2)
    if a.out:
        os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
        with open(a.out, "w", encoding="utf-8") as f:
            f.write(text + "\n")
    else:
        print(text)
    return 0


if __name__ == "__main__":
    sys.exit(main())
