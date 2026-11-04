# Brag Plan: HydraLabs

## What is this app?
HydraLabs is a QA automation workshop: a Jira issue goes in, and an automated test that passes in the framework comes out, with a human approving each step.

## The angle
HydraLabs is a QA product that refuses to trust itself. The builders never grade their own work. The video is a dry, precise walkthrough of that rule, shown through the pipeline and the workbench app.

## Hook (first 2-3 seconds)
The line "Labs never grades its own work." types in on a near-black frame. It is the project's own rule from `testengine/PIPELINE.md`, and it earns the next 17 seconds.

## Key moments (the middle)
- A Jira ticket card (sample ticket) becomes a row of stage chips in order: intake, gate, build, verify, deliver.
- The HydraLabs workbench: the request bar reads `GET https://jsonplaceholder.typicode.com/users`, the cursor hits Send, and the response pill lands on `200 OK`. The activity log gets its line.
- The bottom line of the rule: nothing merges without a person.

## Outro / punchline
"No auto-merge." then the wordmark HydraLabs with the sub-line "HydraPloy + HydraLabs. One repo."

## User flow worth showing
Entry: a request URL typed into the request bar. Key action: Send. Result: status pill `200 OK` and the activity log line `GET ... - 200 OK`. The sidebar's Test Suites tab shows the suite list (Login Tests, API Integration, Performance Tests) as the next beat if there is room.

## Tone
- Preset: polished
- Creative direction: a dry QA lab. Precise, unhurried, and a little smug that the test has to pass in someone else's framework.
- Interpretation: fewer, longer holds; restrained motion; a single dry sound per payoff. No jokes that the project doesn't earn.

## Format: landscape — 1920x1080
## Duration: 20.5 seconds

## Visual identity (from the project)
- Background: `#0F1419` (`--background: 210 15% 8%`)
- Accent: `#9C528B` plum (`--primary: 314 31% 47%`)
- Text: `#D7CDCC` (`--foreground: 15 10% 85%`)
- Card surface: `#1F252B` (`--card: 210 12% 15%`)
- Muted text: hsl(210 10% 60%) (`--muted-foreground`)
- Display font: system UI stack (the project declares no web font)
- Body font: system UI stack; monospace for URLs and code
- Strongest visual element: the request bar (method select, URL field, plum Send button) and the `200 OK` status pill.

## Share copy (draft)
Labs never grades its own work. HydraLabs makes the QA tests, HydraPloy checks them, and a human approves every PR.

## Audio direction
- Role: sparse professional accents over a steady, clean bed
- Music: `happy-beats-business-moves-vol-12-by-ende-dot-app.mp3` (steady and clean, best for `polished`)
- Music treatment: fades in over the first 0.6s, volume 0.35, fades out under the final logo.
- Music cue guidance: preset read from `assets/music/cues/happy-beats-business-moves-vol-12-by-ende-dot-app.music-cues.md`. Beat grid runs 0.56s, 1.09s, ... Strong cues used: 9.29s (window reveal), 17.47s (outro line), 18.56s (logo landing). Chip reveals snap to every other beat from 3.82s.
- Audio-reactive treatment: none. Extraction helper ships with the `media-use` skill, which is not installed here; the glow stays static.
- SFX posture: sparse, motion-matched. Roughly seven cues across the video.
- Audio-coupled moments: chip reveals, window slide-in, Send click, response landing, outro line, logo.
- Restraint rule: no SFX under the spoken or typed hook; no more than one SFX per 0.8s.

## Storyboard

### Scene 1 — Hook — 0.0 to 3.0s (3.0s)
Near-black `#0F1419` frame. The line "Labs never grades its own work." types in at 0.0–1.2s, then holds until 3.0s. Plum underline draws under "never" at 1.2s.
Sequential/interaction: none
Audio intent: quiet, precise. Music fades in.
Audio-coupled idea: none (no SFX under the hook)
Music: steady bed fading in from silence
Transition mood: clean → Scene 2

### Scene 2 — Pipeline — 3.0 to 9.29s (6.29s)
- 3.0s: a sample Jira ticket card ("QA-142 · Login rejects a valid password", labelled sample) fades up on `#1F252B`.
- 3.82s (beat): chip "intake" appears under the card. 4.91s: "gate" (with a small plum "human" tag). 6.00s: "build" (label "HydraLabs"). 7.09s: "verify" (label "HydraPloy"). 8.19s: "deliver" (label "PR for review").
- Each chip is text the viewer reads, so each holds at least 0.8s settled. Chips snap to every other beat, and the full row holds from 8.19s until 9.29s.
Sequential/interaction: yes — five chips appear one by one, in pipeline order.
Audio intent: a few clean taps, one per chip, landing with the visual.
Audio-coupled idea: beat-grid chip reveals; ticket card drop
Transition mood: clean wipe → Scene 3

### Scene 3 — Workbench — 9.29 to 17.47s (8.18s)
- 9.29s (strong cue): the HydraLabs window slides into frame on `#0F1419`: sidebar with Test Suites, request bar, response area, activity panel. Real labels: "Collections", "Test Suites", "Enter request URL", "Headers", "Body", "Auth", "Send".
- 10.2s: URL text types into the bar: `https://jsonplaceholder.typicode.com/users` (GET selected).
- 12.0s: cursor moves to Send and clicks; the button presses.
- 12.4–13.1s: "Sending request..." state.
- 13.11s (strong cue): response pill "200 OK" lands in the project's success tint (in this theme `--success` is the plum accent), and the activity log line "GET https://jsonplaceholder.typicode.com/users - 200 OK" appears.
- 14.5–17.0s: hold on the full frame; "HydraLabs is the workshop: agents build here." caption appears small under the window at 14.5s and holds to 17.0s.
Sequential/interaction: yes — the URL types character by character, Send is clicked, the response arrives.
Audio intent: the window slide has a card-slide cue; the click is the Send tap; the response has a soft impact.
Audio-coupled idea: typed URL (no keyboard SFX to keep it clean), Send click, response impact.
Transition mood: clean → Scene 4

### Scene 4 — Outro — 17.47 to 20.5s (3.03s)
- 17.47s (strong cue): "No auto-merge." appears large on `#0F1419`, with a plum rule under it. Holds settled until 18.9s, then fades out.
- 19.10s (beat): wordmark "HydraLabs" lands centered; sub-line "HydraPloy + HydraLabs. One repo." appears at 19.6s and holds to 20.5s.
Both lines hold at least 0.8s settled.
Sequential/interaction: none
Audio intent: a dry single hit on the line, a bell on the logo.
Audio-coupled idea: outro reveal
Transition mood: fade to black

**Music mood for this video:** steady, clean, corporate-adjacent. Restrained.
**Audio summary:** silence under the hook, a soft bed with light taps through the pipeline, one Send click and one response hit in the workbench, a dry hit on the outro line, and a bell on the wordmark.
