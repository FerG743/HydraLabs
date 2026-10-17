"""What Hydra's generated tests need, and nothing else: one browser for the run, a `driver` per test, state cleared
between tests. With --html (hydra deliver --show) the report gets each test's steps and final screen, and the run's
video is linked from the summary."""
import base64
import os

import pytest
from playwright.sync_api import sync_playwright

from config.settings import Config
from utils import templete_report as tr

VIEWPORT = {"width": 1280, "height": 900}


class Driver:
    """The generated code uses driver.page (Playwright), driver.get(url) and driver.delete_all_cookies()."""

    def __init__(self, page):
        self.page = page

    def get(self, url):
        self.page.goto(url)
        try:  # let the app settle (client code, first data); a page that never goes quiet is still usable
            self.page.wait_for_load_state("networkidle", timeout=5000)
        except Exception:
            pass

    def delete_all_cookies(self):
        self.page.context.clear_cookies()


@pytest.fixture(scope="session")
def _browser(request):
    html = request.config.getoption("htmlpath", default=None)
    videos = os.path.join(os.path.dirname(os.path.abspath(html)), "videos") if html else None
    with sync_playwright() as pw:
        browser = pw.chromium.launch(headless=Config.HEADLESS, slow_mo=int(os.getenv("SLOW_MO", "0") or 0))
        context = browser.new_context(viewport=VIEWPORT, ignore_https_errors=True,
                                      **({"record_video_dir": videos, "record_video_size": VIEWPORT} if videos else {}))
        page = context.new_page()
        page.set_default_timeout(Config.PLAYWRIGHT_TIMEOUT * 1000)
        yield page
        context.close()  # writes the video
        if videos and page.video:
            request.config._hydra_video = page.video.path()
        browser.close()


@pytest.fixture
def driver(_browser):
    tr.STEPS.clear()
    yield Driver(_browser)
    try:  # the page is shared: a job id or other storage left by this test would change the next one's UI
        _browser.evaluate("() => { try { localStorage.clear(); sessionStorage.clear(); } catch (e) {} }")
    except Exception:
        pass
    _browser.context.clear_cookies()
    tr.set_business_mode(False)


@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_makereport(item, call):
    outcome = yield
    report = outcome.get_result()
    html = item.config.pluginmanager.getplugin("html")
    page = item.funcargs.get("driver") and item.funcargs["driver"].page
    if report.when != "call" or not html or not page:
        return
    extras = getattr(report, "extras", [])
    if tr.STEPS:
        extras.append(html.extras.html("<ol>" + "".join(f"<li>{s}</li>" for s in tr.STEPS) + "</ol>"))
    try:
        extras.append(html.extras.png(base64.b64encode(page.screenshot()).decode(), "Final screen"))
    except Exception:
        pass
    report.extras = extras


def pytest_html_results_summary(prefix, summary, postfix, session):
    video = getattr(session.config, "_hydra_video", None)
    if video:
        prefix.append(f'<p>Video of the run: <a href="videos/{os.path.basename(video)}">{os.path.basename(video)}</a></p>')
