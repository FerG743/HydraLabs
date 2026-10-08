#!/usr/bin/env python3
"""Generates hydra-pipeline.json: n8n as a thin, visible shell around the Go `hydra` binary.

  Manual: poll now        -> hydra poll          (every REGISTERED project, projects.json: new ones are onboarded first)
  Manual: run one issue   -> hydra poll --only KEY --force [--intent ...]   (edit the "Pick issue" node, click Execute)
  Manual: crawl the app   -> hydra crawl --project KEY  (read-only; add --watch :8099 yourself to show people)
  Weekdays 07:00          -> hydra poll          (the morning run)

Every manual trigger is a separate entry point, so while testing nothing waits on a timer. Each run becomes one row
(key, intent, tier, status, tokens, ms, detail) in the "Runs" node. All logic lives in the binary; n8n only calls it.
Jira/Chat secrets are NOT stored here: put JIRA_BASE_URL, JIRA_EMAIL, JIRA_TOKEN, HYDRA_CHAT_WEBHOOK in ~/.hydra.env
(start.sh loads it). Re-run this script after editing.
"""
import json, os

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
C = "$('Config').first().json"
nodes, conns = [], {}


def node(name, type_, params, pos, version=1):
    nodes.append({"parameters": params, "name": name, "type": type_, "typeVersion": version, "position": pos,
                  "id": name.lower().replace(" ", "-").replace(":", "")})


def link(a, b):
    conns.setdefault(a, {"main": [[]]})["main"][0].append({"node": b, "type": "main", "index": 0})


def sh(cmd):  # an Execute Command node; stderr is merged so errors show up in the table instead of vanishing
    return {"executeOnce": True, "command": "=" + cmd + " 2>&1"}


node("Manual: poll now", "n8n-nodes-base.manualTrigger", {}, [0, 100])
node("Manual: run one issue", "n8n-nodes-base.manualTrigger", {}, [0, 300])
node("Manual: crawl the app", "n8n-nodes-base.manualTrigger", {}, [0, 500])
node("Weekdays 07:00", "n8n-nodes-base.scheduleTrigger",
     {"rule": {"interval": [{"field": "cronExpression", "expression": "0 7 * * 1-5"}]}}, [0, 700], 1.2)

def config(name, y):  # one Config per branch so each command can name its own node; same three values
    node(name, "n8n-nodes-base.set", {"assignments": {"assignments": [
        {"id": k, "name": k, "value": v, "type": "string"} for k, v in {
            "hydra": os.path.join(ROOT, "bin", "hydra"),
            "project": "DWQ",  # only the crawl button uses it: which registered project to crawl
            "cwd": ROOT}.items()]}, "options": {}}, [240, y], 3.4)


def hydra(cfg, args):  # no --profile: projects.json says which Jira projects exist and where each app's profile and repo are
    return sh(f'cd "{{{{ $(\'{cfg}\').first().json.cwd }}}}" && "{{{{ $(\'{cfg}\').first().json.hydra }}}}" {args}')


# poll now + the morning schedule -> hydra poll
config("Config", 100)
node("hydra poll", "n8n-nodes-base.executeCommand", hydra("Config", "poll --json"), [480, 100])
link("Manual: poll now", "Config"); link("Weekdays 07:00", "Config"); link("Config", "hydra poll")

# one issue on demand: edit "Pick issue" (key, intent), then Execute
node("Pick issue", "n8n-nodes-base.set", {"assignments": {"assignments": [
    {"id": "key", "name": "key", "value": "DWQ-130", "type": "string"},
    {"id": "intent", "name": "intent", "value": "automate", "type": "string"}]}, "options": {}}, [240, 200], 3.4)
config("Config Run", 300)
node("hydra run one", "n8n-nodes-base.executeCommand", hydra(
    "Config Run", "poll --json --only \"{{ $('Pick issue').first().json.key }}\" --intent \"{{ $('Pick issue').first().json.intent }}\""), [480, 300])
link("Manual: run one issue", "Pick issue"); link("Pick issue", "Config Run"); link("Config Run", "hydra run one")

# crawl the app (read-only)
config("Config Crawl", 500)
node("hydra crawl", "n8n-nodes-base.executeCommand", hydra("Config Crawl", "crawl --project \"{{ $('Config Crawl').first().json.project }}\""), [480, 500])
link("Manual: crawl the app", "Config Crawl"); link("Config Crawl", "hydra crawl")

node("Runs", "n8n-nodes-base.code", {"mode": "runOnceForAllItems", "jsCode": """
// One row per run. Lines that are not JSON (errors, crawl progress) are kept as a 'log' row so nothing is hidden.
const out = [], text = ($input.first().json.stdout || '') + ($input.first().json.stderr || '');
for (const line of text.split('\\n').filter(Boolean)) {
  try { const r = JSON.parse(line); out.push({ json: { key: r.key, intent: r.intent, tier: r.tier, status: r.status,
    tokens: r.tokens, ms: r.ms, detail: (r.detail || []).join(' | ') } }); }
  catch (e) { out.push({ json: { log: line } }); }
}
return out.length ? out : [{ json: { log: 'No runs: nothing new, or no issue carries the automate/execute label (poll), or the command printed nothing.' } }];
"""}, [760, 240], 2)
for n in ("hydra poll", "hydra run one", "hydra crawl"):
    link(n, "Runs")

out = os.path.join(HERE, "hydra-pipeline.json")
json.dump({"id": "hydra-pipeline", "name": "HydraPloy: Jira -> automated tests", "nodes": nodes, "connections": conns, "active": False,
           "settings": {"executionOrder": "v1"}}, open(out, "w"), indent=2)
print("wrote", out, "-", len(nodes), "nodes")
