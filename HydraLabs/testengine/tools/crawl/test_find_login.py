"""find_login on the kinds of sign-in forms apps really have; each found locator must fill/click the right element."""
import os, sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from crawl import find_login  # noqa: E402

pw = pytest.importorskip("playwright.sync_api")

FORMS = {
    "ids": """<form><label for=u>Email</label><input id=u type=email><label for=p>Password</label><input id=p type=password>
              <button type=submit>Sign in</button></form>""",
    "testids": """<form><input data-testid=login-user name=user><input data-testid=login-pass type=password name=pass>
                  <button data-testid=login-go>Entrar</button></form>""",
    "search box first": """<input placeholder=Search><form><input name=username placeholder=Usuario><input type=password name=pw>
                           <input type=submit value=Ingresar></form>""",
    "no form, JS button": """<div><input aria-label="User name"><input type=password aria-label=Password>
                             <div role=button onclick="">Log in</div><button>Help</button></div>""",
    "generated ids": """<form><input id=":r1:" name=email type=email><input id=":r2:" type=password name=password>
                        <button type=submit id="mui-12345">Continue</button></form>""",
}


@pytest.fixture(scope="module")
def page():
    with pw.sync_playwright() as p:
        try:
            b = p.chromium.launch()
        except Exception as e:
            pytest.skip(f"no chromium: {e}")
        yield b.new_page()
        b.close()


@pytest.mark.parametrize("kind", FORMS)
def test_finds_the_three_fields(page, kind):
    page.set_content(FORMS[kind] + "<script>document.addEventListener('click', e => window.clicked = e.target.outerHTML)</script>")
    r = find_login(page)
    assert r["found"], r
    page.fill(r["userField"], "qa@shop.test")
    page.fill(r["passwordField"], "s3cr3t")
    assert page.evaluate("document.activeElement.type") == "password"
    users = page.evaluate("[...document.querySelectorAll('input')].filter(i => i.value == 'qa@shop.test').map(i => i.outerHTML)")
    assert len(users) == 1 and "Search" not in users[0], (kind, r, users)  # the user field, not the search box
    page.click(r["submit"], timeout=2000)
    assert "Help" not in (page.evaluate("window.clicked") or ""), (kind, r)
    assert not r["weak"], r


def test_says_why_when_there_is_no_sign_in(page):
    page.set_content("<h1>Catalog</h1><a href=/about>About</a>")
    page.wait_for_timeout(10)
    r = find_login(page, wait_ms=200)
    assert not r["found"] and "no password field" in r["why"]
    page.set_content("<form><input name=e><input type=password><input type=password><button>Create account</button></form>")
    assert "several password fields" in find_login(page, wait_ms=200)["why"]


TABBED = """<div role=tablist><button role=tab onclick="document.getElementById('f').hidden=false">Adquirente</button></div>
<div id=f hidden><label for=m>Mensaje</label><select id=m><option>Greet</option></select><button>Enviar</button>
<button onclick="window.deleted=true">Eliminar todo</button></div>"""


def test_walking_a_case_maps_what_its_clicks_reveal_and_never_types(page):
    from crawl import walk_case, EXTRACT

    class Ev:
        def __init__(self): self.got = []
        def emit(self, kind, **d): self.got.append((kind, d.get("text", "")))

    page.set_content(TABBED)
    mapped, ev = [], Ev()
    steps = ['Click "Adquirente".', "Enter order number `98`.", 'Click "Eliminar todo".', 'Click "Nada".']
    walk_case(page, "http://x", steps, lambda url=None: True,
              lambda settled, after=None: mapped.append((after, [e["name"] for e in page.evaluate(EXTRACT)])), ev)
    assert [a for a, _ in mapped] == ['Click "Adquirente".'], mapped  # only the safe click with a real target was walked
    assert "Mensaje" in mapped[0][1] and "Enviar" in mapped[0][1], mapped  # what the tab revealed is in the map now
    assert not page.evaluate("window.deleted"), "a destructive-looking click must never be made"
    skipped = " ".join(t for k, t in ev.got if k == "skip")
    assert "Enter order number" in skipped and "Eliminar todo" in skipped and "Nada" in skipped, ev.got


SHADCN = """<div><label>Mensaje</label><button type=button role=combobox><span>Solicitud de Plan 86 (Tiempo Aire)</span></button></div>
<div><label>Destino</label><button type=button role=combobox><span>sim — 172.22.64.229</span></button></div>
<div><label>Puerto del stub</label><input value=8101></div>
<div><label for=c>Cuenta</label><input id=c></div>
<div><label>Ayuda</label><button>Ver ayuda</button></div>"""


def test_a_field_is_named_by_the_unattached_label_next_to_it(page):
    from crawl import EXTRACT, best_locator
    page.set_content(SHADCN)
    els = {e["name"]: e for e in page.evaluate(EXTRACT)}
    for name, role in (("Mensaje", "combobox"), ("Destino", "combobox"), ("Puerto del stub", "textbox")):
        assert name in els, list(els)  # not "Solicitud de Plan 86...", "sim — ...", "8101": those change with the value
        loc, strategy, stable = best_locator(els[name])
        assert strategy == "label" and stable and page.locator(loc).count() == 1, (name, loc)
        assert page.locator(loc).get_attribute("role") == ("combobox" if role == "combobox" else None) or role == "textbox"
    assert best_locator(els["Cuenta"])[0] == "#c"  # an attached label: the element's own id stays first
    assert "Ver ayuda" in els and not els["Ver ayuda"].get("byLabel")  # a plain button is not a field: its own text names it
