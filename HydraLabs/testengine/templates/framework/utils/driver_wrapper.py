from config.settings import Config
from utils.logger import get_logger
from utils.templete_report import add_step

logger = get_logger("DriverWrapper")


class PlaywrightElementSimulator:
    def __init__(self, page, selector):
        self.page = page
        self.selector = selector

    @property
    def text(self):
        try:
            return self.page.text_content(self.selector) or ""
        except Exception as e:
            logger.error(f"[-] Error al obtener texto del elemento {self.selector}: {e}")
            return ""


class PlaywrightWaitSimulator:
    def __init__(self, page, timeout_ms):
        self.page = page
        self.timeout_ms = timeout_ms

    def until(self, selector):
        try:
            self.page.wait_for_selector(selector, state="visible", timeout=self.timeout_ms)
            return PlaywrightElementSimulator(self.page, selector)
        except Exception as e:
            raise AssertionError(f"El elemento '{selector}' no apareció tras la espera de Playwright.") from e


class DriverWrapper:
    def __init__(self, driver):
        self.driver = driver
        self.page = driver.page
        self.logger = get_logger("DriverWrapper")
        self.wait = PlaywrightWaitSimulator(self.page, Config.PLAYWRIGHT_TIMEOUT * 1000)

    def click(self, selector, description=None):
        label = description if description else f"elemento {selector}"
        msg = f"Click en {label}"
        self.logger.info(msg)
        add_step(msg)
        self.page.click(selector, timeout=Config.PLAYWRIGHT_TIMEOUT * 1000)

    def send_keys(self, selector, text, description=None, is_secret=False):
        label = description if description else f"campo {selector}"
        msg = f"Ingresando en {label}"
        self.logger.info(msg)
        add_step(msg)
        self.page.fill(selector, text, timeout=Config.PLAYWRIGHT_TIMEOUT * 1000)

    def switch_to_frame(self, url_pattern=None):
        import time
        iframe_el = self.driver.page.wait_for_selector("iframe", timeout=Config.PLAYWRIGHT_TIMEOUT * 1000)
        timeout_ms = Config.PLAYWRIGHT_TIMEOUT * 1000
        deadline = time.time() + Config.PLAYWRIGHT_TIMEOUT
        target = None
        while time.time() < deadline:
            target = iframe_el.content_frame()
            if target:
                break
            time.sleep(0.5)
        if not target:
            raise RuntimeError(
                f"No se pudo acceder al content_frame del iframe (patrón: {url_pattern}) tras {Config.PLAYWRIGHT_TIMEOUT}s"
            )
        self.logger.info(f"Cambiando contexto al iframe: {target.url}")
        self.page = target
        self.wait = PlaywrightWaitSimulator(self.page, timeout_ms)

    def get_performance_metrics(self):
        try:
            metrics = self.page.evaluate("""
                () => {
                    const perf = window.performance.getEntriesByType('navigation')[0];
                    if (!perf) return { loadTime: 0, domReady: 0, dnsLookup: 0 };
                    return {
                        'loadTime': perf.loadEventEnd - perf.startTime,
                        'domReady': perf.domContentLoadedEventEnd - perf.startTime,
                        'dnsLookup': perf.domainLookupEnd - perf.domainLookupStart
                    };
                }
            """)
            self.logger.info(f"Métricas de rendimiento: {metrics}")
            return metrics
        except Exception as e:
            self.logger.error(f"Error al capturar métricas de rendimiento: {e}")
            return {
                'loadTime': 0,
                'domReady': 0,
                'dnsLookup': 0
            }
