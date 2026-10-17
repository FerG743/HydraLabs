from config.settings import Config
from utils.templete_report import add_step


class DriverWrapper:
    """The two actions the generated modules use, each recorded as a step."""

    def __init__(self, driver):
        self.page = driver.page

    def click(self, selector, description=None):
        add_step(f"Click en {description or selector}")
        self.page.click(selector, timeout=Config.PLAYWRIGHT_TIMEOUT * 1000)

    def send_keys(self, selector, text, description=None, is_secret=False):
        add_step(f"Ingresando en {description or selector}")  # never the text: it may be a secret
        self.page.fill(selector, text, timeout=Config.PLAYWRIGHT_TIMEOUT * 1000)
