"""pytest plugin for state discovery: records each named control's enabled and visible state at every step.

Loaded with `-p discover_states` (PYTHONPATH must include this folder). It wraps the framework's add_step and
DriverWrapper so nothing in the generated test changes. Each add_step call takes a snapshot, so the change between two
snapshots is what the step before the second call did. Written to the file in DISCOVER_OUT when the session ends.
"""
import json
import os

_steps = []      # [{"step": text, "controls": {name: {"enabled": bool, "visible": bool}}}]
_driver = {}     # the last driver the framework wrapped (its .page is the Playwright page)
_controls = []   # [(role, name)] from the crawl


def snapshot(page):
    out = {}
    for role, name in _controls:
        try:
            loc = page.get_by_role(role, name=name, exact=True).first
            if loc.count() == 0:  # not on this page right now: never wait for it (a wait is 30 s per control)
                out[name] = {"visible": False, "enabled": None}
                continue
            out[name] = {"visible": loc.is_visible(), "enabled": loc.is_enabled(timeout=1000)}
        except Exception:
            out[name] = {"visible": False, "enabled": None}
    return out


def _install():
    from utils import templete_report, driver_wrapper
    real_add_step = templete_report.add_step
    real_init = driver_wrapper.DriverWrapper.__init__

    def add_step(message, status=None, level=None):
        page = _driver.get("page")
        if page is not None:
            _steps.append({"step": message, "controls": snapshot(page)})
        return real_add_step(message, status, level)

    def init(self, driver):
        _driver["page"] = getattr(driver, "page", None)
        real_init(self, driver)

    templete_report.add_step = add_step
    driver_wrapper.DriverWrapper.__init__ = init


def _load_controls():
    crawl = os.environ.get("DISCOVER_CRAWL", "")
    if not crawl or not os.path.exists(crawl):
        return
    with open(crawl, encoding="utf-8") as f:
        data = json.load(f)
    seen = set()
    for page in data.get("pages", []):
        for e in page.get("elements", []):
            role, name = e.get("role"), e.get("name")
            if role in ("button", "textbox", "combobox") and name and (role, name) not in seen:
                seen.add((role, name))
                _controls.append((role, name))


def pytest_configure(config):
    _load_controls()
    try:
        _install()
    except ImportError:
        pass


def pytest_sessionfinish(session, exitstatus):
    out = os.environ.get("DISCOVER_OUT")
    if out:
        with open(out, "w", encoding="utf-8") as f:
            json.dump(_steps, f, ensure_ascii=False)
