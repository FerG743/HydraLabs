"""pytest plugin (load with -p hydra_guard): generated tests run read-only. Any real non-GET request is aborted and the test
fails loudly. Page-level routes (a test's own RequestMock) take precedence over this context-level guard, so deliberately
mocked POSTs still work.

Writes go through only when the owning team allowed them (the writes config, passed in as env by runner.run_test): HYDRA_ALLOW_WRITES=1,
and only to the hosts in HYDRA_WRITE_HOSTS and, when HYDRA_WRITE_ENDPOINTS is set, only to those "METHOD /path" endpoints.
Nothing is writable by default, loopback included: a local frontend started in PROD mode would otherwise send real writes to production."""
import os
from urllib.parse import urlparse

import pytest

try:  # the framework's sign-in (utils/login.py); repos created before it have none
    from utils import login as _login
except ImportError:
    _login = None

def write_allowed(method, host, allow_writes, extra_hosts=(), path="/", endpoints=()):
    if method in ("GET", "HEAD", "OPTIONS"):
        return True
    if not (allow_writes and host in extra_hosts):
        return False
    return not endpoints or f"{method} {path}" in endpoints


@pytest.fixture(autouse=True)
def _hydra_guard(request):
    if "driver" not in request.fixturenames:
        yield
        return
    driver = request.getfixturevalue("driver")
    allow = os.environ.get("HYDRA_ALLOW_WRITES") == "1"
    extra = [h for h in os.environ.get("HYDRA_WRITE_HOSTS", "").split(",") if h]
    endpoints = [e for e in os.environ.get("HYDRA_WRITE_ENDPOINTS", "").split(",") if e]
    blocked = []

    def guard(route, req):
        u = urlparse(req.url)
        if write_allowed(req.method, u.hostname, allow, extra, u.path or "/", endpoints) or (_login and _login.ACTIVE):
            route.continue_()
        else:
            blocked.append(f"{req.method} {req.url}")
            route.abort()

    driver.page.context.route("**/*", guard)
    yield
    driver.page.context.unroute("**/*", guard)  # the page is shared: left on, it would block the next test's sign-in
    if blocked:
        pytest.fail("HydraLabs bloqueó escrituras reales (solo lectura, o fuera de loopback): " + "; ".join(blocked))
