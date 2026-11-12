#!/usr/bin/env python3
"""Routes: the fewest clicks from the start page to every crawled page, as the links the crawl found.

    python3 tools/routes.py --crawl out/appmap/PortalQA.crawl.json --app PortalQA --out out/routes.json

Each hop names the page it goes to, the link's locator and href, so a route can be replayed. Pages no link reaches are
listed as unreachable. The search is breadth-first, and neighbours are visited in URL order, so the same crawl always
gives the same routes. It uses links only: what a button or tab reveals is not an edge here.
"""
import argparse, json, os, random, sys
from collections import deque
from itertools import islice
from datetime import datetime
from urllib.parse import urljoin, urlparse, urlunparse


def norm(u):
    p = urlparse(u)
    return urlunparse(p._replace(fragment="")).rstrip("/")


def routes(crawl, app, max_routes=50, rng=None):
    """Per page: every shortest route (up to max_routes), and one of them chosen at random."""
    rng = rng or random.Random()
    pages = crawl.get("pages", [])
    by_url = {norm(p["url"]): p for p in pages}
    if not pages:
        return {"app": app, "generatedAt": datetime.now().isoformat(timespec="seconds"), "start": None, "routes": [], "unreachable": []}
    start = norm(crawl.get("baseUrl") or pages[0]["url"])
    links = {}  # page -> [(target, hop)], one edge per target, in URL order
    for p in pages:
        src, seen, out = norm(p["url"]), set(), []
        for e in p.get("elements", []):
            if not e.get("href"):
                continue
            t = norm(urljoin(p["url"], e["href"]))
            if t == src or t not in by_url or t in seen:
                continue
            seen.add(t)
            out.append((t, {"locator": e.get("locator"), "href": e["href"], "stable": bool(e.get("stable"))}))
        links[src] = sorted(out, key=lambda x: x[0])
    # breadth-first: the distance of each page, and every edge that enters it on a shortest route
    dist, into = {start: 0}, {start: []}
    queue = deque([start])
    while queue:
        u = queue.popleft()
        for t, hop in links.get(u, []):
            if t not in dist:
                dist[t], into[t] = dist[u] + 1, []
                queue.append(t)
            if dist[t] == dist[u] + 1:
                into[t].append((u, dict(hop, **{"from": u, "to": t})))
    found = []
    for url in sorted(by_url):
        if url not in dist:
            continue
        ways = list(islice(_shortest(url, start, into), max_routes))
        chosen = ways[rng.randrange(len(ways))]
        found.append({"url": by_url[url]["url"], "clicks": dist[url], "hops": chosen, "routes": ways})
    return {"app": app, "generatedAt": datetime.now().isoformat(timespec="seconds"), "start": start, "routes": found,
            "unreachable": [by_url[u]["url"] for u in sorted(by_url) if u not in dist]}


def _shortest(node, start, into):
    """Every hop list from start to node that uses only shortest-route edges, in a stable order."""
    if node == start:
        yield []
        return
    for u, hop in into.get(node, []):
        for path in _shortest(u, start, into):
            yield path + [hop]


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--crawl", required=True, help="the app map (<App>.crawl.json)")
    ap.add_argument("--app", required=True)
    ap.add_argument("--out", help="write here instead of stdout")
    ap.add_argument("--max-routes", type=int, default=50, help="most shortest routes kept per page")
    ap.add_argument("--seed", type=int, help="pick the same route on every run")
    a = ap.parse_args(argv)
    with open(a.crawl, encoding="utf-8") as f:
        result = routes(json.load(f), a.app, a.max_routes, random.Random(a.seed))
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
