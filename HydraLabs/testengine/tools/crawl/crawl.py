#!/usr/bin/env python3
"""Read-only crawler: every reachable page of a web app -> every interactive element with its best locator.

Why Python and not Go: it uses the same Playwright engine the generated tests run on, so a locator recorded here is
resolved by exactly the code that will use it later.

Read-only by construction: every non-GET request is aborted, links that look like logout/delete are never followed,
and clicks are limited to --safe-clicks (open a combobox/tab to record its options, then Escape). Typed values in the
login are never written to events or output (they come from ENV: tokens, same convention as the framework's CSVs).

    python3 crawl.py --base-url http://host:3000 --out appmap/App.crawl.json --events out/crawl/events.jsonl \
        [--paths /cobro_ordenes,/ordenes] [--max-pages 25] [--safe-clicks] [--headed] [--slow 150] [--login login.json] [--env-file .env]

--slow MS reveals the found elements one at a time (boxes + locator labels, colored by quality) for showing people; the viewer is
`hydra crawl --watch :8099`, and --headed also shows the real browser scrolling along.

login.json: [{"fill": "#user", "value": "ENV:CORREO_ADMIN"}, {"fill": "#pw", "value": "ENV:MI_PASSWORD"}, {"click": "button[type=submit]"}]
"""
import argparse, hashlib, json, os, re, sys, time
from urllib.parse import urljoin, urlparse, urldefrag

from playwright.sync_api import sync_playwright

AVOID = re.compile(r"logout|log-out|signout|sign-out|salir|cerrar.?sesi|delete|eliminar|borrar|remove", re.I)
UNSTABLE_ID = re.compile(r"^(radix-|:r|react-|headlessui-|mui-|\d)|\d{4,}")  # generated ids change between loads

EXTRACT = """() => {
  const sel = 'a[href],button,input,select,textarea,[role=combobox],[role=tab],[role=checkbox],[role=switch],[role=radio],[role=menuitem],[role=option],[role=link],[role=button]';
  const implicit = {A:'link',BUTTON:'button',SELECT:'combobox',TEXTAREA:'textbox'};
  const itype = {checkbox:'checkbox',radio:'radio',button:'button',submit:'button',file:'button',number:'spinbutton',range:'slider'};
  const vis = e => { const r = e.getBoundingClientRect(), s = getComputedStyle(e);
    return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none'; };
  const nameOf = e => {
    const a = e.getAttribute('aria-label'); if (a) return a.trim();
    const lb = e.getAttribute('aria-labelledby');
    if (lb) { const t = lb.split(/\\s+/).map(i => (document.getElementById(i)||{}).innerText||'').join(' ').trim(); if (t) return t; }
    if (e.id) { const l = document.querySelector('label[for="'+CSS.escape(e.id)+'"]'); if (l && l.innerText.trim()) return l.innerText.trim(); }
    const wl = e.closest('label'); if (wl && wl.innerText.trim()) return wl.innerText.trim();
    return (e.getAttribute('placeholder') || e.innerText || e.getAttribute('title') || e.getAttribute('alt') || e.value || '').trim().replace(/\\s+/g,' ').slice(0,80);
  };
  const out = [];
  document.querySelectorAll(sel).forEach(e => {
    if (!vis(e)) return;
    const tag = e.tagName; let role = e.getAttribute('role') || implicit[tag];
    if (tag === 'INPUT') role = e.getAttribute('role') || itype[e.type] || 'textbox';
    const b = e.getBoundingClientRect();
    out.push({rect: {x: b.left + scrollX, y: b.top + scrollY, w: b.width, h: b.height}, tag: tag.toLowerCase(), role: role || null, name: nameOf(e), id: e.id || null,
      testid: e.getAttribute('data-testid') || e.getAttribute('data-test') || e.getAttribute('data-qa') || null,
      type: e.getAttribute('type'), placeholder: e.getAttribute('placeholder'), href: e.getAttribute('href'),
      disabled: e.disabled === true || e.getAttribute('aria-disabled') === 'true',
      expanded: e.getAttribute('aria-expanded')});
  });
  return out;
}"""


def best_locator(el):
    """Framework priority: data-testid, then a stable id, then role+accessible name, then a CSS fallback."""
    if el["testid"]:
        return f'[data-testid="{el["testid"]}"]', "testid", True
    if el["id"] and not UNSTABLE_ID.search(el["id"]):
        return f'#{el["id"]}', "id", True
    if el["role"] and el["name"]:
        return f'role={el["role"]}[name="{el["name"].replace(chr(34), chr(92) + chr(34))}"]', "role+name", True
    css = el["tag"] + (f'[type="{el["type"]}"]' if el["type"] else "")
    return css, "css-fallback", False  # not unique and not meaningful: flagged so nobody trusts it blindly


GREEN, BLUE, ORANGE = "#16a34a", "#2563eb", "#ea580c"  # stable id/testid | role+name | weak or not unique


def color_of(el):
    if not el["stable"]:
        return ORANGE
    return GREEN if el["strategy"] in ("testid", "id") else BLUE


def label_of(el):
    return el["locator"] if el["strategy"] in ("testid", "id") else f'{el["role"]} "{el["name"][:22]}"'


DRAW = """(items) => { for (const [r, color, label] of items) {
  const d = document.createElement('div'); d.className = '__hydra';
  d.style.cssText = `position:absolute;left:${r.x}px;top:${r.y}px;width:${r.w}px;height:${r.h}px;border:2px solid ${color};background:${color}22;z-index:2147483647;pointer-events:none;box-sizing:border-box`;
  const l = document.createElement('span'); l.textContent = label;
  l.style.cssText = `position:absolute;left:-2px;top:-15px;font:10px/13px monospace;color:#fff;background:${color};padding:0 3px;white-space:nowrap;max-width:240px;overflow:hidden`;
  d.appendChild(l); document.body.appendChild(d); } }"""


def draw(page, els, slow_ms, ev, shots):
    """Boxes + locator labels over what the crawler found. With slow_ms they appear one by one (and as events), so a
    person watching the real browser or the viewer sees the elements being discovered; otherwise all at once."""
    items = [(el["rect"], color_of(el), label_of(el)) for el in els if el["rect"]["w"] > 0 and el["rect"]["h"] > 0]
    if not slow_ms:
        page.evaluate(DRAW, items)
        return
    for n, (el, item) in enumerate(zip([e for e in els if e["rect"]["w"] > 0 and e["rect"]["h"] > 0], items), 1):
        page.evaluate(DRAW, [item])
        page.evaluate("(y) => window.scrollTo(0, Math.max(0, y - 280))", el["rect"]["y"])  # follow the element (a headed browser scrolls too)
        live = n % 3 == 0 or n == len(items)  # an interim picture every few elements keeps the viewer in step
        if live:  # viewport-sized on purpose: a full page of 90 elements shrinks the labels to nothing
            page.screenshot(path=os.path.join(shots, "live.png"))
        ev.emit("element", url=page.url, role=el["role"], name=el["name"][:40], locator=el["locator"], strategy=el["strategy"],
                stable=el["stable"], color=item[1], live=live, n=n, of=len(items))
        page.wait_for_timeout(slow_ms)


def resolve_env(v):
    return os.environ.get(v[4:], v) if isinstance(v, str) and v.startswith("ENV:") else v


def load_env(path):
    for line in open(path, encoding="utf-8") if path and os.path.exists(path) else []:
        k, sep, v = line.strip().partition("=")
        if sep and not k.startswith("#"):
            os.environ.setdefault(k.strip(), v.strip())


class Events:
    def __init__(self, path):
        self.f = open(path, "w", encoding="utf-8") if path else None

    def emit(self, kind, **data):
        line = {"t": round(time.time(), 2), "kind": kind, **data}
        if self.f:
            self.f.write(json.dumps(line, ensure_ascii=False) + "\n")
            self.f.flush()
        print(f"[{kind}] " + " ".join(f"{k}={v}" for k, v in data.items() if k != "shot"), file=sys.stderr)


def record_options(page, el, ev):
    """--safe-clicks: open a combobox/tab just to read its options, then close it. Never selects anything."""
    loc = el["locator"]
    try:
        page.locator(loc).first.click(timeout=2000)
        page.wait_for_timeout(250)
        opts = [t.strip() for t in page.get_by_role("option").all_inner_texts() if t.strip()][:40]
        page.keyboard.press("Escape")
        return opts
    except Exception as e:  # a failed peek is information, not an error
        ev.emit("note", url=page.url, text=f"could not open {loc}: {type(e).__name__}")
        return []


def crawl(a):
    load_env(a.env_file)
    ev = Events(a.events)
    base = a.base_url.rstrip("/")
    origin = urlparse(base).netloc
    shots = os.path.join(os.path.dirname(a.events or a.out), "shots")
    os.makedirs(shots, exist_ok=True)
    queue = [base + p for p in (a.paths.split(",") if a.paths else ["/"])]
    seen, pages = set(), []
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=not a.headed)
        page = browser.new_page(viewport={"width": 1280, "height": 800})
        page.route("**/*", lambda route, req: route.continue_() if req.method == "GET" else route.abort())
        if a.login:
            page.goto(base + (a.login_path or "/"), wait_until="networkidle")
            for step in json.load(open(a.login, encoding="utf-8")):
                if "fill" in step:
                    page.fill(step["fill"], resolve_env(step["value"]))
                    ev.emit("login", action="fill", selector=step["fill"], value="***")  # never the value
                else:
                    page.click(step["click"])
                    ev.emit("login", action="click", selector=step["click"])
            page.wait_for_load_state("networkidle")
        while queue and len(pages) < a.max_pages:
            url = urldefrag(queue.pop(0))[0]
            if url in seen:
                continue
            seen.add(url)
            try:
                page.goto(url, wait_until="networkidle", timeout=20000)
            except Exception as e:
                ev.emit("error", url=url, text=f"{type(e).__name__}")
                continue
            els = page.evaluate(EXTRACT)
            for el in els:
                el["locator"], el["strategy"], el["stable"] = best_locator(el)
            count = {}
            for el in els:
                count[el["locator"]] = count.get(el["locator"], 0) + 1
            for el in els:  # a locator that matches several elements is not a locator (7 carousel dots share one testid)
                el["unique"] = count[el["locator"]] == 1
                el["stable"] = el["stable"] and el["unique"]
            if a.safe_clicks:
                for el in els:
                    if el["role"] in ("combobox", "tab") and el["expanded"] == "false" and el["stable"]:
                        el["options"] = record_options(page, el, ev)
            sig = hashlib.sha1("|".join(sorted(f'{e["role"]}:{e["name"]}:{e["id"]}' for e in els)).encode()).hexdigest()[:10]
            shot = f"{len(pages):02d}.png"
            if a.slow:
                page.wait_for_timeout(a.slow * 4)  # let the page be seen before the boxes start appearing
            draw(page, els, a.slow, ev, shots)
            page.evaluate("() => window.scrollTo(0, 0)")
            page.screenshot(path=os.path.join(shots, shot), full_page=True)
            page.evaluate("() => document.querySelectorAll('.__hydra').forEach(e => e.remove())")
            for el in els:
                el.pop("rect", None)  # positions are for the picture only; the map stays compact
            pages.append({"url": page.url, "title": page.title(), "hash": sig, "shot": shot, "elements": els,
                          "headings": [h.strip() for h in page.locator("h1,h2,h3").all_inner_texts() if h.strip()][:10]})
            by = {}
            for el in els:
                by[el["strategy"]] = by.get(el["strategy"], 0) + 1
            ev.emit("page", url=page.url, title=page.title(), elements=len(els), strategies=by, shot=shot)
            for el in els:
                if el["href"] and not AVOID.search(el["href"] + " " + el["name"]):
                    nxt = urldefrag(urljoin(page.url, el["href"]))[0]
                    if urlparse(nxt).netloc == origin and nxt not in seen and not nxt.lower().endswith((".pdf", ".zip", ".csv")):
                        queue.append(nxt)
                elif el["href"]:
                    ev.emit("skip", url=el["href"], text="looks destructive or leaves the session; not followed")
        browser.close()
    out = {"baseUrl": base, "crawledAt": time.strftime("%Y-%m-%dT%H:%M:%S"), "readOnly": True, "pages": pages}
    os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
    json.dump(out, open(a.out, "w", encoding="utf-8"), indent=1, ensure_ascii=False)
    total = sum(len(pg["elements"]) for pg in pages)
    weak = sum(1 for pg in pages for e in pg["elements"] if not e["stable"])
    ev.emit("done", pages=len(pages), elements=total, weak=weak, out=a.out)
    return out


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--events", default="")
    ap.add_argument("--paths", default="")
    ap.add_argument("--max-pages", type=int, default=25)
    ap.add_argument("--safe-clicks", action="store_true")
    ap.add_argument("--headed", action="store_true")
    ap.add_argument("--slow", type=int, default=0, metavar="MS", help="reveal elements one at a time, MS apart (for showing people)")
    ap.add_argument("--login", default="")
    ap.add_argument("--login-path", default="")
    ap.add_argument("--env-file", default="")
    crawl(ap.parse_args())
