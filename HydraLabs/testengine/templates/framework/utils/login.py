"""Sign-in before each test, for apps that have one. hydra/login.json (written by hydra init) holds the sign-in page
and its steps; values are ENV:NAME tokens read from the environment or .env, never the secrets themselves. An app
without that file has no sign-in: tests start on it directly. A test marked @pytest.mark.signed_out skips it."""
import json
import os

ACTIVE = False  # True while signing in: Hydra's read-only guard lets these requests through (a sign-in is a POST)


def load(root):
    path = os.path.join(root, "hydra", "login.json")
    if not os.path.exists(path):
        return None
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def _value(v):
    if not (isinstance(v, str) and v.startswith("ENV:")):
        return v
    if v[4:] not in os.environ:  # never type the token itself into a form
        raise RuntimeError(f"sign-in: {v[4:]} is not set (add it to .env)")
    return os.environ[v[4:]]


def sign_in(page, base_url, login):
    global ACTIVE
    ACTIVE = True
    try:
        _steps(page, base_url, login)
    finally:
        ACTIVE = False


def _steps(page, base_url, login):
    page.goto(base_url.rstrip("/") + login.get("path", "/"))
    for step in login["steps"]:
        if "fill" in step:
            page.fill(step["fill"], _value(step["value"]))
        elif "select" in step:
            page.select_option(step["select"], _value(step["value"]))
        elif "keypad" in step:  # on-screen keypad: click each character
            for ch in _value(step["keypad"]):
                page.click(f'role=button[name="{ch}"]')
        elif "wait" in step:
            page.wait_for_timeout(step["wait"])
        else:
            page.click(step["click"])
    try:  # let the app finish signing in (redirect, first data)
        page.wait_for_load_state("networkidle", timeout=5000)
    except Exception:
        pass
