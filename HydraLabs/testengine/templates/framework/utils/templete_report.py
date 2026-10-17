"""Steps of the running test (the name is kept: generated tests import it). conftest adds them to the HTML report;
Hydra's verify harness reads STEPS."""
STEPS = []
_business = False


def set_business_mode(enabled):
    global _business
    _business = bool(enabled)


def add_step(message, status=None, level=None):
    STEPS.append(message if not status or status == "passed" else f"{message} ({status})")
