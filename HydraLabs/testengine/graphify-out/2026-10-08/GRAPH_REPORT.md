# Graph Report - testengine  (2026-10-08)

## Corpus Check
- 126 files · ~78,660 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 942 nodes · 1544 edges · 63 communities (51 shown, 6 thin omitted)
- Extraction: 95% EXTRACTED · 5% INFERRED · 0% AMBIGUOUS · INFERRED: 74 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `7ebf4147`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- RunContext
- agent.py
- .f
- App.jsx
- Config
- test_spec_render.py
- testing.T
- jira2case.py
- conftest.py
- render_pytest.py
- matcher.py
- join/main.go
- AgentLoop
- skeleton/main.go
- HydraPloy + HydraLabs: scope, stages, rules
- PortalE2E
- scorecard.py
- Sandbox
- intake.go
- properties
- harness.py
- type
- properties
- suite/main.go
- case.schema.json
- properties
- catalog.py
- RequestMock
- write_allowed
- system_es.md
- build_jira_workflow.py
- meta
- test_files.py
- start.sh
- id
- name
- verify_locators.py
- testengine
- crawl.py
- casecheck/main.go
- Framework de Automatización E2E con Playwright
- .Crawl
- CobroOrdenes.md
- bridge.go
- Cómo Agregar Nuevos Tests
- Jira
- 3. Configurar variables de entorno y mapearlas con el archivo CSV
- xml_reader.py
- lint
- Buenas Prácticas y Modelo SOLID
- build_hydra_workflow.py
- Cómo Ejecutar los Tests (Ejecución Granular)
- 2. Pruebas de API HTTPS (GET, POST, PUT, DELETE)
- pdf_reader.py
- .Validate
- Escalabilidad y Soporte Multi-Aplicación
- Paso 1: Definir los localizadores

## God Nodes (most connected - your core abstractions)
1. `HydraPloy + HydraLabs: scope, stages, rules` - 19 edges
2. `RunContext` - 18 edges
3. `MCPClient` - 14 edges
4. `render()` - 14 edges
5. `AgentLoop` - 14 edges
6. `Config` - 12 edges
7. `fakeEnv()` - 12 edges
8. `build()` - 12 edges
9. `Status` - 12 edges
10. `Framework de Automatización E2E con Playwright` - 12 edges

## Surprising Connections (you probably didn't know these)
- `registry()` --calls--> `NewRegistry()`  [EXTRACTED]
  cmd/demo/main.go → engine/registry.go
- `main()` --calls--> `LoadGraph()`  [EXTRACTED]
  cmd/demo/main.go → engine/graph.go
- `loadBlock` --references--> `Graph`  [EXTRACTED]
  cmd/demo/loadblock.go → engine/graph.go
- `loadBlock` --references--> `Registry`  [EXTRACTED]
  cmd/demo/loadblock.go → engine/registry.go
- `runFailed()` --references--> `RunResult`  [EXTRACTED]
  cmd/demo/loadblock.go → engine/interpreter.go

## Import Cycles
- None detected.

## Communities (63 total, 6 thin omitted)

### Community 0 - "RunContext"
Cohesion: 0.06
Nodes (43): orDefault(), loadFactory(), percentile(), runFailed(), main(), printSteps(), registry(), reportValidation() (+35 more)

### Community 1 - "agent.py"
Cohesion: 0.05
Nodes (35): chat(), compact(), dispatch(), local_tools(), map_summary(), openai_tools(), LM agent: one normalized case + the real page (via Playwright MCP, read-only)…, Keeps the conversation inside the model's context: old tool results are… (+27 more)

### Community 2 - ".f"
Cohesion: 0.05
Nodes (27): submit_spec(), Build, camel(), module_header(), plain(), spec + Jira case -> README-layout files, deterministically. The agent never…, Writes the files. Returns {"files": [...], "warnings": [...], "errors": [...]}…, {NAME: (value, dynamic)} from an existing locators file. (+19 more)

### Community 3 - "App.jsx"
Cohesion: 0.08
Nodes (28): react, react-dom, vite, @vitejs/plugin-react, @xyflow/react, dependencies, react, react-dom (+20 more)

### Community 4 - "Config"
Cohesion: 0.11
Nodes (28): argValue(), cfgFlags(), Config, main(), poll(), pollAll(), report(), Case (+20 more)

### Community 5 - "test_spec_render.py"
Cohesion: 0.12
Nodes (9): case(), EndToEnd, framework(), golden(), Portal, read(), Renderer, Validator (+1 more)

### Community 6 - "testing.T"
Cohesion: 0.07
Nodes (52): crawl(), el(), TestEnglishCaseWordsMatchSpanishPage(), TestHumanFileWinsOverLearned(), TestModelOnlyChoosesFromTheOfferedList(), TestRunConfirmsGuessesAndMarksCustomWidgets(), TestUnsureStaysUnresolvedWithoutAModel(), copyTree() (+44 more)

### Community 7 - "jira2case.py"
Cohesion: 0.06
Nodes (30): HTMLParser, generate(), plain(), Normalized case JSON -> Gherkin feature. Deterministic: every step and expected…, tag(), clean(), convert(), walk() (+22 more)

### Community 8 - "conftest.py"
Cohesion: 0.07
Nodes (24): hookimpl, Config, clear_steps_before_test(), driver(), PlaywrightDriverAdapter, fixture, pytest_html_results_summary(), pytest_runtest_makereport() (+16 more)

### Community 9 - "render_pytest.py"
Cohesion: 0.17
Nodes (21): analyse(), App, app_slug(), build_calls(), build_expected(), button(), csv_rows(), main() (+13 more)

### Community 10 - "matcher.py"
Cohesion: 0.11
Nodes (18): ask(), baseline(), bench(), catalog_text(), combine(), decide(), gold(), gold_cases() (+10 more)

### Community 11 - "join/main.go"
Cohesion: 0.14
Nodes (20): join(), load(), main(), matchesAny(), norm(), TestJoin(), find(), main() (+12 more)

### Community 12 - "AgentLoop"
Cohesion: 0.15
Nodes (4): AgentLoop, fake_lm(), FakeMCP, Agent loop mechanics with a scripted fake LM and a fake MCP. No real model,…

### Community 13 - "skeleton/main.go"
Cohesion: 0.22
Nodes (19): autoMap(), build(), collapse(), fieldsOf(), main(), prep(), removeEntry(), tagsOf() (+11 more)

### Community 14 - "HydraPloy + HydraLabs: scope, stages, rules"
Cohesion: 0.09
Nodes (21): App map (the crawl), Boards, apps and repos, Contracts, Environment, secrets, data (the framework's own convention), Failure triage, HydraPloy + HydraLabs: scope, stages, rules, Known data issues (facturación sample), Measuring it (+13 more)

### Community 15 - "PortalE2E"
Cohesion: 0.15
Nodes (5): skipUnless, Driver, Handler, PortalE2E, Drives the GENERATED TC-3 test in a real Chromium against a tiny fake portal…

### Community 16 - "scorecard.py"
Cohesion: 0.25
Nodes (15): anchors(), chat(), jaccard(), lint(), literals(), main(), match_all(), norm() (+7 more)

### Community 17 - "Sandbox"
Cohesion: 0.19
Nodes (3): build(), Sandbox, StructureCheck

### Community 18 - "intake.go"
Cohesion: 0.41
Nodes (12): attr(), clean(), isEl(), item(), subItems(), text(), ToCase(), topItems() (+4 more)

### Community 19 - "properties"
Cohesion: 0.22
Nodes (11): items, allOf, properties, required, type, note, status, steps (+3 more)

### Community 20 - "harness.py"
Cohesion: 0.14
Nodes (10): Run one generated test against a real URL. Read-only by default: every non-GET…, Driver, Runs a GENERATED test against a real URL without needing the framework repo:…, Reads always pass. A write passes only with allow_writes AND only to a loopback…, run(), net(), _tests_dir(), write_allowed() (+2 more)

### Community 21 - "type"
Cohesion: 0.25
Nodes (8): description, items, type, type, depends_on, warnings, items, type

### Community 22 - "properties"
Cohesion: 0.29
Nodes (7): properties, required, type, data, lists, rows, shared

### Community 23 - "suite/main.go"
Cohesion: 0.53
Nodes (5): lines(), main(), nonBlank(), slug(), Case

### Community 24 - "case.schema.json"
Cohesion: 0.33
Nodes (5): description, required, $schema, title, type

### Community 25 - "properties"
Cohesion: 0.33
Nodes (6): type, items, type, properties, cases, orphans

### Community 26 - "catalog.py"
Cohesion: 0.60
Nodes (5): functions(), locators(), main(), parse(), Reads a framework repo (README layout: apps/<App>/{modules,locators},…

### Community 28 - "write_allowed"
Cohesion: 0.27
Nodes (6): _hydra_guard(), guard(), fixture, pytest plugin (load with -p hydra_guard): generated tests run read-only. Any…, write_allowed(), PytestPluginGuard

### Community 29 - "system_es.md"
Cohesion: 0.40
Nodes (4): Cómo trabajar, Honestidad, La spec (JSON), Seguridad (no negociable)

### Community 31 - "meta"
Cohesion: 0.40
Nodes (5): type, additionalProperties, description, type, meta

### Community 32 - "test_files.py"
Cohesion: 0.40
Nodes (4): binary_payload(), csv_payload(), Archivo binario determinista (contiene bytes NUL, por lo tanto nunca es…, CSV con el encabezado y las filas dadas (valores arbitrarios: vacíos, enormes,…

### Community 33 - "start.sh"
Cohesion: 0.50
Nodes (3): N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS, NODES_EXCLUDE, start.sh script

### Community 34 - "id"
Cohesion: 0.50
Nodes (4): description, minLength, type, id

### Community 35 - "name"
Cohesion: 0.67
Nodes (3): minLength, type, name

### Community 40 - "crawl.py"
Cohesion: 0.19
Nodes (13): best_locator(), color_of(), crawl(), draw(), Events, label_of(), load_env(), --safe-clicks: open a combobox/tab just to read its options, then close it.… (+5 more)

### Community 41 - "casecheck/main.go"
Cohesion: 0.36
Nodes (6): Case, Step, check(), main(), ready(), TestCheck()

### Community 42 - "Framework de Automatización E2E con Playwright"
Cohesion: 0.18
Nodes (10): 1. Cambiar de navegador localmente, 2. Emulación de dispositivos móviles o tablets, **¿Cuándo debemos usar e interactuar con esta carpeta?**, **Descripción de componentes:**, Ejecución Multiplataforma y Emulación, Estructura del Proyecto, Estructura y Uso de la Carpeta `utils`, Framework de Automatización E2E con Playwright (+2 more)

### Community 43 - ".Crawl"
Cohesion: 0.47
Nodes (3): Config, join(), Serve()

### Community 45 - "bridge.go"
Cohesion: 0.22
Nodes (9): fold(), Config, inRoles(), loadConfirmed(), tokens(), Bridge, Confirmed, Crawl (+1 more)

### Community 46 - "Cómo Agregar Nuevos Tests"
Cohesion: 0.20
Nodes (10): **1. ¿Quién lo construye y de dónde viene?**, **2. ¿Cómo funciona y de dónde sale el objeto `driver`?**, **3. ¿Cómo sé si lo estoy llamando bien? (Métodos Comunes)**, Cómo Agregar Nuevos Tests, **Guía Didáctica: ¿Cómo usar `DriverWrapper` de forma correcta?**, **Opción A: Escenario de Negocio Único (Enfoque Recomendado para E2E )**, **Opción B: Escenarios Parametrizados (Multi-caso con Decorador)**, Paso 2: Crear las acciones reutilizables (Módulo de Acción) (+2 more)

### Community 47 - "Jira"
Cohesion: 0.42
Nodes (4): JiraFromEnv(), net/url.Values, Issue, Jira

### Community 48 - "3. Configurar variables de entorno y mapearlas con el archivo CSV"
Cohesion: 0.22
Nodes (9): 1. Instalar dependencias, 2. Instalar navegadores de Playwright, 3. Configurar variables de entorno y mapearlas con el archivo CSV, Instalación y Configuración, **Paso 1: Crear o editar tu archivo `.env`**, **Paso 2: Vincular las variables en el CSV de datos**, **Paso 3: Funcionamiento automático en ejecución**, **Paso 4: Uso del archivo `.gitignore` para seguridad** (+1 more)

### Community 49 - "xml_reader.py"
Cohesion: 0.25
Nodes (8): assert_cfdi_matches(), assert_cfdi_tiene_conceptos(), extract_cfdi_data(), _find_comprobante_ns(), Detecta si el XML es CFDI 4.0 o 3.3 según el namespace del root., Parsea un CFDI (XML) y extrae los campos más relevantes para pruebas: fecha,…, Compara campos puntuales del CFDI extraído contra los valores esperados de la…, Valida que el CFDI tenga al menos `minimo` conceptos (artículos).

### Community 50 - "lint"
Cohesion: 0.43
Nodes (5): lint(), main(), TestLint(), Opts, Result

### Community 51 - "Buenas Prácticas y Modelo SOLID"
Cohesion: 0.29
Nodes (7): 1. **Single Responsibility Principle (SRP) - Principio de Responsabilidad Única**, 2. **Open/Closed Principle (OCP) - Principio de Abierto/Cerrado**, 3. **Liskov Substitution Principle (LSP) - Principio de Sustitución de Liskov**, 4. **Interface Segregation Principle (ISP) - Principio de Segregación de Interfaces**, 5. **Dependency Inversion Principle (DIP) - Principio de Inversión de Dependencias**, Buenas Prácticas y Modelo SOLID, Otras Buenas Prácticas Generales

### Community 52 - "build_hydra_workflow.py"
Cohesion: 0.38
Nodes (5): config(), hydra(), node(), Generates hydra-pipeline.json: n8n as a thin, visible shell around the Go…, sh()

### Community 53 - "Cómo Ejecutar los Tests (Ejecución Granular)"
Cohesion: 0.40
Nodes (5): **1. Ejecutar sin generar reportes (Más rápido para pruebas locales)**, **2. Ejecutar generando el Reporte HTML**, Cómo Ejecutar los Tests (Ejecución Granular), Opción A: A través de `runner_central.py` (Recomendado para Suites Completas), Opción B: Ejecución Granular vía `pytest` (Desarrollo y Depuración)

### Community 54 - "2. Pruebas de API HTTPS (GET, POST, PUT, DELETE)"
Cohesion: 0.40
Nodes (5): 1. Pruebas de Rendimiento con JMeter, 2. Pruebas de API HTTPS (GET, POST, PUT, DELETE), Integración de Pruebas de API (HTTPS) y Rendimiento (JMeter), **Opción A: Cliente API nativo de Playwright**, **Opción B: Uso de la librería `requests` (Tests de API puros)**

### Community 55 - "pdf_reader.py"
Cohesion: 0.40
Nodes (4): assert_invoice_contains(), extract_invoice_data(), Extrae el texto completo de un PDF y busca el UUID fiscal (CFDI), que sigue el…, Valida que ciertos fragmentos de texto (ej. RFC, nombre, forma de pago) estén…

### Community 57 - "Escalabilidad y Soporte Multi-Aplicación"
Cohesion: 0.50
Nodes (4): 1. Estructura de Carpetas, 2. Actualizar el Orquestador Central (`runner_central.py`), 3. Ejecución de la nueva aplicación, Escalabilidad y Soporte Multi-Aplicación

### Community 58 - "Paso 1: Definir los localizadores"
Cohesion: 0.50
Nodes (4): **Ejemplo de Archivo de Localizadores:**, **Guía de Buenas Prácticas: ¿Cómo elegir el mejor localizador?**, **Mini-Guía: ¿Cómo construir un Selector CSS estable?**, Paso 1: Definir los localizadores

## Knowledge Gaps
- **110 isolated node(s):** `Case`, `RuleKind`, `testengine`, `start.sh script`, `NODES_EXCLUDE` (+105 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 300 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `add_test_result()` connect `conftest.py` to `.f`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **Why does `writeJSON()` connect `Config` to `bridge.go`, `testing.T`?**
  _High betweenness centrality (0.013) - this node is a cross-community bridge._
- **Why does `driver()` connect `conftest.py` to `.f`?**
  _High betweenness centrality (0.012) - this node is a cross-community bridge._
- **What connects `Case`, `RuleKind`, `testengine` to the rest of the system?**
  _110 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `RunContext` be split into smaller, more focused modules?**
  _Cohesion score 0.05970149253731343 - nodes in this community are weakly interconnected._
- **Should `agent.py` be split into smaller, more focused modules?**
  _Cohesion score 0.05191256830601093 - nodes in this community are weakly interconnected._
- **Should `.f` be split into smaller, more focused modules?**
  _Cohesion score 0.0544464609800363 - nodes in this community are weakly interconnected._