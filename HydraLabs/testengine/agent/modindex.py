"""What the repo's modules already have for a page: its locator constants, by the page their functions open.

The model is told these so it reuses a name and a selector the repo already has, instead of reading the locator files
and writing a new one. The renderer merges locators by name and refuses a name that maps to two selectors, so reuse also
avoids a conflict.
"""
import ast, importlib.util, os

_tool = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "tools", "coverage.py")
_spec = importlib.util.spec_from_file_location("hydra_coverage", _tool)
cov = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(cov)


def page_locators(repo, app):
    """norm path -> {NAME: selector}. A module file is one page, so every page its functions open gets all its selectors."""
    mod_dir = os.path.join(repo, "apps", app, "modules")
    pages = {}
    if not os.path.isdir(mod_dir):
        return pages
    for fname in sorted(os.listdir(mod_dir)):
        if not fname.endswith(".py") or fname.startswith("_"):
            continue
        path = os.path.join(mod_dir, fname)
        names = {n.name for n in cov.tree(path).body if isinstance(n, ast.FunctionDef) and not n.name.startswith("_")}
        consts, paths = cov.module_uses(path, names)
        found = {name: cov.locator_values(repo, app, lf, cls).get(name) for (lf, cls), name in consts}
        found = {k: v for k, v in found.items() if v}
        for p in paths:
            pages.setdefault(cov.norm_path(p), {}).update(found)
    return pages


def locator_lines(repo, app, case_paths, limit=2000):
    """'NAME = selector' lines for the case's pages, one heading per page; empty when the repo has none yet."""
    have = page_locators(repo, app)
    out, size = {}, 0
    for p in case_paths:
        names = have.get(cov.norm_path(p), {})
        if not names:
            continue
        lines = [f"{k} = {v}" for k, v in sorted(names.items())]
        if size + len(p) + sum(len(x) for x in lines) > limit:
            break
        out[p] = lines
        size += len(p) + sum(len(x) for x in lines)
    return out
