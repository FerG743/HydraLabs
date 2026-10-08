#!/usr/bin/env python3
"""Scores local LMs on strict instruction following: test case JSON -> Gherkin.

Every check is deterministic (no model judges another model):
  lint        parses as Gherkin, right language, exactly one scenario, has a Then
  steps       every source step appears as a When/And step, in order (anchors + word overlap)
  expected    every expected result appears as a Then/And step, in order
  extra       generated steps that match no source step (additions)
  invented    literals (numbers, quoted/backticked strings, paths) not present in the source
  clean       raw reply is only the feature: starts with "# language:", no fences, no prose
A run is a STRICT PASS only if all of them are perfect. Stdlib only.

    python3 tools/scorecard/scorecard.py [-m model1,model2] [-n 3] [-t 0.2] [-u http://127.0.0.1:1234/v1]
"""
import argparse, glob, json, os, re, subprocess, sys, time, urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
LINT = os.path.join(os.path.dirname(os.path.dirname(HERE)), "bin", "lint")

KEYWORDS = {  # keyword -> stage (g=context, a=action, o=outcome, c=conjunction)
    "given": "g", "when": "a", "then": "o", "and": "c", "but": "c",
    "dado": "g", "cuando": "a", "entonces": "o", "y": "c", "pero": "c",
}


def norm(s):
    return re.sub(r"\s+", " ", re.sub(r"[`\"']", "", s)).strip().lower()


def words(s):
    return set(re.findall(r"\w+", norm(s)))


def jaccard(a, b):
    a, b = words(a), words(b)
    return len(a & b) / len(a | b) if a | b else 0.0


def anchors(text):  # values a faithful step must keep: `code` spans and "quoted" labels
    return [norm(a) for a in re.findall(r"`([^`]+)`", text) + re.findall(r'"([^"]+)"', text)]


def step_lines(feature):
    """[(stage, text)] with 'and/but' resolved to the previous stage."""
    out, prev = [], "g"
    for line in feature.splitlines():
        m = re.match(r"\s*(\w+)\s+(.*\S)\s*$", line)
        if not m or m.group(1).lower() not in KEYWORDS:
            continue
        stage = KEYWORDS[m.group(1).lower()]
        stage = prev if stage == "c" else stage
        prev = stage
        out.append((stage, m.group(2)))
    return out


def match_all(sources, lines):
    """For each source text, the index of the matching generated line (or None); plus unmatched lines."""
    used, idx = set(), []
    for src in sources:
        best, best_j = None, 0.0
        for i, ln in enumerate(lines):
            if i in used:
                continue
            a_ok = all(a in norm(ln) for a in anchors(src))
            j = jaccard(src, ln)
            if a_ok and j >= (0.4 if anchors(src) else 0.6) and j > best_j:
                best, best_j = i, j
        if best is not None:
            used.add(best)
        idx.append(best)
    extra = [i for i in range(len(lines)) if i not in used]
    return idx, extra


def literals(text):
    t = text
    found = re.findall(r"`([^`]+)`|\"([^\"]+)\"", t)
    found = [a or b for a, b in found]
    found += re.findall(r"\b\d{3,}\b", t) + re.findall(r"\S*[/_.]\S*\w", t)
    return {norm(x).strip(".,;:") for x in found if x.strip()}


def lint(feature, lang):
    r = subprocess.run([LINT, "-lang", lang, "-max-scenarios", "1", "-min-then", "1"],
                       input=feature, capture_output=True, text=True)
    try:
        return json.loads(r.stdout)
    except Exception:
        return {"ok": False, "errors": ["lint output unreadable: " + r.stderr[:100]]}


def score(case, raw):
    feature = raw.strip()
    lines = step_lines(feature)
    acts = [t for s, t in lines if s == "a"]
    outs = [t for s, t in lines if s == "o"]
    src_steps = [s["action"] for s in case["steps"]]
    src_exp = [e["text"] for e in case["expected"]]

    ai, a_extra = match_all(src_steps, acts)
    oi, o_extra = match_all(src_exp, outs)
    in_order = all(i is not None for i in ai + oi) and ai == sorted(ai) and oi == sorted(oi)

    source_text = json.dumps(case, ensure_ascii=False)
    body = "\n".join(l for l in feature.splitlines() if not l.strip().startswith(("#", "@")))
    invented = [x for x in sorted(literals(body)) if x not in norm(source_text)]

    li = lint(feature, case.get("language", "en"))
    clean = bool(re.match(r"#\s*language:", feature)) and "```" not in raw
    m = {
        "lint": li["ok"],
        "steps": sum(i is not None for i in ai) / len(ai),
        "expected": sum(i is not None for i in oi) / len(oi),
        "order": in_order,
        "extra": len(a_extra) + len(o_extra),
        "invented": len(invented),
        "clean": clean,
    }
    m["strict"] = (m["lint"] and m["steps"] == 1 and m["expected"] == 1 and m["order"]
                   and m["extra"] == 0 and m["invented"] == 0 and m["clean"])
    m["notes"] = {"lint": li.get("errors", []), "invented": invented[:5],
                  "missing_steps": [s for s, i in zip(src_steps, ai) if i is None],
                  "missing_expected": [s for s, i in zip(src_exp, oi) if i is None]}
    return m


def chat(url, model, system, user, temperature, effort):
    body = {"model": model, "temperature": temperature,
            "messages": [{"role": "system", "content": system}, {"role": "user", "content": user}]}
    if effort:
        body["reasoning_effort"] = effort
    req = urllib.request.Request(url + "/chat/completions", json.dumps(body).encode(),
                                 {"Content-Type": "application/json"})
    t = time.time()
    r = json.load(urllib.request.urlopen(req, timeout=600))
    dt = time.time() - t
    raw = re.sub(r"(?s)<think>.*?</think>", "", r["choices"][0]["message"]["content"])
    return raw, dt, r.get("usage", {}).get("completion_tokens", 0)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("-m", "--models", help="comma list; default = every non-embedding model LM Studio lists")
    ap.add_argument("-n", "--runs", type=int, default=3)
    ap.add_argument("-t", "--temperature", type=float, default=0.2)
    ap.add_argument("-e", "--effort", default="none", help="reasoning_effort; '' to omit")
    ap.add_argument("-u", "--url", default="http://127.0.0.1:1234/v1")
    ap.add_argument("-c", "--cases", default=os.path.join(HERE, "cases"))
    ap.add_argument("-k", "--filter", default="", help="only cases whose id contains this text")
    ap.add_argument("--rescore", action="store_true", help="re-score saved outputs in results/ without calling any model")
    ap.add_argument("--log", help="with --rescore: an earlier console log, to recover seconds per run")
    a = ap.parse_args()

    results_path = os.path.join(HERE, "results", "results.json")
    results = json.load(open(results_path)) if os.path.exists(results_path) else {}
    system = open(os.path.join(HERE, "prompt.txt"), encoding="utf-8").read()
    cases = [json.load(open(f, encoding="utf-8")) for f in sorted(glob.glob(os.path.join(a.cases, "*.json")))]
    cases = [c for c in cases if a.filter in c["id"]]

    if a.rescore:
        secs = {}
        if a.log:
            for m in re.finditer(r"^(\S+) (\S+) run(\d+):.* ([\d.]+)s$", open(a.log).read(), re.M):
                secs[m.groups()[:3]] = float(m.group(4))
        models = a.models.split(",") if a.models else sorted(
            d for d in os.listdir(os.path.join(HERE, "results")) if os.path.isdir(os.path.join(HERE, "results", d)))
    else:
        models = a.models.split(",") if a.models else [
            m["id"] for m in json.load(urllib.request.urlopen(a.url + "/models"))["data"] if "embed" not in m["id"]]

    for model in models:
        out_dir = os.path.join(HERE, "results", model.replace("/", "_"))
        os.makedirs(out_dir, exist_ok=True)
        runs = []
        try:
            for case in cases:
                for k in range(a.runs):
                    path = os.path.join(out_dir, f"{case['id']}-run{k + 1}.feature")
                    if a.rescore:
                        if not os.path.exists(path):
                            continue
                        raw, dt, toks = open(path, encoding="utf-8").read(), secs.get((model.replace("_", "/", 1), case["id"], str(k + 1)), 0), 0
                    else:
                        raw, dt, toks = chat(a.url, model, system, json.dumps(case, ensure_ascii=False), a.temperature, a.effort)
                        open(path, "w", encoding="utf-8").write(raw)
                    s_ = score(case, raw)
                    s_.update(case=case["id"], run=k + 1, seconds=round(dt, 1), tokens=toks)
                    runs.append(s_)
                    print(f"{model} {case['id']} run{k + 1}: {'PASS' if s_['strict'] else 'fail'}  "
                          f"steps={s_['steps']:.2f} exp={s_['expected']:.2f} extra={s_['extra']} inv={s_['invented']} "
                          f"lint={s_['lint']} clean={s_['clean']} {s_['seconds']}s", flush=True)
        except urllib.error.HTTPError as e:  # e.g. LM Studio will not auto-load a second model
            print(f"!! {model}: HTTP {e.code} - is it loaded in LM Studio? Skipping; load it and re-run with -m {model}", flush=True)
        if runs:
            name = model.replace("_", "/", 1) if a.rescore and "/" not in model else model
            ran = {r["case"] for r in runs}  # keep earlier runs of cases this invocation did not touch
            results[name] = [r for r in results.get(name, []) if r["case"] not in ran] + runs
            json.dump(results, open(results_path, "w"), indent=2, ensure_ascii=False)
    write_report(results, a)


def write_report(results, a):
    rows = ["| Model | Strict pass | Lint ok | Steps kept | Expected kept | Order ok | Extra steps | Invented | Clean | s/run |",
            "|---|---|---|---|---|---|---|---|---|---|"]
    for model, runs in results.items():
        n = len(runs)
        avg = lambda k: sum(float(r[k]) for r in runs) / n
        rows.append(f"| {model} | {sum(r['strict'] for r in runs)}/{n} | {avg('lint'):.0%} | {avg('steps'):.0%} | "
                    f"{avg('expected'):.0%} | {avg('order'):.0%} | {avg('extra'):.1f} | {avg('invented'):.1f} | "
                    f"{avg('clean'):.0%} | {avg('seconds'):.0f} |")
    per_case = ["", "Per case (strict passes / runs):", "", "| Model | " + " | ".join(sorted({r['case'] for rs in results.values() for r in rs})) + " |",
                "|---|" + "---|" * len({r['case'] for rs in results.values() for r in rs})]
    for model, runs in results.items():
        cs = sorted({r["case"] for r in runs})
        per_case.append(f"| {model} | " + " | ".join(
            f"{sum(r['strict'] for r in runs if r['case'] == c)}/{sum(1 for r in runs if r['case'] == c)}" for c in cs) + " |")
    fails = ["", "Most common problems:", ""]
    for model, runs in results.items():
        seen = {}
        for r in runs:
            for k in ("lint", "invented", "missing_steps", "missing_expected"):
                for x in r["notes"][k]:
                    seen[f"{k}: {x}"[:140]] = seen.get(f"{k}: {x}"[:140], 0) + 1
        for text, c in sorted(seen.items(), key=lambda kv: -kv[1])[:6]:
            fails.append(f"- {model}: {text}  (x{c})")
    head = f"# Model scorecard\n\nRuns per case: {a.runs} | temperature: {a.temperature} | reasoning_effort: {a.effort or 'default'}\n\n"
    text = head + "\n".join(rows + per_case + fails) + "\n"
    open(os.path.join(HERE, "results", "report.md"), "w", encoding="utf-8").write(text)
    print("\n" + text)


if __name__ == "__main__":
    main()
