import logging
import os
from datetime import datetime


def get_logger(name):
    if not hasattr(get_logger, "_log_file_name"):
        log_dir = os.path.join(os.getcwd(), "logs")
        os.makedirs(log_dir, exist_ok=True)
        get_logger._log_file_name = os.path.join(log_dir, f"test_run_{datetime.now().strftime('%Y%m%d_%H%M%S')}.log")

    logger = logging.getLogger(name)

    if not logger.handlers:
        logger.setLevel(logging.INFO)
        formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')

        console_handler = logging.StreamHandler()
        console_handler.setFormatter(formatter)
        logger.addHandler(console_handler)

        file_handler = logging.FileHandler(get_logger._log_file_name)
        file_handler.setFormatter(formatter)
        logger.addHandler(file_handler)

    return logger
