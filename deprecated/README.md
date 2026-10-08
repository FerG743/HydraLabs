# deprecated/

Everything here is **not part of the HydraLabs pipeline** (`HydraLabs/testengine`: Jira → gate → crawl → render → verify → deliver). It is kept, not deleted, in case something is worth reviving.

## Rules

- Move with `git mv`, so history follows the file. Keep the original relative path under `deprecated/` (e.g. `deprecated/testengine/engine/`).
- Nothing in the pipeline may import from, call or depend on this folder. If a move breaks the pipeline's build or tests, it is not deprecated yet.
- Put a line in the table below when you move something: what it was, why it was retired, and what replaced it.

## Moved so far

| Path | What it was | Why retired | Replaced by |
|---|---|---|---|
| `testengine/engine/`, `testengine/cmd/demo/` | A Go block-graph engine (blocks, rules, an interpreter) and its demo runner (`login_then_buy.json`) | Never part of the Jira → test pipeline; the pipeline needed no engine | `cmd/hydra` (the router) and the tiers in PIPELINE.md |
| `testengine/web/` (`testengine-canvas`), `testengine/examples/` | A React Flow canvas for drawing block graphs for that engine (its catalog mirrored the engine's `BlockMeta` and it exported the JSON the engine consumed), and the sample graph the demo used (a byte-identical copy of the demo's) | Only existed to serve the engine; nothing referenced them | n/a |

Code here is **not built** with the pipeline: it lives outside the `testengine` Go module, so its imports (e.g. `testengine/engine`) do not resolve from this folder. To revive something, move it back with `git mv` first.

## Candidates (not moved; dependencies checked)

PIPELINE.md lists these as "parked". Moving each one needs the dependency on the right resolved first.

| Candidate | Still used by | Safe to move when |
|---|---|---|
| Tauri desktop app (`HydraLabs/src`, `src-tauri`, `package.json`, `vite.config.ts`, ...) | nothing in `testengine` | you decide the pipeline needs no dashboard (note: `src/App.tsx` has your own uncommitted change) |
| `uiagent/` (vision planner) | nothing in `testengine` references it (I did not search the desktop app) | any time, after checking the app |
| `testengine/cmd/skeleton/`, `cmd/lint/` (Gherkin output) | the old Gherkin n8n workflows, `tools/jira/gherkin_gen.py`, `jira2feature.py`, `resolve_refs.py`, `tools/scorecard/` | those old workflows and scripts move with them |
| `n8n/jira-pipeline.json`, `jira-pipeline-test.json`, `build_jira_workflow.py` (Jira → Gherkin) | n8n still has them imported (`jira-gherkin`, `jira-gherkin-test`) | you delete or deactivate them in n8n |
| `n8n/facturacion-pipeline.json`, `build_workflow.py`, `n8n/profiles/facturacion.json` (CSV matrix → Gherkin) | `cmd/matrix`, `cmd/suite`, `cmd/join` (CSV adapters), imported in n8n as `gherkin-test` | the CSV source is retired |
| `testengine/cmd/matrix`, `cmd/suite`, `cmd/join`, `cmd/casecheck`, `schema/` (CSV adapters + contract) | `tools/matcher/matcher.py` reads matrix output | the matcher benchmark is retired |

`cmd/hydra/resolve.go` is the Go port of `tools/jira/resolve_refs.py`. The Python version imports `gherkin_gen`, so it can retire together with the Gherkin path above.
