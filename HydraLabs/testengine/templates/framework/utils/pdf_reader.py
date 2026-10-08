# utils/pdf_reader.py
#
# Utilidad genérica y transversal del framework para leer PDFs (ej. facturas
# CFDI descargadas durante las pruebas) y extraer/validar su contenido.
# No es específica de ninguna app, por lo que vive en utils/ (ver README,
# sección "Estructura y Uso de la Carpeta utils").
#
# Requiere agregar a requirements.txt:
#   pdfplumber>=0.10

import re
import pdfplumber

UUID_REGEX = re.compile(
    r"[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}"
)


def extract_invoice_data(pdf_path: str) -> dict:
    """
    Extrae el texto completo de un PDF y busca el UUID fiscal (CFDI),
    que sigue el formato estándar 8-4-4-4-12 en hexadecimal.

    Retorna: {"raw_text": str, "uuid": str | None}
    """
    raw_text = ""
    with pdfplumber.open(pdf_path) as pdf:
        for page in pdf.pages:
            texto_pagina = page.extract_text() or ""
            raw_text += texto_pagina + "\n"

    match = UUID_REGEX.search(raw_text)
    return {"raw_text": raw_text, "uuid": match.group(0) if match else None}


def assert_invoice_contains(raw_text: str, expected_fragments: list[str]) -> None:
    """
    Valida que ciertos fragmentos de texto (ej. RFC, nombre, forma de pago)
    estén presentes en el texto extraído del PDF. Lanza AssertionError con
    el detalle de lo que falta si alguno no se encuentra.
    """
    texto_lower = raw_text.lower()
    faltantes = [f for f in expected_fragments if f.lower() not in texto_lower]

    assert not faltantes, (
        f"La factura en PDF no contiene los siguientes datos esperados: "
        f"{', '.join(faltantes)}"
    )
