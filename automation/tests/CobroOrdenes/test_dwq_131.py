# tests/CobroOrdenes/test_dwq_131.py
#
# DWQ-131 - TC-2 – Bulk CSV upload using the portal's sample file (Happy Path)
# Generado por HydraLabs desde Jira. Pasos pendientes: ninguno

import os
import pytest

from apps.CobroOrdenes.modules import cobro_ordenes_module as co
from utils.csv_reader import cargar_datos_csv
from config.settings import Config
from utils.templete_report import set_business_mode, add_step

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

CASO_ID = 'DWQ-131'
DOWNLOADS_DIR = os.path.join(ROOT_DIR, "reports", "downloads")


def test_dwq_131_bulk_csv_upload_using_the_portal_s_sample_file_h(driver):

    set_business_mode(True)

    # ----- Precondición -----
    add_step('Precondición: Same as TC-1: Portal is deployed and reachable. The "Un vistazo a tus ordenes" table is empty.', level="business")

    # ----- Paso 1: Navigate to /cobro_ordenes. -----
    add_step('Navigate to /cobro_ordenes.', level="business")
    co.navegar_a_cobro_ordenes_flujo(driver)

    # ----- Paso 2: Click "Descargar CSV" and capture the downloaded file. -----
    add_step('Click "Descargar CSV" and capture the downloaded file.', level="business")
    archivo_descargado = co.descargar_con_descargar_csv_flujo(driver, DOWNLOADS_DIR, CASO_ID)

    # ----- Paso 3: Upload that file via the drop zone. -----
    add_step('Upload that file via the drop zone.', level="business")
    co.subir_archivo_flujo(driver, archivo_descargado)

    # ----- Paso 4: Verify "Subir y Procesar" is enabled. -----
    add_step('Verify "Subir y Procesar" is enabled.', level="business")
    co.verificar_subir_y_procesar_enabled_flujo(driver)

    # ----- Paso 5: Click "Subir y Procesar". -----
    add_step('Click "Subir y Procesar".', level="business")
    co.hacer_clic_subir_y_procesar_flujo(driver)

    # ----- Resultados esperados -----
    add_step('Esperado: The download succeeds.', level="business")
    assert archivo_descargado and os.path.exists(archivo_descargado), "La descarga no generó un archivo."
    add_step('Esperado: Orders 9800074297 (LIVR) and 9800074362 (SBB) appear in the results table.', level="business")
    co.verificar_texto_visible_flujo(driver, '9800074297')
    co.verificar_texto_visible_flujo(driver, '9800074362')
    add_step('Esperado: The sample file round-trips through the upload without errors.', level="business")
    # PENDIENTE: resultado esperado sin aserción automática

    driver.delete_all_cookies()
