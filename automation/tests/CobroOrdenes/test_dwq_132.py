# tests/CobroOrdenes/test_dwq_132.py
#
# DWQ-132 - TC-3 – Invalid input and backend failure under duplicate submission (Negative / Resilience)
# Generado por HydraLabs desde Jira. Pasos pendientes: 2

import os
import pytest

from apps.CobroOrdenes.modules import cobro_ordenes_module as co
from utils.csv_reader import cargar_datos_csv
from config.settings import Config
from utils.templete_report import set_business_mode, add_step

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

CASO_ID = 'DWQ-132'


def test_dwq_132_invalid_input_and_backend_failure_under_duplicat(driver):
    pytest.skip("Pasos sin automatizar: 2: Upload evil.csv.exe (binary content, application/octet-stream). | 3: Upload junk.csv containing: an empty order number; an unknown store (XXXX); a 10,000-character order number; an SQL-injection string ('; DROP TABLE x;--)")

    set_business_mode(True)

    # ----- Precondición -----
    add_step('Precondición: The POST endpoint is mocked to return HTTP 500 after a 1.5 s delay. A request counter is attached.', level="business")

    # ----- Paso 1: Navigate to /cobro_ordenes. -----
    add_step('Navigate to /cobro_ordenes.', level="business")
    co.navegar_a_cobro_ordenes_flujo(driver)

    # ----- Paso 2: Upload evil.csv.exe (binary content, application/octet-stream). -----
    add_step('Upload evil.csv.exe (binary content, application/octet-stream).', level="business")
    # PENDIENTE: sin regla para este paso

    # ----- Paso 3: Upload junk.csv containing: an empty order number; an unknown store (XXXX); a 10,000-character order number; an SQL-injection string ('; DROP TABLE x;--) -----
    add_step("Upload junk.csv containing: an empty order number; an unknown store (XXXX); a 10,000-character order number; an SQL-injection string ('; DROP TABLE x;--)", level="business")
    # PENDIENTE: sin regla para este paso

    # ----- Paso 4: Double-click "Subir y Procesar". -----
    add_step('Double-click "Subir y Procesar".', level="business")
    co.doble_clic_subir_y_procesar_flujo(driver)

    # ----- Paso 5: Wait for the response. -----
    add_step('Wait for the response.', level="business")
    co.esperar_respuesta_flujo(driver)

    # ----- Resultados esperados -----
    add_step('Esperado: The non-CSV file is rejected client-side and "Subir y Procesar" stays disabled.', level="business")
    # PENDIENTE: resultado esperado sin aserción automática
    add_step('Esperado: The double-click sends at most one POST request (no duplicate charge).', level="business")
    # PENDIENTE: resultado esperado sin aserción automática
    add_step('Esperado: An error toast or alert is shown to the user.', level="business")
    # PENDIENTE: resultado esperado sin aserción automática
    add_step('Esperado: The button returns to the enabled state with no stuck spinner.', level="business")
    # PENDIENTE: resultado esperado sin aserción automática
    add_step('Esperado: No Next.js "Application error" crash screen appears.', level="business")
    co.verificar_texto_oculto_flujo(driver, 'Application error')

    driver.delete_all_cookies()
