# utils/network_mock.py
#
# Generado por HydraLabs. Herramienta genérica: simula la respuesta de un endpoint y cuenta las peticiones
# que recibe (útil para validar que un doble clic no envíe dos veces, o cómo reacciona la UI ante un HTTP 500).

class RequestMock:
    def __init__(self, page, url_pattern="**/*", method="POST", status=500, delay_ms=0, body="{}"):
        self.page = page
        self.url_pattern = url_pattern
        self.method = method.upper()
        self.status = int(status)
        self.delay_ms = int(delay_ms)
        self.body = body
        self.requests = []  # URLs de cada petición interceptada

    def install(self):
        self.page.route(self.url_pattern, self._handle)
        return self

    @property
    def count(self):
        return len(self.requests)

    def _handle(self, route, request):
        if request.method.upper() != self.method:
            route.continue_()
            return
        self.requests.append(request.url)
        if self.delay_ms:
            self.page.wait_for_timeout(self.delay_ms)
        route.fulfill(status=self.status, content_type="application/json", body=self.body)
