# Hyperframes Composition Brief: HydraLabs

## Objective
Create a short launch-style brag video for HydraLabs, a QA automation workshop where a Jira issue becomes an automated test that a human approves.

## Output
- Composition directory: `composition/`
- Rendered video: `brag.mp4`
- Format: landscape — 1920x1080, 30fps
- Duration: 20.5 seconds

## Source Material
- Project root: `HydraLabs/HydraLabs/` (desktop app) and `HydraLabs/testengine/` (pipeline)
- Primary files read: `HydraLabs/src/App.tsx`, `HydraLabs/src/app.css`, `testengine/PIPELINE.md`
- Product name: HydraLabs (workshop), with HydraPloy (pipeline) in the same repo
- Tagline / strongest claim: "Labs never grades its own work."
- Key UI or visual moment to recreate: the request bar (method select, URL field, plum Send button), the `200 OK` status pill, and the activity log line.
- Copy that must appear verbatim:
  - "Labs never grades its own work."
  - "Human gates." / "No auto-merge."
  - "HydraPloy + HydraLabs. One repo."
  - "HydraLabs is the workshop: agents build here."
  - Stage names: intake, gate, build, verify, deliver
  - Request bar: `GET`, `https://jsonplaceholder.typicode.com/users`, `Send`, `Headers`, `Body`, `Auth`
  - Response pill: `200 OK`
  - Activity line: `GET https://jsonplaceholder.typicode.com/users - 200 OK`
- The Jira ticket card is a labelled sample ("QA-142 · Login rejects a valid password"), not a real ticket.

## Creative Direction
- Tone preset: polished
- Creative direction: a dry QA lab. Precise, unhurried, a little smug that a test has to pass in someone else's framework.
- Interpretation: fewer, longer holds; restrained motion; one sound per payoff.
- Angle: HydraLabs builds the tests; a different system grades them; a person approves every step.
- Hook: "Labs never grades its own work." types in over the first 1.2s.
- Outro / punchline: "No auto-merge." then the HydraLabs wordmark.
- Avoid:
  - Generic SaaS language
  - Abstract filler visuals
  - Unrelated visual redesign
  - Claims the project does not make (no speed or count claims)

## Visual Identity
- Background: `#0F1419` (`--background`)
- Text: `#D7CDCC` (`--foreground`)
- Accent: `#9C528B` plum (`--primary`); for small text use a lighter plum tint for contrast
- Card surface: `#1F252B` (`--card`)
- Muted text: hsl(210 10% 60%) (`--muted-foreground`); check contrast on the card
- Display font: system UI stack (`-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif`); the project declares no web font
- Body font: same system stack; monospace (`ui-monospace, Menlo, Consolas, monospace`) for URLs
- Visual references from the project: the dark slate panels, the plum accent, the request bar and status pill layout from `App.tsx`

## Storyboard
Use the storyboard in `brag-plan.md` as the creative contract.

Scene summary:
1. Hook — 0.0–3.0s — "Labs never grades its own work." types in
2. Pipeline — 3.0–9.29s — sample ticket card, then five chips (intake, gate, build, verify, deliver) on beats 3.82, 4.91, 6.00, 7.09, 8.19
3. Workbench — 9.29–17.47s — window slides in on beat 9.29; URL types; Send clicked; `200 OK` lands on 13.11; caption holds
4. Outro — 17.47–20.5s — "No auto-merge." on 17.47; wordmark HydraLabs on 19.10; sub-line holds

## Audio
- Audio role: steady bed with sparse, motion-matched accents
- Audio arc: silence-to-bed under the hook; light taps through the pipeline; one click and one soft hit in the workbench; dry hit on the outro line; bell on the wordmark
- Music: `happy-beats-business-moves-vol-12-by-ende-dot-app.mp3` (copied to `composition/assets/music/`)
- Music treatment: fade in 0.6s, volume 0.35, fade out 0.8s at the end
- Music cue guidance: preset `assets/music/cues/happy-beats-business-moves-vol-12-by-ende-dot-app.music-cues.md` (bundled, tempo ~110 BPM). Strong cues in use: 9.29s, 17.47s, 18.56s (the bundled preset lists 18.56 as a strong cue; use 19.10 beat for the wordmark). Beat grid used for chip reveals.
- Audio-reactive treatment: none. The `media-use` skill with its extraction helper is not available in this environment, so skip audio-reactive and keep the glow static. Documented here, does not block the render.
- Audio-coupled moments:
  - Pipeline chips — tap on each chip (interface/click_002 on 3.82 and 6.00; interface/drop_001 on 4.91, 7.09, 8.19)
  - Window slide — casino/card-slide-1 at 9.29
  - Send click — interface/click_001 at 12.0
  - Response landing — impact/impactSoft_medium_001 at 13.11
  - Outro line — impact/impactSoft_medium_002 at 17.47
  - Logo — impact/impactBell_heavy_000 at 19.10
- SFX selection guidance: keep to the seven files above (copied to `composition/assets/sfx/`). Low HF-risk picks per `sfx-analysis.md`, except card-slide-1 (medium).
- Exact SFX choice: as listed above.
- Audio files: copied into `composition/assets/`

## Hyperframes Instructions
- Root: `<div id="root" data-composition-id="root" data-width="1920" data-height="1080">` with a paused GSAP timeline registered on `window.__timelines["root"]`.
- Scene elements carry `class="clip"` and `data-start`/`data-duration`.
- Audio elements use relative `assets/...` paths, `data-track-index` ascending from music (10) to SFX (11+), and `data-volume` per the brag volume policy (music 0.35; SFX 0.55–0.85).
- Keep all text readable: short labels hold at least 0.8s settled; full sentences at least 0.3s per word.
- Run `npx hyperframes check` before render (single gate). Fix all errors, including WCAG contrast failures.
- Keep creation and rendering local.
