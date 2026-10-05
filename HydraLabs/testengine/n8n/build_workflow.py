#!/usr/bin/env python3
"""Generates gherkin-pipeline.json (the n8n workflow). Edit here, re-run, re-import.
Kept as a script so the prompt and JS stay readable instead of JSON-escaped.

Flow: Prepare (matrix+suite+join) -> per ready case: Skeleton -> Lint -> Save.
Usage: python3 n8n/build_workflow.py [profile]   (profile = n8n/profiles/<name>.json)
The model is a side loop, used only to map data fields the automatic pass missed:
  Parse Lint -> Ask LM? -> Plan -> LM Studio -> Extract Map -> back to Skeleton."""
import json, os, sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(HERE)
DL = os.path.expanduser("~/Downloads")

MAP_PROMPT = """Recibes los pasos de un caso de prueba y una lista de "fields": campos de datos que ningún paso usa todavía.
Para cada campo, busca en los pasos el texto literal que representa ese dato (copiado EXACTAMENTE como aparece en el paso).
Responde SOLO un objeto JSON: {"<n del paso>": [{"field": "<campo>", "literal": "<texto exacto del paso>"}]}.
Si un campo no aparece en ningún paso, omítelo. No inventes campos ni texto."""

# Per-project settings live in n8n/profiles/<name>.json; these defaults are project-independent.
DEFAULTS = {
    "binDir": "{root}/bin",
    "outDir": "{root}/out",
    "tagCols": "prioridad,ciclo,nivel_de_prueba,parent",  # meta keys emitted as @tags
    "useLm": "auto",  # auto = only when a data column would be dropped | always | never
    "lmUrl": "http://127.0.0.1:1234/v1",
    "lmModel": "qwen/qwen3.5-9b",
    "reasoningEffort": "none",  # thinking models burn minutes reasoning; empty = omit
    "onlyIds": "",  # comma list of case ids; empty = all ready cases
    "mapPrompt": MAP_PROMPT,
}

def load_profile(name):
    with open(os.path.join(HERE, "profiles", name + ".json"), encoding="utf-8") as f:
        prof = {k: v for k, v in json.load(f).items() if not k.startswith("_")}
    cfg = {**DEFAULTS, **prof}
    return {k: os.path.expanduser(v.replace("{root}", ROOT)) if isinstance(v, str) else v for k, v in cfg.items()}

PROFILE = sys.argv[1] if len(sys.argv) > 1 else "facturacion"
CONFIG = load_profile(PROFILE)
missing = [k for k in ("matrixCsv", "suiteCsv", "nameCol", "stepCol", "expectedCol", "keyCol") if not CONFIG.get(k)]
if missing:
    sys.exit(f"profile {PROFILE!r} is missing: {', '.join(missing)}")

C = "$('Config').first().json"
nodes, conns = [], {}

def node(name, type_, params, pos, version=1):
    nodes.append({"parameters": params, "name": name, "type": type_, "typeVersion": version,
                  "position": pos, "id": name.lower().replace(" ", "-").replace("?", "")})

PER_ITEM = ("Skeleton", "Lint", "Plan", "Save")  # Execute Command defaults to "execute once": must run per case

def link(a, b, out=0):
    conns.setdefault(a, {"main": []})
    while len(conns[a]["main"]) <= out:
        conns[a]["main"].append([])
    conns[a]["main"][out].append({"node": b, "type": "main", "index": 0})

def code(js):  # Code node, JS, run once per item
    return {"mode": "runOnceForEachItem", "jsCode": js}

def sh(expr):  # Execute Command parameter: "=" makes n8n evaluate the {{ }} parts
    return "=" + expr

node("Manual Trigger", "n8n-nodes-base.manualTrigger", {}, [0, 0])

node("Config", "n8n-nodes-base.set", {"assignments": {"assignments": [
    {"id": k, "name": k, "value": v, "type": "string"} for k, v in CONFIG.items()]},
    "options": {}}, [220, 0], 3.4)

# pipefail + casecheck: an adapter that breaks the case contract fails the run here, before anything is generated
node("Prepare", "n8n-nodes-base.executeCommand", {"command": sh(
    f'set -o pipefail; mkdir -p "{{{{ {C}.outDir }}}}" && cd "{{{{ {C}.binDir }}}}" && '
    f'./matrix -name "{{{{ {C}.nameCol }}}}" -step "{{{{ {C}.stepCol }}}}" -expected "{{{{ {C}.expectedCol }}}}" -num "{{{{ {C}.numCol }}}}" "{{{{ {C}.matrixCsv }}}}" > "{{{{ {C}.outDir }}}}/matrix.json" && '
    f'./suite -key "{{{{ {C}.keyCol }}}}" "{{{{ {C}.suiteCsv }}}}" > "{{{{ {C}.outDir }}}}/suite.json" && '
    f'./join -blocked "{{{{ {C}.blockedRegex }}}}" "{{{{ {C}.outDir }}}}/matrix.json" "{{{{ {C}.outDir }}}}/suite.json" | ./casecheck')},
    [440, 0])

node("Pick Ready Cases", "n8n-nodes-base.code", {"mode": "runOnceForAllItems", "jsCode": """
const out = JSON.parse($input.first().json.stdout);
const cfg = $('Config').first().json;
const only = cfg.onlyIds ? cfg.onlyIds.split(',').map(s => s.trim()) : null;
const b64 = s => Buffer.from(s).toString('base64');
const skipped = {};
for (const c of out.cases) if (c.status !== 'ready') skipped[c.status] = (skipped[c.status] || 0) + 1;
return out.cases
  .filter(c => c.status === 'ready' && (!only || only.includes(String(c.id))))
  .map(c => ({ json: { id: c.id, name: c.name, caseB64: b64(JSON.stringify(c)),
    minThen: c.steps.filter(s => s.expected).length,   // lint: one Entonces per expected result
    mapB64: b64('{}'), lmTried: false, skipped } }));
"""}, [660, 0], 2)

node("Skeleton Input", "n8n-nodes-base.code", code("return { json: $json };"), [880, 0], 2)  # loop entry

node("Skeleton", "n8n-nodes-base.executeCommand", {"command": sh(
    f'{{{{ "echo " + $json.caseB64 + " | base64 -d | \\"" + {C}.binDir + "/skeleton\\" -tags \\"" + {C}.tagCols + "\\" -map-b64 \\"" + $json.mapB64 + "\\"" }}}}')},
    [1100, 0])

node("Parse Skeleton", "n8n-nodes-base.code", code("""
const st = $('Skeleton Input').item.json;
let r;
try { r = JSON.parse($json.stdout); } catch (e) { r = { feature: '', warnings: ['skeleton failed: ' + $json.stderr] }; }
const dropped = [...r.warnings.join('\\n').matchAll(/data column "([^"]+)" is not used/g)].map(m => m[1]);
return { json: { ...st, feature: r.feature, warnings: r.warnings, dropped,
  b64: Buffer.from(r.feature).toString('base64') } };
"""), [1320, 0], 2)

node("Lint", "n8n-nodes-base.executeCommand", {"command": sh(
    f'{{{{ "echo " + $json.b64 + " | base64 -d | \\"" + {C}.binDir + "/lint\\" -min-then " + $json.minThen + " -max-scenarios 1" }}}}')},
    [1540, 0])

node("Parse Lint", "n8n-nodes-base.code", code("""
const st = $('Parse Skeleton').item.json, cfg = $('Config').first().json;
let r;
try { r = JSON.parse($json.stdout); } catch (e) { r = { ok: false, errors: ['lint output was not JSON: ' + $json.stderr], needs_input: [] }; }
// the model is only worth calling when data would otherwise be dropped (or when forced)
const ask = !st.lmTried && (cfg.useLm === 'always' || (cfg.useLm === 'auto' && st.dropped.length > 0));
const notes = [...r.errors.map(e => 'LINT: ' + e), ...st.warnings, ...r.needs_input].join('\\n');
// clean = lint passes AND no data column was dropped; anything else goes to features/review/
return { json: { ...st, ok: r.ok, clean: r.ok && st.dropped.length === 0, errors: r.errors, ask,
  file: st.name.normalize('NFD').replace(/[\\u0300-\\u036f]/g, '').replace(/[^\\w.-]+/g, '_') + '.feature',
  notesB64: notes ? Buffer.from(notes + '\\n').toString('base64') : '' } };
"""), [1760, 0], 2)

node("Ask LM?", "n8n-nodes-base.if", {"conditions": {
    "options": {"caseSensitive": True, "typeValidation": "strict"},
    "conditions": [{"id": "ask", "leftValue": "={{ $json.ask }}", "rightValue": True,
                    "operator": {"type": "boolean", "operation": "true", "singleValue": True}}],
    "combinator": "and"}}, [1980, 0], 2.2)

node("Plan", "n8n-nodes-base.executeCommand", {"command": sh(
    f'{{{{ "echo " + $json.caseB64 + " | base64 -d | \\"" + {C}.binDir + "/skeleton\\" -plan -tags \\"" + {C}.tagCols + "\\" -map-b64 \\"" + $json.mapB64 + "\\"" }}}}')},
    [2200, 120])

MAP_SCHEMA = {"type": "object", "additionalProperties": {"type": "array", "items": {
    "type": "object", "properties": {"field": {"type": "string"}, "literal": {"type": "string"}},
    "required": ["field", "literal"]}}}
# The request body is built here, not in the HTTP node: a "}}" inside an n8n {{ }} expression ends it early.
node("Build LM Prompt", "n8n-nodes-base.code", code("""
const st = $('Parse Lint').item.json, cfg = $('Config').first().json;
const body = { model: cfg.lmModel || undefined, reasoning_effort: cfg.reasoningEffort || undefined, temperature: 0.1,
  messages: [{ role: 'system', content: cfg.mapPrompt }, { role: 'user', content: $json.stdout }],
  response_format: { type: 'json_schema', json_schema: { name: 'map', schema: %s } } };
return { json: { ...st, body: JSON.stringify(body) } };
""" % json.dumps(MAP_SCHEMA)), [2420, 120], 2)

node("LM Studio", "n8n-nodes-base.httpRequest", {
    "method": "POST", "url": f"={{{{ {C}.lmUrl }}}}/chat/completions",
    "sendBody": True, "specifyBody": "json", "jsonBody": "={{ $json.body }}",
    "options": {"timeout": 300000}}, [2640, 120], 4.2)

node("Extract Map", "n8n-nodes-base.code", code("""
const st = $('Build LM Prompt').item.json;
let t = $json.choices[0].message.content.replace(/<think>[\\s\\S]*?<\\/think>/g, '').trim();
t = t.replace(/^```[a-z]*\\n?/i, '').replace(/\\n?```$/, '');
let m = {};
try { const p = JSON.parse(t); if (p && typeof p === 'object' && !Array.isArray(p)) m = p; } catch (e) {}  // bad JSON = no extra mapping
return { json: { ...st, mapB64: Buffer.from(JSON.stringify(m)).toString('base64'), lmTried: true } };
"""), [2860, 120], 2)

node("Save", "n8n-nodes-base.executeCommand", {"command": sh(
    f'{{{{ "d=\\"" + {C}.outDir + "/features" + ($json.clean ? "" : "/review") + "\\"; mkdir -p \\"$d\\" && echo " + $json.b64 + " | base64 -d > \\"$d/" + $json.file + "\\"" '
    f'+ ($json.notesB64 ? " && echo " + $json.notesB64 + " | base64 -d > \\"$d/" + $json.file + ".notes.txt\\"" : "") + " && echo \\"$d/" + $json.file + "\\"" }}}}')},
    [2200, -120])

for a, b in [("Manual Trigger", "Config"), ("Config", "Prepare"), ("Prepare", "Pick Ready Cases"),
             ("Pick Ready Cases", "Skeleton Input"), ("Skeleton Input", "Skeleton"), ("Skeleton", "Parse Skeleton"),
             ("Parse Skeleton", "Lint"), ("Lint", "Parse Lint"), ("Parse Lint", "Ask LM?"),
             ("Plan", "Build LM Prompt"), ("Build LM Prompt", "LM Studio"), ("LM Studio", "Extract Map"),
             ("Extract Map", "Skeleton Input")]:
    link(a, b)
link("Ask LM?", "Plan", 0)   # true: map leftovers with the model, then loop back
link("Ask LM?", "Save", 1)   # false: done

for n in nodes:
    if n["name"] in PER_ITEM:
        n["parameters"]["executeOnce"] = False  # a node parameter, not a node-level flag
wf = {"name": f"Matrix → Gherkin ({PROFILE})", "nodes": nodes, "connections": conns,
      "settings": {"executionOrder": "v1"}, "active": False}
OUT = os.path.join(HERE, f"{PROFILE}-pipeline.json")
with open(OUT, "w") as f:
    json.dump(wf, f, indent=2, ensure_ascii=False)
print("wrote", os.path.relpath(OUT, ROOT), "with", len(nodes), "nodes")
