# Framework de Automatización E2E con Playwright

Framework de automatización de pruebas E2E basado en **Python + Playwright + Pytest** para aplicaciones web con interacción DOM.

---

## Requisitos Previos

- **Python 3.8+** instalado y agregado al PATH del sistema.
- **Git** (opcional pero recomendado para control de versiones).

---

## Instalación y Configuración

### 1. Instalar dependencias
Abre una terminal en la raíz del proyecto y ejecuta:
```bash
pip install -r requirements.txt
```

### 2. Instalar navegadores de Playwright
Instala el navegador Chromium utilizado para las ejecuciones automáticas:
```bash
playwright install chromium
```

### 3. Configurar variables de entorno y mapearlas con el archivo CSV

Para proteger datos sensibles (como contraseñas, tokens de API o URLs privadas) y evitar subirlos al repositorio de código, el framework soporta el uso de variables de entorno combinadas con archivos CSV.

#### **Paso 1: Crear o editar tu archivo `.env`**
1. En la raíz del proyecto, localiza el archivo `.env` (si no existe, créalo).
2. Agrega las variables clave-valor que necesites. Por ejemplo:
   ```ini
   # Configuración de ejecución
   BROWSER=chrome
   BASE_URL=https://example.com/login
   HEADLESS=false
   EXPLICIT_WAIT=20

   # Credenciales sensibles
   MI_PASSWORD_SECRETO=Liverpool2026!
   CORREO_ADMIN=tester_admin@liverpool.com.mx
   ```

#### **Paso 2: Vincular las variables en el CSV de datos**
Cuando prepares los archivos de datos parametrizados (por ejemplo, en `apps/MiApp/data/users.csv`), no escribas la contraseña o el correo directamente en el archivo si son sensibles.
1. Utiliza el prefijo especial `ENV:` seguido del nombre exacto de la variable de entorno que definiste en tu `.env`.
2. Ejemplo de estructura en `users.csv`:
   ```csv
   usuario,password,descripcion_caso
   usuario_comun,ENV:MI_PASSWORD_SECRETO,Login con credenciales comunes
   ENV:CORREO_ADMIN,ENV:MI_PASSWORD_SECRETO,Login como administrador
   ```

#### **Paso 3: Funcionamiento automático en ejecución**
Cuando ejecutes los tests, el componente utilitario `csv_reader.py` escaneará cada valor del CSV. Al detectar el prefijo `ENV:`, buscará la variable de entorno correspondiente (`MI_PASSWORD_SECRETO` o `CORREO_ADMIN`) en la memoria del sistema y reemplazará dinámicamente el valor en el test. Si no se encuentra la variable en el entorno, el framework mantendrá el texto literal `ENV:MI_PASSWORD_SECRETO` como medida de seguridad.

#### **Paso 4: Uso del archivo `.gitignore` para seguridad**
El framework incluye un archivo `.gitignore` en la raíz del proyecto configurado con reglas clave para proteger el repositorio de fugas de información y evitar subir archivos innecesarios:
1. **Protección del `.env`**: El archivo `.env` está explícitamente añadido a `.gitignore`. Esto asegura que tus credenciales locales reales (como `MI_PASSWORD_SECRETO`) nunca se suban al repositorio de Git.
2. **Exclusión de Reportes y Logs**: Las carpetas de reportes dinámicos (`reports/` y `allure-results/`) y los archivos de logs (`*.log`) están excluidos, permitiendo que cada tester genere sus evidencias de forma local sin ensuciar el repositorio compartido de Git.
3. **Buenas prácticas**: Nunca subas tu archivo `.env` personal al control de versiones. Si necesitas compartir la estructura de variables del entorno, se recomienda crear un archivo de ejemplo llamado `.env.example` con valores ficticios.

#### **Paso 5: Configurar Resultados Esperados (Expected Result) para Cortinillas y Reportes**
Para que los videos de evidencia y el reporte HTML muestren el resultado esperado de cada prueba:
1. Localiza o crea el archivo `expected_result.csv` dentro de la carpeta `data/` de tu aplicación (ej. `apps/MiApp/data/expected_result.csv`).
2. Define la cabecera `test_id,expected_result`.
3. Registra el identificador de la prueba (que coincida con el nombre del test, por ejemplo, buscando tokens como `ICD_7`) y su respectivo resultado esperado:
   ```csv
   test_id,expected_result
   ACC_ICD_7,Validar que el sistema permita registrar y procesar las órdenes C&C correctamente.
   ```
4. El framework leerá dinámicamente este archivo para construir las cortinillas de entrada en los videos de evidencia y para renderizar la caja informativa de resultado esperado en el reporte interactivo.

---

## Estructura del Proyecto

```text
NuevoFramework/
├── .env                          # Variables de entorno
├── conftest.py                   # Configuración central de Pytest + Playwright
├── pytest.ini                    # Configuración de descubrimiento de pruebas
├── requirements.txt              # Dependencias del proyecto
├── runner_central.py             # Orquestador CLI de ejecución
├── config/
│   └── settings.py               # Clase Config que lee variables del .env
├── apps/
│   └── MiApp/
│       ├── data/
│       │   ├── users.csv         # Datos parametrizados de prueba
│       │   └── expected_result.csv # Resultados esperados para cortinillas y reporte
│       ├── locators/
│       │   ├── login_locators.py # Selectores de la página de login
│       │   └── home_locators.py  # Selectores de la página de inicio
│       └── modules/
│           └── auth_module.py    # Acciones reutilizables (login, login fallido)
├── tests/
│   └── MiApp/
│       └── test_login.py         # Pruebas de login parametrizadas
├── utils/
│   ├── driver_wrapper.py         # Wrapper de acciones con Playwright (clicks, inputs, etc.)
│   ├── csv_reader.py             # Lector de CSV con resolución de ENV tokens
│   ├── data_loader.py            # Cargador de CSV a diccionarios
│   ├── logger.py                 # Logger con salida a consola y archivo
│   └── templete_report.py        # Sistema de generación de reportes Elite
├── templates/
│   └── mock_report.html          # Plantilla del reporte HTML visual (¡NO ELIMINAR!)
├── reports/                      # Reportes generados (PDF/HTML)
│   └── Logo/                     # Logos corporativos (¡NO ELIMINAR!)
└── logs/                         # Logs de ejecución
```

---

## Cómo Ejecutar los Tests (Ejecución Granular)

Tienes dos opciones para ejecutar las pruebas según tu necesidad: usar el orquestador central (que genera el reporte automáticamente) o invocar `pytest` directamente (para un control granular y desarrollo).

### Opción A: A través de `runner_central.py` (Recomendado para Suites Completas)
El orquestador central se encarga de configurar las variables del entorno, invocar a pytest y guardar un reporte con timestamp en la carpeta `reports/`.

```bash
# Ejecutar la suite de una aplicación específica
python runner_central.py --suite=MiApp

# Ejecutar todas las suites de prueba en el framework
python runner_central.py --suite=all

# Ejecutar sin abrir el navegador físicamente (Modo Headless / Sin interfaz)
python runner_central.py --suite=MiApp --headless
```

### Opción B: Ejecución Granular vía `pytest` (Desarrollo y Depuración)
Para mayor control durante el desarrollo de scripts, puedes usar los comandos nativos de `pytest`.

#### **1. Ejecutar sin generar reportes (Más rápido para pruebas locales)**
*   **Ejecutar un archivo de pruebas específico:**
    ```bash
    pytest tests/MiApp/test_login.py -v
    ```
*   **Ejecutar una única prueba específica dentro de un archivo:**
    ```bash
    pytest tests/MiApp/test_login.py::test_login_exitoso -v
    ```
*   **Ejecutar una prueba específica con un parámetro específico del CSV:**
    Si usas parametrización, puedes filtrar por una fila específica usando `-k` y un fragmento del valor de entrada:
    ```bash
    pytest tests/MiApp/test_login.py -k "usuario_comun" -v
    ```
*   **Ejecutar múltiples pruebas que coincidan con un texto en su nombre:**
    ```bash
    pytest -k "login or autenticacion" -v
    ```
*   **Mostrar prints y salida estándar en consola en tiempo real:**
    ```bash
    pytest tests/MiApp/test_login.py -s -v
    ```

#### **2. Ejecutar generando el Reporte HTML**
*   **Ejecutar un archivo específico con reporte individualizado:**
    ```bash
    pytest tests/MiApp/test_login.py --html=reports/reporte_login_individual.html --self-contained-html -v
    ```
*   **Ejecutar un test específico con reporte individualizado:**
    ```bash
    pytest tests/MiApp/test_login.py::test_login_exitoso --html=reports/reporte_caso_exitoso.html --self-contained-html -v
    ```

---

## Cómo Agregar Nuevos Tests

La arquitectura de este framework sigue un patrón modular basado en la separación de responsabilidades: **Localizadores, Módulos de Acción, Datos Parametrizados y Pruebas**.

A continuación, se detalla el flujo de trabajo paso a paso para agregar un nuevo caso de prueba para una funcionalidad llamada "Búsqueda de Productos" en la aplicación "MiApp":

### Paso 1: Definir los localizadores
Crea o edita el archivo de localizadores en `apps/MiApp/locators/buscar_locators.py`. Aquí solo declaramos los selectores DOM y no realizamos interacciones.

#### **Guía de Buenas Prácticas: ¿Cómo elegir el mejor localizador?**
Para asegurar que tus pruebas sean estables y no fallen cuando la interfaz cambie, utiliza el siguiente **orden de prioridad**:
1. **Atributos de Test (`data-testid`)**: Si el HTML posee atributos de prueba como `[data-testid="boton-enviar"]` o `[data-qa="..."]`, utilízalos siempre. Son independientes del diseño visual.
2. **Selectores CSS (El Estándar Recomendado)**: Son increíblemente rápidos, legibles y la opción predilecta en Playwright.
3. **XPath (Último recurso)**: **Evita** los XPaths absolutos (ej. `/html/body/div[2]/div/form/button`). Son muy frágiles y se rompen al mínimo cambio en el HTML. Solo usa XPath si necesitas buscar un elemento basado en su texto o subir niveles en el árbol DOM (padres).

---

#### **Mini-Guía: ¿Cómo construir un Selector CSS estable?**
Aquí tienes las reglas básicas de sintaxis para crear selectores CSS de forma sencilla:

*   **Por ID (`#`)**: Si el elemento tiene un atributo `id`, usa `#` seguido del ID.
    - *HTML*: `<input id="search-input">`
    - *CSS*: `"#search-input"`
*   **Por Clase (`.`)**: Si el elemento tiene un atributo `class`, usa `.` seguido de la clase.
    - *HTML*: `<button class="search-btn">`
    - *CSS*: `".search-btn"`
*   **Por Atributo (`[...]`)**: Utiliza corchetes para buscar por cualquier otro atributo (como `name`, `type`, `placeholder` o `data-testid`).
    - *HTML*: `<input type="email" placeholder="Ingresa tu correo">`
    - *CSS*: `"[type='email']"` o `"[placeholder='Ingresa tu correo']"`
*   **Combinaciones (Para mayor precisión)**: Puedes unir etiquetas, clases y atributos sin dejar espacios.
    - *HTML*: `<button class="btn btn-primary" type="submit">`
    - *CSS*: `"button.btn-primary[type='submit']"`
*   **Relación de descendencia (Espacio)**: Deja un espacio en blanco para buscar un elemento secundario dentro de otro elemento contenedor.
    - *HTML*: `<div class="header-container"><input type="text"></div>`
    - *CSS*: `"div.header-container input"`

---

#### **Ejemplo de Archivo de Localizadores:**
```python
class BuscarLocators:
    CAMPO_BUSQUEDA = "#search-input"                        # Selector por ID
    BOTON_BUSCAR = "button.search-btn[type='submit']"       # Combinación de tag, clase y atributo
    RESULTADOS_BUSQUEDA = ".product-item"                   # Selector por Clase
    MENSAJE_SIN_RESULTADOS = "[data-testid='no-results-alert']"  # Atributo data-testid
```

### Paso 2: Crear las acciones reutilizables (Módulo de Acción)
Crea o edita el archivo en `apps/MiApp/modules/buscar_module.py`.

#### **Guía Didáctica: ¿Cómo usar `DriverWrapper` de forma correcta?**

Para interactuar con los elementos del navegador (como hacer clics o escribir textos), **nunca** debes usar comandos nativos directos de Playwright. En su lugar, usamos la clase **`DriverWrapper`**. A continuación explicamos detalladamente su funcionamiento:

##### **1. ¿Quién lo construye y de dónde viene?**
La clase `DriverWrapper` es un componente central del framework ya pre-construido y listo para usarse. Está ubicada físicamente en el archivo `utils/driver_wrapper.py`. Tú no necesitas programar sus funciones internas de interacción, solo debes importarlo en tu módulo de acción de la siguiente forma:
```python
from utils.driver_wrapper import DriverWrapper
```

##### **2. ¿Cómo funciona y de dónde sale el objeto `driver`?**
El objeto `driver` representa la sesión/ventana activa del navegador y su contexto:
1. **En la Prueba (Paso 4)**: Pytest se encarga de crear el navegador automáticamente y lo inyecta a la prueba mediante un "fixture" (una función que prepara el entorno) llamado `driver`. Por eso tu función de prueba siempre tiene el parámetro `driver`.
2. **En las Acciones**: Al llamar a tu función de flujo (como `buscar_producto_flujo`), le transfieres ese objeto `driver` como primer parámetro.
3. **Instanciación**: Dentro del flujo de acción, creas tu envoltorio personalizado escribiendo: `wrapper = DriverWrapper(driver)`. A partir de esa línea, interactúas usando el objeto `wrapper`.

##### **3. ¿Cómo sé si lo estoy llamando bien? (Métodos Comunes)**
El wrapper encapsula la complejidad de Playwright. Los métodos esenciales que utilizarás son:

*   **`click(selector, description)`**: Hace un clic izquierdo en el elemento.
    - `selector`: El localizador string (generalmente importado desde tu clase de `Locators`).
    - `description`: Un texto descriptivo en español (ej. `"el botón de buscar"`). Sirve para que en el log y reporte se lea claramente: *"Click en el botón de buscar"*.
    - **Uso**: `wrapper.click(BuscarLocators.BOTON_BUSCAR, description="el botón de buscar")`

*   **`send_keys(selector, text, description)`**: Escribe texto en un campo de entrada.
    - `selector`: El localizador del campo de texto.
    - `text`: El texto que deseas ingresar.
    - `description`: Descripción amigable del campo (ej. `"el campo de búsqueda"`).
    - **Uso**: `wrapper.send_keys(BuscarLocators.CAMPO_BUSQUEDA, texto_producto, description="el campo de búsqueda")`

*   **`wait.until(selector)`**: Detiene la ejecución del test hasta que el elemento DOM especificado aparezca físicamente en la pantalla (útil antes de hacer validaciones).
    - **Uso**: `wrapper.wait.until(BuscarLocators.RESULTADOS_BUSQUEDA)`

> [!TIP]
> **Ayuda del Editor (Autocompletado)**: Al escribir `wrapper.` en tu editor de código (como VS Code o PyCharm), el IDE te mostrará de forma automática los métodos disponibles, sus parámetros y los tipos de datos que requieren. Si escribes un argumento incorrecto o falta alguno obligatorio, el editor te lo marcará con una alerta visual (subrayado rojo).

Aquí tienes el ejemplo de cómo estructurar e instanciar tu módulo de acción usando el wrapper:

```python
from utils.driver_wrapper import DriverWrapper
from apps.MiApp.locators.buscar_locators import BuscarLocators
from utils.templete_report import add_step

def buscar_producto_flujo(driver, texto_producto):
    """
    Realiza el flujo completo de buscar un producto y verifica que se realice la acción.
    """
    wrapper = DriverWrapper(driver)
    
    # 1. Ingresar texto de búsqueda
    wrapper.send_keys(BuscarLocators.CAMPO_BUSQUEDA, texto_producto, description=f"el campo de búsqueda con el producto '{texto_producto}'")
    add_step(f"Se ingresó el término '{texto_producto}' en el buscador.", "Exitoso")

    # 2. Hacer clic en buscar
    wrapper.click(BuscarLocators.BOTON_BUSCAR, description="el botón de buscar")
    add_step("Se presionó el botón de búsqueda.", "Exitoso")
```

### Paso 3: Preparar los datos de prueba
Crea un archivo CSV en `apps/MiApp/data/productos.csv` con las variaciones de datos que quieres validar:

```csv
termino_busqueda,debe_encontrar_resultados,descripcion
laptop,true,Búsqueda de un producto existente
productoInexistente123,false,Búsqueda de un producto no existente
```

### Paso 4: Escribir el script de prueba
Crea el script en `tests/MiApp/test_buscar.py`. Este archivo debe ser lo más limpio y legible posible, evitando lógica de interacción DOM directa (la cual ya reside en los módulos y el wrapper). 

Su propósito es estructurar las aserciones (validaciones finales), inyectar los datos y organizar los pasos. Para que el reporte HTML sea ejecutivo (vista gerencial/business), debemos activar el modo de negocio con `set_business_mode(True)` y definir los pasos principales con `add_step(..., level="business")`.

Dependiendo de las necesidades de tu escenario, puedes estructurar tu prueba de dos maneras diferentes:

---

#### **Opción A: Escenario de Negocio Único (Enfoque Recomendado para E2E )**
Se utiliza cuando tu script representa **un flujo funcional transaccional de inicio a fin** (por ejemplo: iniciar sesión, llenar un manifiesto, validar firmas y salir). Este test solo correrá una vez, consumiendo un registro específico del CSV de datos. **No requiere decoradores.**

*   **Cuándo usarlo**: Para flujos E2E complejos y secuenciales.
*   **Ejemplo**:

```python
import pytest
import os
from apps.MiApp.modules.buscar_module import buscar_producto_flujo
from utils.csv_reader import cargar_datos_csv
from config.settings import Config
from utils.templete_report import set_business_mode, add_step

# 1. Cargar el CSV completo en memoria
ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
CSV_PATH = os.path.join(ROOT_DIR, "apps", "MiApp", "data", "productos.csv")
DATA_PRODUCTOS = cargar_datos_csv(CSV_PATH)

# 2. Extraer manualmente la fila de datos para el caso específico mediante su ID
datos_caso = next((fila for fila in DATA_PRODUCTOS if fila[0] == "BUSCAR_001"), None)

def test_busqueda_producto_especifico_e2e(driver):
    # Validar que los datos del caso existan
    if not datos_caso:
        pytest.skip("No se encontraron los datos para el caso BUSCAR_001 en el CSV.")

    # Desestructurar los datos de la fila
    caso_id, termino_busqueda, debe_encontrar, descripcion = datos_caso
    
    # Activar el formato ejecutivo del reporte (Modo Negocio)
    set_business_mode(True)

    # Paso de Negocio 1: Navegar al portal
    add_step(f"Navegando a la tienda online: {Config.BASE_URL}", level="business")
    driver.get(Config.BASE_URL)

    # Paso de Negocio 2: Ejecutar el flujo modular
    add_step(f"Ejecutando flujo de búsqueda para el producto: {termino_busqueda}", level="business")
    buscar_producto_flujo(driver, termino_busqueda)

    # Paso de Negocio 3: Validar que la acción se completó
    add_step(f"Validando resultados en pantalla para el caso {caso_id}", level="business")
    # ... Aserciones finales aquí ...
```

---

#### **Opción B: Escenarios Parametrizados (Multi-caso con Decorador)**
Se utiliza cuando quieres **ejecutar la misma función de prueba repetidas veces**, alimentada por cada una de las filas del CSV. Para esto se emplea el decorador `@pytest.mark.parametrize` provisto por Pytest.

*   **Cuándo usarlo**: Para validaciones de formularios, coberturas de múltiples tipos de inputs o flujos rápidos que varían solo por los datos de entrada.
*   **Cómo se reporta**: Si tu CSV tiene 10 registros, Pytest generará y mostrará 10 pruebas individuales en la consola y en el reporte HTML.
*   **Ejemplo**:

```python
import pytest
import os
from apps.MiApp.modules.buscar_module import buscar_producto_flujo
from utils.csv_reader import cargar_datos_csv
from config.settings import Config
from utils.templete_report import set_business_mode, add_step
from utils.driver_wrapper import DriverWrapper
from apps.MiApp.locators.buscar_locators import BuscarLocators

# 1. Cargar todos los registros del CSV
ROOT_DIR = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
CSV_PATH = os.path.join(ROOT_DIR, "apps", "MiApp", "data", "productos.csv")
DATA_PRODUCTOS = cargar_datos_csv(CSV_PATH)

# El decorador inyectará automáticamente cada fila del CSV como variables a la función de prueba
@pytest.mark.parametrize("caso_id, termino_busqueda, debe_encontrar, descripcion", DATA_PRODUCTOS)
def test_busqueda_productos_parametrizado(driver, caso_id, termino_busqueda, debe_encontrar, descripcion):
    set_business_mode(True)

    add_step(f"Navegando a la tienda online: {Config.BASE_URL}", level="business")
    driver.get(Config.BASE_URL)

    add_step(f"Ejecutando flujo de búsqueda para el producto: {termino_busqueda}", level="business")
    buscar_producto_flujo(driver, termino_busqueda)

    add_step("Validando los resultados de la búsqueda en la pantalla", level="business")
    wrapper = DriverWrapper(driver)
    espera_resultados = (debe_encontrar.lower() == "true")

    if espera_resultados:
        assert wrapper.is_visible(BuscarLocators.RESULTADOS_BUSQUEDA), f"Error: No se mostraron resultados para '{termino_busqueda}'"
    else:
        assert wrapper.is_visible(BuscarLocators.MENSAJE_SIN_RESULTADOS), f"Error: Se esperaban 0 resultados pero no se visualizó la alerta de no resultados."
```

---

## Estructura y Uso de la Carpeta `utils`

La carpeta `utils/` contiene módulos transversales que dotan de funcionalidad base al framework. 

### **¿Cuándo debemos usar e interactuar con esta carpeta?**
Los testers deben consumir los archivos de `utils/` para interactuar con el navegador, leer datos, o escribir logs personalizados. **Solo se deben modificar o agregar nuevos archivos en `utils/` cuando se requiera crear una herramienta genérica y aplicable a todo el framework** (por ejemplo: soporte para leer archivos Excel, conectores de base de datos, cifrado de datos, etc.). No se deben poner localizadores ni flujos específicos de una aplicación aquí.

### **Descripción de componentes:**
1.  **`driver_wrapper.py`**: Contiene métodos simplificados para interactuar de forma segura con el navegador (escribir text, hacer click, esperar elementos, etc.). Siempre utiliza este wrapper en lugar de los métodos nativos directos de Playwright para garantizar el logging y captura automática de pantallas ante fallos.
2.  **`csv_reader.py` y `data_loader.py`**: Utilidades dedicadas a leer archivos de configuración CSV y mapear sus filas. Resuelven dinámicamente las variables de entorno asociadas (`ENV:VAR_NAME`).
3.  **`logger.py`**: Proporciona una consola estandarizada y logs en archivo de texto. Úsalo importándolo en tus módulos para depurar variables o dejar trazas del flujo de negocio.
4.  **`templete_report.py`**: Es el motor encargado de dar formato al reporte Elite. Gestiona los pasos registrados con `add_step()`, recopila las evidencias en Base64 y formatea la salida visual.

---

## Generación de Reportes

El framework genera automáticamente reportes visuales con:
- Gráfico de donut con porcentaje de éxito.
- Dashboard con tests pasados/fallidos y tiempo de ejecución.
- Lista interactiva de tests con detalle de pasos y capturas de pantalla embebidas en Base64.
- Filtros interactivos por estado.
- Alternador de modo claro y oscuro.

> [!IMPORTANT]
> **No eliminar la carpeta `templates/` ni su contenido (como `mock_report.html`), ni la carpeta `reports/Logo/`**. 
> Estas carpetas contienen las plantillas HTML estructuradas y los recursos gráficos corporativos esenciales. Si se eliminan o se alteran, el generador de reportes no podrá renderizar la interfaz visual ni inyectar los datos de ejecución, provocando que los reportes fallen o se muestren sin estilos.

---

## Buenas Prácticas y Modelo SOLID

Para asegurar que los scripts de prueba sean mantenibles, legibles y escalables a largo plazo, seguimos las siguientes buenas prácticas alineadas con los **Principios SOLID**:

### 1. **Single Responsibility Principle (SRP) - Principio de Responsabilidad Única**
Cada archivo y clase debe tener un solo motivo para cambiar:
*   **Localizadores (`locators/`)**: Su única función es almacenar las direcciones DOM de los elementos web. Si un botón cambia de ID en la web, solo modificas este archivo.
*   **Módulos (`modules/`)**: Su única función es estructurar flujos de interacción e instrucciones de negocio.
*   **Pruebas (`tests/`)**: Su única función es definir el orden de ejecución, la inyección de datos (parametrización) y la evaluación de resultados (aserciones).

### 2. **Open/Closed Principle (OCP) - Principio de Abierto/Cerrado**
Los componentes base del framework (como `DriverWrapper`) están **abiertos a la extensión pero cerrados a la modificación**. Si requieres una interacción especial del navegador que el wrapper no soporte actualmente, añade un nuevo método especializado en él en lugar de alterar los métodos existentes y romper scripts antiguos.

### 3. **Liskov Substitution Principle (LSP) - Principio de Sustitución de Liskov**
Los wrappers y drivers respetan la interfaz base de Playwright de forma consistente. Si en el futuro cambiamos a otro adaptador de driver, éste debe implementar los mismos métodos de control (`click`, `send_keys`, `is_visible`) sin alterar el funcionamiento de los módulos de acción y los scripts de prueba.

### 4. **Interface Segregation Principle (ISP) - Principio de Segregación de Interfaces**
Es preferible tener múltiples módulos de acción específicos y compactos (por ejemplo, `auth_module.py`, `checkout_module.py`, `search_module.py`) en lugar de un único archivo gigante de interacciones. Esto evita dependencias innecesarias y simplifica la importación de código.

### 5. **Dependency Inversion Principle (DIP) - Principio de Inversión de Dependencias**
Los scripts de prueba y módulos de acción no controlan ni configuran la inicialización del navegador directamente. Dependen de una abstracción (el fixture `driver` inyectado por Pytest en `conftest.py`), el cual a su vez depende de la configuración global (`settings.py`). Esto facilita cambiar la configuración global (como alternar a modo Headless o cambiar de navegador) desde el archivo `.env` sin cambiar una sola línea de código en los scripts de prueba.

### Otras Buenas Prácticas Generales
*   **Limpieza de Estado**: Mantén tus pruebas aisladas. Limpia cookies e historial al final de cada prueba con `driver.delete_all_cookies()`.
*   **Reportes Claros**: Documenta la ejecución registrando pasos lógicos con `add_step("Descripción del paso")` en tus módulos de acción.
*   **Importaciones Absolutas**: Escribe siempre importaciones absolutas a partir de la raíz del proyecto para evitar errores de ruta en ejecuciones multi-suite (ej. `from utils.driver_wrapper import DriverWrapper`).

---

## Ejecución Multiplataforma y Emulación

Playwright es un motor multiplataforma nativo. Esto te permite ejecutar las pruebas en distintos sistemas operativos (Windows, Linux, macOS) y simular diferentes navegadores o resoluciones de dispositivos sin duplicar código.

### 1. Cambiar de navegador localmente
El tester puede definir qué navegador desea utilizar modificando la variable `BROWSER` en su archivo `.env`:
```ini
BROWSER=chrome   # Opciones: chrome, firefox, webkit (para simular Safari)
```

### 2. Emulación de dispositivos móviles o tablets
Si el caso de uso requiere validar la visualización en un dispositivo específico (por ejemplo, Safari en un iPhone), puedes modificar la inicialización del contexto en `conftest.py`:
```python
# Ejemplo de emulación en conftest.py
iphone_13 = p.devices['iPhone 13']
context = browser.new_context(**iphone_13)
```

---

## Integración de Pruebas de API (HTTPS) y Rendimiento (JMeter)

Para expandir el alcance de la automatización a pruebas de backend (APIs) y pruebas de rendimiento, se recomienda la siguiente estructura:

### 1. Pruebas de Rendimiento con JMeter
Para evitar mezclar los scripts de interfaz de usuario de Selenium/Playwright con las pruebas de carga, crea una carpeta raíz dedicada llamada `jmeter/`:

```text
NuevoFramework/
├── ...
├── jmeter/                           # Carpeta exclusiva para scripts de JMeter
│   ├── MiApp_LoadTest.jmx            # Script de prueba de carga de JMeter
│   ├── datos_jmeter.csv              # Datos parametrizados exclusivos para JMeter
│   └── reports/                      # Reportes de rendimiento HTML generados
```
**Uso**: Abre y ejecuta tus archivos `.jmx` utilizando JMeter de forma independiente. Mantener los archivos dentro del repositorio ayuda a que todo el equipo de automatización tenga acceso a la misma suite de carga.

### 2. Pruebas de API HTTPS (GET, POST, PUT, DELETE)
Si deseas automatizar flujos de servicios web REST, puedes usar el soporte nativo de Playwright o la librería estándar `requests` de Python.

#### **Opción A: Cliente API nativo de Playwright**
Playwright cuenta con `APIRequestContext`, que permite hacer llamadas HTTP directamente compartiendo la sesión/cookies del navegador si es necesario:
```python
# Ejemplo en un módulo de acción (apps/MiApp/modules/api_module.py)
def obtener_perfil_usuario(driver, user_id):
    # page.request nos permite lanzar peticiones HTTPS
    response = driver.page.request.get(f"https://api.example.com/users/{user_id}")
    assert response.ok, f"Error al consultar el perfil de usuario: {response.status}"
    return response.json()
```

#### **Opción B: Uso de la librería `requests` (Tests de API puros)**
Si prefieres realizar peticiones independientes del navegador:
1. Asegúrate de tener `requests` en tu `requirements.txt`.
2. Puedes crear un cliente helper en `utils/api_client.py`:
   ```python
   import requests

   class APIClient:
       def __init__(self, base_url):
           self.base_url = base_url

       def get(self, endpoint, headers=None):
           return requests.get(f"{self.base_url}{endpoint}", headers=headers)

       def put(self, endpoint, data=None, headers=None):
           return requests.put(f"{self.base_url}{endpoint}", json=data, headers=headers)
   ```
3. Implementa tus pruebas en `tests/MiApp/test_api.py` importando `APIClient` y parametrizando los casos con tu cargador de CSV.

---

## Escalabilidad y Soporte Multi-Aplicación

El framework está diseñado bajo una arquitectura desacoplada que permite dar soporte a pruebas de múltiples aplicaciones web en un mismo repositorio de forma independiente.

Si necesitas agregar una nueva aplicación llamada **`OtraApp`**, sigue estos pasos:

### 1. Estructura de Carpetas
Crea las carpetas correspondientes bajo `apps/` y `tests/` respetando el patrón modular:
```text
NuevoFramework/
├── apps/
│   └── OtraApp/
│       ├── data/            # CSVs de datos de prueba para OtraApp
│       ├── locators/        # Localizadores e IDs de elementos de OtraApp
│       └── modules/         # Flujos y acciones reutilizables de OtraApp
└── tests/
    └── OtraApp/
        └── test_login.py    # Scripts de prueba de Pytest para OtraApp
```

### 2. Actualizar el Orquestador Central (`runner_central.py`)
Para poder ejecutar las pruebas de `OtraApp` de manera independiente o en conjunto desde la línea de comandos, registra la nueva aplicación en el orquestador:

1. **Añadir a la lista de opciones (`choices`)**:
   En `runner_central.py`, modifica el argumento `--suite` para agregar tu nueva aplicación:
   ```python
   parser.add_argument(
       '--suite',
       choices=['MiApp', 'OtraApp', 'all'], # Agrega 'OtraApp' aquí
       default='MiApp',
       help='Suite a ejecutar.'
   )
   ```

2. **Configurar el mapeo de rutas**:
   Agrega la lógica para asignar los directorios correctos al PYTHONPATH y a las rutas de pytest cuando se ejecute la suite de la nueva app:
   ```python
   if args.suite == 'MiApp':
       test_targets.append(os.path.join(tests_dir, "MiApp"))
       report_name = "report_MiApp.html"
       app_dir = os.path.join(root_dir, "apps", "MiApp")
   elif args.suite == 'OtraApp':
       test_targets.append(os.path.join(tests_dir, "OtraApp"))
       report_name = "report_OtraApp.html"
       app_dir = os.path.join(root_dir, "apps", "OtraApp")
   ```

### 3. Ejecución de la nueva aplicación
Una vez configurado, puedes ejecutar las pruebas mediante los comandos habituales:
```bash
# Ejecutar solo los escenarios de la nueva aplicación con reportes automáticos
python runner_central.py --suite=OtraApp

# Ejecutar todas las aplicaciones (MiApp y OtraApp) de forma secuencial
python runner_central.py --suite=all

# Ejecutar de forma granular en pytest sin reportes estáticos
pytest tests/OtraApp/ -v
```


