# Hyperframes Composition Brief: HydraLabs

## Objective
Create a short launch-style brag video for HydraLabs, deadpan tone.

## Output
- Composition directory: `brag-output/composition/`
- Rendered video: `brag-output/brag.mp4`
- Format: landscape, 1920x1080
- Duration: 21.0 seconds

## Source Material
- Project root: the HydraLabs repo (QA pipeline; no web app)
- Primary files read: `HydraLabs/testengine/PIPELINE.md`, `HydraLabs/testengine/cmd/hydra/live.go` (watch-page colors and legend), `HydraLabs/testengine/cmd/hydra/router.go` (status names)
- Product name: HydraLabs
- Tagline / strongest claim: "Labs never grades its own work."
- Key UI or visual moment to recreate: the stage strip (Intake, Gate, Plan, Build, Verify, Deliver, tagged Ploy or Labs), the gate card with `needs-clarification`, the locator legend in the watch page's colors
- Copy that must appear verbatim (all grounded in the pipeline docs):
  - "Whoever writes the test does not grade it."
  - "Labs builds. Ploy decides."
  - "needs-clarification", "0 tokens. Nothing else runs."
  - "free / render / model", "0 tokens / seconds / minutes, tens of thousands of tokens"
  - "stable id / test id", "role + name", "guess, not confirmed"
  - "Pass every run, or nothing is delivered."
  - "flaky: passed some, failed some. Never delivered."
  - "HydraLabs", "Automated means it passed."

## Creative Direction
- Tone preset: deadpan
- Creative direction: a QA pipeline that will not grade its own homework. Calm, dry, exact.
- Interpretation: long holds, slow crossfades, one short line at a time, no exclamation marks, no motion for its own sake.
- Angle: the agent that writes a test is never the one that decides it passed. The video states that rule and lets its own strictness be the joke.
- Hook: "Whoever writes the test does not grade it." (0-3.5s)
- Outro / punchline: "Automated means it passed." under the wordmark (18.56-21.0s)
- Avoid:
  - Generic SaaS language
  - Abstract filler visuals
  - Internal hostnames, Jira keys, names, or user data (none appear)

## Visual Identity
- Background: #ffffff
- Text: #1a1a1a
- Accent: #2563eb (role + name, the rule under the wordmark)
- Status colors: #16a34a / #15803d (stable, passed run), #ea580c (guess, dot only)
- Card surface: #f6f6f4, line #e3e3df
- Display font: system-ui stack (same stack as the watch page)
- Body font: system-ui stack
- Visual references from the project: `live.go` legend, the stage table in `PIPELINE.md`

## Storyboard
Use the storyboard in `brag-output/brag-plan.md` as the creative contract.

Scene summary:
1. Hook — 3.5s — "Whoever writes the test does not grade it."
2. Stage strip — 4.0s — six stage chips on beats, "Labs builds. Ploy decides."
3. Gate card — 3.43s — `needs-clarification`, three cost tiers, cheapest first
4. Locator legend — 3.57s — the three legend rows, "Orange means guess. We show the guess."
5. Stability runs — 4.06s — three ticks on consecutive beats, "Pass every run, or nothing is delivered."
6. Outro — 2.44s — wordmark, "Automated means it passed."

## Audio
- Audio role: warm bed, sparse accents (deadpan)
- Audio arc: bed from 0s, a dry hit on the hook, small clicks on the stage chips, one card sound at the gate card, a bell on the wordmark, bed fades 20.2-21.0s
- Music: `happy-beats-business-moves-vol-12-by-ende-dot-app.mp3`, at 0.15 volume (absolute, via the fade tween)
- Music treatment: starts at 0s; fades out 20.2-21.0s
- Music cue guidance: source is the bundled preset `happy-beats-business-moves-vol-12-by-ende-dot-app.music-cues.md`. Strong locks: 10.93s (legend), 18.56s (wordmark). Beat-grid: chips on 3.82, 4.39, 4.91, 5.34, 6.00, 6.56s; ticks on 14.73, 15.29, 15.84s.
- Audio-reactive treatment: none (deadpan asks for stillness)
- Audio-coupled moments:
  - hook — soft impact at 0.2s
  - stage strip — click per chip (6)
  - gate card — card-place at 7.5s
  - stability runs — ticks on beats, no separate SFX
  - outro — bell at 18.56s
- SFX selection guidance: sparse, one per moment; low-HF-risk files from the approved list
- Exact SFX choice: `impact/impactSoft_medium_000.ogg`, `interface/click_001.ogg` (x6), `casino/card-place-1.ogg`, `impact/impactBell_heavy_000.ogg`
- Audio files: copied to `brag-output/composition/assets/` (music and the five SFX above)

## Hyperframes Instructions
Composition built from `hyperframes init --example blank`, one paused GSAP root timeline registered on `window.__timelines["main"]`. Timed elements use `data-start`, `data-duration` and `class="clip"`. The composition has no `Date.now()`, `Math.random()` or network fetches beyond the GSAP CDN script the scaffold provides.

Gate: `npx hyperframes check` returns 0 errors. Remaining warnings are advisory: nested scene structure and one contrast sample at 15.17s, which lands on a tick mid-transition (pending grey to green).
