#!/usr/bin/env python3
"""Reports which generated locators resolve on the real page (GET only; nothing is clicked or sent).

    python3 verify_locators.py --locators automation/apps/CobroOrdenes/locators/cobro_ordenes_locators.py --url http://host:3000/path
"""
import argparse, sys

from playwright.sync_api import sync_playwright

if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--locators", required=True)
    ap.add_argument("--url", required=True)
    a = ap.parse_args()
    ns = {}
    exec(compile(open(a.locators, encoding="utf-8").read(), a.locators, "exec"), ns)
    cls = next(v for k, v in ns.items() if k.endswith("Locators"))
    rows = [(k, v) for k, v in vars(cls).items() if k.isupper() and isinstance(v, str)
            and not k.startswith(("API_", "CSV_"))]  # selectors only
    bad = 0
    with sync_playwright() as p:
        b = p.chromium.launch(headless=True)
        pg = b.new_page()
        pg.route("**/*", lambda route, req: route.continue_() if req.method == "GET" else route.abort())
        pg.goto(a.url, wait_until="networkidle")
        print(f"{'locator':26} {'matches':>7}  selector")
        for k, v in rows:
            n = pg.locator(v).count()
            bad += n == 0
            print(f"{k:26} {n:>7}  {v}{'   <-- NOT FOUND' if n == 0 else ''}")
        b.close()
    sys.exit(1 if bad else 0)
