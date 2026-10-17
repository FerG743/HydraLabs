# hydra/

Everything HydraPloy produces for this application. Do not rename the folders: the pipeline reads them.

| Path | What | Who writes it |
|---|---|---|
| `crawl/` | `crawl.json` and `CRAWL.md`: every page and element found by the last stored crawl, with the weak locators listed | the pipeline, on a review branch |
| `locators/learned.json` | locators the pipeline matched to the real page, so they are not asked of the model twice | the pipeline, with a delivery |
| `locators/confirmed.json` | locators a person verified; they always win over the learned ones | a person |
| `knowledge.md` | what is peculiar about this app (custom widgets, which actions really write, unstable ids); the model tier reads it | a person |
| `cases/` | each delivered test's case: the Jira text (`.case.json`) and the same steps as Gherkin (`.feature`) | the pipeline, with a delivery |
| `out/` | working files: run records, bundles, crawl screenshots and video, agent logs, state. **Git-ignored.** | the pipeline |

Durable files only change on a review branch (`hydra/...`); a person merges them.
