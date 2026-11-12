# Layout proposal: the app's repo and Hydra's output

Status: proposal, for review. Nothing here has been moved yet.

The layout was shaped for a viewer that no longer runs. Hydra is a CI/CD pipeline, so the layout only needs to suit
what CI runs and what a person opens. This document says what each folder is, what should change, and in what order.

## Today

The app's repo (`Hydra-PortalQA`) has two kinds of folder at the top level:

| Folder or file | What it is | Who uses it |
|---|---|---|
| `apps/`, `tests/`, `utils/`, `config/`, `conftest.py`, `pytest.ini`, `requirements.txt` | the generated test project | `pytest tests/PortalQA` (the README's command) |
| `cases/` | the human-written test cases, as markdown (`DWQ-L1.md`) | Hydra reads them as input (`localCaseDir`) |
| `reports/` | pytest HTML reports, and `reports/downloads` for files the tests download | the generated tests write here (`DOWNLOADS_DIR` in `agent/render.py` and `tools/jira/render_pytest.py`) |
| `hydra/` | Hydra's own working folder | Hydra only |
| `README.md` | how to run the tests | people |

Inside `hydra/`, `out/` holds everything the pipeline produces in one flat level of about 25 entries. Some are folders,
some are single files, and the names are not grouped by what they are.

## Proposed

### The top level: what CI and people see

```
Hydra-PortalQA/
  README.md
  apps/  tests/  utils/  config/        the generated test project
  conftest.py  pytest.ini  requirements.txt
  cases/                                the human-written cases (input)
  reports/                              what a person opens: pytest HTML, downloads
  hydra/                                everything Hydra writes
```

- **Unchanged:** the generated project, so `pytest tests/PortalQA` keeps working without any change.
- **Kept at the top:** `reports/`. It is the one output a person opens after a run, so it stays visible.
- **Kept at the top:** `cases/`. It is the input a person writes, so it stays where people look. It is not a duplicate of
  `hydra/cases/`, which holds the JSON that Hydra builds from Jira.

### Inside `hydra/`: Hydra's own folders

```
hydra/
  README.md                 what each folder is, in one page
  cases/                    the JSON cases built from Jira (today: hydra/cases)
  playlists/                saved flows (today: hydra/playlists)
  out/
    map/                    the app map and what is derived from it
                            crawl.json, learned.json, coverage.json, routes.json
    pipeline/               the stages of a run
                            cases/, stage/, bundles/, verify/, deliver/, deliveries/, design/, runs/
    logs/                   what happened, append-only or per run
                            agent-runs/, executions.jsonl, live/events.jsonl, crawl/events.jsonl
    media/                  screenshots and videos, kept apart so they can be cleared
                            crawl/shots, crawl/videos, live/shots, verify/videos
    state.json              the current state
    posted.json             what was posted and where
```

Why this grouping: each folder has one owner and one purpose. `map/` is written by the crawl and the derived builders,
`pipeline/` by the run stages, `logs/` by everything that records what happened, and `media/` can be deleted without
losing any data.

## What needs code changes

Every path is written in code, so moving a folder means changing the code that writes it, the code that reads it, and any
test that names it. The paths are spread across Go (`cmd/hydra/*.go`) and Python (`tools/`, `agent/`).

- `out/` paths are built in `router.go` (`caseDir`, `bundleDir`) and in the other `Config` helpers. Changing them in one
  table makes the move a single change instead of about 60 edits.
- `reports/downloads` is written into the generated tests by `agent/render.py` and `tools/jira/render_pytest.py`. It
  only moves if the generated project's README and templates move with it. This proposal keeps it where it is.
- `hydra/cases` and `hydra/playlists` are already under `hydra/`. They only change if the table above is applied.
- `cases/` at the top is read by `localCaseDir` and by the init seed. It stays unless we decide otherwise.

## Order of work

1. **A path table.** One Go table and one Python module that name every `out/` path. Nothing moves. The existing tests
   must still pass. This is the step that makes the rest safe.
2. **Move `out/` into the groups above.** Only the table changes. Old runs keep working if `state.json` is rewritten to
   the new paths in the same step.
3. **Remove `.DS_Store` and ignore `out/` in git.** Nothing reads these files.
4. **Write `hydra/README.md` and point the app's README at `reports/`.** Documentation only.

## Risks

- **Old runs:** a run that started before the move keeps its paths in `state.json`. The move must rewrite those paths or
  the run can't resume.
- **Delivered copies:** `deliver.go` copies files into the repo. Anything copied before the move keeps the old layout.
- **Tests that name paths:** some tests compare paths as strings. They are found by searching for each folder name before
  the move.
- **Other readers:** nothing outside this repo reads these folders today, since the old viewer has stopped. If that
  changes, the table in step 1 is the place to update.

## Decisions for review

1. Keep `reports/` at the top, or move it into `hydra/out/media/`?
2. Keep `cases/` at the top, or move it into `hydra/` as `hydra/local-cases/`?
3. Group `out/` as proposed, or keep it flat and only add a `README.md`?
