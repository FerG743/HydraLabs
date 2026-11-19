#!/usr/bin/env python3
"""Token bench: the same cases run with the hints (the crawl's page locators and the repo's selectors) and without.

    python3 agent/bench.py --repo-src <bundle> --app PortalQA --base-url http://127.0.0.1:3000 \
        --map <PortalQA.crawl.json> --cases DWQ-131 DWQ-140 --cases-dir <out/cases> --reps 2 --out results.jsonl

Each run works on a fresh copy of the repo, so no run sees another run's files. Only the tokens the model processed are
counted (agent.USAGE), with the turns and whether the run finished. The report prints the average per mode.
"""
import argparse, json, os, shutil, sys, tempfile, time
from statistics import mean

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import agent  # noqa: E402
import runner  # noqa: E402


def one(case, args, hints):
    repo = tempfile.mkdtemp(prefix="bench-")
    shutil.copytree(args.repo_src, repo, dirs_exist_ok=True)
    before, t0 = agent.USAGE["tokens"], time.time()
    logs = os.path.join(os.path.dirname(os.path.abspath(args.out)), "bench-logs")  # kept: the temp repo is not
    os.makedirs(logs, exist_ok=True)
    log = os.path.join(logs, f"{case['id']}-{'hints' if hints else 'nohints'}-{int(t0)}.jsonl")
    res = agent.run(case, repo, args.app, args.base_url, args.model, args.lm_url, args.max_turns, crawl=args.map,
                    max_tokens=args.max_tokens, hints=hints, log_path=log, live_gate=args.live_gate,
                    writes=runner.load_writes(args.writes_config))
    return {"case": case["id"], "hints": hints, "tokens": agent.USAGE["tokens"] - before, "turns": res.get("turns"),
            "finished": res.get("finished"), "seconds": round(time.time() - t0),
            "blocked_writes": len(res.get("blocked_writes", []))}


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--repo-src", required=True)
    ap.add_argument("--app", required=True)
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--map", required=True)
    ap.add_argument("--cases-dir", required=True)
    ap.add_argument("--cases", nargs="+", required=True)
    ap.add_argument("--reps", type=int, default=1)
    ap.add_argument("--live-gate", action="store_true", help="finish also needs each test to pass against the app")
    ap.add_argument("--model", default="prism-ml/bonsai-27b")
    ap.add_argument("--lm-url", default="http://127.0.0.1:1234/v1")
    ap.add_argument("--max-turns", type=int, default=15)
    ap.add_argument("--max-tokens", type=int, default=60000)
    ap.add_argument("--out", required=True)
    ap.add_argument("--writes-config", default="", help="the owning team's writes config: without it, test runs cannot write")
    ap.add_argument("--hints", choices=["on", "off", "both"], default="both", help="which hint modes to run")
    args = ap.parse_args()
    rows = []
    with open(args.out, "a", encoding="utf-8") as out:
        for rep in range(args.reps):
            for cid in args.cases:
                with open(os.path.join(args.cases_dir, cid + ".case.json"), encoding="utf-8") as f:
                    case = json.load(f)
                for hints in {"on": (True,), "off": (False,), "both": (True, False)}[args.hints]:
                    row = dict(one(case, args, hints), rep=rep)
                    rows.append(row)
                    out.write(json.dumps(row) + "\n")
                    out.flush()
                    print(row, flush=True)
    for mode in (True, False):
        xs = [r for r in rows if r["hints"] == mode]
        if xs:
            print(f"hints={mode}: runs {len(xs)}, mean tokens {mean(r['tokens'] for r in xs):.0f}, "
                  f"mean turns {mean(r['turns'] or 0 for r in xs):.1f}, finished {sum(bool(r['finished']) for r in xs)}/{len(xs)}")


if __name__ == "__main__":
    main()
