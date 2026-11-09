#!/usr/bin/env python3
"""Read-only crawler: every reachable page of a web app -> every interactive element with its best locator.

Why Python and not Go: it uses the same Playwright engine the generated tests run on, so a locator recorded here is
resolved by exactly the code that will use it later.

Read-only by construction: every non-GET request is aborted, links that look like logout/delete are never followed,
and clicks are limited to --safe-clicks (open a combobox/tab to record its options, then Escape). Typed values in the
login are never written to events or output (they come from ENV: tokens, same convention as the framework's CSVs).

    python3 crawl.py --base-url http://host:3000 --out appmap/App.crawl.json --events out/crawl/events.jsonl \
        [--paths /cobro_ordenes,/ordenes] [--max-pages 25] [--safe-clicks] [--headed] [--slow 150] [--video DIR] [--login login.json] [--env-file .env]

--slow MS reveals the found elements one at a time (boxes + locator labels, colored by quality) for showing people; the viewer is
`hydra crawl --watch :8099`, and --headed also shows the real browser scrolling along.

login.json: [{"fill": "#user", "value": "ENV:CORREO_ADMIN"}, {"fill": "#pw", "value": "ENV:MI_PASSWORD"}, {"click": "button[type=submit]"}]
"""
import argparse, hashlib, json, os, re, sys, time
from urllib.parse import urljoin, urlparse, urldefrag

from playwright.sync_api import sync_playwright

AVOID = re.compile(r"logout|log-out|signout|sign-out|salir|cerrar.?sesi|delete|eliminar|borrar|remove", re.I)
GENERATED_PART = re.compile(r"[0-9A-Z]{20,}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}|\d{5,}")  # ULID, UUID, db ids in a testid
UNSTABLE_ID = re.compile(r"^(radix-|:r|react-|headlessui-|mui-|\d)|\d{4,}")  # generated ids change between loads

EXTRACT = """() => {
  const sel = 'a[href],button,input,select,textarea,[role=combobox],[role=tab],[role=checkbox],[role=switch],[role=radio],[role=menuitem],[role=option],[role=link],[role=button]';
  const implicit = {A:'link',BUTTON:'button',SELECT:'combobox',TEXTAREA:'textbox'};
  const itype = {checkbox:'checkbox',radio:'radio',button:'button',submit:'button',file:'button',number:'spinbutton',range:'slider'};
  const vis = e => { const r = e.getBoundingClientRect(), s = getComputedStyle(e);
    return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none'; };
  const ownName = e => {  // a name the page gives the element itself (what Playwright's role selectors also see)
    const a = e.getAttribute('aria-label'); if (a) return a.trim();
    const lb = e.getAttribute('aria-labelledby');
    if (lb) { const t = lb.split(/\\s+/).map(i => (document.getElementById(i)||{}).innerText||'').join(' ').trim(); if (t) return t; }
    if (e.id) { const l = document.querySelector('label[for="'+CSS.escape(e.id)+'"]'); if (l && l.innerText.trim()) return l.innerText.trim(); }
    const wl = e.closest('label'); if (wl && wl.innerText.trim()) return wl.innerText.trim();
    return '';
  };
  const nameOf = e => ownName(e) ||
    (e.getAttribute('placeholder') || e.innerText || e.getAttribute('title') || e.getAttribute('alt') || e.value || '').trim().replace(/\\s+/g,' ').slice(0,80);
  // a field whose <label> sits right before it but is not attached (no for=): the label, not its current value, names it
  const nearLabel = (e, role) => {
    const p = e.previousElementSibling, control = ['INPUT','SELECT','TEXTAREA'].includes(e.tagName) || role === 'combobox';
    return control && !ownName(e) && p && p.tagName === 'LABEL' && !p.htmlFor ? p.innerText.trim().replace(/\\s+/g,' ').slice(0,80) : '';
  };
  const out = [];
  document.querySelectorAll(sel).forEach(e => {
    if (!vis(e)) return;
    const tag = e.tagName; let role = e.getAttribute('role') || implicit[tag];
    if (tag === 'INPUT') role = e.getAttribute('role') || itype[e.type] || 'textbox';
    const b = e.getBoundingClientRect();
    const near = nearLabel(e, role);
    out.push({rect: {x: b.left + scrollX, y: b.top + scrollY, w: b.width, h: b.height}, tag: tag.toLowerCase(), role: role || null, name: near || nameOf(e), id: e.id || null,
      byLabel: near ? true : undefined,
      testid: e.getAttribute('data-testid') || e.getAttribute('data-test') || e.getAttribute('data-qa') || null,
      type: e.getAttribute('type'), placeholder: e.getAttribute('placeholder'), href: e.getAttribute('href'),
      disabled: e.disabled === true || e.getAttribute('aria-disabled') === 'true',
      expanded: e.getAttribute('aria-expanded'), login: e.dataset.hydraLogin || undefined,
      nameAttr: e.dataset.hydraLogin ? e.getAttribute('name') : undefined});
  });
  return out;
}"""


# Marks the sign-in form's three elements (data-hydra-login) so EXTRACT + best_locator describe them like any other.
# One visible password field; the user field is the last text/email/tel input before it in the same form; the button
# is the one that submits that form, else a button whose text says sign in.
FIND_LOGIN = """() => {
  const vis = e => { const r = e.getBoundingClientRect(), s = getComputedStyle(e);
    return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none'; };
  const pw = [...document.querySelectorAll('input[type=password]')].filter(vis);
  if (pw.length !== 1) return pw.length ? 'several password fields (a sign-up form?)' : 'no password field on this page';
  const p = pw[0], form = p.form || p.closest('form'), scope = form || document.body;
  const texts = [...scope.querySelectorAll('input')].filter(e => vis(e) && ['text', 'email', 'tel'].includes(e.type));
  const before = texts.filter(e => e.compareDocumentPosition(p) & Node.DOCUMENT_POSITION_FOLLOWING);
  const u = before[before.length - 1];
  const words = /log ?in|sign ?in|entrar|acceder|iniciar|ingresar|continuar|continue|submit|next|siguiente/i;
  const btns = [...scope.querySelectorAll('button,input[type=submit],[role=button]')].filter(vis);
  const s = (form && btns.find(b => (b.tagName === 'BUTTON' && b.type === 'submit') || (b.tagName === 'INPUT' && b.type === 'submit')))
    || btns.find(b => words.test(b.innerText || b.value || b.getAttribute('aria-label') || ''));
  if (!u) return 'no user field before the password';
  if (!s) return 'no sign-in button';
  u.dataset.hydraLogin = 'user'; p.dataset.hydraLogin = 'password'; s.dataset.hydraLogin = 'submit';
  return '';
}"""


def find_login(page, wait_ms=30000):
    """The sign-in form on the page as login steps fields, or why it was not found. Read-only: nothing is typed."""
    try:  # a login rendered by client code appears after load; micro-frontend shells take 10-15 s in QA
        page.wait_for_selector("input[type=password]", state="visible", timeout=wait_ms)
    except Exception:
        pass
    why = page.evaluate(FIND_LOGIN)
    if why:
        return {"found": False, "why": why, "path": urlparse(page.url).path}
    els = page.evaluate(EXTRACT)
    out = {"found": True, "path": urlparse(page.url).path or "/", "weak": []}
    for el in els:
        if not el.get("login"):
            continue
        loc, strategy, stable = best_locator(el)
        if el["login"] == "password" and strategy not in ("testid", "id"):
            stable = False  # the one visible password field: input[type=password] says so better than a role guess
        if stable and sum(1 for e in els if best_locator(e)[0] == loc) > 1:
            stable = False  # role + name shared with another element: not a locator
        if not stable:
            loc = {"password": "input[type=password]"}.get(el["login"]) or (f'{el["tag"]}[name="{el["nameAttr"]}"]' if el.get("nameAttr") else loc)
            if el["login"] != "password" and not el.get("nameAttr"):
                out["weak"].append(el["login"])
        out[{"user": "userField", "password": "passwordField", "submit": "submit"}[el["login"]]] = loc
    return out


def path_under(base, url):
    """url's path relative to the base URL (an app under /app: /app/login -> /login); None when outside it."""
    b, u = urlparse(base), urlparse(url)
    bp, up = b.path.rstrip("/"), u.path
    if b.netloc != u.netloc or (up != bp and not up.startswith(bp + "/")):
        return None
    return up[len(bp):] or "/"


def detect_login(a):
    """--detect-login: open the app (it usually redirects to its sign-in page) and print the form's locators."""
    base = a.base_url.rstrip("/")
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        page.route("**/*", lambda route, req: route.continue_() if req.method == "GET" else route.abort())  # read-only
        tried = []
        for path in dict.fromkeys([a.login_path or "/", "/login"]):  # the given (or home) page, then the usual one
            try:
                page.goto(base + path, wait_until="domcontentloaded", timeout=20000)
                try:
                    page.wait_for_load_state("networkidle", timeout=5000)
                except Exception:
                    pass
            except Exception as e:
                tried.append(f"{path}: {type(e).__name__}")
                continue
            r = find_login(page)
            if r["found"]:
                browser.close()
                rel = path_under(base, page.url)
                if rel is None:  # e.g. a single sign-on page on another app: its path cannot be joined to the base URL
                    return {"found": False, "why": f"the sign-in page {page.url} is outside {base}"}
                r["path"] = rel
                return r
            tried.append(f'{r["path"]}: {r["why"]}')
        browser.close()
    return {"found": False, "why": "; ".join(tried)}


def best_locator(el):
    """Framework priority: data-testid, then a stable id, then role+accessible name, then a CSS fallback."""
    if el["testid"] and not GENERATED_PART.search(el["testid"]):
        return f'[data-testid="{el["testid"]}"]', "testid", True
    if el["id"] and not UNSTABLE_ID.search(el["id"]):
        return f'#{el["id"]}', "id", True
    if el.get("byLabel"):  # named by the label next to it, which role selectors cannot see: reach it through that label
        target = el["tag"] if el["tag"] in ("input", "select", "textarea") else f'[role={el["role"]}]'
        return f'label:text-is("{el["name"].replace(chr(34), chr(92) + chr(34))}") + {target}', "label", True
    if el["role"] and el["name"]:
        return f'role={el["role"]}[name="{el["name"].replace(chr(34), chr(92) + chr(34))}"]', "role+name", True
    if el["testid"]:  # only a testid with a record id in it: works here, not after a data reset or in another environment
        return f'[data-testid="{el["testid"]}"]', "testid", False
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
    if not (isinstance(v, str) and v.startswith("ENV:")):
        return v
    if v[4:] not in os.environ:  # never type the token itself into a form
        sys.exit(f"login: {v[4:]} is not set (add it to the --env-file)")
    return os.environ[v[4:]]


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


STEP_NAV = re.compile(r'^Navigate to (/\S*?)\.?$')  # the renderer's own wording (tools/jira/render_pytest.py ACTIONS)
STEP_CLICK = re.compile(r'^(?:Click|Press) "(.+?)"\.?$')


def clickable(page, name):
    """The one visible tab, button, link or menu item with exactly this name; None if there is none or several."""
    for role in ("tab", "button", "link", "menuitem"):
        loc = page.get_by_role(role, name=name, exact=True)
        if loc.count() == 1 and loc.is_visible():
            return loc
    return None


def walk_case(page, base, steps, settle, map_here, ev):
    """Follow a case's navigation and clicks (read-only: a click that would send a form is aborted by the route) and map
    the screen after each one. Other steps (typing, choosing) are skipped: nothing is ever typed into the app."""
    for text in steps:
        t = text.replace("`", "").strip()
        nav, click = STEP_NAV.match(t), STEP_CLICK.match(t)
        try:
            if nav:
                settled = settle(base + nav.group(1))
            elif click and not AVOID.search(click.group(1)):
                target = clickable(page, click.group(1))
                if target is None:
                    ev.emit("skip", text=f"{t} nothing clickable has that exact name on this screen")
                    continue
                target.click(timeout=5000)
                settled = settle()
            else:
                ev.emit("skip", text=f"{t} not walked (only navigation and safe clicks)")
                continue
        except Exception as e:
            ev.emit("error", text=f"{t} {type(e).__name__}")
            continue
        map_here(settled, after=t)


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
        # --video: Playwright records the crawler's own browser (scrolling, boxes appearing with --slow); saved when the context closes
        context = browser.new_context(viewport={"width": 1280, "height": 800},
                                      **({"record_video_dir": a.video, "record_video_size": {"width": 1280, "height": 800}} if a.video else {}))
        page = context.new_page()
        login_now = {"on": False}  # the crawl is read-only; only the login steps may send forms (connection, sign-in)
        page.route("**/*", lambda route, req: route.continue_() if req.method == "GET" or login_now["on"] else route.abort())

        def settle(url=None):
            # DOM ready is required; network quiet is best effort: a third-party tracker that never answers
            # (e.g. an analytics beacon blocked on this network) must not cost the whole page
            if url:
                page.goto(url, wait_until="domcontentloaded", timeout=20000)
            try:
                page.wait_for_load_state("networkidle", timeout=5000)
                return True
            except Exception:
                return False  # mapped anyway, but the page says so: content loaded by those requests may be missing

        if a.login:
            settle(base + (a.login_path or "/"))
            login_now["on"] = True
            for step in json.load(open(a.login, encoding="utf-8")):
                if "fill" in step:
                    page.fill(step["fill"], resolve_env(step["value"]))
                    ev.emit("login", action="fill", selector=step["fill"], value="***")  # never the value
                elif "select" in step:
                    page.select_option(step["select"], resolve_env(step["value"]))
                    ev.emit("login", action="select", selector=step["select"])
                elif "wait" in step:  # the POS screen loads its data after sign-in; settle() returns too early
                    page.wait_for_timeout(step["wait"])
                elif "keypad" in step:  # on-screen keypad: click each character of the value (it ignores typed text)
                    for ch in resolve_env(step["keypad"]):
                        page.click(f'role=button[name="{ch}"]')
                    ev.emit("login", action="keypad", value="***")
                else:
                    page.click(step["click"])
                    ev.emit("login", action="click", selector=step["click"])
            login_now["on"] = False
            settle()
        def map_here(settled, after=None):
            """Every element on screen now -> a page entry of the app map (after: the case step that revealed it)."""
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
            pages.append({"url": page.url, "title": page.title(), "hash": sig, "shot": shot, "settled": settled, "elements": els,
                          "headings": [h.strip() for h in page.locator("h1,h2,h3").all_inner_texts() if h.strip()][:10],
                          **({"after": after} if after else {})})
            by = {}
            for el in els:
                by[el["strategy"]] = by.get(el["strategy"], 0) + 1
            ev.emit("page", url=page.url, title=page.title(), elements=len(els), strategies=by, shot=shot, settled=settled)
            return els

        if a.guide:  # walk one case's steps instead of following links: what a tab or button reveals gets mapped too
            walk_case(page, base, json.load(open(a.guide, encoding="utf-8")), settle, map_here, ev)
            queue = []
        reuse = bool(a.login)  # after sign-in, the page already on screen is not reloaded: on some apps (the POS) that drops the session
        while queue and len(pages) < a.max_pages:
            url = urldefrag(queue.pop(0))[0]
            if url in seen:
                continue
            seen.add(url)
            try:
                # reuse it only when it is the page asked for; any other first path (/orders after landing on /home) is opened
                settled = settle(None if reuse and urldefrag(page.url)[0] == url else url)
                reuse = False
                if urldefrag(page.url)[0] in seen - {url}:  # a redirect to a page already mapped (back to /home, or to the login)
                    continue
                seen.add(urldefrag(page.url)[0])
            except Exception as e:
                ev.emit("error", url=url, text=f"{type(e).__name__}")
                continue
            els = map_here(settled)
            for el in els:
                if el["href"] and not AVOID.search(el["href"] + " " + el["name"]):
                    nxt = urldefrag(urljoin(page.url, el["href"]))[0]
                    if urlparse(nxt).netloc == origin and nxt not in seen and not nxt.lower().endswith((".pdf", ".zip", ".csv")):
                        queue.append(nxt)
                elif el["href"]:
                    ev.emit("skip", url=el["href"], text="looks destructive or leaves the session; not followed")
        video = page.video.path() if a.video else None
        context.close()  # flushes the recording
        browser.close()
    if video:
        os.makedirs(a.video, exist_ok=True)
        final = os.path.join(a.video, time.strftime("crawl-%Y%m%d-%H%M%S.webm"))
        os.replace(video, final)
        ev.emit("video", path=final, mb=round(os.path.getsize(final) / 1e6, 1))
    out = {"baseUrl": base, "crawledAt": time.strftime("%Y-%m-%dT%H:%M:%S"), "readOnly": True, "pages": pages}
    if a.guide and os.path.exists(a.out):  # a walk adds to the app map; a page it mapped again replaces the old entry
        old = json.load(open(a.out, encoding="utf-8"))
        merged = {(pg["url"], pg.get("after")): pg for pg in old["pages"] + pages}  # same screen again: the newest wins
        out = {**old, "pages": list(merged.values())}
    os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
    json.dump(out, open(a.out, "w", encoding="utf-8"), indent=1, ensure_ascii=False)
    total = sum(len(pg["elements"]) for pg in pages)
    weak = sum(1 for pg in pages for e in pg["elements"] if not e["stable"])
    ev.emit("done", pages=len(pages), elements=total, weak=weak, out=a.out)
    return out


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--out", default="")
    ap.add_argument("--events", default="")
    ap.add_argument("--paths", default="")
    ap.add_argument("--max-pages", type=int, default=25)
    ap.add_argument("--safe-clicks", action="store_true")
    ap.add_argument("--headed", action="store_true")
    ap.add_argument("--video", default="", metavar="DIR", help="record the crawl as a .webm video into DIR")
    ap.add_argument("--slow", type=int, default=0, metavar="MS", help="reveal elements one at a time, MS apart (for showing people)")
    ap.add_argument("--login", default="")
    ap.add_argument("--login-path", default="")
    ap.add_argument("--env-file", default="")
    ap.add_argument("--guide", default="", help="JSON list of one case's steps: walk them instead of following links, add to --out")
    ap.add_argument("--detect-login", action="store_true", help="only find the sign-in form and print its locators as JSON")
    args = ap.parse_args()
    if args.detect_login:
        print(json.dumps(detect_login(args)))
    elif not args.out:
        ap.error("--out is required")
    else:
        crawl(args)
