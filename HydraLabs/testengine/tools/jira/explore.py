"""State exploration by input: fill the text fields and pick the dropdown options of one page, snapshotting after each
action. It never clicks a button: buttons write data, and they are only clicked inside a case recipe (the deploy gate).

  python3 explore.py --crawl <crawl.json> --path /Tarjetas --base-url http://127.0.0.1:3000 \
      --value "Ingresa la cuenta=QA_CARD" --runs 3 --out <file>

--value NAME=ENVNAME fills the text field NAME with the environment variable ENVNAME (read from the project .env).
"""
import argparse, json, os, sys
from urllib.parse import urlparse

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import discover  # noqa: E402  (transitions and the 3-passes gate are shared with the case discovery)


def page_elements(crawl, path):
    """The named fields and dropdowns of the page at `path` (links and buttons are not inputs here)."""
    with open(crawl, encoding="utf-8") as f:
        data = json.load(f)
    for page in data.get("pages", []):
        if urlparse(page.get("url", "")).path == path:
            return page.get("elements", [])
    return []


def plan(elements, values):
    """-> ordered actions: ("fill", name, text) for text fields with a value, ("select", name, option) per option.
    A text field with no value is left alone; a dropdown with no options is skipped. Buttons never appear here."""
    actions = []
    for e in elements:
        if e.get("role") == "textbox" and e.get("name") in values:
            actions.append(("fill", e["name"], values[e["name"]]))
    for e in elements:
        if e.get("role") == "combobox" and e.get("name"):
            for opt in e.get("options") or []:
                actions.append(("select", e["name"], opt))
    return actions


def writes_allowed(base_url):
    """Clicks that write data only run against a local app or one labelled as a test environment (QA, dev, preprod)."""
    host = (urlparse(base_url).hostname or "").lower()
    if host in ("127.0.0.1", "localhost", "::1") or host.endswith((".test", ".localhost")):
        return True
    labels = host.replace("-", ".").split(".")
    return any(l in ("qa", "dev", "dev1", "preprod", "pre", "staging", "test") or l.startswith(("qa", "dev", "preprod"))
               for l in labels)


def controls_of(elements):
    return [(e["role"], e["name"]) for e in elements if e.get("name") and e.get("role") in ("button", "textbox", "combobox")]


def run_once(base_url, path, elements, values, clicks=()):
    from playwright.sync_api import sync_playwright
    from discover_states import snapshot
    steps = []
    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=True)
        page = browser.new_page()
        page.goto(base_url.rstrip("/") + path)
        page.wait_for_load_state("networkidle")
        controls = controls_of(elements)
        steps.append({"step": "open " + path, "controls": _snap(page, controls, snapshot)})
        for kind, name, value in plan(elements, values):
            try:
                if kind == "fill":
                    page.get_by_role("textbox", name=name, exact=True).first.fill(value)
                    label = f'fill "{name}"'
                else:
                    loc = page.get_by_role("combobox", name=name, exact=True).first
                    loc.click()
                    page.get_by_role("option", name=value, exact=True).first.click()
                    label = f'select "{name}" = {value}'
            except Exception as err:  # a control that is not there right now: note it, keep going
                label = f'{kind} "{name}" failed: {type(err).__name__}'
            steps.append({"step": label, "controls": _snap(page, controls, snapshot)})
        for name in clicks:  # only the buttons named on the command line, after the inputs, in the order given
            try:
                page.get_by_role("button", name=name, exact=True).first.click()
                label = f'click "{name}"'
            except Exception as err:
                label = f'click "{name}" failed: {type(err).__name__}'
            steps.append({"step": label, "controls": _snap(page, controls, snapshot)})
        browser.close()
    return steps


def _snap(page, controls, snapshot):
    # the plugin's snapshot reads a module-level list; set it to this page's controls, then read it
    import discover_states
    discover_states._controls[:] = controls
    return snapshot(page)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--crawl", required=True)
    ap.add_argument("--path", required=True)
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--value", action="append", default=[], help="NAME=ENVNAME for a text field")
    ap.add_argument("--env", default="", help="project .env, for the ENVNAME values")
    ap.add_argument("--click", action="append", default=[], help="a button to click, in order (writes data)")
    ap.add_argument("--allow-writes", action="store_true", help="required with --click")
    ap.add_argument("--runs", type=int, default=3)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    env = discover.load_env(os.path.dirname(a.env)) if a.env else {}
    values = {}
    for pair in a.value:
        name, var = pair.split("=", 1)
        values[name] = env.get(var) or os.environ.get(var, "")
    if a.click and not (a.allow_writes and writes_allowed(a.base_url)):
        sys.exit("refused: --click writes data; it needs --allow-writes and a local or QA/dev/preprod host, got " + a.base_url)
    elements = page_elements(a.crawl, a.path)
    runs = [transitions_of(run_once(a.base_url, a.path, elements, values, a.click)) for _ in range(a.runs)]
    kept = discover.gate(runs, need=a.runs)
    result = {"path": a.path, "runs": a.runs,
              "candidates": [{"after_action": s, "control": c, "field": f, "from": x, "to": y} for s, c, f, x, y in kept]}
    os.makedirs(os.path.dirname(os.path.abspath(a.out)), exist_ok=True)
    with open(a.out, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
    print(json.dumps(result, ensure_ascii=False, indent=2))


def transitions_of(steps):
    return discover.transitions(steps)


if __name__ == "__main__":
    main()
