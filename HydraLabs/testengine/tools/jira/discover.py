"""State discovery: run one generated case several times with the discover_states plugin and keep only the state
changes every run agrees on. The output is candidates (data), never cases: a person or the drafting step picks them.

  python3 discover.py --repo <app repo> --bundle <bundle folder> --token pqa_7 --crawl <crawl.json> --runs 3 --out <file>
"""
import argparse, json, os, shutil, subprocess, sys, tempfile
from collections import Counter

HERE = os.path.dirname(os.path.abspath(__file__))


def transitions(steps):
    """The state changes between consecutive snapshots: each is caused by the step before the second snapshot."""
    out = []
    for prev, cur in zip(steps, steps[1:]):
        for name, now in cur["controls"].items():
            before = prev["controls"].get(name)
            if before is None:
                continue
            for field in ("enabled", "visible"):
                if before[field] is not None and now[field] is not None and before[field] != now[field]:
                    out.append((prev["step"], name, field, before[field], now[field]))
    return out


def gate(runs, need):
    """Keep a change only if at least `need` runs saw it (the 3-passes gate)."""
    counts = Counter(t for run in runs for t in set(run))
    return sorted((t for t, n in counts.items() if n >= need), key=lambda t: (t[0], t[1], t[2]))


def load_env(repo):
    env = {}
    try:
        with open(os.path.join(repo, ".env"), encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    k, v = line.split("=", 1)
                    env.setdefault(k.strip(), v.strip().strip("\"'"))
    except OSError:
        pass
    return env


def run_once(repo, bundle, token, crawl, workdir):
    """One pass in a scratch copy of the repo with the bundle's generated files on top; returns its snapshots."""
    work = os.path.join(workdir, "repo")
    shutil.copytree(repo, work, ignore=shutil.ignore_patterns(".git", "hydra", "out", "__pycache__", ".env"))
    for part in ("apps", "tests", "utils"):
        src = os.path.join(bundle, part)
        if os.path.isdir(src):
            shutil.copytree(src, os.path.join(work, part), dirs_exist_ok=True)
    out = os.path.join(workdir, "steps.json")
    env = dict(os.environ, **load_env(repo))
    env.update({"PYTHONPATH": f"{work}{os.pathsep}{HERE}", "DISCOVER_CRAWL": crawl, "DISCOVER_OUT": out,
                "HEADLESS": env.get("HEADLESS", "1")})
    test = os.path.join(work, "tests", "PortalQAdev", f"test_{token}.py")
    subprocess.run([sys.executable, "-m", "pytest", test, "-q", "-p", "discover_states", "-p", "no:cacheprovider"],
                   cwd=work, env=env, capture_output=True, text=True)
    if not os.path.exists(out):
        raise RuntimeError("no snapshots written: the test did not start")
    with open(out, encoding="utf-8") as f:
        return json.load(f)


def discover(repo, bundle, token, crawl, runs=3, need=3):
    runs_t, raw = [], []
    for _ in range(runs):
        with tempfile.TemporaryDirectory() as tmp:
            steps = run_once(repo, bundle, token, crawl, tmp)
        raw.append(steps)
        runs_t.append(transitions(steps))
    kept = gate(runs_t, need)
    return {"token": token, "runs": runs, "need": need,
            "candidates": [{"after_step": s, "control": c, "field": f, "from": a, "to": b} for s, c, f, a, b in kept],
            "seen_per_run": [len(t) for t in runs_t]}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--repo", required=True)
    ap.add_argument("--bundle", required=True)
    ap.add_argument("--token", required=True)
    ap.add_argument("--crawl", required=True)
    ap.add_argument("--runs", type=int, default=3)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    result = discover(a.repo, a.bundle, a.token, a.crawl, a.runs, need=a.runs)
    os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
    with open(a.out, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
