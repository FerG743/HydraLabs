import os

from dotenv import load_dotenv

load_dotenv()  # .env next to this repo; Hydra passes BASE_URL, HEADLESS and SLOW_MO itself


class Config:
    BASE_URL = os.getenv("BASE_URL")
    HEADLESS = os.getenv("HEADLESS", "false").lower() == "true"
    PLAYWRIGHT_TIMEOUT = int(os.getenv("PLAYWRIGHT_TIMEOUT", "10"))  # seconds
