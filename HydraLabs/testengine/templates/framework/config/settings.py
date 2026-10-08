import os
from dotenv import load_dotenv

_env_file = os.getenv("ENV_FILE")
if _env_file and os.path.exists(_env_file):
    load_dotenv(_env_file, override=True)
else:
    load_dotenv(override=True)


class Config:
    BROWSER = os.getenv("BROWSER", "chrome").lower()
    BASE_URL = os.getenv("BASE_URL")

    PLAYWRIGHT_TIMEOUT = int(os.getenv("PLAYWRIGHT_TIMEOUT", os.getenv("EXPLICIT_WAIT", 10)))

    HEADLESS = os.getenv("HEADLESS", "false").lower() == "true"

    USER_ADMIN_NAME = os.getenv("USER_ADMIN_NAME")
    USER_ADMIN_PASS = os.getenv("USER_ADMIN_PASS")

    USER_SALE_NAME = os.getenv("USER_SALE_NAME")
    USER_SALE_PASS = os.getenv("USER_SALE_PASS")

    USER_INACTIVE_NAME = os.getenv("USER_INACTIVE_NAME")
    USER_INACTIVE_PASS = os.getenv("USER_INACTIVE_PASS")

    REPORT_PATH = os.path.join(os.getcwd(), "reports", "report.html")
