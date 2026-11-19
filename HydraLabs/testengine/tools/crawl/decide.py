#!/usr/bin/env python3
"""The team's answer to each write endpoint the crawl found (the map's "decisions" list).

    python3 decide.py <app.crawl.json>                       list the decisions and their choices
    python3 decide.py <app.crawl.json> --writes <writes.json> --set "POST /uploadFile" allow
    python3 decide.py <app.crawl.json> --set "POST /uploadFile" stub|keep_blocked
    python3 decide.py --writes <writes.json> --show-writes     the writes config as it stands
    python3 decide.py --writes <writes.json> --writes-off      no writes for this app (the switch)

allow: the endpoint is added to the writes config (its host to "hosts", "METHOD /path" to "endpoints"), so a walk may really send it.
stub: recorded for the test stubs (the stubbing itself is not built yet). keep_blocked: the default, nothing is sent.
"""
import argparse, json, os, sys
from urllib.parse import urlparse

CHOICES = ("allow", "stub", "keep_blocked")


def load(path):
    return json.load(open(path, encoding="utf-8"))


def listing(decisions):
    for d in decisions:
        print(f'{d["method"]:6} {d["path"]:<32} host={d["host"]:<14} {d["status"]:<8} choice={d["choice"]}')


def show_writes(writes_path):
    cfg = load(writes_path) if os.path.exists(writes_path) else None
    print(json.dumps(cfg, indent=1, ensure_ascii=False) if cfg else "no writes config: nothing is writable")


def writes_off(writes_path):
    """Keeps the hosts and endpoints the team listed; only the switch changes."""
    cfg = load(writes_path) if os.path.exists(writes_path) else {"hosts": [], "endpoints": []}
    cfg["allow_writes"] = False
    json.dump(cfg, open(writes_path, "w", encoding="utf-8"), indent=1, ensure_ascii=False)
    print("writes off: allow_writes=false (hosts and endpoints kept)")


def set_choice(map_path, writes_path, endpoint, choice):
    if choice not in CHOICES:
        raise SystemExit(f"choice must be one of {CHOICES}")
    method, path = endpoint.split(" ", 1)
    doc = load(map_path)
    hit = [d for d in doc.get("decisions", []) if d["method"] == method.upper() and d["path"] == path]
    if not hit:
        raise SystemExit(f"no decision for {method} {path} in {map_path}")
    for d in hit:
        d["choice"] = choice
    json.dump(doc, open(map_path, "w", encoding="utf-8"), indent=1, ensure_ascii=False)
    if choice == "allow":
        if not writes_path:
            raise SystemExit("allow needs --writes <writes.json>: the writes config the crawl reads")
        cfg = load(writes_path) if os.path.exists(writes_path) else {"allow_writes": True, "hosts": [], "endpoints": []}
        cfg["allow_writes"] = True
        host = hit[0]["host"]
        if host not in cfg.get("hosts", []):
            cfg.setdefault("hosts", []).append(host)
        key = f"{method.upper()} {path}"
        if key not in cfg.get("endpoints", []):
            cfg.setdefault("endpoints", []).append(key)
        json.dump(cfg, open(writes_path, "w", encoding="utf-8"), indent=1, ensure_ascii=False)
    print(f"{method.upper()} {path}: {choice}")


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("map", nargs="?", default="")
    ap.add_argument("--writes", default="")
    ap.add_argument("--set", nargs=2, metavar=("ENDPOINT", "CHOICE"))
    ap.add_argument("--show-writes", action="store_true")
    ap.add_argument("--writes-off", action="store_true")
    a = ap.parse_args()
    if a.show_writes or a.writes_off:
        if not a.writes:
            raise SystemExit("--writes <writes.json> is needed")
        show_writes(a.writes) if a.show_writes else writes_off(a.writes)
    elif not a.map:
        raise SystemExit("give the app map, or --writes with --show-writes / --writes-off")
    elif a.set:
        set_choice(a.map, a.writes, a.set[0], a.set[1])
    else:
        listing(load(a.map).get("decisions", []))
