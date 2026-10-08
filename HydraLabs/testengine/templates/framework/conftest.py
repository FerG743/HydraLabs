import pytest
import os
import base64
from datetime import datetime
from playwright.sync_api import sync_playwright

from config.settings import Config
from utils.logger import get_logger
from utils.templete_report import add_test_result, generate_html_summary, clear_results, add_step, _CURRENT_STEPS

logger = get_logger("Conftest")


class PlaywrightDriverAdapter:
    def __init__(self, browser, context, page):
        self.browser = browser
        self.context = context
        self.page = page

    def get(self, url):
        self.page.goto(url)
        self.page.wait_for_timeout(5000)

    def delete_all_cookies(self):
        try:
            self.context.clear_cookies()
            logger.info("[*] Cookies del contexto de Playwright eliminadas con éxito.")
        except Exception as e:
            logger.error(f"[-] Error al eliminar cookies de Playwright: {e}")

    def get_screenshot_as_base64(self):
        try:
            screenshot_bytes = self.page.screenshot()
            return base64.b64encode(screenshot_bytes).decode('utf-8')
        except Exception as e:
            logger.error(f"[-] Error capturando pantalla con Playwright: {e}")
            return None

    def quit(self):
        try:
            self.browser.close()
        except Exception as e:
            logger.error(f"[-] Error al cerrar el navegador: {e}")


@pytest.fixture(autouse=True)
def clear_steps_before_test():
    _CURRENT_STEPS.clear()
    yield


@pytest.fixture(scope="session")
def _shared_playwright_session(request):
    """Fixture de sesión: lanza UN solo browser/context/page para toda la suite.
    La grabación de video se activa SOLO si se está generando un reporte HTML."""
    import ctypes
    import time

    # Detectar si se configuró un reporte HTML en pytest
    html_path = request.config.getoption("htmlpath", default=None)
    grabar_video = html_path is not None

    if grabar_video:
        logger.info("[*] Inicializando sesión de Playwright con GRABACIÓN DE VIDEO activa...")
    else:
        logger.info("[*] Inicializando sesión de Playwright (Sin grabación de video)...")

    pw = sync_playwright().start()
    headless_mode = Config.HEADLESS

    # Obtener dinámicamente la resolución real de la pantalla del sistema (Windows)
    try:
        user32 = ctypes.windll.user32
        width = user32.GetSystemMetrics(0)
        height = user32.GetSystemMetrics(1)
    except Exception:
        width = 1920
        height = 1080

    browser = pw.chromium.launch(
        headless=headless_mode,
        args=[
            "--disable-save-password-bubble",
            "--disable-translate",
            "--disable-infobars",
            "--disable-notifications",
            "--disable-features=Translate,PasswordGeneration,AutofillAddressEnabled,AutofillCreditCardEnabled",
            f"--window-size={width},{height}",
            "--window-position=0,0"
        ]
    )

    reports_dir = os.path.dirname(Config.REPORT_PATH) or os.path.join(os.getcwd(), "reports")
    videos_dir = os.path.join(reports_dir, "videos")
    
    context_args = {
        "ignore_https_errors": True,
        "viewport": {"width": width, "height": height}
    }
    
    if grabar_video:
        os.makedirs(videos_dir, exist_ok=True)
        context_args["record_video_size"] = {"width": width, "height": height}
        context_args["record_video_dir"] = videos_dir

    context = browser.new_context(**context_args)
    page = context.new_page()

    # Establecer timeout global por defecto (clics, búsquedas, etc.)
    page.set_default_timeout(Config.PLAYWRIGHT_TIMEOUT * 1000)

    yield {
        "browser": browser,
        "context": context,
        "page": page,
        "videos_dir": videos_dir,
        "grabar_video": grabar_video
    }

    # ── Teardown de sesión: cerrar todo y renombrar el video ──
    video_path = None
    if grabar_video:
        try:
            video_path = page.video.path() if page.video else None
        except Exception:
            pass

    try:
        context.close()
        browser.close()
    except Exception:
        pass

    try:
        pw.stop()
    except Exception:
        pass

    if grabar_video and video_path and os.path.exists(video_path):
        try:
            timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
            new_video_name = f"suite_execution_{timestamp}.webm"
            new_video_path = os.path.join(videos_dir, new_video_name)

            for _ in range(5):
                try:
                    os.rename(video_path, new_video_path)
                    logger.info(f"[*] Video de la suite guardado con éxito en: {new_video_path}")
                    break
                except Exception:
                    time.sleep(0.5)
        except Exception as e:
            logger.error(f"[-] Error al renombrar el video de la suite: {e}")


@pytest.fixture(scope="function")
def driver(request, _shared_playwright_session):
    """Fixture por test: reutiliza la sesión compartida, muestra cortinilla
    con el nombre del test y el resultado esperado, y limpia el estado entre ejecuciones."""
    import time
    import re
    import csv

    session = _shared_playwright_session
    page = session["page"]
    context = session["context"]
    browser = session["browser"]

    # ── Mostrar cortinilla de presentación para este test ──
    try:
        test_name = request.node.name
        test_display_name = test_name.replace("test_", "").replace("_", " ").title()

        # Buscar el resultado esperado en el CSV de forma dinámica
        expected_result_text = None
        try:
            id_match = re.search(r"ICD_\d+", test_name, re.IGNORECASE)
            if id_match:
                test_id = f"ACC_{id_match.group(0).upper()}" # fallback o genérico
                base_dir = os.path.dirname(os.path.abspath(__file__))
                apps_dir = os.path.join(base_dir, "apps")
                if os.path.exists(apps_dir):
                    for app_name in os.listdir(apps_dir):
                        app_path = os.path.join(apps_dir, app_name)
                        if os.path.isdir(app_path):
                            csv_path = os.path.join(app_path, "data", "expected_result.csv")
                            if os.path.exists(csv_path):
                                with open(csv_path, "r", encoding="utf-8") as f:
                                    reader = csv.reader(f)
                                    next(reader, None)  # Saltar encabezado
                                    for fila in reader:
                                        if len(fila) >= 2 and (fila[0].strip() == test_id or fila[0].strip().endswith(id_match.group(0).upper())):
                                            expected_result_text = fila[1].strip()
                                            break
                                if expected_result_text:
                                    break
        except Exception as e_csv:
            logger.warning(f"[-] No se pudo leer expected_result.csv para cortinilla: {e_csv}")

        # Generar el bloque HTML del resultado esperado (si existe)
        expected_box_html = ""
        if expected_result_text:
            expected_box_html = f"""
                <div class="expected-box">
                    <div class="expected-title">📋 Resultado Esperado</div>
                    <p class="expected-text">{expected_result_text}</p>
                </div>
            """

        intro_html = f"""
        <!DOCTYPE html>
        <html lang="es">
        <head>
            <meta charset="UTF-8">
            <style>
                body {{
                    margin: 0;
                    padding: 0;
                    background: linear-gradient(135deg, #78256f 0%, #3e0d37 100%);
                    color: white;
                    font-family: 'Helvetica Neue', Arial, sans-serif;
                    display: flex;
                    flex-direction: column;
                    justify-content: center;
                    align-items: center;
                    height: 100vh;
                    overflow: hidden;
                }}
                .container {{
                    text-align: center;
                    max-width: 900px;
                    padding: 0 40px;
                    animation: fadeIn 1s ease-out;
                }}
                h1 {{
                    font-size: 2.5rem;
                    font-weight: 700;
                    margin-bottom: 12px;
                    letter-spacing: -0.5px;
                    max-width: 800px;
                    word-wrap: break-word;
                }}
                p.app-label {{
                    font-size: 1.1rem;
                    color: rgba(255, 255, 255, 0.6);
                    margin: 0 0 30px 0;
                    text-transform: uppercase;
                    letter-spacing: 2px;
                }}
                .badge {{
                    background-color: rgba(255, 255, 255, 0.12);
                    padding: 6px 16px;
                    border-radius: 20px;
                    font-size: 0.85rem;
                    font-weight: 600;
                    margin-bottom: 25px;
                    display: inline-block;
                    border: 1px solid rgba(255, 255, 255, 0.2);
                    letter-spacing: 1px;
                }}
                .expected-box {{
                    background: rgba(255, 255, 255, 0.07);
                    border: 1px dashed rgba(255, 255, 255, 0.25);
                    border-radius: 12px;
                    padding: 20px 25px;
                    margin-top: 20px;
                    text-align: left;
                    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
                }}
                .expected-title {{
                    font-size: 0.9rem;
                    font-weight: 700;
                    text-transform: uppercase;
                    letter-spacing: 1px;
                    color: #d88fd1;
                    margin-bottom: 8px;
                    display: flex;
                    align-items: center;
                    gap: 8px;
                }}
                .expected-text {{
                    font-size: 1.1rem;
                    line-height: 1.5;
                    color: rgba(255, 255, 255, 0.9);
                    margin: 0;
                }}
                @keyframes fadeIn {{
                    from {{ opacity: 0; transform: translateY(10px); }}
                    to {{ opacity: 1; transform: translateY(0); }}
                }}
            </style>
        </head>
        <body>
            <div class="container">
                <div class="badge">EJECUCIÓN DE PRUEBA DE AUTOMATIZACIÓN</div>
                <h1>{test_display_name}</h1>
                <p class="app-label">Aplicativo WEB Liverpool</p>
                {expected_box_html}
            </div>
        </body>
        </html>
        """
        encoded_html = base64.b64encode(intro_html.encode('utf-8')).decode('utf-8')
        page.goto(f"data:text/html;base64,{encoded_html}")
        time.sleep(3.0)
    except Exception as e:
        logger.error(f"[-] Error al mostrar pantalla de intro en el video: {e}")

    adapter = PlaywrightDriverAdapter(browser, context, page)
    yield adapter

    # ── Limpieza entre tests (sin cerrar browser/context) ──
    try:
        context.clear_cookies()
        logger.info("[*] Cookies del contexto de Playwright eliminadas con éxito al finalizar el test.")
    except Exception as e:
        logger.error(f"[-] Error al eliminar cookies al finalizar el test: {e}")

    # Intentar desactivar modo negocio si la función existe en templete_report (por compatibilidad)
    try:
        from utils.templete_report import set_business_mode
        set_business_mode(False)
        logger.info("[*] Modo de negocio desactivado al finalizar el test.")
    except ImportError:
        pass
    except Exception as e:
        logger.error(f"[-] Error al desactivar modo de negocio al finalizar el test: {e}")

    # Navegar a página en blanco para separar visualmente los tests en el video
    try:
        page.goto("about:blank")
    except Exception:
        pass


@pytest.hookimpl(hookwrapper=True)
def pytest_runtest_makereport(item, call):
    pytest_html = item.config.pluginmanager.getplugin('html')
    outcome = yield
    report = outcome.get_result()
    extra = getattr(report, 'extras', [])

    driver = item.funcargs.get('driver')
    screenshot_b64 = add_test_result(item, report, driver)

    if screenshot_b64 and pytest_html:
        extra.append(pytest_html.extras.image(screenshot_b64, 'Evidencia de Ejecución'))

    report.extras = extra


def pytest_configure(config):
    htmlpath = config.getoption("htmlpath")
    if htmlpath:
        dir_name = os.path.dirname(htmlpath) or "reports"
        if not os.path.exists(dir_name):
            os.makedirs(dir_name, exist_ok=True)

        timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
        _, ext = os.path.splitext(os.path.basename(htmlpath))
        config.option.htmlpath = os.path.join(dir_name, f"report_{timestamp}{ext}")


def pytest_html_report_title(report):
    report.title = "Reporte de Automatización - Liverpool"


def pytest_html_results_summary(prefix, summary, postfix):
    generate_html_summary(prefix, None)


def pytest_sessionstart(session):
    clear_results()


def pytest_unconfigure(config):
    import requests
    import json
    from datetime import datetime
    import urllib3
    
    # Silenciar advertencias de SSL no verificado (InsecureRequestWarning)
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    
    # 1. Verificar si se generó un reporte HTML
    html_path = getattr(config.option, "htmlpath", None)
    if not html_path:
        logger.info("[!] No se generó reporte HTML. Se cancela el envío de la notificación a Google Chat.")
        return

    # 2. Obtener la URL del Webhook
    webhook_url = os.getenv("AUTO_ACC")
    if not webhook_url:
        logger.warning("[!] Webhook URL (AUTO_ACC) no encontrada en las variables de entorno. No se enviará notificación.")
        return
        
    # 3. Calcular los resultados acumulados
    from utils.templete_report import _ELITE_CASES
    
    passed = sum(1 for c in _ELITE_CASES if c["outcome"] == "passed")
    failed = sum(1 for c in _ELITE_CASES if c["outcome"] in ["failed", "error"])
    total = passed + failed
    
    if total == 0:
        logger.warning("[!] No se ejecutaron pruebas. No se enviará notificación.")
        return
        
    # 4. Subir el reporte HTML a Nexus
    report_url = "https://nexushub.up.railway.app/"  # Fallback
    api_key = os.getenv("NEXUS_API_KEY")
    
    headers = {}
    if api_key:
        headers["X-API-Key"] = api_key

    if html_path and os.path.exists(html_path):
        try:
            logger.info(f"[*] Subiendo reporte HTML a Nexus: {html_path}")
            nombre_reporte = os.path.basename(html_path)
            with open(html_path, 'rb') as f:
                r_upload = requests.post(
                    "https://nexushub.up.railway.app/api/reports/upload",
                    files={'file': (nombre_reporte, f)},
                    headers=headers,
                    verify=False
                )
            if r_upload.status_code == 201:
                filename = r_upload.json().get("filename", nombre_reporte)
                report_url = f"https://nexushub.up.railway.app/reports/{filename}"
                logger.info(f"[*] Reporte subido con éxito: {report_url}")
            else:
                logger.error(f"[-] Error al subir reporte: {r_upload.status_code} - {r_upload.text}")
        except Exception as e:
            logger.error(f"[-] Excepción al subir reporte HTML: {e}")
            
    # 5. Subir el video a Nexus si existe
    video_url = None
    if html_path:
        reports_dir = os.path.dirname(html_path)
        videos_dir = os.path.join(reports_dir, "videos")
        if os.path.exists(videos_dir):
            videos = [os.path.join(videos_dir, f) for f in os.listdir(videos_dir) if f.lower().endswith((".webm", ".mp4"))]
            if videos:
                # Ordenar por fecha de modificación (el más reciente primero)
                videos.sort(key=os.path.getmtime, reverse=True)
                latest_video = videos[0]
                try:
                    logger.info(f"[*] Subiendo video de evidencia a Nexus: {latest_video}")
                    nombre_video = os.path.basename(latest_video)
                    with open(latest_video, 'rb') as f:
                        r_upload_vid = requests.post(
                            "https://nexushub.up.railway.app/api/videos/upload",
                            files={'file': (nombre_video, f)},
                            headers=headers,
                            verify=False
                        )
                    if r_upload_vid.status_code == 201:
                        filename_vid = r_upload_vid.json().get("filename", nombre_video)
                        video_url = f"https://nexushub.up.railway.app/reports/{filename_vid}"
                        logger.info(f"[*] Video subido con éxito: {video_url}")
                    else:
                        logger.error(f"[-] Error al subir video: {r_upload_vid.status_code} - {r_upload_vid.text}")
                except Exception as e:
                    logger.error(f"[-] Excepción al subir video: {e}")
        
    # 6. Determinar icono dinámico del encabezado
    if failed > 0:
        header_icon = "https://upload.wikimedia.org/wikipedia/commons/thumb/8/8f/Flat_cross_icon.svg/120px-Flat_cross_icon.svg.png"
    else:
        header_icon = "https://upload.wikimedia.org/wikipedia/commons/thumb/7/73/Flat_tick_icon.svg/120px-Flat_tick_icon.svg.png"
        
    # 7. Formatear la fecha
    fecha_str = datetime.now().strftime("El %Y-%m-%d a las %H:%M hrs")
    
    # 8. Construir botones dinámicos
    buttons = [
         {
             "text": "Ver reportes",
             "onClick": {
                 "openLink": {
                     "url": "https://nexushub.up.railway.app/"
                 }
             }
         }
    ]
    if video_url:
        buttons.append({
            "text": "Ver video",
            "onClick": {
                "openLink": {
                    "url": video_url
                }
            }
        })
    
    # 9. Crear el payload de la tarjeta Card V2
    widgets = [
        {
            "decoratedText": {
                "startIcon": {
                    "iconUrl": "https://upload.wikimedia.org/wikipedia/commons/thumb/7/73/Flat_tick_icon.svg/120px-Flat_tick_icon.svg.png"
                },
                "text": f"Exitosos: {passed}"
            }
        },
        {
            "decoratedText": {
                "startIcon": {
                    "iconUrl": "https://upload.wikimedia.org/wikipedia/commons/thumb/8/8f/Flat_cross_icon.svg/120px-Flat_cross_icon.svg.png"
                },
                "text": f"Fallidos: {failed}"
            }
        },
        {
            "decoratedText": {
                "startIcon": {
                    "knownIcon": "CLOCK"
                },
                "text": f"Fecha: {fecha_str}"
            }
        },
        {
            "decoratedText": {
                "startIcon": {
                    "iconUrl": "https://upload.wikimedia.org/wikipedia/commons/thumb/e/e4/Infobox_info_icon.svg/120px-Infobox_info_icon.svg.png"
                },
                "text": "Nota: Es necesario ingresar al reporte de histórico de Nexus para ver el reporte",
                "wrapText": True
            }
        }
    ]
    
    if video_url:
        widgets.append({
            "decoratedText": {
                "startIcon": {
                    "knownIcon": "VIDEO_CAMERA"
                },
                "text": "Evidencia de video: Expira y se elimina en 24 hrs. Haz clic en 'Ver video' para reproducirlo.",
                "wrapText": True
            }
        })
        
    widgets.append({
        "buttonList": {
            "buttons": buttons
        }
    })

    payload = {
        "cardsV2": [
            {
                "cardId": "test_execution_report",
                "card": {
                    "header": {
                        "title": "AUTOMATIZACIÓN DE PRUEBAS",
                        "subtitle": "MPA WEB_No afectación",
                        "imageUrl": header_icon,
                        "imageType": "CIRCLE"
                    },
                    "sections": [
                        {
                            "header": "Resumen de ejecución",
                            "widgets": widgets
                        }
                    ]
                }
            }
        ]
    }
    
    # 10. Enviar petición POST a Google Chat Webhook
    try:
        r_webhook = requests.post(
            webhook_url,
            json=payload,
            headers={"Content-Type": "application/json; charset=UTF-8"}
        )
        logger.info(f"[*] Notificación de Google Chat enviada. Status Code: {r_webhook.status_code}")
    except Exception as e:
        logger.error(f"[-] Error al enviar notificación a Google Chat: {e}")
