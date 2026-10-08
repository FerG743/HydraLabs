# utils/xml_reader.py
#
# Utilidad genérica para leer y validar comprobantes fiscales CFDI (XML),
# el esquema estándar del SAT (México) para facturas electrónicas. No es
# específica de ninguna app — vive en utils/ por el mismo criterio que
# pdf_reader.py.
#
# Soporta CFDI 3.3 y 4.0 (los namespaces más comunes). Si tu XML real usa
# un namespace distinto, ajusta NAMESPACES abajo.

import xml.etree.ElementTree as ET

NAMESPACES = {
    "cfdi40": "http://www.sat.gob.mx/cfd/4",
    "cfdi33": "http://www.sat.gob.mx/cfd/3",
    "tfd": "http://www.sat.gob.mx/TimbreFiscalDigital",
}


def _find_comprobante_ns(root) -> str:
    """Detecta si el XML es CFDI 4.0 o 3.3 según el namespace del root."""
    tag = root.tag
    if tag.startswith("{" + NAMESPACES["cfdi40"] + "}"):
        return "cfdi40"
    if tag.startswith("{" + NAMESPACES["cfdi33"] + "}"):
        return "cfdi33"
    raise ValueError(f"No se reconoce el namespace del CFDI: {tag}")


def extract_cfdi_data(xml_path: str) -> dict:
    """
    Parsea un CFDI (XML) y extrae los campos más relevantes para pruebas:
    fecha, forma_pago, metodo_pago, receptor (rfc/nombre/domicilio),
    conceptos (lista) y uuid (del Timbre Fiscal Digital).
    """
    tree = ET.parse(xml_path)
    root = tree.getroot()
    ns_key = _find_comprobante_ns(root)
    cfdi_ns = NAMESPACES[ns_key]
    ns = {"cfdi": cfdi_ns, "tfd": NAMESPACES["tfd"]}

    comprobante_attrs = root.attrib

    receptor_el = root.find("cfdi:Receptor", ns)
    receptor = dict(receptor_el.attrib) if receptor_el is not None else {}

    conceptos = []
    conceptos_container = root.find("cfdi:Conceptos", ns)
    if conceptos_container is not None:
        for concepto_el in conceptos_container.findall("cfdi:Concepto", ns):
            conceptos.append(dict(concepto_el.attrib))

    timbre_el = root.find(".//tfd:TimbreFiscalDigital", ns)
    uuid = timbre_el.attrib.get("UUID") if timbre_el is not None else None

    return {
        "fecha": comprobante_attrs.get("Fecha"),
        "forma_pago": comprobante_attrs.get("FormaPago"),
        "metodo_pago": comprobante_attrs.get("MetodoPago"),
        "serie": comprobante_attrs.get("Serie"),
        "folio": comprobante_attrs.get("Folio"),
        "total": comprobante_attrs.get("Total"),
        "receptor_rfc": receptor.get("Rfc"),
        "receptor_nombre": receptor.get("Nombre"),
        "receptor_domicilio_fiscal": receptor.get("DomicilioFiscalReceptor"),
        "conceptos": conceptos,
        "uuid": uuid,
        "_raw_comprobante_attrs": comprobante_attrs,
    }


def assert_cfdi_matches(datos_cfdi: dict, esperado: dict):
    """
    Compara campos puntuales del CFDI extraído contra los valores esperados
    de la prueba (ej. esperado={"receptor_rfc": "XAXX010101000",
    "forma_pago": "01"}). Lanza AssertionError detallando cualquier
    discrepancia.
    """
    discrepancias = []
    for campo, valor_esperado in esperado.items():
        valor_real = datos_cfdi.get(campo)
        if str(valor_real) != str(valor_esperado):
            discrepancias.append(
                f"{campo}: esperado='{valor_esperado}', real='{valor_real}'"
            )

    assert not discrepancias, (
        "El XML (CFDI) no coincide con los datos esperados:\n"
        + "\n".join(discrepancias)
    )


def assert_cfdi_tiene_conceptos(datos_cfdi: dict, minimo: int = 1):
    """Valida que el CFDI tenga al menos `minimo` conceptos (artículos)."""
    total = len(datos_cfdi.get("conceptos", []))
    assert total >= minimo, (
        f"El XML (CFDI) tiene {total} concepto(s), se esperaban al menos {minimo}."
    )
