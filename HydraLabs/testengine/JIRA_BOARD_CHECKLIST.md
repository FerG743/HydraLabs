# Jira board readiness checklist

Use this before registering a Jira board (one board = one application under test) with HydraLabs. Items marked **person** need a human decision; the pipeline will not guess them.

## 1. Board and access

- [ ] The board has a Jira project key (for example `DWQ`). **person**
- [ ] The Jira token used by the pipeline can read that project's issues. **person**
- [ ] If write-back is wanted, the token can comment on the board's issues. **person**
- [ ] The app's name is decided (it becomes the repo name `Hydra-<App>`). **person**

## 2. Registry entry

- [ ] The key is added to `HydraLabs/testengine/projects.json` with `app` and `profile`. Registration is explicit on purpose.
- [ ] If the cases live in a different Jira project than the registry key, set `jiraProject` in the profile (for example PQA takes DWQ issues).

## 3. Issue labels

- [ ] Issues to generate a test from carry the label `automate`.
- [ ] Issues that already have a stored test carry the label `execute`. If both labels are present, `execute` wins.

## 4. Case content

Each case the pipeline should act on needs:

- [ ] Numbered steps, each with an action.
- [ ] An expected result for each step.
- [ ] A precondition, if the case depends on one.
- [ ] No placeholders (`TBD`, `pendiente`).
- [ ] No unresolved references such as "Same as TC-1".

Cases that fail this gate go back to the author with specific questions. They cost no tokens.

## 5. Target environment

- [ ] A QA URL for the app, set in the profile as `baseUrl`. Jira cannot know where the app runs, so the first run stops at `needs-onboarding` until this is set. **person**
- [ ] The environment is confirmed not to be production. **person**
- [ ] `allowWrites` is enabled only if the QA environment may be changed. Cases that write run once per stability pass, so the environment must tolerate repeats. **person**

## 6. Add-ons

- [ ] **Ploy** (generate and run tests from Jira): a JQL that selects the cases. **person**
- [ ] **Labs** (crawl the app and pick a model tier): crawl paths and a model tier. **person**
- [ ] Add-ons are enabled only for what the board needs. A profile without `addons` keeps the basic behaviour.

## 7. First run

- [ ] Run `hydra init` (interactively, or `--answers team.json --yes` in CI). It checks each answer, previews one crawled page, and writes nothing until confirmed. **person**
- [ ] Onboarding creates `Hydra-<App>` next to the other repos, with one baseline commit.
- [ ] Nothing is pushed or merged. Generated tests land on a local review branch, and a person merges them.
- [ ] Knowledge notes (`hydra/knowledge.md`) and confirmed locators are written by a person once the board is running.
