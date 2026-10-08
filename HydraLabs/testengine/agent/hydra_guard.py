"""pytest plugin (load with -p hydra_guard): generated tests run read-only. Any real non-GET request is aborted and the test
fails loudly. Page-level routes (a test's own RequestMock) take precedence over this context-level guard, so deliberately
mocked POSTs still work.

HYDRA_ALLOW_WRITES=1 lets writes through, but ONLY to loopback hosts (or those named in HYDRA_WRITE_HOSTS): a local
frontend started in PROD mode would otherwise send real writes to production. Same rule as tools/jira/support/harness.py."""
import os
from urllib.parse import urlparse

import pytest

LOOPBACK = {"127.0.0.1", "localhost", "::1"}


def write_allowed(method, host, allow_writes, extra_hosts=()):
    return method in ("GET", "HEAD", "OPTIONS") or (allow_writes and (host in LOOPBACK or host in extra_hosts))


@pytest.fixture(autouse=True)
def _hydra_guard(request):
    if "driver" not in request.fixturenames:
        yield
        return
    driver = request.getfixturevalue("driver")
    allow = os.environ.get("HYDRA_ALLOW_WRITES") == "1"
    extra = [h for h in os.environ.get("HYDRA_WRITE_HOSTS", "").split(",") if h]
    blocked = []

    def guard(route, req):
        if write_allowed(req.method, urlparse(req.url).hostname, allow, extra):
            route.continue_()
        else:
            blocked.append(f"{req.method} {req.url}")
            route.abort()

    driver.page.context.route("**/*", guard)
    yield
    if blocked:
        pytest.fail("HydraLabs bloqueó escrituras reales (solo lectura, o fuera de loopback): " + "; ".join(blocked))
