#!/usr/bin/env python3
"""Benchmark: can a model (or plain code) pick the existing automation function that implements a manual step?

  gold   builds the answer key from hand-written tests that mark their steps ("# ----- Pasos 04-07: ... -----")
         plus the matrix steps (cmd/matrix output); the accepted answers are the functions called in that block.
  bench  scores a lexical baseline and, with --models, LM Studio models. One closed-choice question per step:
         the catalog (tools/catalog.py output, step ranges stripped) + the step -> {"function": "<name>|none"}.

    python3 matcher.py gold --repo <framework repo> --matrix out/matrix.json --cases 1,3 > gold.json
    python3 matcher.py bench --gold gold.json --catalog out/catalog.json --app FacturacionWeb [--models a,b]
"""
import argparse, json, os, re, sys, time, unicodedata, urllib.request

MARK = re.compile(r"#\s*-{3,}\s*Paso(?:s)?\s+(\d+)(?:\s*[-–]\s*(\d+))?[^:]*:")
CHECKPOINT = re.compile(r"#\s*-{3,}\s*Checkpoint")
CALL = re.compile(r"\b([a-z][a-z_]*_flujo)\(")


def norm(s):
    return unicodedata.normalize("NFKD", s).encode("ascii", "ignore").decode().lower()


def words(s):
    return set(re.findall(r"[a-z]{3,}", norm(s)))


# ---- gold ---------------------------------------------------------------------------------------------
def parse_test(path):
    """{step number: set(function names)} from the step markers of one test file."""
    steps, cur, last = {}, [], []
    for line in open(path, encoding="utf-8"):
        m = MARK.search(line)
        if m:
            a, b = int(m.group(1)), int(m.group(2) or m.group(1))
            cur = list(range(a, b + 1))
            last = cur
            continue
        if CHECKPOINT.search(line):
            cur = last[-1:]  # a checkpoint belongs to the step it follows
            continue
        for fn in CALL.findall(line):
            if line.lstrip().startswith("def "):
                continue
            for n in cur:
                steps.setdefault(n, set()).add(fn)
    return steps


def gold(a):
    matrix = {c["id"]: c for c in json.load(open(a.matrix, encoding="utf-8"))}
    out = []
    for cid in a.cases.split(","):
        test = os.path.join(a.repo, "tests", "FacturacionWeb", f"test_fact_{int(cid):04d}.py")
        for n, fns in sorted(parse_test(test).items()):
            st = next((s for s in matrix[cid]["steps"] if int(s["n"]) == n), None)
            if st:
                prev = [" ".join(x["action"].split())[:140] for x in matrix[cid]["steps"] if int(x["n"]) < n][-4:]
                out.append({"case": cid, "case_name": matrix[cid]["name"], "previous": prev, "n": n, "text": " ".join(st["action"].split()),
                            "expected": " ".join(st.get("expected", "").split()), "accepted": sorted(fns)})
    json.dump(out, sys.stdout, indent=2, ensure_ascii=False)


def gold_cases(a):
    """Answer key for Jira-sourced cases: the function render_pytest maps each step to ('none' where it has no rule)."""
    sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "jira"))
    import glob
    import render_pytest as rp
    files = sorted(glob.glob(os.path.join(a.cases, "*.case.json")) + glob.glob(os.path.join(a.cases, "review", "*.case.json")))
    app, out = rp.App(), []
    for f in files:
        case = json.load(open(f, encoding="utf-8"))
        an = rp.analyse(case)
        calls, _ = rp.build_calls(app, an)
        texts = [rp.plain(x["action"]) for x in case["steps"]]
        for i, (st, line) in enumerate(calls):
            m = re.search(r"\bco\.(\w+)\(", line or "")
            args = {rp.slug(st["g"][0]): st["g"][1]} if st["op"] in ("fill", "select") else {}
            out.append({"case": case["id"], "case_name": rp.plain(case["name"]), "previous": texts[max(0, i - 4):i], "n": i + 1,
                        "text": texts[i], "expected": "", "accepted": [m.group(1) if m else "none"], "args": args})
    json.dump(out, sys.stdout, indent=2, ensure_ascii=False)


# ---- catalog for the prompt ---------------------------------------------------------------------------
def catalog_text(cat):
    rows = []
    for f in cat["functions"]:
        doc = re.sub(r"(?i)^pasos?\s+[\d\-–, y]+\s*[:.]\s*", "", f["doc"]).strip()
        doc = re.sub(r"(?i)\bpasos?\s+\d+(?:\s*[-–]\s*\d+)?\b:?", "", doc).strip()  # no step numbers: they would give the answer away
        doc = re.sub(r"\bFACT_\d+\b", "", doc).strip()
        rows.append(f"- {f['name']}({', '.join(f['params'])})  [{f['module'].split('.')[-1]}]: {doc}")
    return "\n".join(rows)


SYSTEM = """You match one manual test step to the existing automation function that implements it.
Choose exactly ONE function name from the catalog, copied exactly, or "none" if no function implements the step.
Answer ONLY with JSON: {"function": "<name or none>"}

Catalog:
%s"""


SYSTEM_ARGS = """You match one manual test step to the existing automation function that implements it, and extract its arguments.
Choose exactly ONE function name from the catalog, copied exactly, or "none" if no function implements the step (never invent one).
"args" maps each parameter of the chosen function (except driver) that the step gives a value for to that value, copied exactly from the step; use {} if none.
Answer ONLY with JSON: {"function": "<name or none>", "args": {"<param>": "<value>"}}

Catalog:
%s"""


# ---- agreement rule -----------------------------------------------------------------------------------
def decide(primary, secondary, names):
    """Two matchers must agree on a catalog function to auto-accept. Otherwise the step goes to review,
    carrying the primary's pick (or the secondary's if the primary abstained) so a human starts from a guess.
    -> (status, pick) with status 'accept' | 'review'."""
    valid = lambda a: a in names
    if primary == secondary and valid(primary):
        return "accept", primary
    return "review", primary if valid(primary) else (secondary if valid(secondary) else None)


# ---- baseline + scoring -------------------------------------------------------------------------------
def baseline(item, cat):
    q = words(item["case_name"] + " " + item["text"] + " " + item["expected"])
    best, score = "none", 0.0
    for f in cat["functions"]:
        d = words(f["name"].replace("_", " ") + " " + f["doc"])
        s = len(q & d) / (len(d) ** 0.5 or 1)
        if s > score:
            best, score = f["name"], s
    return best


def ask(url, model, system, item, effort, context=False, screen=None, want_args=False):
    body = {"model": model, "temperature": 0, "max_tokens": 60,
            "messages": [{"role": "system", "content": system},
                         {"role": "user", "content": json.dumps({"test_case": item["case_name"], **({"earlier_steps": item["previous"]} if context else {}), **({"current_screen": screen} if screen else {}), "step": item["text"], "expected_result": item["expected"]}, ensure_ascii=False)}]}
    if effort:
        body["reasoning_effort"] = effort
    req = urllib.request.Request(url + "/chat/completions", json.dumps(body).encode(), {"Content-Type": "application/json"})
    t = time.time()
    r = json.load(urllib.request.urlopen(req, timeout=600))
    raw = re.sub(r"(?s)<think>.*?</think>", "", r["choices"][0]["message"]["content"]).strip()
    args = None
    try:
        data = json.loads(re.sub(r"^```\w*|```$", "", raw).strip())
        fn = re.sub(r"\(.*$", "", str(data["function"])).strip()  # "name(args)" is still the name
        args = {k: str(v) for k, v in (data.get("args") or {}).items()} if want_args else None
    except Exception:
        fn = "<unparseable>"
    return (fn, time.time() - t, args) if want_args else (fn, time.time() - t)


def score(answers, gold_items, names):
    ok = sum(a in g["accepted"] for a, g in zip(answers, gold_items))
    return {"accuracy": ok / len(gold_items), "invalid": sum(a not in names and a != "none" for a in answers),
            "none": sum(a == "none" for a in answers)}


def bench(a):
    gold_items = json.load(open(a.gold, encoding="utf-8"))
    cat = json.load(open(a.catalog, encoding="utf-8"))[a.app]
    names = {f["name"] for f in cat["functions"]}
    system = (SYSTEM_ARGS if a.args else SYSTEM) % catalog_text(cat)
    results = {}

    base = [baseline(g, cat) for g in gold_items]
    results["keyword baseline (no model)"] = {**score(base, gold_items, names), "seconds": 0, "answers": base}

    module_of = {f["name"]: f["module"].split(".")[-1] for f in cat["functions"]}
    for model in (a.models.split(",") if a.models else []):
        answers, arg_answers, secs, case, screen = [], [], 0.0, None, None
        for g in gold_items:
            if g["case"] != case:  # new case: forget the screen
                case, screen = g["case"], None
            r = ask(a.url, model, system, g, a.effort, a.context, screen if a.screen else None, a.args)
            fn, dt = r[0], r[1]
            arg_answers.append(r[2] if a.args else None)
            if fn.startswith("navegar_a_") and fn in module_of:  # the model's own navigation answer sets the screen
                screen = {"module": module_of[fn], "after": fn}
            answers.append(fn)
            secs += dt
            print(f"{model} {g['case']}/{g['n']:>2}: {'ok  ' if fn in g['accepted'] else 'MISS'} {fn}  (want {g['accepted']})", flush=True)
        entry = {**score(answers, gold_items, names), "seconds": secs / len(gold_items), "answers": answers}
        if a.args:  # exact-match arguments on the steps that carry values
            want = [(i, g["args"]) for i, g in enumerate(gold_items) if g.get("args")]
            entry["args_correct"] = sum(arg_answers[i] == w for i, w in want)
            entry["args_total"] = len(want)
            entry["arg_answers"] = arg_answers
        results[model + (" +earlier steps" if a.context else "") + (" +screen" if a.screen else "")] = entry

    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), f"results{a.tag}.json")
    old = json.load(open(path)) if os.path.exists(path) else {}
    old.update(results)  # merge: models run in separate invocations
    json.dump(old, open(path, "w"), indent=2, ensure_ascii=False)
    report(old, gold_items, a.tag)


def report(results, gold_items, tag=""):
    n = len(gold_items)
    lines = [f"\n# Function-matching benchmark ({n} steps)\n", "| Matcher | Accuracy | Invented names | Said none | s/step |" + (" Args exact |" if any("args_correct" in r for r in results.values()) else ""), "|---|---|---|---|---|" + ("---|" if any("args_correct" in r for r in results.values()) else "")]
    for name, r in results.items():
        extra = f" | {r['args_correct']}/{r['args_total']}" if "args_correct" in r else ""
        lines.append(f"| {name} | {r['accuracy']:.0%} | {r['invalid']} | {r['none']} | {r['seconds']:.1f}{extra} |")
    lines += ["", "Misses (first matcher that is a model):", ""]
    for model in [k for k in results if "baseline" not in k]:
        lines.append(f"**{model}**")
        for ans, g in zip(results[model]["answers"], gold_items):
            if ans not in g["accepted"]:
                lines.append(f"- case {g['case']} step {g['n']}: {g['text'][:70]!r} -> {ans} (want {', '.join(g['accepted'])})")
        lines.append("")
    text = "\n".join(lines) + "\n"
    open(os.path.join(os.path.dirname(os.path.abspath(__file__)), f"report{tag}.md"), "w", encoding="utf-8").write(text)
    print(text)


def combine(a):
    gold_items = json.load(open(a.gold, encoding="utf-8"))
    names = {f["name"] for f in json.load(open(a.catalog, encoding="utf-8"))[a.app]["functions"]}
    res = json.load(open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "results.json")))
    name_only = lambda x: re.sub(r"\(.*$", "", x).strip()
    p, q = ([name_only(x) for x in res[k]["answers"]] for k in (a.primary, a.secondary))
    acc, rev = [], []
    for g, x, y in zip(gold_items, p, q):
        status, pick = decide(x, y, names)
        (acc if status == "accept" else rev).append((g, pick))
    ok = lambda lst: sum(pick in g["accepted"] for g, pick in lst)
    n = len(gold_items)
    print(f"primary  : {a.primary}\nsecondary: {a.secondary}\n")
    print(f"auto-accepted (both agree) : {len(acc):>2}/{n} steps ({len(acc) / n:.0%}) -> correct {ok(acc)}/{len(acc)}")
    print(f"sent to review             : {len(rev):>2}/{n} steps ({len(rev) / n:.0%}) -> primary's pick correct {ok(rev)}/{len(rev)}")
    wrong = [(g["case"], g["n"], pick) for g, pick in acc if pick not in g["accepted"]]
    print("confident-but-wrong (accepted, incorrect):", wrong or "none")


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    g = sub.add_parser("gold")
    g.add_argument("--repo", required=True)
    g.add_argument("--matrix", required=True)
    g.add_argument("--cases", default="1,3")
    gc = sub.add_parser("gold-cases", help="answer key from Jira case.json files + the renderer's mapping")
    gc.add_argument("--cases", required=True)
    b = sub.add_parser("bench")
    b.add_argument("--gold", required=True)
    b.add_argument("--catalog", required=True)
    b.add_argument("--app", default="FacturacionWeb")
    b.add_argument("--models", default="")
    b.add_argument("--url", default="http://127.0.0.1:1234/v1")
    b.add_argument("--effort", default="none")
    b.add_argument("--context", action="store_true", help="also show the model the 4 preceding steps of the case")
    b.add_argument("--args", action="store_true", help="also ask for the function's arguments and score them")
    b.add_argument("--tag", default="", help="suffix for results/report files (keeps benchmarks apart)")
    b.add_argument("--screen", action="store_true", help="also tell the model which screen the last navigation step opened")
    c = sub.add_parser("combine", help="evaluate the agreement rule on two saved result sets")
    c.add_argument("--gold", required=True)
    c.add_argument("--catalog", required=True)
    c.add_argument("--app", default="FacturacionWeb")
    c.add_argument("--primary", required=True, help="results.json key whose pick is used on disagreement")
    c.add_argument("--secondary", required=True)
    a = ap.parse_args()
    {"gold": gold, "gold-cases": gold_cases, "bench": bench, "combine": combine}[a.cmd](a)
