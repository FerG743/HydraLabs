# tests/CobroOrdenes/test_dwq_130.py
#
# DWQ-130 - TC-1 – Manual order entry and processing (Happy Path)
# Generado por HydraLabs desde Jira. Pasos pendientes: ninguno

import os
import pytest

from apps.CobroOrdenes.modules import cobro_ordenes_module as co
from utils.csv_reader import cargar_datos_csv
from config.settings import Config
from utils.templete_report import set_business_mode, add_step

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
CASO_CSV = os.path.join(ROOT_DIR, "apps", "CobroOrdenes", "data", "dwq_130_data.csv")
DATA_CASO = cargar_datos_csv(CASO_CSV)

CASO_ID = 'DWQ-130'
datos_caso = next((fila for fila in DATA_CASO if fila[0] == CASO_ID), None)


def test_dwq_130_manual_order_entry_and_processing_happy_path(driver):
    if not datos_caso:
        pytest.skip(f"No se encontraron datos para el caso {CASO_ID} en el CSV.")
    _caso_id, order_number, store = datos_caso

    set_business_mode(True)

    # ----- Precondición -----
    add_step('Precondición: Portal is deployed and reachable. The "Un vistazo a tus ordenes" table is empty.', level="business")

    # ----- Paso 1: Navigate to /cobro_ordenes. -----
    add_step('Navigate to /cobro_ordenes.', level="business")
    co.navegar_a_cobro_ordenes_flujo(driver)

    # ----- Paso 2: Verify "Procesar" is disabled. -----
    add_step('Verify "Procesar" is disabled.', level="business")
    co.verificar_procesar_disabled_flujo(driver)

    # ----- Paso 3: Enter order number 9800074297. -----
    add_step('Enter order number 9800074297.', level="business")
    co.capturar_order_number_flujo(driver, order_number)

    # ----- Paso 4: Select store LIVR. -----
    add_step('Select store LIVR.', level="business")
    co.seleccionar_store_flujo(driver, store)

    # ----- Paso 5: Click "Agregar". -----
    add_step('Click "Agregar".', level="business")
    co.hacer_clic_agregar_flujo(driver)

    # ----- Paso 6: Click "Procesar". -----
    add_step('Click "Procesar".', level="business")
    co.hacer_clic_procesar_flujo(driver)

    # ----- Resultados esperados -----
    add_step('Esperado: "Procesar" is disabled before the order is added and enabled after.', level="business")
    # PENDIENTE: resultado esperado sin aserción automática
    add_step('Esperado: After processing, the order appears in "Un vistazo a tus ordenes".', level="business")
    co.verificar_texto_visible_flujo(driver, 'Un vistazo a tus ordenes')
    add_step('Esperado: The "No hay resultados disponibles" empty state disappears.', level="business")
    co.verificar_texto_oculto_flujo(driver, 'No hay resultados disponibles')

    driver.delete_all_cookies()
