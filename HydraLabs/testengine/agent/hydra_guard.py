"""pytest plugin (load with -p hydra_guard): generated tests run read-only. Any real non-GET request is aborted and the test
fails loudly. Page-level routes (a test's own RequestMock) take precedence over this context-level guard, so deliberately
mocked POSTs still work. Set HYDRA_ALLOW_WRITES=1 to disable (never the default)."""
import os

import pytest


@pytest.fixture(autouse=True)
def _hydra_readonly(request):
    if os.environ.get("HYDRA_ALLOW_WRITES") == "1" or "driver" not in request.fixturenames:
        yield
        return
    driver = request.getfixturevalue("driver")
    blocked = []

    def guard(route, req):
        if req.method in ("GET", "HEAD", "OPTIONS"):
            route.continue_()
        else:
            blocked.append(f"{req.method} {req.url}")
            route.abort()

    driver.page.context.route("**/*", guard)
    yield
    if blocked:
        pytest.fail("HydraLabs bloqueó escrituras reales (modo solo lectura): " + "; ".join(blocked))
