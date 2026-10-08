#!/usr/bin/env python3
"""Resolves "Same as TC-1" preconditions across the cases already generated in a directory.

A reference like TC-1 is matched to a case by key, else by title prefix ("TC-1 – Manual order entry...";
Jira's own link for "TC-1" points at an unrelated issue, so the title is the reliable key). The referenced
precondition is appended to the original text, which is kept verbatim. Chains (TC-3 -> TC-2 -> TC-1) resolve
transitively; cycles, ambiguity and unknown references stay unresolved and are reported, never guessed.

    python3 resolve_refs.py --dir out/jira --lint bin/lint [--lang en]

Rebuilds the .feature of every case it changes, re-lints it, and moves it between <dir> and <dir>/review
by the same rule the n8n workflow uses (lint ok, nothing unresolved, no blocking warning).
"""
import argparse, glob, json, os, re, subprocess, sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gherkin_gen import generate  # noqa: E402

REF_WARNING = "precondition refers to"


def find_target(ref, cases, self_id):
    hits = [c for c in cases if c["id"] == ref]
    if not hits:  # "TC-1" matches "TC-1 – ..." and "TC-1: ..." but never "TC-10"
        hits = [c for c in cases if re.match(rf"^{re.escape(ref)}(?![\w-])", c["name"].strip())]
    hits = [c for c in hits if c["id"] != self_id]
    if len(hits) == 1:
        return hits[0], None
    return None, ("not found" if not hits else "ambiguous: " + ", ".join(c["id"] for c in hits))


def precondition_of(case, cases, seen=()):
    """-> (text, source id, error). Follows chains; `seen` catches cycles."""
    ref = case.get("precondition_ref")
    if not ref:
        return case["precondition"], None, None
    original = case.get("precondition_original") or case["precondition"]
    if case["id"] in seen:
        return None, None, "cycle: " + " -> ".join(list(seen) + [case["id"]])
    target, err = find_target(ref, cases, case["id"])
    if err:
        return None, None, err
    text, _, err = precondition_of(target, cases, seen + (case["id"],))
    if err:
        return None, None, err
    return f"{original.rstrip('. ')}: {text}", target["id"], None


def resolve(cases):
    """Updates cases in place; returns {id: 'resolved from X' | reason it stayed unresolved}."""
    out = {}
    for c in cases:
        if not c.get("precondition_ref"):
            continue
        c.setdefault("precondition_original", c["precondition"])
        original = c["precondition_original"]
        keep = [w for w in c.get("warnings", []) if not w.startswith(REF_WARNING)]
        text, source, error = precondition_of({**c, "precondition": original}, cases)
        if text is None:
            info = error
            c["precondition"] = original
            c.pop("precondition_resolved_from", None)
            c["warnings"] = keep + [f"{REF_WARNING} {c['precondition_ref']}: {info}"]
            out[c["id"]] = info
        else:
            c["precondition"], c["precondition_resolved_from"], c["warnings"] = text, source, keep
            out[c["id"]] = f"resolved from {source}"
    return out


def lint(binary, feature, lang):
    r = subprocess.run([binary, "-lang", lang, "-max-scenarios", "1"], input=feature, capture_output=True, text=True)
    try:
        return json.loads(r.stdout)
    except Exception:
        return {"ok": False, "errors": ["lint output unreadable: " + r.stderr[:80]], "needs_input": []}


def run(directory, binary, lang):
    paths = {}  # case id -> path of its .case.json (clean dir or review/)
    for p in glob.glob(os.path.join(directory, "*.case.json")) + glob.glob(os.path.join(directory, "review", "*.case.json")):
        paths[json.load(open(p, encoding="utf-8"))["id"]] = p
    cases = [json.load(open(p, encoding="utf-8")) for p in paths.values()]
    before = {c["id"]: json.dumps(c, sort_keys=True) for c in cases}
    status = resolve(cases)

    moved = []
    for c in cases:
        if not c.get("precondition_ref") or json.dumps(c, sort_keys=True) == before[c["id"]]:
            continue  # untouched: leave its files alone
        feature = generate(c)
        r = lint(binary, feature, lang)
        blocking = [w for w in c.get("warnings", []) if not w.startswith("priority conflict")]
        clean = r["ok"] and not r["needs_input"] and not blocking
        old = paths[c["id"]]
        stem = os.path.basename(old)[: -len(".case.json")]
        dest = directory if clean else os.path.join(directory, "review")
        for ext in (".feature", ".case.json", ".notes.txt"):  # remove the old location's files
            for d in (directory, os.path.join(directory, "review")):
                if os.path.exists(os.path.join(d, stem + ext)):
                    os.remove(os.path.join(d, stem + ext))
        os.makedirs(dest, exist_ok=True)
        notes = [f"LINT: {e}" for e in r["errors"]] + r["needs_input"] + c.get("warnings", [])
        if c.get("precondition_resolved_from"):
            notes.append(f"precondition resolved from {c['precondition_resolved_from']}")
        with open(os.path.join(dest, stem + ".feature"), "w", encoding="utf-8") as f:
            f.write(feature)
        with open(os.path.join(dest, stem + ".case.json"), "w", encoding="utf-8") as f:
            json.dump(c, f, ensure_ascii=False)
        if notes:
            with open(os.path.join(dest, stem + ".notes.txt"), "w", encoding="utf-8") as f:
                f.write("\n".join(notes) + "\n")
        moved.append({"case": c["id"], "status": status[c["id"]], "clean": clean})
    return {"resolved": status, "rewritten": moved}


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--dir", required=True)
    ap.add_argument("--lint", required=True, help="path to the Go lint binary")
    ap.add_argument("--lang", default="en")
    a = ap.parse_args()
    print(json.dumps(run(a.dir, a.lint, a.lang), indent=2, ensure_ascii=False))
