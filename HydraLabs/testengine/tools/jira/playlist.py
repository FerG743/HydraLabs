#!/usr/bin/env python3
"""Playlists: an ordered list of generated cases run as one flow, with values passed forward (testengine/PLAYLISTS.md).

  python3 playlist.py check --playlist flow.json --cases cases.json --out automation
  python3 playlist.py run   --playlist flow.json --cases cases.json --out automation --base-url http://host:3000
        [--allow-writes] [--headed] [--login steps.json --login-path /] [--record out/playlist.json]

A playlist file:
  {"id": "order-lifecycle", "stopOnFailure": true,
   "steps": [
     {"case": "DWQ-101", "captures": {"order_id": "ORDER_NUMBER"}},
     {"case": "DWQ-102", "with": {"order_number": "$order_id"}},
     {"case": "DWQ-103", "with": {"api_key": "ENV:QA_API_KEY"}}]}

- captures: {name: locator} or {name: {"source": locator, "log": true}}. The value is the locator's text (or input
  value) after that case passes. Not written to the record unless log is true.
- with: {column: value}. Column = a data column of that case's generated CSV. Value: a literal, "$name" (a capture from
  an EARLIER step), or "ENV:NAME" (from the environment). It overrides the case's own data for this run only.
"""
import argparse, glob, importlib.util, json, os, re, sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "support"))
import harness  # noqa: E402


def slug(s):  # must match render_pytest.slug: the generated file is test_<slug(case id)>.py
    return re.sub(r"[^a-z0-9]+", "_", s.lower()).strip("_")


def _ref(v):
    if isinstance(v, str) and v.startswith("$"):
        return "capture", v[1:]
    if isinstance(v, str) and v.startswith("ENV:"):
        return "env", v[4:]
    return "literal", v


def _source(cap):
    return cap["source"] if isinstance(cap, dict) else cap


def _logged(cap):
    return isinstance(cap, dict) and cap.get("log") is True


def csv_header(out_dir, token):
    """Columns of this case's generated data file ([caso_id, ...]), or None if the case was never rendered."""
    import csv
    paths = glob.glob(os.path.join(out_dir, "apps", "*", "data", f"{token}_data.csv"))
    if not paths:
        return None
    with open(paths[0], newline="", encoding="utf-8") as f:
        return next(csv.reader(f), None)


def load_selectors(out_dir):
    """Locator constants of the generated app ({NAME: selector}), read by importing the generated locators module."""
    sel = {}
    for i, path in enumerate(glob.glob(os.path.join(out_dir, "apps", "*", "locators", "*_locators.py"))):
        spec = importlib.util.spec_from_file_location(f"playlist_locators_{i}", path)
        mod = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(mod)
        for cls in vars(mod).values():
            if isinstance(cls, type) and cls.__name__.endswith("Locators"):
                sel.update({k: v for k, v in vars(cls).items() if k.isupper() and isinstance(v, str)})
    return sel


def validate(pl, cases, out_dir, selectors):
    """Static checks, no browser. Returns (errors, warnings)."""
    errors, warnings = [], []
    steps = pl.get("steps") or []
    if not steps:
        errors.append("playlist has no steps")
    produced = {}  # capture name -> 1-based step that produces it
    ran = []  # case ids in earlier steps
    used = set()
    for i, st in enumerate(steps, 1):
        at, cid = f"step {i}", st.get("case")
        case = cases.get(cid)
        if case is None:
            errors.append(f"{at}: case {cid!r} is not in the cases file")
            continue
        if case.get("status") != "ready":
            errors.append(f"{at}: {cid} is {case.get('status')}; only ready cases can run in a playlist")
        for dep in case.get("depends_on") or []:
            if dep not in ran:
                errors.append(f"{at}: {cid} depends_on {dep}, which must run in an earlier step")
        header = csv_header(out_dir, slug(cid))
        if header is None:
            errors.append(f"{at}: no generated test for {cid} in {out_dir} (render it first)")
        for name, cap in (st.get("captures") or {}).items():
            if name in produced:
                errors.append(f"{at}: capture {name} is already produced in step {produced[name]}")
            elif not re.fullmatch(r"[A-Za-z_]\w*", name):
                errors.append(f"{at}: capture name {name!r} is not a plain identifier")
            if selectors is not None and _source(cap) not in selectors:
                errors.append(f"{at}: capture {name}: no locator {_source(cap)!r} in the app's locators")
            produced[name] = i
        for col, v in (st.get("with") or {}).items():
            if header is not None and col not in header[1:]:
                errors.append(f"{at}: {cid} has no data column {col!r} (columns: {', '.join(header[1:]) or 'none'})")
            kind, val = _ref(v)
            if kind == "capture":
                used.add(val)
                if produced.get(val, i) >= i:
                    errors.append(f"{at}: with {col}: ${val} must be captured by an earlier step")
        ran.append(cid)
    for name in produced:
        if name not in used:
            warnings.append(f"capture {name} is never used")
    return errors, warnings


class Unresolved(Exception):
    pass


def resolve(v, values):
    kind, val = _ref(v)
    if kind == "capture":
        if val not in values:
            raise Unresolved(f"${val} was not captured (its step did not pass)")
        return values[val]
    if kind == "env":
        if val not in os.environ:
            raise Unresolved(f"ENV:{val} is not set")
        return os.environ[val]
    return v


def run(pl, cases, out_dir, base_url, allow_writes=False, headed=False, login=None, runner=harness.run):
    selectors = load_selectors(out_dir)
    errors, warnings = validate(pl, cases, out_dir, selectors)
    if errors:
        return {"playlist": pl.get("id"), "status": "invalid", "errors": errors, "warnings": warnings, "steps": []}
    stop_on_failure = pl.get("stopOnFailure", True)
    values, report, stopped = {}, [], False
    for i, st in enumerate(pl["steps"], 1):
        cid = st["case"]
        entry = {"step": i, "case": cid, "status": None, "error": None, "captures": {}}
        if stopped:
            entry.update(status="skipped", error="an earlier step did not pass")
            report.append(entry)
            continue
        try:
            overrides = {col: resolve(v, values) for col, v in (st.get("with") or {}).items()}
        except Unresolved as e:
            entry.update(status="skipped", error=str(e))
            stopped = stop_on_failure
            report.append(entry)
            continue
        caps = st.get("captures") or {}
        r = runner(out_dir, slug(cid), base_url, allow_writes=allow_writes, headed=headed, login=login,
                   overrides=overrides or None, captures={n: selectors[_source(c)] for n, c in caps.items()} or None)
        entry.update(status=r["status"], error=r["error"])
        if r["status"] == "passed":
            for name, cap in caps.items():
                values[name] = r["captured"][name]
                entry["captures"][name] = values[name] if _logged(cap) else "<redacted>"
        else:
            stopped = stop_on_failure
        report.append(entry)
    ok = all(e["status"] == "passed" for e in report)
    return {"playlist": pl.get("id"), "status": "passed" if ok else "failed", "errors": [], "warnings": warnings,
            "steps": report}


def main(argv=None):
    ap = argparse.ArgumentParser()
    ap.add_argument("command", choices=["check", "run"])
    ap.add_argument("--playlist", required=True)
    ap.add_argument("--cases", required=True, help="normalized cases JSON ({\"cases\": [...]}) the tests were rendered from")
    ap.add_argument("--out", required=True, help="the generated automation folder")
    ap.add_argument("--base-url", default="")
    ap.add_argument("--allow-writes", action="store_true")
    ap.add_argument("--headed", action="store_true")
    ap.add_argument("--login", default="", help="the crawl's sign-in steps JSON (ENV: tokens)")
    ap.add_argument("--login-path", default="/")
    ap.add_argument("--record", default="", help="also write the result JSON to this file")
    a = ap.parse_args(argv)
    with open(a.playlist, encoding="utf-8") as f:
        pl = json.load(f)
    with open(a.cases, encoding="utf-8") as f:
        cases = {c["id"]: c for c in json.load(f)["cases"]}
    if a.command == "check":
        errors, warnings = validate(pl, cases, a.out, load_selectors(a.out))
        for w in warnings:
            print("warning:", w, file=sys.stderr)
        for e in errors:
            print("error:", e, file=sys.stderr)
        print("ok" if not errors else f"{len(errors)} error(s)")
        return 1 if errors else 0
    login = None
    if a.login:
        with open(a.login, encoding="utf-8") as f:
            login = {"path": a.login_path or "/", "steps": json.load(f)}
    result = run(pl, cases, a.out, a.base_url, a.allow_writes, a.headed, login)
    text = json.dumps(result, indent=2, ensure_ascii=False)
    if a.record:
        with open(a.record, "w", encoding="utf-8") as f:
            f.write(text + "\n")
    print(text)
    return 0 if result["status"] == "passed" else 1


if __name__ == "__main__":
    sys.exit(main())
