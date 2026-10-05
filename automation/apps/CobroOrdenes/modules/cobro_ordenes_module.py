# apps/CobroOrdenes/modules/cobro_ordenes_module.py
#
# Generado por HydraLabs: un flujo reutilizable por acción distinta de los casos.

import os
from playwright.sync_api import expect
from utils.driver_wrapper import DriverWrapper
from apps.CobroOrdenes.locators.cobro_ordenes_locators import CobroOrdenesLocators as L
from config.settings import Config
from utils.templete_report import add_step


def navegar_a_cobro_ordenes_flujo(driver):
    """Navega a /cobro_ordenes."""
    driver.get(Config.BASE_URL.rstrip("/") + '/cobro_ordenes')
    add_step("El sistema muestra la pantalla /cobro_ordenes.", "Exitoso")


def verificar_procesar_disabled_flujo(driver):
    """Verifica que el botón Procesar esté deshabilitado."""
    assert driver.page.locator(L.BOTON_PROCESAR).is_disabled(), "El botón 'Procesar' debería estar disabled."
    add_step("El botón Procesar está disabled.", "Exitoso")


def capturar_order_number_flujo(driver, order_number):
    """Captura el campo order number."""
    DriverWrapper(driver).send_keys(L.INPUT_ORDER_NUMBER, order_number, description="el campo order number")


def seleccionar_store_flujo(driver, store):
    """Selecciona store."""
    add_step("Seleccionando store.")
    driver.page.select_option(L.SELECT_STORE, value=store)


def hacer_clic_agregar_flujo(driver):
    """Clic en el botón Agregar."""
    DriverWrapper(driver).click(L.BOTON_AGREGAR, description="el botón Agregar")


def hacer_clic_procesar_flujo(driver):
    """Clic en el botón Procesar."""
    DriverWrapper(driver).click(L.BOTON_PROCESAR, description="el botón Procesar")


def verificar_texto_visible_flujo(driver, texto):
    """Verifica que el texto esté visible."""
    expect(driver.page.get_by_text(texto).first).to_be_visible()
    add_step(f"El texto '{texto}' está visible.", "Exitoso")


def verificar_texto_oculto_flujo(driver, texto):
    """Verifica que el texto esté oculto."""
    expect(driver.page.get_by_text(texto).first).to_be_hidden()
    add_step(f"El texto '{texto}' está oculto.", "Exitoso")


def descargar_con_descargar_csv_flujo(driver, downloads_dir, file_base_name):
    """Clic en Descargar CSV y guarda el archivo descargado; devuelve su ruta."""
    os.makedirs(downloads_dir, exist_ok=True)
    with driver.page.expect_download() as info:
        DriverWrapper(driver).click(L.BOTON_DESCARGAR_CSV, description="el botón Descargar CSV")
    download = info.value
    path = os.path.join(downloads_dir, f"{file_base_name}_{download.suggested_filename}")
    download.save_as(path)
    add_step(f"Archivo descargado: {os.path.basename(path)}", "Exitoso")
    return path


def subir_archivo_flujo(driver, ruta_archivo):
    """Sube un archivo mediante la zona de carga."""
    add_step(f"Subiendo el archivo {os.path.basename(ruta_archivo)}.")
    driver.page.set_input_files(L.INPUT_ARCHIVO, ruta_archivo)


def verificar_subir_y_procesar_enabled_flujo(driver):
    """Verifica que el botón Subir y Procesar esté habilitado."""
    assert driver.page.locator(L.BOTON_SUBIR_Y_PROCESAR).is_enabled(), "El botón 'Subir y Procesar' debería estar enabled."
    add_step("El botón Subir y Procesar está enabled.", "Exitoso")


def hacer_clic_subir_y_procesar_flujo(driver):
    """Clic en el botón Subir y Procesar."""
    DriverWrapper(driver).click(L.BOTON_SUBIR_Y_PROCESAR, description="el botón Subir y Procesar")


def doble_clic_subir_y_procesar_flujo(driver):
    """Doble clic en el botón Subir y Procesar."""
    add_step("Doble clic en el botón Subir y Procesar.")
    driver.page.dblclick(L.BOTON_SUBIR_Y_PROCESAR)


def esperar_respuesta_flujo(driver):
    """Espera a que termine la actividad de red."""
    driver.page.wait_for_load_state("networkidle")
    add_step("La respuesta del servidor fue recibida.", "Exitoso")
