# Playlists: ordered test flows with values passed between cases

Status: **built** (`tools/jira/playlist.py`, tests in `tools/jira/test_playlist.py`). Not yet wired into `hydra` (no `hydra run` command). Where the build differs from the first draft, the "Built" notes below say how.

## Problem

Some cases cannot run alone. An order must exist before it can be cancelled, a
customer ID created in one case is searched in the next. Today the pipeline
marks these `depends` (free text such as "use another case's inputs") and
blocks them, and the normalized case schema has `depends_on` with no way to
say which value crosses over.

A **playlist** is an ordered list of cases that run as one flow, where earlier
cases can hand values to later ones.

## Scope

In:
- An ordered list of case ids, run in one session, stopping on the first failure.
- Named values captured by one case and consumed by a later case in the same run.
- Static checks that reject broken flows before anything runs.

Out (for v1):
- Parallel execution, branches, loops, conditional steps.
- Values that survive past one run (no persistence between runs).
- Values passed from a case that did not run in this playlist.
- A GUI editor.

## Where it lives

- Playlist files: `<app repo>/hydra/playlists/<name>.json`, next to `cases/`, committed on a review branch like other durable app files.
- Cases are unchanged: the playlist refers to them by id.
- The runner is a run-and-report concern, so it belongs to **Ploy** (see [PIPELINE.md](PIPELINE.md), "Dividing rule"). Test code generation is untouched.

## Playlist format

```json
{
  "id": "order-lifecycle",
  "name": "Create order, then cancel it",
  "stopOnFailure": true,
  "steps": [
    { "case": "DWQ-101", "captures": { "order_id": "ORDER_NUMBER" } },
    { "case": "DWQ-102", "with": { "order_number": "$order_id" } },
    { "case": "DWQ-103", "with": { "order_number": "$order_id", "reason": "Cliente duplicado", "api_key": "ENV:QA_API_KEY" } }
  ]
}
```

- `case`: a case id that exists in the cases file.
- `captures`: `{ name: source }` or `{ name: { "source": ..., "log": true } }`. Values this step produces (see below).
- `with`: `{ column: value }`. Overrides data for this case, for this run only. `column` is a data column of that case's generated CSV. A value is a literal, `$name` (a capture from an earlier step), or `ENV:NAME` (read from the environment). Built: `$` is the capture marker, so a literal value that starts with `$` is not possible; use `ENV:` for secrets.
- `stopOnFailure`: default `true`.
- `stopOnFailure`: default `true`. A flow that continues after a failed step produces misleading downstream failures.

Keeping the grammar this small is deliberate. There are no expressions, no
string templating, no conditionals.

## How a value is captured

A capture reads a value that is already on screen during the run, so it names
where to read it, never a fixed value.

`source` is the name of a locator constant in the generated app's locators
class (for example `ORDER_NUMBER`). The runner reads that element's text, or
its value for an input, once the case has passed.

Built: the read happens at the **end of the producing case**, not right after
the step inside it. Generated tests are one function each, so a mid-test read
would need changes to the renderer. For a case whose result is its final state
(an order number on the confirmation screen), this is the same value.

If the element is missing, the case is reported **failed** with
`capture order_id: could not read #orderNumber`. A capture never silently
becomes an empty string. Decision still open: whether to also allow a data-row
field as a capture source. Not built; a data-row value is already known.

## Runtime

Built: the runner runs each case through the existing harness, one at a time,
in playlist order. It does not generate one combined pytest module.

- Each case gets its own browser, like any generated test. Values pass
  through the runner's memory, not through the browser. Cases share the app's
  server-side state, which is what flows like create-then-cancel rely on.
- A `with` value is written into that case's data row in the harness's
  working copy (`_override_row`). The generated test is unchanged, and the
  source folder is never modified.
- Captured values are kept in memory. The record shows them only for captures
  marked `"log": true`; the rest show `<redacted>`.
- Result: `playlist`, `status` (`passed` | `failed` | `invalid`), `errors`,
  `warnings`, and `steps` (status, error, captures per step). Steps after a
  stop are `skipped`. Written to stdout, and to `--record` if given. Not yet
  written into the `hydra` run record.

## Static checks (run before any browser opens)

Reject the playlist if:

1. A `case` id does not exist, or its status is not `ready`.
2. A `with` value names a capture that no earlier step produces (producer must come first).
3. A case's `depends_on` lists a case that is not earlier in the playlist.
4. The same capture name is produced twice.
5. A step with `stopOnFailure` disabled consumes a value from a failed step (checked at runtime, reported as skipped).

Warn (do not reject) when a capture is produced but never consumed.

Built in `playlist.py check` (Python, not a Go `cmd/`): the execution side uses
the Python harness, so the checks sit with it. Covered by `test_playlist.py`.

## Secrets

Captured values can be sensitive. Rules:
- Captured values are kept in memory only and are not written to logs, the
  report, or the run record unless the capture is marked `"log": true`.
- `with` literals that are secrets use `ENV:NAME` tokens, the same convention
  `utils/login.py` already uses. The runner never writes a literal credential
  into a playlist file.

## Open decisions

1. Capture source: locator text only (proposed), or also data-row fields.
2. Whether `stopOnFailure: false` is needed in v1 at all. Proposed: keep the field, default true, but do not document it until a real flow needs it.
3. Whether a playlist can include another playlist. Proposed: no.

## Build order (once this is approved)

1. Done: static checker and 14 unit tests (`python3 -m unittest test_playlist`).
2. Done: sequential runner with `with` overrides (fake-runner tests).
3. Done: capture support, checked against a real Chromium page (an input's value and a text node read correctly; a missing element fails with a clear message).
4. Not done: `hydra run <playlist>` in `cmd/hydra`, and the `playlist` block in the run record under `hydra/out/runs/`. Next step.

See also: [PIPELINE.md](PIPELINE.md) for the stage model, and
`schema/case.schema.json` for the case shape this builds on.
