import csv
import os


def cargar_datos_csv(file_path):
    """Rows after the header, as tuples. A value ENV:NAME is read from the environment (secrets never live in CSVs)."""
    if not os.path.exists(file_path):
        return []
    with open(file_path, encoding="utf-8", newline="") as f:
        rows = list(csv.reader(f))[1:]
    return [tuple(os.getenv(v[4:], v) if v.startswith("ENV:") else v for v in row) for row in rows if row]
