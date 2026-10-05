# apps/CobroOrdenes/locators/cobro_ordenes_locators.py
#
# Generado por HydraLabs. Los selectores son CANDIDATOS derivados de los textos del caso
# (role + nombre accesible); los marcados TODO son suposiciones: confirmar con Codegen o el DOM real.


class CobroOrdenesLocators:
    BOTON_PROCESAR = 'role=button[name="Procesar"]'
    INPUT_ORDER_NUMBER = 'role=textbox[name="order number" i]'  # TODO: confirmar el selector del campo 'order number'
    SELECT_STORE = 'role=combobox[name="store" i]'  # TODO: confirmar el selector de 'store' y si la opción va por value o por label
    BOTON_AGREGAR = 'role=button[name="Agregar"]'
    BOTON_DESCARGAR_CSV = 'role=button[name="Descargar CSV"]'
    INPUT_ARCHIVO = "input[type='file']"  # TODO: confirmar el selector de la zona de carga (drop zone)
    BOTON_SUBIR_Y_PROCESAR = 'role=button[name="Subir y Procesar"]'
