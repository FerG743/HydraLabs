# CobroOrdenes (Portal QA): what is peculiar about this app

Read by the model tier on every run and edited by people. Add a line whenever a failure taught us something about the
app. Facts only; how to find things belongs in the app map, what should happen belongs in the case.

- Two environments, set by `NEXT_PUBLIC_API_ENV` in `Oms_Automation/FrontEnd/sistema_cobros/.env.local` and started with `./dev.sh`: **MAC = QA** (frontend `127.0.0.1:3000`, backends `127.0.0.1:8000-8002`) and **PROD = `172.22.64.228`**. Tests with real writes run only against MAC/QA. The card simulator agent is remote (`172.22.64.229:8090`), so the VPN is needed even for QA.
- Entry page for cobro de órdenes: `/cobro_ordenes`.
- The UI is Spanish; cases are written in English. "order number" is the field labeled "Ingresa el número de orden" (`#orderNumber`), "store" is "Ingresa la tienda" (`#name`).
- The store field is a Radix combobox (a `<button role=combobox>`, not a `<select>`): open it, then pick an option. Known options: LIVR, SBB.
- Almost nothing has `data-testid`; the only ones (carousel dots) are not unique. Prefer stable ids (`#orderNumber`, `#name`, `#file-upload`), then role + name.
- "Procesar" and "Subir y Procesar" send `POST :5000/uploadFile` (a real write; `:5000` is PROD, QA uses `:8000`). Read-only runs abort it, so assertions after processing cannot pass without `allowWrites`.
- The orders table starts with the empty state "No hay resultados disponibles"; preconditions that say "table is empty" refer to it.
- Radix generates ids like `radix-:R19al7rmj6:` that change on every load: never use them as locators.
