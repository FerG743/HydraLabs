#!/usr/bin/env python3
"""Generates jira-pipeline.json: poll Jira for test cases -> .feature files (no model involved).

  Schedule/Manual -> Config -> Search Jira (JQL) -> Split -> New Or Changed -> Get Issue
  -> Build Input -> Convert (tools/jira/jira2feature.py) -> Lint (bin/lint) -> Decide -> Save -> Remember
  -> Resolve References (tools/jira/resolve_refs.py: "Same as TC-1" across everything already saved)

Polling, not the Jira Trigger node: Jira Cloud cannot call a webhook on localhost.
Re-run after editing; `--test` writes a variant whose two Jira HTTP nodes return canned fixtures.
"""
import json, os, sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
TEST = "--test" in sys.argv

CONFIG = {
    "jiraBaseUrl": "https://liverpooldigital.atlassian.net",
    "jql": "parent = DWQ-129 ORDER BY key ASC",  # add "AND labels = automate" to make it opt-in
    "lang": "en",  # Gherkin language of the cases
    "pollMinutes": "5",
    "toolsDir": os.path.join(ROOT, "tools", "jira"),
    "binDir": os.path.join(ROOT, "bin"),
    "outDir": os.path.join(ROOT, "out", "jira"),
    "python": "python3",
    "version": "4",  # bump to reprocess every issue (e.g. after a parser fix); otherwise an issue runs once per Jira edit
}
C = "$('Config').first().json"
nodes, conns = [], {}


def node(name, type_, params, pos, version=1, **extra):
    nodes.append({"parameters": params, "name": name, "type": type_, "typeVersion": version, "position": pos,
                  "id": name.lower().replace(" ", "-").replace("?", ""), **extra})


def link(a, b, out=0):
    conns.setdefault(a, {"main": []})
    while len(conns[a]["main"]) <= out:
        conns[a]["main"].append([])
    conns[a]["main"][out].append({"node": b, "type": "main", "index": 0})


def code(js, each=True):
    return {"mode": "runOnceForEachItem" if each else "runOnceForAllItems", "jsCode": js}


# optional: JIRA_CRED_ID=<id from n8n> pre-selects the credential on the Jira HTTP nodes
CRED = ({"jiraSoftwareCloudApi": {"id": os.environ["JIRA_CRED_ID"], "name": "Jira SW Cloud account"}}
        if os.environ.get("JIRA_CRED_ID") else None)
JIRA_AUTH = {"authentication": "predefinedCredentialType", "nodeCredentialType": "jiraSoftwareCloudApi"}

node("Schedule", "n8n-nodes-base.scheduleTrigger", {"rule": {"interval": [{"field": "minutes", "minutesInterval": 5}]}}, [0, 120], 1.2)
node("Manual Trigger", "n8n-nodes-base.manualTrigger", {}, [0, -80])
node("Config", "n8n-nodes-base.set", {"assignments": {"assignments": [
    {"id": k, "name": k, "value": v, "type": "string"} for k, v in CONFIG.items()]}, "options": {}}, [220, 0], 3.4)

if not TEST:
    node("Search Jira", "n8n-nodes-base.httpRequest", {
        "method": "GET", "url": f"={{{{ {C}.jiraBaseUrl }}}}/rest/api/3/search/jql", **JIRA_AUTH,
        "sendQuery": True, "queryParameters": {"parameters": [
            {"name": "jql", "value": f"={{{{ {C}.jql }}}}"}, {"name": "fields", "value": "updated"},
            {"name": "maxResults", "value": "100"}]}}, [440, 0], 4.2, **({"credentials": CRED} if CRED else {}))
else:
    fixtures = {k: open(os.path.join(ROOT, "tools", "jira", "fixtures", k + ".html"), encoding="utf-8").read()
                for k in ("DWQ-130", "DWQ-131", "DWQ-132")}
    fixtures["DWQ-999"] = "<ul><li><p><strong>Preconditions:</strong> Only a precondition, no steps.</p></li></ul>"
    node("Search Jira", "n8n-nodes-base.code", code(
        "return { json: { issues: " + json.dumps([{"key": k, "fields": {"updated": "2026-10-05T09:00:00.000-0500"}} for k in fixtures]) + " } };"),
        [440, 0], 2)
    node("Get Issue", "n8n-nodes-base.code", code(
        "const html = " + json.dumps(fixtures) + ";\nconst k = $json.key;\n"
        "return { json: { key: k, fields: { summary: 'Fixture ' + k + ' \"quoted\"', priority: { name: 'Medium' }, parent: { key: 'DWQ-129' },"
        " updated: $json.fields.updated }, renderedFields: { description: html[k] } } };"), [1100, 0], 2)

node("Split Issues", "n8n-nodes-base.splitOut", {"fieldToSplitOut": "issues", "options": {}}, [660, 0], 1)

node("New Or Changed", "n8n-nodes-base.code", code("""
// remember (key -> updated) in workflow static data: process each issue once per edit
const seen = $getWorkflowStaticData('global').seen || {};
const v = $('Config').first().json.version;
return $input.all().filter(i => seen[i.json.key] !== i.json.fields.updated + '|' + v);
""", each=False), [880, 0], 2)

if not TEST:
    node("Get Issue", "n8n-nodes-base.httpRequest", {
        "method": "GET", "url": f"={{{{ {C}.jiraBaseUrl }}}}/rest/api/3/issue/{{{{ $json.key }}}}", **JIRA_AUTH,
        "sendQuery": True, "queryParameters": {"parameters": [
            {"name": "expand", "value": "renderedFields"},
            {"name": "fields", "value": "summary,description,priority,parent,updated"}]}}, [1100, 0], 4.2, **({"credentials": CRED} if CRED else {}))

node("Build Input", "n8n-nodes-base.code", code("""
const cfg = $('Config').first().json, f = $json.fields;
const input = { key: $json.key, summary: f.summary, html: ($json.renderedFields || {}).description || '',
  lang: cfg.lang, parent: f.parent && f.parent.key, jira_priority: f.priority && f.priority.name };
return { json: { key: $json.key, summary: f.summary, updated: f.updated,
  inputB64: Buffer.from(JSON.stringify(input)).toString('base64') } };
"""), [1320, 0], 2)

node("Convert", "n8n-nodes-base.executeCommand", {"executeOnce": False, "command":
    f'={{{{ "echo " + $json.inputB64 + " | base64 -d | " + {C}.python + " \\"" + {C}.toolsDir + "/jira2feature.py\\"" }}}}'}, [1540, 0])

node("Parse Convert", "n8n-nodes-base.code", code("""
const st = $('Build Input').item.json;
let r;
try { r = JSON.parse($json.stdout); } catch (e) { r = { feature: '', case: {}, warnings: ['convert failed: ' + ($json.stderr || e.message)] }; }
return { json: { ...st, feature: r.feature, caseJson: JSON.stringify(r.case), warnings: r.warnings,
  featureB64: Buffer.from(r.feature).toString('base64') } };
"""), [1760, 0], 2)

node("Lint", "n8n-nodes-base.executeCommand", {"executeOnce": False, "command":
    f'={{{{ "echo " + $json.featureB64 + " | base64 -d | \\"" + {C}.binDir + "/lint\\" -lang " + {C}.lang + " -max-scenarios 1" }}}}'}, [1980, 0])

node("Decide", "n8n-nodes-base.code", code("""
const st = $('Parse Convert').item.json;
let r;
try { r = JSON.parse($json.stdout); } catch (e) { r = { ok: false, errors: ['lint output was not JSON'], needs_input: [] }; }
// clean = parses, and nothing is missing or unresolved; otherwise a human looks at it in review/.
// A priority mismatch is reported in the notes but does not block.
const blocking = st.warnings.filter(w => !/^priority conflict/.test(w));
const clean = r.ok && r.needs_input.length === 0 && blocking.length === 0;
const notes = [...r.errors.map(e => 'LINT: ' + e), ...r.needs_input, ...st.warnings].join('\\n');
const base = st.key + '_' + st.summary.normalize('NFD').replace(/[\\u0300-\\u036f]/g, '').replace(/[^\\w.-]+/g, '_').slice(0, 60);
const b64 = s => Buffer.from(s).toString('base64');
return { json: { key: st.key, updated: st.updated, clean, base, featureB64: st.featureB64,
  caseB64: b64(st.caseJson), notesB64: notes ? b64(notes + '\\n') : '' } };
"""), [2200, 0], 2)

node("Save", "n8n-nodes-base.executeCommand", {"executeOnce": False, "command":
    f'={{{{ "d=\\"" + {C}.outDir + ($json.clean ? "" : "/review") + "\\"; mkdir -p \\"$d\\" && echo " + $json.featureB64 + " | base64 -d > \\"$d/" + $json.base + ".feature\\" && echo " + $json.caseB64 + " | base64 -d > \\"$d/" + $json.base + ".case.json\\"" '
    f'+ ($json.notesB64 ? " && echo " + $json.notesB64 + " | base64 -d > \\"$d/" + $json.base + ".notes.txt\\"" : "") + " && echo \\"$d/" + $json.base + ".feature\\"" }}}}'}, [2420, 0])

node("Remember", "n8n-nodes-base.code", code("""
// only after a successful Save: a failed run is retried on the next poll
const sd = $getWorkflowStaticData('global');
sd.seen = sd.seen || {};
const v = $('Config').first().json.version;
for (const d of $('Decide').all()) sd.seen[d.json.key] = d.json.updated + '|' + v;
return $input.all();
""", each=False), [2640, 0], 2)

node("Resolve References", "n8n-nodes-base.executeCommand", {"executeOnce": True, "command":
    f'={{{{ {C}.python + " \\"" + {C}.toolsDir + "/resolve_refs.py\\" --dir \\"" + {C}.outDir + "\\" --lint \\"" + {C}.binDir + "/lint\\" --lang " + {C}.lang }}}}'},
    [2860, 160])

node("Comment on Jira (off)", "n8n-nodes-base.httpRequest", {
    "method": "POST", "url": f"={{{{ {C}.jiraBaseUrl }}}}/rest/api/3/issue/{{{{ $('Decide').item.json.key }}}}/comment", **JIRA_AUTH,
    "sendBody": True, "specifyBody": "json",
    "jsonBody": "={{ JSON.stringify({ body: { type: 'doc', version: 1, content: [{ type: 'paragraph', content: [{ type: 'text', text: "
                "'Gherkin generated: ' + ($('Decide').item.json.clean ? 'clean' : 'needs review') }] }] } }) }}"},
    [2860, 0], 4.2, disabled=True)  # visible to the whole team: enable only deliberately

for a, b in [("Schedule", "Config"), ("Manual Trigger", "Config"), ("Config", "Search Jira"),
             ("Search Jira", "Split Issues"), ("Split Issues", "New Or Changed"), ("New Or Changed", "Get Issue"),
             ("Get Issue", "Build Input"), ("Build Input", "Convert"), ("Convert", "Parse Convert"),
             ("Parse Convert", "Lint"), ("Lint", "Decide"), ("Decide", "Save"), ("Save", "Remember"),
             ("Remember", "Resolve References"), ("Remember", "Comment on Jira (off)")]:
    link(a, b)

wf = {"name": "Jira → Gherkin" + (" (test)" if TEST else ""), "nodes": nodes, "connections": conns,
      "settings": {"executionOrder": "v1"}, "active": False}
wf["id"] = "jira-gherkin-test" if TEST else "jira-gherkin"
out = os.path.join(HERE, "jira-pipeline-test.json" if TEST else "jira-pipeline.json")
json.dump(wf, open(out, "w", encoding="utf-8"), indent=2, ensure_ascii=False)
print("wrote", os.path.basename(out), "with", len(nodes), "nodes")
