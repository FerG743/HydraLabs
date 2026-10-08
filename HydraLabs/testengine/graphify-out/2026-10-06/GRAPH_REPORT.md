# Graph Report - testengine  (2026-10-06)

## Corpus Check
- 106 files · ~63,250 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 746 nodes · 1239 edges · 54 communities (43 shown, 9 thin omitted)
- Extraction: 97% EXTRACTED · 3% INFERRED · 0% AMBIGUOUS · INFERRED: 43 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `76f514c9`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- RunContext
- agent.py
- .f
- App.jsx
- .Route
- test_spec_render.py
- testing.T
- jira2case.py
- resolve_refs.py
- render_pytest.py
- matcher.py
- join/main.go
- AgentLoop
- skeleton/main.go
- HydraPloy + HydraLabs: scope, stages, rules
- PortalE2E
- scorecard.py
- test_agent_parts.py
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
- _hydra_readonly
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
- ResolveRefs
- .Crawl
- CobroOrdenes.md
- bridge.go
- playwright_command
- structure_check.py
- MCPClient
- Jira
- runner.py
- Sandbox
- build_hydra_workflow.py
- Sandbox

## God Nodes (most connected - your core abstractions)
1. `RunContext` - 18 edges
2. `HydraPloy + HydraLabs: scope, stages, rules` - 18 edges
3. `MCPClient` - 14 edges
4. `render()` - 14 edges
5. `Registry` - 14 edges
6. `build()` - 12 edges
7. `Status` - 12 edges
8. `run()` - 11 edges
9. `Renderer` - 11 edges
10. `Config` - 11 edges

## Surprising Connections (you probably didn't know these)
- `registry()` --calls--> `NewRegistry()`  [EXTRACTED]
  cmd/demo/main.go → engine/registry.go
- `main()` --calls--> `LoadGraph()`  [EXTRACTED]
  cmd/demo/main.go → engine/graph.go
- `GuardTest` --uses--> `MCPClient`  [INFERRED]
  agent/test_guard.py → agent/mcp_client.py
- `loadBlock` --references--> `Graph`  [EXTRACTED]
  cmd/demo/loadblock.go → engine/graph.go
- `loadBlock` --references--> `Registry`  [EXTRACTED]
  cmd/demo/loadblock.go → engine/registry.go

## Import Cycles
- None detected.

## Communities (54 total, 9 thin omitted)

### Community 0 - "RunContext"
Cohesion: 0.06
Nodes (43): orDefault(), loadFactory(), percentile(), runFailed(), main(), printSteps(), registry(), reportValidation() (+35 more)

### Community 1 - "agent.py"
Cohesion: 0.21
Nodes (9): chat(), compact(), local_tools(), openai_tools(), LM agent: one normalized case + the real page (via Playwright MCP, read-only)…, Keeps the conversation inside the model's context: old tool results are…, run(), Layout the README requires, in advance and idempotently. Returns warnings (e.g.… (+1 more)

### Community 2 - ".f"
Cohesion: 0.07
Nodes (23): submit_spec(), Build, camel(), module_header(), plain(), spec + Jira case -> README-layout files, deterministically. The agent never…, Writes the files. Returns {"files": [...], "warnings": [...], "errors": [...]}…, {NAME: (value, dynamic)} from an existing locators file. (+15 more)

### Community 3 - "App.jsx"
Cohesion: 0.08
Nodes (28): react, react-dom, vite, @vitejs/plugin-react, @xyflow/react, dependencies, react, react-dom (+20 more)

### Community 4 - ".Route"
Cohesion: 0.14
Nodes (20): cfgFlags(), Config, main(), poll(), report(), Case, Maturity(), ChatNotify() (+12 more)

### Community 5 - "test_spec_render.py"
Cohesion: 0.13
Nodes (9): case(), EndToEnd, framework(), golden(), Portal, read(), Renderer, Validator (+1 more)

### Community 6 - "testing.T"
Cohesion: 0.14
Nodes (26): crawl(), el(), TestEnglishCaseWordsMatchSpanishPage(), TestHumanFileWinsOverLearned(), TestModelOnlyChoosesFromTheOfferedList(), TestRunConfirmsGuessesAndMarksCustomWidgets(), TestUnsureStaysUnresolvedWithoutAModel(), asJSON() (+18 more)

### Community 7 - "jira2case.py"
Cohesion: 0.12
Nodes (15): HTMLParser, clean(), convert(), walk(), item(), Node, Jira issue description (HTML) -> normalized case JSON. Deterministic, stdlib…, Inline text of a node, skipping nested lists; <code> keeps its backticks (value… (+7 more)

### Community 8 - "resolve_refs.py"
Cohesion: 0.13
Nodes (15): generate(), plain(), Normalized case JSON -> Gherkin feature. Deterministic: every step and expected…, tag(), find_target(), lint(), precondition_of(), Resolves "Same as TC-1" preconditions across the cases already generated in a… (+7 more)

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
Cohesion: 0.16
Nodes (4): AgentLoop, fake_lm(), FakeMCP, Agent loop mechanics with a scripted fake LM and a fake MCP. No real model,…

### Community 13 - "skeleton/main.go"
Cohesion: 0.22
Nodes (19): autoMap(), build(), collapse(), fieldsOf(), main(), prep(), removeEntry(), tagsOf() (+11 more)

### Community 14 - "HydraPloy + HydraLabs: scope, stages, rules"
Cohesion: 0.10
Nodes (20): App map (the crawl), Contracts, Environment, secrets, data (the framework's own convention), Failure triage (target), HydraPloy + HydraLabs: scope, stages, rules, Known data issues (facturación sample), Measuring it, Model roles (all bounded, all validated) (+12 more)

### Community 15 - "PortalE2E"
Cohesion: 0.15
Nodes (5): skipUnless, Driver, Handler, PortalE2E, Drives the GENERATED TC-3 test in a real Chromium against a tiny fake portal…

### Community 16 - "scorecard.py"
Cohesion: 0.25
Nodes (15): anchors(), chat(), jaccard(), lint(), literals(), main(), match_all(), norm() (+7 more)

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

### Community 28 - "_hydra_readonly"
Cohesion: 0.40
Nodes (3): _hydra_readonly(), pytest plugin (load with -p hydra_guard): generated tests run read-only. Any…, fixture

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
Cohesion: 0.24
Nodes (9): best_locator(), crawl(), Events, load_env(), Read-only crawler: every reachable page of a web app -> every interactive…, Framework priority: data-testid, then a stable id, then role+accessible name,…, --safe-clicks: open a combobox/tab just to read its options, then close it.…, record_options() (+1 more)

### Community 41 - "casecheck/main.go"
Cohesion: 0.36
Nodes (6): Case, Step, check(), main(), ready(), TestCheck()

### Community 42 - "ResolveRefs"
Cohesion: 0.80
Nodes (5): findTarget(), Case, original(), preconditionOf(), ResolveRefs()

### Community 43 - ".Crawl"
Cohesion: 0.47
Nodes (3): Config, join(), Serve()

### Community 45 - "bridge.go"
Cohesion: 0.22
Nodes (9): fold(), Config, inRoles(), loadConfirmed(), tokens(), Bridge, Confirmed, Crawl (+1 more)

### Community 46 - "playwright_command"
Cohesion: 0.20
Nodes (6): playwright_command(), Minimal MCP client over stdio (newline-delimited JSON-RPC): initialize,…, Playwright MCP with the read-only guard injected. Pinned: option names change…, GuardTest, Handler, Proves the read-only guard: through the real Playwright MCP, a click that makes…

### Community 47 - "structure_check.py"
Cohesion: 0.31
Nodes (8): _calls_on(), check(), _imports(), _page_calls(), Deterministic README-compliance check for what the agent wrote under…, (attr, line) for every <name>.<attr>(...) call, e.g. wrapper.is_visible()., page.<attr>() reached as driver.page.<attr>() or page.<attr>()., _read()

### Community 49 - "Jira"
Cohesion: 0.42
Nodes (4): JiraFromEnv(), net/url.Values, Issue, Jira

### Community 50 - "runner.py"
Cohesion: 0.36
Nodes (7): dispatch(), _pytest(), Checks the agent can run on what it wrote: README structure, compile,…, Which static locators of apps/<App>/locators/<page>_locators.py resolve on the…, run_checks(), run_test(), verify_locators()

### Community 52 - "build_hydra_workflow.py"
Cohesion: 0.38
Nodes (5): config(), hydra(), node(), Generates hydra-pipeline.json: n8n as a thin, visible shell around the Go…, sh()

## Knowledge Gaps
- **71 isolated node(s):** `Case`, `RuleKind`, `testengine`, `start.sh script`, `NODES_EXCLUDE` (+66 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 223 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **9 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `ToCase()` connect `intake.go` to `.Route`, `testing.T`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **Why does `MCPClient` connect `MCPClient` to `agent.py`, `playwright_command`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Why does `run()` connect `skeleton/main.go` to `testing.T`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **What connects `Case`, `RuleKind`, `testengine` to the rest of the system?**
  _71 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `RunContext` be split into smaller, more focused modules?**
  _Cohesion score 0.057971014492753624 - nodes in this community are weakly interconnected._
- **Should `.f` be split into smaller, more focused modules?**
  _Cohesion score 0.06547619047619048 - nodes in this community are weakly interconnected._
- **Should `App.jsx` be split into smaller, more focused modules?**
  _Cohesion score 0.0784313725490196 - nodes in this community are weakly interconnected._