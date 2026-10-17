# Graph Report - HydraLabs  (2026-10-14)

## Corpus Check
- 249 files · ~105,559 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1564 nodes · 2712 edges · 124 communities (92 shown, 16 thin omitted)
- Extraction: 94% EXTRACTED · 6% INFERRED · 0% AMBIGUOUS · INFERRED: 167 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `283aabd4`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- AgentLoop
- jira2case.py
- conftest.py
- RunContext
- .SaveCrawl
- cn
- web/package.json
- testing.T
- Poll
- use-toast.ts
- lucide-react
- render_pytest.py
- matcher.py
- join/main.go
- writeJSON
- Action
- HydraLabs/package.json
- HydraPloy + HydraLabs: scope, stages, rules
- test_spec_render.py
- init.go
- skeleton/main.go
- compilerOptions
- ActivityPanel.tsx
- harness.py
- components.json
- testengine/cmd/demo/main.go
- tauri.conf.json
- context.Context
- dependencies
- agent.py
- crawl.py
- PortalE2E
- scorecard.py
- ApiSetup.tsx
- APITesting.tsx
- .f
- SideExplorer.tsx
- fileExists
- intake.go
- git
- render
- App.tsx
- MCPClient
- UrlInput.tsx
- .Observe
- queue_test.go
- apiConstants.ts
- write_allowed
- properties
- Framework de Automatización E2E con Playwright
- Sandbox
- structure_check.py
- Cómo Agregar Nuevos Tests
- ollama.go
- SideExplorer/components/index.tsx
- TestSuiteCard.tsx
- devDependencies
- AdvancedSection.tsx
- Sandbox
- 3. Configurar variables de entorno y mapearlas con el archivo CSV
- xml_reader.py
- src/hooks/index.ts
- casecheck/main.go
- init_test.go
- spec.py
- type
- compilerOptions
- .crawlArgs
- runner.py
- lint
- build_hydra_workflow.py
- bridge_test.go
- properties
- Buenas Prácticas y Modelo SOLID
- Agent
- default.json
- scripts
- suite/main.go
- case.schema.json
- properties
- catalog.py
- RequestMock
- deprecated/
- TestIntakeMatchesPythonGolden
- system_es.md
- build_jira_workflow.py
- meta
- Cómo Ejecutar los Tests (Ejecución Granular)
- 2. Pruebas de API HTTPS (GET, POST, PUT, DELETE)
- pdf_reader.py
- test_files.py
- .Act
- .Validate
- lib.rs
- start.sh
- id
- Escalabilidad y Soporte Multi-Aplicación
- Paso 1: Definir los localizadores
- Screen
- Tauri + React + Typescript
- useSearch.ts
- name
- hydra/README.md
- verify_locators.py
- README.md
- hydralabs
- testengine
- uiagent

## God Nodes (most connected - your core abstractions)
1. `cn()` - 44 edges
2. `lucide-react` - 34 edges
3. `fileExists()` - 21 edges
4. `answers` - 20 edges
5. `Poll()` - 19 edges
6. `HydraPloy + HydraLabs: scope, stages, rules` - 19 edges
7. `Init()` - 18 edges
8. `compilerOptions` - 18 edges
9. `git()` - 17 edges
10. `writeJSON()` - 16 edges

## Surprising Connections (you probably didn't know these)
- `AlertTitle` --calls--> `cn()`  [EXTRACTED]
  HydraLabs/src/components/ui/alert.tsx → HydraLabs/src/lib/utils.ts
- `AlertDescription` --calls--> `cn()`  [EXTRACTED]
  HydraLabs/src/components/ui/alert.tsx → HydraLabs/src/lib/utils.ts
- `SelectScrollUpButton` --calls--> `cn()`  [EXTRACTED]
  HydraLabs/src/components/ui/select.tsx → HydraLabs/src/lib/utils.ts
- `SelectScrollDownButton` --calls--> `cn()`  [EXTRACTED]
  HydraLabs/src/components/ui/select.tsx → HydraLabs/src/lib/utils.ts
- `SelectLabel` --calls--> `cn()`  [EXTRACTED]
  HydraLabs/src/components/ui/select.tsx → HydraLabs/src/lib/utils.ts

## Import Cycles
- 3-file cycle: `HydraLabs/src/components/ActivityPanel/components/ActivityItem.tsx -> HydraLabs/src/components/ActivityPanel/index.ts -> HydraLabs/src/components/ActivityPanel/components/index.tsx -> HydraLabs/src/components/ActivityPanel/components/ActivityItem.tsx`
- 4-file cycle: `HydraLabs/src/components/ActivityPanel/ActivityPanel.tsx -> HydraLabs/src/components/ActivityPanel/components/index.tsx -> HydraLabs/src/components/ActivityPanel/components/ActivityItem.tsx -> HydraLabs/src/components/ActivityPanel/index.ts -> HydraLabs/src/components/ActivityPanel/ActivityPanel.tsx`

## Communities (124 total, 16 thin omitted)

### Community 0 - "AgentLoop"
Cohesion: 0.15
Nodes (4): AgentLoop, fake_lm(), FakeMCP, Agent loop mechanics with a scripted fake LM and a fake MCP. No real model,…

### Community 1 - "jira2case.py"
Cohesion: 0.06
Nodes (30): HTMLParser, generate(), plain(), Normalized case JSON -> Gherkin feature. Deterministic: every step and expected…, tag(), clean(), convert(), walk() (+22 more)

### Community 2 - "conftest.py"
Cohesion: 0.07
Nodes (24): hookimpl, Config, clear_steps_before_test(), driver(), PlaywrightDriverAdapter, fixture, pytest_html_results_summary(), pytest_runtest_makereport() (+16 more)

### Community 3 - "RunContext"
Cohesion: 0.07
Nodes (28): unmarshalInto(), Block, Factory, RunContext, NewRunContext(), Graph, LoadGraph(), RunResult (+20 more)

### Community 4 - ".SaveCrawl"
Cohesion: 0.11
Nodes (28): Bridge, Confirmed, Crawl, crawlDoc, CrawlElement, Stored, fold(), Config (+20 more)

### Community 5 - "cn"
Cohesion: 0.15
Nodes (22): Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle, DialogContent, DialogDescription (+14 more)

### Community 6 - "web/package.json"
Cohesion: 0.08
Nodes (28): dependencies, react, react-dom, @xyflow/react, devDependencies, vite, @vitejs/plugin-react, react (+20 more)

### Community 7 - "testing.T"
Cohesion: 0.12
Nodes (30): testing.T, TestADamagedLineDoesNotHideTheRest(), TestExecutionsAreAppendedNeverRewritten(), TestStatusCountsEachCasesHistory(), TestACommandCannotCreateAnAppsFoldersBeforeItsRepoExists(), TestAFreshCloneIsSeededFromTheCommittedFiles(), TestAnExplicitOutIsRespected(), TestAppOutputsLiveInTheAppsRepo() (+22 more)

### Community 8 - "Poll"
Cohesion: 0.19
Nodes (16): PollOpts, Case, Maturity(), BlockedMessage(), ChatNotify(), Message(), needsPerson(), classify() (+8 more)

### Community 9 - "use-toast.ts"
Cohesion: 0.15
Nodes (17): ToastActionElement, ToastProps, Toaster(), Action, ActionType, actionTypes, addToRemoveQueue(), dispatch() (+9 more)

### Community 10 - "lucide-react"
Cohesion: 0.14
Nodes (16): QuickHeaders(), QuickParams(), Button, ButtonProps, buttonVariants, Input, SelectContent, SelectItem (+8 more)

### Community 11 - "render_pytest.py"
Cohesion: 0.17
Nodes (21): analyse(), App, app_slug(), build_calls(), build_expected(), button(), csv_rows(), main() (+13 more)

### Community 12 - "matcher.py"
Cohesion: 0.11
Nodes (18): ask(), baseline(), bench(), catalog_text(), combine(), decide(), gold(), gold_cases() (+10 more)

### Community 13 - "join/main.go"
Cohesion: 0.14
Nodes (20): regexp.Regexp, join(), load(), main(), matchesAny(), norm(), TestJoin(), find() (+12 more)

### Community 14 - "writeJSON"
Cohesion: 0.12
Nodes (19): net/url.Values, Issue, Jira, Run, Verdict, CommentADF(), JiraFromEnv(), Config (+11 more)

### Community 15 - "Action"
Cohesion: 0.19
Nodes (16): ActionKind, Rect, fakePlanner, Action, Element, Goal, Observation, Step (+8 more)

### Community 16 - "HydraLabs/package.json"
Cohesion: 0.07
Nodes (28): react, react-dom, vite, @vitejs/plugin-react, name, private, type, version (+20 more)

### Community 17 - "HydraPloy + HydraLabs: scope, stages, rules"
Cohesion: 0.09
Nodes (22): App map (the crawl), Boards, apps and repos, Contracts, Environment, secrets, data (the framework's own convention), Failure triage, HydraPloy + HydraLabs: scope, stages, rules, Known data issues (facturación sample), Measuring it (+14 more)

### Community 18 - "test_spec_render.py"
Cohesion: 0.12
Nodes (9): case(), EndToEnd, framework(), golden(), Portal, read(), Renderer, Validator (+1 more)

### Community 19 - "init.go"
Cohesion: 0.12
Nodes (30): bufio.Reader, io.Reader, io.Writer, addon, answers, initRun, question, addonNames() (+22 more)

### Community 20 - "skeleton/main.go"
Cohesion: 0.22
Nodes (19): autoMap(), build(), collapse(), fieldsOf(), main(), prep(), removeEntry(), tagsOf() (+11 more)

### Community 21 - "compilerOptions"
Cohesion: 0.10
Nodes (20): compilerOptions, allowImportingTsExtensions, baseUrl, isolatedModules, jsx, lib, module, moduleResolution (+12 more)

### Community 22 - "ActivityPanel.tsx"
Cohesion: 0.17
Nodes (7): ActivityItem(), MetricCard(), ResultCard(), useMetricCard(), useResultCard(), copyToClipboard(), formatTime()

### Community 23 - "harness.py"
Cohesion: 0.14
Nodes (10): Run one generated test against a real URL. Read-only by default: every non-GET…, Driver, Runs a GENERATED test against a real URL without needing the framework repo:…, Reads always pass. A write passes only with allow_writes AND only to a loopback…, run(), net(), _tests_dir(), write_allowed() (+2 more)

### Community 24 - "components.json"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 25 - "testengine/cmd/demo/main.go"
Cohesion: 0.16
Nodes (16): buyBlock, loadBlock, logBlock, loginBlock, loadFactory(), percentile(), runFailed(), main() (+8 more)

### Community 26 - "tauri.conf.json"
Cohesion: 0.11
Nodes (17): app, security, windows, build, beforeBuildCommand, beforeDevCommand, devUrl, frontendDist (+9 more)

### Community 27 - "context.Context"
Cohesion: 0.27
Nodes (6): assertBlock, httpRequestBlock, orDefault(), context.Context, testengine/engine.RunContext, testengine/engine.Status

### Community 28 - "dependencies"
Cohesion: 0.12
Nodes (16): dependencies, class-variance-authority, clsx, lucide-react, @radix-ui/react-checkbox, @radix-ui/react-collapsible, @radix-ui/react-dialog, @radix-ui/react-select (+8 more)

### Community 29 - "agent.py"
Cohesion: 0.17
Nodes (11): chat(), compact(), local_tools(), map_summary(), openai_tools(), LM agent: one normalized case + the real page (via Playwright MCP, read-only)…, Keeps the conversation inside the model's context: old tool results are…, The crawl, compressed for the model: stable, unique locators only, one line… (+3 more)

### Community 30 - "crawl.py"
Cohesion: 0.14
Nodes (18): best_locator(), color_of(), crawl(), draw(), Events, label_of(), load_env(), --safe-clicks: open a combobox/tab just to read its options, then close it.… (+10 more)

### Community 31 - "PortalE2E"
Cohesion: 0.15
Nodes (5): Driver, Handler, PortalE2E, Drives the GENERATED TC-3 test in a real Chromium against a tiny fake portal…, skipUnless

### Community 32 - "scorecard.py"
Cohesion: 0.25
Nodes (15): anchors(), chat(), jaccard(), lint(), literals(), main(), match_all(), norm() (+7 more)

### Community 33 - "ApiSetup.tsx"
Cohesion: 0.22
Nodes (9): ApiSetup(), QuickAuthSelector(), RequestHeader(), useApiRequest(), useKeyValuePairs(), usePasswordVisibility(), useExpandableItems(), UseExpandableItemsOptions (+1 more)

### Community 34 - "APITesting.tsx"
Cohesion: 0.28
Nodes (5): ActivityPanel(), DEFAULT_ICONS, STATUS_STYLES, useActivityPanel(), APITesting()

### Community 35 - ".f"
Cohesion: 0.07
Nodes (14): Build, load_profile(), Generates gherkin-pipeline.json (the n8n workflow). Edit here, re-run, re-…, cargar_datos_csv(), load_csv_data(), ButtonStateTest, case_from_fixture(), OnlyTest (+6 more)

### Community 36 - "SideExplorer.tsx"
Cohesion: 0.23
Nodes (5): ScrollableContainer(), ResizeHandle(), ResizeHandleProps, useExplorerState(), SideExplorer()

### Community 37 - "fileExists"
Cohesion: 0.17
Nodes (11): copyFile(), Config, Registry, newRegistry(), TestNewProjectStopsAtTheBaseUrlGate(), TestOnboardingCreatesTheAppsOwnRepo(), TestOnboardingIsIdempotent(), TestOnboardingNeverWritesIntoAForeignFolder() (+3 more)

### Community 38 - "intake.go"
Cohesion: 0.41
Nodes (12): golang.org/x/net/html.Node, Case, Expected, Step, attr(), clean(), isEl(), item() (+4 more)

### Community 39 - "git"
Cohesion: 0.06
Nodes (47): flag.FlagSet, CaseOutcome, CaseStatus, Delivery, Execution, junitCase, junitSuite, Onboarding (+39 more)

### Community 40 - "render"
Cohesion: 0.23
Nodes (11): submit_spec(), camel(), module_header(), plain(), spec + Jira case -> README-layout files, deterministically. The agent never…, Writes the files. Returns {"files": [...], "warnings": [...], "errors": [...]}…, {NAME: (value, dynamic)} from an existing locators file., read_functions() (+3 more)

### Community 41 - "App.tsx"
Cohesion: 0.20
Nodes (6): defaultActivities, defaultFileTree, defaultTestSuites, getStatusIcon(), getTypeColor(), SideExplorer()

### Community 42 - "MCPClient"
Cohesion: 0.14
Nodes (7): MCPClient, playwright_command(), Minimal MCP client over stdio (newline-delimited JSON-RPC): initialize,…, Playwright MCP with the read-only guard injected. Pinned: option names change…, GuardTest, Handler, Proves the read-only guard: through the real Playwright MCP, a click that makes…

### Community 43 - "UrlInput.tsx"
Cohesion: 0.36
Nodes (6): UrlInput(), Alert, AlertDescription, AlertTitle, alertVariants, validateUrl()

### Community 44 - ".Observe"
Cohesion: 0.27
Nodes (7): fakeSurface, newFakeSurface(), onoff(), savedLabel(), main(), outcome(), printTranscript()

### Community 45 - "queue_test.go"
Cohesion: 0.29
Nodes (12): job, Config, order(), fakeJira(), fakeProject(), Config, jb(), keys() (+4 more)

### Community 46 - "apiConstants.ts"
Cohesion: 0.18
Nodes (9): BodyEditor(), MethodSelector(), AUTH_TYPES, BODY_TYPES, COMMON_HEADERS, CONTENT_TYPES, getDefaultRequestTemplate(), HTTP_METHODS (+1 more)

### Community 47 - "write_allowed"
Cohesion: 0.27
Nodes (6): _hydra_guard(), guard(), fixture, pytest plugin (load with -p hydra_guard): generated tests run read-only. Any…, write_allowed(), PytestPluginGuard

### Community 48 - "properties"
Cohesion: 0.22
Nodes (11): items, allOf, properties, required, type, note, status, steps (+3 more)

### Community 49 - "Framework de Automatización E2E con Playwright"
Cohesion: 0.18
Nodes (10): 1. Cambiar de navegador localmente, 2. Emulación de dispositivos móviles o tablets, **¿Cuándo debemos usar e interactuar con esta carpeta?**, **Descripción de componentes:**, Ejecución Multiplataforma y Emulación, Estructura del Proyecto, Estructura y Uso de la Carpeta `utils`, Framework de Automatización E2E con Playwright (+2 more)

### Community 51 - "structure_check.py"
Cohesion: 0.31
Nodes (8): _calls_on(), check(), _imports(), _page_calls(), Deterministic README-compliance check for what the agent wrote under…, (attr, line) for every <name>.<attr>(...) call, e.g. wrapper.is_visible()., page.<attr>() reached as driver.page.<attr>() or page.<attr>()., _read()

### Community 52 - "Cómo Agregar Nuevos Tests"
Cohesion: 0.20
Nodes (10): **1. ¿Quién lo construye y de dónde viene?**, **2. ¿Cómo funciona y de dónde sale el objeto `driver`?**, **3. ¿Cómo sé si lo estoy llamando bien? (Métodos Comunes)**, Cómo Agregar Nuevos Tests, **Guía Didáctica: ¿Cómo usar `DriverWrapper` de forma correcta?**, **Opción A: Escenario de Negocio Único (Enfoque Recomendado para E2E )**, **Opción B: Escenarios Parametrizados (Multi-caso con Decorador)**, Paso 2: Crear las acciones reutilizables (Módulo de Acción) (+2 more)

### Community 53 - "ollama.go"
Cohesion: 0.31
Nodes (7): net/http.Client, main(), NewOllama(), chatMessage, chatReq, chatResp, Ollama

### Community 54 - "SideExplorer/components/index.tsx"
Cohesion: 0.22
Nodes (5): FileTreeItem(), FileTreeSection(), FooterStats(), Header(), SectionTabs()

### Community 55 - "TestSuiteCard.tsx"
Cohesion: 0.33
Nodes (6): TestSuiteCard(), TestSuitesSection(), defaultFileTree, defaultTestSuites, getStatusIcon(), getTypeColor()

### Community 56 - "devDependencies"
Cohesion: 0.18
Nodes (11): devDependencies, autoprefixer, postcss, tailwindcss, @tauri-apps/cli, @types/node, @types/react, @types/react-dom (+3 more)

### Community 57 - "AdvancedSection.tsx"
Cohesion: 0.27
Nodes (5): AnimatedCollapse(), AdvancedSection(), ExpandableSection(), KeyValueEditor(), Textarea

### Community 58 - "Sandbox"
Cohesion: 0.19
Nodes (3): build(), Sandbox, StructureCheck

### Community 59 - "3. Configurar variables de entorno y mapearlas con el archivo CSV"
Cohesion: 0.22
Nodes (9): 1. Instalar dependencias, 2. Instalar navegadores de Playwright, 3. Configurar variables de entorno y mapearlas con el archivo CSV, Instalación y Configuración, **Paso 1: Crear o editar tu archivo `.env`**, **Paso 2: Vincular las variables en el CSV de datos**, **Paso 3: Funcionamiento automático en ejecución**, **Paso 4: Uso del archivo `.gitignore` para seguridad** (+1 more)

### Community 60 - "xml_reader.py"
Cohesion: 0.25
Nodes (8): assert_cfdi_matches(), assert_cfdi_tiene_conceptos(), extract_cfdi_data(), _find_comprobante_ns(), Detecta si el XML es CFDI 4.0 o 3.3 según el namespace del root., Parsea un CFDI (XML) y extrae los campos más relevantes para pruebas: fecha,…, Compara campos puntuales del CFDI extraído contra los valores esperados de la…, Valida que el CFDI tenga al menos `minimo` conceptos (artículos).

### Community 62 - "casecheck/main.go"
Cohesion: 0.36
Nodes (6): Case, Step, check(), main(), ready(), TestCheck()

### Community 63 - "init_test.go"
Cohesion: 0.40
Nodes (9): Registry, initRegistry(), offline(), readJSON(), TestAutomateNeedsLabs(), TestInitAsksFixesATypoAndWritesOnlyAfterYes(), TestInitFromAnswersFileWithoutPrompts(), TestInitWritesNothingWhenNotConfirmedOrInvalid() (+1 more)

### Community 64 - "spec.py"
Cohesion: 0.43
Nodes (6): as_list(), _check_errors(), locator_selector(), The declarative spec the agent fills in (instead of writing code), and its…, -> list of error strings the agent must fix (empty = valid)., validate()

### Community 65 - "type"
Cohesion: 0.25
Nodes (8): description, items, type, type, depends_on, warnings, items, type

### Community 66 - "compilerOptions"
Cohesion: 0.25
Nodes (7): compilerOptions, allowSyntheticDefaultImports, composite, module, moduleResolution, skipLibCheck, include

### Community 67 - ".crawlArgs"
Cohesion: 0.43
Nodes (3): Config, join(), Serve()

### Community 68 - "runner.py"
Cohesion: 0.36
Nodes (7): dispatch(), _pytest(), Checks the agent can run on what it wrote: README structure, compile,…, Which static locators of apps/<App>/locators/<page>_locators.py resolve on the…, run_checks(), run_test(), verify_locators()

### Community 70 - "lint"
Cohesion: 0.43
Nodes (5): lint(), main(), TestLint(), Opts, Result

### Community 71 - "build_hydra_workflow.py"
Cohesion: 0.38
Nodes (5): config(), hydra(), node(), Generates hydra-pipeline.json: n8n as a thin, visible shell around the Go…, sh()

### Community 72 - "bridge_test.go"
Cohesion: 0.54
Nodes (7): crawl(), el(), TestEnglishCaseWordsMatchSpanishPage(), TestHumanFileWinsOverLearned(), TestModelOnlyChoosesFromTheOfferedList(), TestRunConfirmsGuessesAndMarksCustomWidgets(), TestUnsureStaysUnresolvedWithoutAModel()

### Community 73 - "properties"
Cohesion: 0.29
Nodes (7): properties, required, type, data, lists, rows, shared

### Community 74 - "Buenas Prácticas y Modelo SOLID"
Cohesion: 0.29
Nodes (7): 1. **Single Responsibility Principle (SRP) - Principio de Responsabilidad Única**, 2. **Open/Closed Principle (OCP) - Principio de Abierto/Cerrado**, 3. **Liskov Substitution Principle (LSP) - Principio de Sustitución de Liskov**, 4. **Interface Segregation Principle (ISP) - Principio de Segregación de Interfaces**, 5. **Dependency Inversion Principle (DIP) - Principio de Inversión de Dependencias**, Buenas Prácticas y Modelo SOLID, Otras Buenas Prácticas Generales

### Community 75 - "Agent"
Cohesion: 0.40
Nodes (4): Agent, Planner, Surface, fingerprint()

### Community 76 - "default.json"
Cohesion: 0.33
Nodes (5): description, identifier, permissions, $schema, windows

### Community 78 - "scripts"
Cohesion: 0.40
Nodes (5): scripts, build, dev, preview, tauri

### Community 79 - "suite/main.go"
Cohesion: 0.53
Nodes (5): lines(), main(), nonBlank(), slug(), Case

### Community 80 - "case.schema.json"
Cohesion: 0.33
Nodes (5): description, required, $schema, title, type

### Community 81 - "properties"
Cohesion: 0.33
Nodes (6): type, items, type, properties, cases, orphans

### Community 82 - "catalog.py"
Cohesion: 0.60
Nodes (5): functions(), locators(), main(), parse(), Reads a framework repo (README layout: apps/<App>/{modules,locators},…

### Community 84 - "deprecated/"
Cohesion: 0.40
Nodes (4): Candidates (not moved; dependencies checked), deprecated/, Moved so far, Rules

### Community 85 - "TestIntakeMatchesPythonGolden"
Cohesion: 0.67
Nodes (3): asJSON(), TestIntakeMatchesPythonGolden(), TestResolveRefs()

### Community 86 - "system_es.md"
Cohesion: 0.40
Nodes (4): Cómo trabajar, Honestidad, La spec (JSON), Seguridad (no negociable)

### Community 89 - "meta"
Cohesion: 0.40
Nodes (5): type, additionalProperties, description, type, meta

### Community 90 - "Cómo Ejecutar los Tests (Ejecución Granular)"
Cohesion: 0.40
Nodes (5): **1. Ejecutar sin generar reportes (Más rápido para pruebas locales)**, **2. Ejecutar generando el Reporte HTML**, Cómo Ejecutar los Tests (Ejecución Granular), Opción A: A través de `runner_central.py` (Recomendado para Suites Completas), Opción B: Ejecución Granular vía `pytest` (Desarrollo y Depuración)

### Community 91 - "2. Pruebas de API HTTPS (GET, POST, PUT, DELETE)"
Cohesion: 0.40
Nodes (5): 1. Pruebas de Rendimiento con JMeter, 2. Pruebas de API HTTPS (GET, POST, PUT, DELETE), Integración de Pruebas de API (HTTPS) y Rendimiento (JMeter), **Opción A: Cliente API nativo de Playwright**, **Opción B: Uso de la librería `requests` (Tests de API puros)**

### Community 92 - "pdf_reader.py"
Cohesion: 0.40
Nodes (4): assert_invoice_contains(), extract_invoice_data(), Extrae el texto completo de un PDF y busca el UUID fiscal (CFDI), que sigue el…, Valida que ciertos fragmentos de texto (ej. RFC, nombre, forma de pago) estén…

### Community 93 - "test_files.py"
Cohesion: 0.40
Nodes (4): binary_payload(), csv_payload(), Archivo binario determinista (contiene bytes NUL, por lo tanto nunca es…, CSV con el encabezado y las filas dadas (valores arbitrarios: vacíos, enormes,…

### Community 97 - "start.sh"
Cohesion: 0.50
Nodes (3): N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS, NODES_EXCLUDE, start.sh script

### Community 98 - "id"
Cohesion: 0.50
Nodes (4): description, minLength, type, id

### Community 99 - "Escalabilidad y Soporte Multi-Aplicación"
Cohesion: 0.50
Nodes (4): 1. Estructura de Carpetas, 2. Actualizar el Orquestador Central (`runner_central.py`), 3. Ejecución de la nueva aplicación, Escalabilidad y Soporte Multi-Aplicación

### Community 100 - "Paso 1: Definir los localizadores"
Cohesion: 0.50
Nodes (4): **Ejemplo de Archivo de Localizadores:**, **Guía de Buenas Prácticas: ¿Cómo elegir el mejor localizador?**, **Mini-Guía: ¿Cómo construir un Selector CSS estable?**, Paso 1: Definir los localizadores

### Community 106 - "name"
Cohesion: 0.67
Nodes (3): minLength, type, name

## Knowledge Gaps
- **246 isolated node(s):** `$schema`, `style`, `rsc`, `tsx`, `config` (+241 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 494 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **16 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `ParseAction()` connect `Action` to `ollama.go`?**
  _High betweenness centrality (0.048) - this node is a cross-community bridge._
- **Why does `TestParseClickCoords()` connect `Action` to `testing.T`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._
- **Why does `TestParseAction()` connect `Action` to `testing.T`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._
- **Are the 19 inferred relationships involving `fileExists()` (e.g. with `.Deliver()` and `.requireHome()`) actually correct?**
  _`fileExists()` has 19 INFERRED edges - model-reasoned connections that need verification._
- **What connects `$schema`, `style`, `rsc` to the rest of the system?**
  _246 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `jira2case.py` be split into smaller, more focused modules?**
  _Cohesion score 0.06363636363636363 - nodes in this community are weakly interconnected._
- **Should `conftest.py` be split into smaller, more focused modules?**
  _Cohesion score 0.07123034227567067 - nodes in this community are weakly interconnected._