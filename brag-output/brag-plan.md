# Brag Plan: HydraLabs

## What is this app?
HydraLabs is a QA pipeline: a human-written Jira test case goes in, and a passing, human-reviewed automated test comes out. Labs builds the tests, and Ploy decides whether they passed.

## The angle
The whole product rests on one rule: the thing that writes the test never grades it. The video plays that rule completely straight. Deadpan, calm, no hype. The joke is that a QA tool is this strict about its own homework.

## Hook (first 2-3 seconds)
Black screen. White line, slow fade-in: "Whoever writes the test does not grade it." It holds long enough to read twice.

## Key moments (the middle)
- The stage strip, built from the real pipeline: Intake, Gate, Plan, Build, Verify, Deliver, each tagged Ploy or Labs.
- A case card showing the status `needs-clarification` with "0 tokens. Nothing else runs."
- The three cost tiers, cheapest first: free, render, model.
- The locator legend, in the watch page's own colors: green = stable id / test id, blue = role + name, orange = guess, not confirmed.
- Three stability ticks, one per run, and the line "Pass every run, or nothing is delivered."

## Outro / punchline
Logo lands on a beat. Wordmark "HydraLabs". Final line: "Automated means it passed."

## User flow worth showing
none, this is a pipeline, not a web app. The working product is the pipeline itself: a case moves through its stages, and each stage's real output (status, cost, locator confidence, stability runs) is the material the centerpiece scenes show.

## Tone
- Preset: deadpan
- Creative direction: a QA pipeline that will not grade its own homework. Calm, dry, and exact.
- Interpretation: long holds, slow crossfades, one word or one short line per beat, no exclamation marks, no motion for its own sake. Restraint is the joke.

## Format: landscape — 1920x1080
## Duration: 20.5 seconds

## Visual identity (from the project)
- Background: #ffffff (light), #141414 (dark, unused in the render)
- Accent: #2563eb (role + name, the "on" color)
- Text: #1a1a1a
- Status colors used in the render: #16a34a (stable, ok), #dc2626 (fail), #ea580c (guess, dots only, not text)
- Card surface: #f6f6f4, line: #e3e3df
- Display font: system-ui stack (`system-ui, -apple-system, sans-serif`), the same stack the watch page uses
- Body font: system-ui stack
- Strongest visual element: the locator legend and the stage strip, both lifted from the live watch page and the pipeline table

## Share copy (draft)
We built a QA pipeline that will not grade its own homework. Labs writes the test, Ploy decides if it passed.

## Audio direction
- Role: warm bed, sparse accents (deadpan, so very few cues)
- Music: happy-beats-business-moves-vol-12 (steady and clean, the deadpan choice from the audio reference). Volume 0.15.
- Music treatment: starts at 0.0s, fades out over the final 0.8s under the logo. No ducking needed (no voiceover).
- Music cue guidance: preset read from `<skill-dir>/assets/music/cues/happy-beats-business-moves-vol-12-by-ende-dot-app.music-cues.md`. Strong lock targets: 10.93s (legend reveal), 18.56s (logo landing). Beat-grid windows: stage chips on 3.82, 4.39, 4.91, 5.34, 6.00, 6.56s; stability ticks on 14.73, 15.29, 15.84s.
- Audio-reactive treatment: none. Deadpan asks for stillness; skip the audio-reactive layer.
- SFX posture: very sparse. One `impact/impactSoft_medium_000` on the hook line, one `interface/click_001` per stage chip (chips only, not every beat), one `casino/card-place-1` on the gate card, one `impact/impactBell_heavy_000` on the logo landing.
- Audio-coupled moments: stage chips land on beats; stability ticks land on consecutive beats; logo on the strong cue.
- Restraint rule: no SFX on text fades, no SFX on the hook beyond one hit, nothing on the legend.

## Storyboard

### Scene 1 — Hook — 3.5s (0.00–3.50s)
Black, then the line "Whoever writes the test does not grade it." fades in over 0.6s and holds to 3.5s. No other elements.
Sequential/interaction: none
Audio intent: a dry, quiet opening
Audio-coupled idea: one `impactSoft_medium_000` at 0.2s, on the fade-in
Music: vol-12 at 0.15 from 0.0s
Transition mood: soft → Scene 2

### Scene 2 — Stage strip — 4.0s (3.50–7.50s)
Six chips in a row, labeled Intake, Gate, Plan, Build, Verify, Deliver. Each chip shows its owner underneath: Ploy, Ploy, Labs, Labs, Ploy, Ploy. Chips appear one at a time on the beat grid (3.82, 4.39, 4.91, 5.34, 6.00, 6.56s). Once all six are visible, a line under the strip reads "Labs builds. Ploy decides." It holds until 7.50s. The chip labels hold ~0.8s settled each; the full set stays on screen after the last chip.
Sequential/interaction: yes — six chips appear one by one on the beat grid, then the caption
Audio intent: small, precise accents
Audio-coupled idea: one `interface/click_001` per chip at its beat time (chips only)
Music: vol-12 continues
Transition mood: soft → Scene 3

### Scene 3 — Gate card — 3.43s (7.50–10.93s)
A card in the watch page's style (white, #f6f6f4 surface, rounded), with the case label "case" and a status pill reading `needs-clarification` in the watch page's grey. Under it, one line: "0 tokens. Nothing else runs." Then three stacked rows fade in, cheapest first: "free — 0 tokens", "render — seconds", "model — minutes, tens of thousands of tokens". Each row holds ~0.8s settled. The card appears at 7.5s, rows follow on the beat grid after 8.2s, and the scene ends at 10.93s.
Sequential/interaction: yes — three tier rows appear one by one, cheapest first
Audio intent: a clean card landing
Audio-coupled idea: one `casino/card-place-1` as the card appears at 7.5s
Music: vol-12 continues
Transition mood: soft → Scene 4

### Scene 4 — Locator legend — 3.57s (10.93–14.50s)
The legend from the watch page: three dots with labels. Green dot: "stable id / test id". Blue dot: "role + name". Orange dot: "guess, not confirmed". Dots are the colors from the page; the label text stays #1a1a1a. The legend lands on the strong cue at 10.93s. Under it, one line: "Orange means guess. We show the guess." Holds to 14.50s.
Sequential/interaction: yes — the three legend rows appear top to bottom, 0.3s apart
Audio intent: a dry, factual beat
Audio-coupled idea: none (the strong cue lands the legend; no SFX)
Music: vol-12 continues, strong cue lock 10.93s
Transition mood: soft → Scene 5

### Scene 5 — Stability runs — 4.06s (14.50–18.56s)
Three run tokens side by side, labeled "run 1", "run 2", "run 3". Each gets a green tick on consecutive beats (14.73, 15.29, 15.84s). Under them, the line "Pass every run, or nothing is delivered." Holds until 18.56s. A small grey note underneath, "flaky: passed some, failed some. Never delivered." appears at 17.0s.
Sequential/interaction: yes — three ticks land on consecutive beats
Audio intent: a steady, counted rhythm
Audio-coupled idea: the ticks sit on the beat grid, no separate SFX
Music: vol-12 continues
Transition mood: soft → Scene 6

### Scene 6 — Outro — 1.94s (18.56–20.50s)
The wordmark "HydraLabs" lands on the strong cue at 18.56s, in #1a1a1a, with a short blue (#2563eb) rule under it. Under the wordmark: "Automated means it passed." Holds to 20.50s, then the music fades out over the final 0.8s.
Sequential/interaction: none
Audio intent: a single cinematic hit, then quiet
Audio-coupled idea: one `impact/impactBell_heavy_000` on the wordmark landing at 18.56s
Music: vol-12 lands on the strong cue, fades out 19.7–20.5s
Transition mood: hard → end

**Music mood for this video:** deadpan
**Audio summary:** a quiet steady bed, a dry hit on the hook, small clicks on the stage chips, one card sound, and a bell on the logo. The bed fades under the last line.

## Duration check
3.5 + 4.0 + 3.43 + 3.57 + 4.06 + 1.94 = 20.5 seconds. Within 15-25.

## Render notes (what changed during build)
- Final length is 21.0s, not 20.5s. The outro was lengthened so the wordmark and closing line each hold at least 2s, per the composition check.
- Tiers, flaky note and closing line moved earlier to meet the reading floor.
