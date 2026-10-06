#!/usr/bin/env python3
"""Run one generated test against a real URL.  Read-only by default: every non-GET request the test does not
deliberately mock is aborted and reported.  --allow-writes lets real requests through (they can change data).

    python3 run_generated.py --out automation --test dwq_132 --base-url http://host:3000 [--allow-writes] [--headed]
"""
import argparse, json, os, sys

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "support"))
import harness  # noqa: E402

if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--test", required=True, help="e.g. dwq_132")
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--allow-writes", action="store_true")
    ap.add_argument("--headed", action="store_true")
    ap.add_argument("--shots", default="")
    a = ap.parse_args()
    r = harness.run(a.out, a.test, a.base_url, a.allow_writes, a.headed, a.shots or None)
    print(json.dumps(r, indent=2, ensure_ascii=False))
    sys.exit(0 if r["status"] == "passed" else 1)
