# utils/test_files.py
#
# Generado por HydraLabs. Herramienta genérica: arma archivos de prueba EN MEMORIA para subirlos con
# Playwright (page.set_input_files acepta {"name", "mimeType", "buffer"}), sin escribir archivos temporales.

import csv
import io
import random


def binary_payload(name, mime="application/octet-stream", size=2048, seed=0):
    """Archivo binario determinista (contiene bytes NUL, por lo tanto nunca es texto/CSV válido)."""
    rng = random.Random(seed)
    data = b"\x00\xff" + bytes(rng.randrange(256) for _ in range(max(size, 2) - 2))
    return {"name": name, "mimeType": mime, "buffer": data}


def csv_payload(name, header, rows, mime="text/csv"):
    """CSV con el encabezado y las filas dadas (valores arbitrarios: vacíos, enormes, con comillas, etc.)."""
    out = io.StringIO()
    writer = csv.writer(out, lineterminator="\n")
    writer.writerow(header)
    writer.writerows(rows)
    return {"name": name, "mimeType": mime, "buffer": out.getvalue().encode("utf-8")}
