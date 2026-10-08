# HydraPloy + HydraLabs: scope, stages, rules

## Scope (one sentence)

**A test case enters as a Jira issue and leaves as a passing, human-reviewed automated test in the framework repo.**

Everything else is a stage of that, a source feeding it, or a consumer of it. Anything that is none of those is out of scope until this works end to end.

## Two products, one repo

- **HydraPloy** is the pipeline: a fully automated QA/testing CI/CD. It moves work through gates and decides what passed.
- **HydraLabs** is the workshop: where agents actually build the automations (tests, locators, modules, app changes).

The boundary is a contract, not a convention:

```
 Jira issue ──► HydraPloy ──── work order ───► HydraLabs
                 (intake, gates)                (agents build)
                      ▲                              │
                      └──── automation bundle ◄──────┘
                 verify · deliver · report
```

- **Work order** (Ploy → Labs): a normalized case, the profile, and the framework catalog.
- **Automation bundle** (Labs → Ploy): a plain folder of files in framework README layout, plus `bundle.json` (which framework functions were matched, which steps are unresolved, what the model was asked, warnings).
- **Ploy verifies every bundle. Labs never grades its own work.** This is the independent-oracle rule enforced by structure: the agents that build something are never the ones that decide whether it passed.
- **Dividing rule:** if it touches a Jira issue, a gate, a run or a result, it is Ploy. If it produces test code or touches the app under test, it is Labs. A feature that fits neither waits.

| | HydraPloy | HydraLabs |
|---|---|---|
| Owns | triggers, intake adapters, pre-flight, orchestration (n8n), Verify, Deliver, gates, run records | Stage 0 design, Plan, Build, app map + exploration, coding agent (Loop B) |
| Code today | `cmd/hydra` (poll, route, gate, verify), `cmd/matrix`, `cmd/suite`, `cmd/join`, `cmd/casecheck`, `n8n/`, `schema/`, `profiles/` | `tools/crawl/` (crawler), `cmd/hydra/bridge.go` (plan), `tools/jira/render_pytest.py` (renderer), `agent/` (model tier), `knowledge/`, `tools/catalog.py`; parked: `cmd/skeleton`, `cmd/lint`, `uiagent/`, the Tauri app |

### Target layout (not moved yet; the move happens during slice 1)

```text
HydraLabs/                    # repo
├── ploy/                     # HydraPloy: pipeline
│   ├── intake/               #   adapters: csv (matrix, suite, join), jira
│   ├── schema/               #   case.schema.json, work-order, bundle
│   ├── casecheck/            #   contract enforcement
│   ├── orchestration/        #   n8n workflow + profiles/
│   └── verify/ deliver/      #   run bundle in the framework; branch/PR, Jira
├── labs/                     # HydraLabs: agents that build
│   ├── plan/                 #   catalog + matcher
│   ├── build/                #   renderer, skeleton, explore, app map
│   └── agents/               #   coding agent, vision agent (parked)
├── app/                      # the Tauri desktop app (parked)
└── PIPELINE.md
```

## Rules (what keeps this from becoming a behemoth)

1. **One contract per stage.** Each stage is a small command: JSON in, JSON out. It runs and is tested alone, with at least one golden test.
2. **n8n orchestrates, nothing more.** No business logic in workflow nodes beyond glue.
3. **Per-project differences live in a profile file** (`n8n/profiles/<name>.json`), never in code.
4. **Models do bounded, optional jobs.** Code does everything it can; the model fills gaps and its output is always validated. A bad model answer must degrade to "needs review", never to a wrong result.
5. **Independent oracle.** Expected behavior comes from human-written requirements. It is never derived from the implementation, and the agent that writes code never writes the tests that judge it.
6. **Human gates.** A person approves cases before automation and reviews every PR. No auto-merge.
7. **A case counts as automated only if its generated test passes in the framework.**
8. **New work must fit a stage.** If it doesn't, it waits.

## Stages

A **human writes and documents every case in Jira first.** The pipeline starts from that, never from nothing.

| # | Stage | Owner | In → Out | Model? | Status | Code |
|---|---|---|---|---|---|---|
| 0 | **Design** (optional source) | Labs | requirements → draft cases | yes | not started | - |
| 1 | **Intake + maturity gate** | Ploy | Jira issue → normalized case, or specific questions back to the author | no | done (Jira polling untested live) | `cmd/hydra/intake.go`, `resolve.go`, `maturity.go` |
| 2 | **Crawl** (once per app) | Labs | running app → every page, element, best locator, options | no | done | `tools/crawl/crawl.py`, `hydra crawl` |
| 3 | **Plan / bridge** | Labs | case words ↔ crawled elements (confirmed locators, widget kinds) | rarely (closed list) | done | `cmd/hydra/bridge.go` |
| 4 | **Build** | Labs | plan → bundle in the framework README layout | only for gaps (agent) | rules done · agent untested live | `tools/jira/render_pytest.py`, `agent/` |
| 5 | **Verify** | Ploy | bundle → pass/fail in the real app, read-only unless allowed | no | done (harness, not yet the real framework repo) | `tools/jira/run_generated.py`, `router.go` |
| 6 | **Deliver** | Ploy | verified bundle → branch/PR, Jira comment + status | no | Jira/Chat notify only (off by default); no PR yet | `notify.go` |

Cross-cutting: **profile** (one JSON per web app), **knowledge** (`knowledge/<App>.md`, the app's quirks), **run record** (`out/hydra/runs/<KEY>.json`: tier, status, tokens, ms).

### Contracts

- **Normalized case**: `schema/case.schema.json`, enforced by `cmd/casecheck` (rejects bad statuses, ready cases without steps or data rows, empty names, duplicate ids among ready cases). Every adapter must produce it; downstream stages see nothing else.
  **Known gap:** the CSV adapters produce this shape, but the Jira adapter (`cmd/hydra/intake.go`) produces its own (`steps[].n/action`, `expected[].text`, `precondition`, no `status`), which is what the renderer and agent consume. Reconcile (map Jira cases to this schema, or version the schema) before a third source is added.
- **Work order / bundle**: the Ploy↔Labs boundary (see "Two products"). Both get a schema next to `case.schema.json` and are validated at the boundary, like cases.
- **Profile**: which columns/fields mean what, the case key, tag fields, blocked phrase. Missing required keys fail at build time.
- **Output layout** follows the framework README exactly: `apps/<App>/{data,locators,modules}`, `tests/<App>/test_*.py`, `expected_result.csv`, `ENV:` tokens for secrets (never literal credentials), `DriverWrapper` only (no raw Playwright), `add_step(..., level="business")`.

## Boards, apps and repos

One Jira project (board) = one application under test. Humans write and document the cases there; the pipeline starts from that.

- **Registry** (`projects.json`): the Jira projects the pipeline automates, each with its app name and profile (`"DWQ": {"app": "CobroOrdenes", ...}`). Registration is explicit on purpose: a Jira token sees dozens of projects and must never turn each one into a repo.
- **Onboarding** (`hydra onboard --project KEY`, and automatically before every poll): the first time a registered project is seen, the pipeline creates **`Hydra-<App>`** next to the other repos on the machine that runs n8n (`reposRoot`, default `~/Documents/GitHub`). It is built from the framework template (`templates/framework/`), with the suite registered in `runner_central.py` and one baseline commit, so every later review branch shows only generated tests. It never writes into a folder that is not already a Hydra repo, and a second run changes nothing.
- **Onboarding gate**: Jira cannot know where the app runs. A new project gets a read-only default profile with an empty `baseUrl` and stops with `needs-onboarding` until a person sets it (and `allowWrites` only for a QA environment that may be changed). With a URL set, the first crawl runs automatically.
- **Delivery** goes to that app's repo: `hydra deliver --project KEY [--runs N]`. The cases are run N times in a row in the real framework (stability gate, default 3; repeating stops at the first failure). Only a case that passes every run is committed, on a local review branch; a flaky or failing case blocks the whole delivery (exit 3) until a person decides. Re-delivering identical files reports "already delivered" instead of failing. Nothing is pushed or merged. Note: a case that writes (e.g. DWQ-130) writes N times per delivery, so a QA environment must tolerate that.
- n8n polls the whole registry (`hydra poll`); "Manual: run one issue" finds the project from the key's prefix.

## Routing: cheapest tier that can finish, escalating only when it cannot

Entry: a Jira issue labeled `automate` or `execute` (`execute` wins if both). Polled, not webhooked (Jira Cloud cannot call localhost). n8n is a thin, visible shell around the `hydra` binary; every manual trigger is its own entry point.

```
T0 gate     maturity check: steps, expected results, placeholders (TBD/pendiente), unresolved "Same as TC-1"
            incomplete -> needs-clarification: specific questions for the author, 0 tokens, nothing else runs
execute  -> T0 existing   a stored bundle for the key -> run it                          (0 tokens)
automate -> T1 render     parse + render from the case, bridge confirms locators against the crawl
            T2 bridge     a closed-list pick (<=12 candidates) only when rules tie       (~100-250 tokens, cached forever)
            verify        run the generated test in the real app
            T3 agent      only if T1 left unmappable steps or its output failed verification: the model explores the
                          real page (read-only) with the app's knowledge notes, under a hard turn budget
            T4 review     a person, with the bundle and the exact reason; never a retry loop
```

Rules of the router:
- A test that fails only because the read-only guard blocked a real write is `needs-writes`. It never escalates; no model can fix it.
- The agent is called at most once per case, and its output is judged by the same verify as everything else.
- A case is *automated* only if its test passes. "Rendered" is not automated.
- Resolved mappings are cached (`appmap/<App>.learned.json`), so cost falls with every run. A human-written `appmap/<App>.json` always wins.
- Per-run tokens are logged in the run record; the number to improve is successes per token, per tier.

## Environment, secrets, data (the framework's own convention)

- Credentials live in the framework's `.env` and are referenced from CSVs as `ENV:NAME`. The pipeline never reads, writes or logs a secret value; the crawler's login steps use `ENV:` tokens and record `***`.
- Pipeline secrets (`JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_TOKEN`, `HYDRA_CHAT_WEBHOOK`) live in `~/.hydra.env`, loaded by `n8n/start.sh`; never in a profile or the repo.
- Tests run **read-only by default** (every non-GET is aborted). `allowWrites` in the profile turns that off for a test environment that may be changed.
- Still needed per app: how to reset or seed data, and whether writes are allowed in that environment. Without it, mutating cases stop at `needs-writes`.

## Failure triage

A failure says *why*, and only test bugs reach the model:

| Outcome | Meaning | Goes to |
|---|---|---|
| `needs-clarification` | the case is incomplete or ambiguous (gate) | the author, with questions |
| `needs-writes` | only the write guard stopped it | a decision on the environment |
| `env-down` | the app (or VPN, backend) is unreachable | a person; never the model: it would burn tokens on something no model can fix |
| `flaky` (delivery) | passed some stability runs and failed others | a person; never delivered |
| `failed` + "possible app bug" | the elements were found and an **assertion** about behavior failed | a person; never the model, which must not loosen the check |
| `failed` (locator, timeout, strict-mode) | a test bug | T3 agent, once |
| `needs-review` | the pipeline could not finish within its budget | a person, with the bundle and the reason |

Not separated yet: environment problems (VPN down, backend 500) look like test failures.

Real example (DWQ-132 on QA): all 5 steps map at T1, the error toast appears, and the final check "button returns to enabled" fails because the portal clears the selected file after a failed upload, leaving the button disabled. Whether that is an app bug or a wrong expectation is a human call.

Lessons that shaped the rules: running the model on this case four times cost about 600k tokens and never finished. The cause was that the router rendered one case at a time, so the renderer lost the default values it learns from sibling cases, and a deterministic step was pushed to the model. The renderer now takes every stored case and renders one (`--only`).

## Profiles and per-app knowledge

- `profiles/<app>.json`: base URL, Jira query, model, turn budget, crawl entry paths, safe-clicks, login file, synonyms, notification settings. A new web app is a new profile file, not new code.
- `knowledge/<App>.md`: what is peculiar about the app (UI language, custom widgets, which actions are real writes, unstable ids). Read by the model tier on every run; people add a line whenever a failure teaches something. Facts only: how to find things belongs in the app map, what should happen belongs in the case.
- Model requirements: LM Studio context of at least **24000** tokens (the agent refuses to start below that). The model is only used at T2/T3.

## Stage 0: Design (requirements → cases)

1. A story with acceptance criteria goes in (Jira story, or a document a human wrote).
2. **Code** enumerates the data with standard techniques (valid, invalid, empty, boundaries).
3. **The model** only phrases steps and expected results in the matrix format.
4. Cases are created in Jira as **Draft**.
5. **A human approves** (Draft → Ready for automation). That transition is the trigger for Intake.

Source of truth is requirements, not code. If no requirements exist for a feature, write them first.

## App map (the crawl)

`hydra crawl` (read-only) visits every reachable page of an app and records, per element: role, accessible name, id, `data-testid`, the best locator by framework priority (`data-testid`, then a stable id, then role + name, CSS last), whether that locator is stable and **unique**, and the options of comboboxes/tabs (`--safe-clicks`: opens them to read, never selects or submits). Non-GET requests are aborted; links that look like logout/delete are never followed.

- Output: `appmap/<App>.crawl.json`. Watch it live with `hydra crawl --watch :8099` (add `--headed` to see the real browser, and `--slow 150` to reveal elements one at a time: boxes and locator labels colored green = stable id/testid, blue = role + name, orange = weak or not unique).
- Plan and Build **look up the map first**; the model is only asked for elements the map cannot resolve.
- Flags worth acting on: non-unique locators (seven carousel dots share one `data-testid`), generated ids (`radix-:R19...`), and "weak" CSS fallbacks.
- Limit: a read-only crawl sees only what is on screen after load. Results tables, dialogs and error messages appear after an action; the agent adds those when a case first exercises them, and they are remembered.
- The map says **how to find** things, never **what should happen**.
- Portal QA: the form has no `data-testid` (only the carousel does); it does have stable ids (`#orderNumber`, `#name`, `#file-upload`). Adding `data-testid` remains a useful first task for the build loop.

## The two loops

- **Loop A (QA):** approved Jira case → Ploy intake → work order → Labs plan/build → bundle → Ploy verify → deliver.
- **Loop B (build):** feature ticket → Labs coding agent changes the app on a branch → Loop A verifies it (Ploy, independent of the agent). Only after A is stable.

**Portal QA** (`~/Documents/GitHub/Oms_Automation`, runs locally via `./dev.sh`; VPN required for the backends) is the test dummy for every stage. Its automation lives in `~/Documents/GitHub/portal-qa-automation`, created from the framework template.

## Model roles (all bounded, all validated)

| Where | Job | Validated by | On failure |
|---|---|---|---|
| Stage 0 | phrase steps/expected for enumerated data | casecheck + human approval | stays Draft |
| Plan | choose a framework function from a closed list | catalog (name must exist, params must match) | "no match" → gap |
| Build | map data fields to step text (only leftovers) | skeleton warnings, lint | review folder |
| Explore | pick an element from an accessibility snapshot | the action must succeed and the assertion hold | review |
| Onboarding | propose a profile from one sample | human approval, then frozen | rejected |

## Measuring it

- **Pass rate** of generated tests (necessary, not sufficient).
- **Mutation score**: seed known bugs on a branch of the portal (wrong total, broken validation) and measure how many the generated tests catch. This is the honest score for "autonomous QA". Green tests alone prove nothing.
- Regression check: regenerate the hand-written facturación cases (FACT_0001-0004) and diff against the originals.

## Slices (nothing new starts until the previous one works)

1. **Portal, Jira → passing test, locally.** Built: gate, intake, crawl, bridge, renderer, verify, router, n8n manual triggers. Proven on the real portal: DWQ-131 passes at 0 tokens; DWQ-130 reaches `needs-writes`. Not yet: live Jira polling, the model tier on a real case, a reviewed branch/PR in `portal-qa-automation`, and a passing mutating case (needs a test environment decision).
2. **Regeneration check**: facturación FACT_0001-0004 regenerate and match the hand-written tests.
3. **Auto-learn from failures**: failures that teach something become a line in `knowledge/<App>.md`.
4. **Jira write-back** (comment + status) and the Google Chat space, enabled only after 1-3 are trusted (`jiraComments`, `HYDRA_CHAT_WEBHOOK`).
5. ~~Stability gate~~ **Done**: delivery runs the suite `stabilityRuns` times (default 3) in the real framework; a case is delivered only if it passes every run. A mix of pass and fail is reported as `flaky` and never reaches the repo.
6. **Stage 0 design** from requirements; then **Loop B** and the mutation score.

## Parked (not deleted; the retired ones live in `/deprecated`, see its README)

The Tauri desktop app (a possible future HydraPloy/HydraLabs dashboard), `uiagent` (vision planner), Gherkin output (`cmd/skeleton`, `cmd/lint`: valid, tested, optional spec only). Revisit when the slices above are done.

## Open decisions

- Decided: one repo, `ploy/` and `labs/` side by side. Open: the physical move and the final subfolder names.
- Jira: Cloud (`liverpooldigital.atlassian.net`). Intent is signaled by the labels `automate` / `execute`; the human gate is the case being written and ready. Still to confirm against a live poll.
- Test environment per app: where writes are allowed, how data is reset or seeded, and whether a login is needed. Today a person sets `baseUrl`/`allowWrites` in the profile (the onboarding gate).
- Delivery: each app's own `Hydra-<App>` repo, one review branch per delivery (no auto-merge). Open: a GitHub remote for PRs; for now everything stays local.
- Cases 05/11/12: which RFC belongs to which boleta (data question, not a pipeline one).

## Known data issues (facturación sample)

Duplicate case id `FACT_0037.3` in the matrix; blank terminals for boletas 48/72/71 in case 3; `notas` column unplaced in case 7; 37 cases have no data rows yet. The pipeline reports these and never guesses.
