#!/usr/bin/env python3
"""LM agent: one normalized case + the real page (via Playwright MCP, read-only) -> README-structured files in a repo.

    python3 agent.py --case case.json --repo ~/Documents/GitHub/portal-qa-automation --app Portal \
        --base-url http://172.22.64.228:3000 --model prism-ml/bonsai-27b

The model never writes code or files. It fills a declarative spec (spec.py); render.py generates every file in the README
layout. Guardrails live in code, not in the prompt: the browser blocks every non-GET request (guard.cjs), the file tools are
read-only, run_code_unsafe is never offered, and `finish` is refused until a spec was rendered and the checks are clean.
"""
import argparse, json, os, sys, time, urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import render  # noqa: E402
import runner  # noqa: E402
import spec as specmod  # noqa: E402
import structure_check  # noqa: E402
from mcp_client import MCPClient, playwright_command  # noqa: E402
from sandbox import Sandbox  # noqa: E402

BROWSER_TOOLS = {"browser_navigate", "browser_snapshot", "browser_click", "browser_type", "browser_fill_form", "browser_select_option",
                 "browser_press_key", "browser_wait_for", "browser_find", "browser_navigate_back",  # no browser_evaluate: arbitrary JS was a token sink
                 "browser_hover", "browser_file_upload", "browser_network_requests", "browser_console_messages", "browser_tabs"}
RESULT_CHARS = 5000        # a page snapshot can be huge: the model only needs the head of it
FINISH_REFUSALS = 2        # how many times finish() is bounced while problems remain


def local_tools():
    obj = lambda props, req: {"type": "object", "properties": props, "required": req}
    s = {"type": "string"}
    return [
        ("list_files", "Lista archivos del repositorio (carpeta relativa a la raíz).", obj({"path": s}, [])),
        ("read_file", "Lee un archivo del repositorio.", obj({"path": s}, ["path"])),
        ("submit_spec", "Envía la spec del caso; la valida y genera los archivos. Devuelve errores o la lista de archivos.", obj({"spec": {"type": "object"}}, ["spec"])),
        ("verify_locators", "Comprueba en la página REAL (solo lectura) que los localizadores estáticos de la última spec resuelven.", obj({}, [])),
        ("run_test", "Ejecuta un test generado en modo SOLO LECTURA (las escrituras reales se bloquean).", obj({"file": s}, ["file"])),
        ("finish", "Termina. Incluye resumen y pasos NO automatizados/verificados.", obj({"summary": s}, ["summary"])),
    ]


def openai_tools(mcp_tools):
    out = []
    for t in mcp_tools:
        if t["name"] in BROWSER_TOOLS:  # everything else, run_code_unsafe especially, is simply not offered
            out.append({"type": "function", "function": {"name": t["name"], "description": (t.get("description") or "").split(".")[0][:140],
                                                          "parameters": t["inputSchema"]}})
    for name, desc, schema in local_tools():
        out.append({"type": "function", "function": {"name": name, "description": desc, "parameters": schema}})
    return out


USAGE = {"tokens": 0}  # total tokens the model processed this process; the router logs it per case


def chat(url, model, messages, tools, effort="none", temperature=0.2):
    body = {"model": model, "messages": messages, "tools": tools, "temperature": temperature, "max_tokens": 6000}
    if effort:
        body["reasoning_effort"] = effort
    req = urllib.request.Request(url + "/chat/completions", json.dumps(body).encode(), {"Content-Type": "application/json"})
    r = json.load(urllib.request.urlopen(req, timeout=900))
    USAGE["tokens"] += (r.get("usage") or {}).get("total_tokens", 0)
    return r["choices"][0]["message"]


def compact(messages, budget_chars):
    """Keeps the conversation inside the model's context: old tool results are replaced by a marker, newest kept."""
    def size():
        return sum(len(json.dumps(m, ensure_ascii=False)) for m in messages)
    tool_idx = [i for i, m in enumerate(messages) if m.get("role") == "tool"]
    for i in tool_idx[:-3]:
        if size() <= budget_chars:
            break
        if not messages[i]["content"].startswith("[omitido"):
            messages[i]["content"] = f"[omitido: resultado de {len(messages[i]['content'])} caracteres, ya no es necesario]"


def map_summary(path, limit=3500):
    """The crawl, compressed for the model: stable, unique locators only, one line each. Truncated, never the whole map."""
    try:
        crawl = json.load(open(path, encoding="utf-8"))
    except (OSError, ValueError):
        return ""
    lines, seen = [], set()
    for pg in crawl.get("pages", []):
        lines.append(f"# {pg['url'].split('//', 1)[-1].split('/', 1)[-1] or '/'}")
        for e in pg["elements"]:
            if e.get("stable") and e["locator"] not in seen and e.get("role") != "link":
                seen.add(e["locator"])
                lines.append(f"{e['role']} {e['name'][:40]!r} -> {e['locator']}" + (f" opciones={e['options']}" if e.get("options") else ""))
    return "\n".join(lines)[:limit]


def run(case, repo, app, base_url, model, lm_url, max_turns=40, effort="none", mcp=None, ctx_tokens=32000, log_path=None, notes="",
        app_map="", max_tokens=0):
    sb = Sandbox(repo, app)
    state = {"case": case, "spec": None, "rendered": False}
    scaffold_warnings = render.scaffold(repo, app)
    own_mcp = mcp is None
    mcp = mcp or MCPClient(playwright_command(), node_setup="source ~/.nvm/nvm.sh && nvm use 24")
    tools = openai_tools(mcp.tools())
    with open(os.path.join(HERE, "prompts", "system_es.md"), encoding="utf-8") as f:
        system = f.read().replace("{APP}", app)
    task = {"BASE_URL": base_url, "app": app, "case": {k: case[k] for k in ("id", "name", "meta", "precondition", "steps", "expected") if k in case}}
    if scaffold_warnings:
        task["avisos"] = scaffold_warnings
    if app_map:  # what the crawler already found: the model should not rediscover it
        task["mapa_de_la_app"] = app_map
    if notes:  # what is peculiar about this application, kept by humans in knowledge/<App>.md
        task["notas_de_la_app"] = notes
    messages = [{"role": "system", "content": system},
                {"role": "user", "content": "Automatiza este caso.\n" + json.dumps(task, ensure_ascii=False, indent=1)}]
    log = open(log_path, "w", encoding="utf-8") if log_path else None
    refusals, result = 0, {"finished": False, "summary": "", "turns": 0, "problems": []}
    start_tokens, tried = USAGE["tokens"], {}

    def note(kind, data):
        if log:
            log.write(json.dumps({"t": round(time.time(), 1), "kind": kind, "data": data}, ensure_ascii=False) + "\n")
            log.flush()

    try:
        for turn in range(max_turns):
            if max_tokens and USAGE["tokens"] - start_tokens >= max_tokens:  # a stuck run must cost a bounded amount
                result["summary"] = f"stopped: token budget of {max_tokens} reached"
                break
            if turn == 6 and not state["rendered"]:
                messages.append({"role": "user", "content": "Ya exploraste lo suficiente. Llama a submit_spec ahora con lo que sabes; el validador te dirá qué falta."})
            result["turns"] = turn + 1
            compact(messages, int(ctx_tokens * 3.2 * 0.7))
            msg = chat(lm_url, model, messages, tools, effort)
            messages.append({k: v for k, v in msg.items() if k in ("role", "content", "tool_calls") and v is not None} | {"content": msg.get("content") or ""})
            calls = msg.get("tool_calls") or []
            note("assistant", {"content": msg.get("content"), "calls": [(c["function"]["name"], c["function"]["arguments"]) for c in calls]})
            if not calls:
                messages.append({"role": "user", "content": "Usa las herramientas. Cuando termines, llama a finish."})
                continue
            for c in calls:
                name, raw = c["function"]["name"], c["function"]["arguments"]
                try:
                    args = json.loads(raw) if isinstance(raw, str) else (raw or {})
                except ValueError:
                    args = {}
                    out = "argumentos JSON inválidos"
                else:
                    sig = name + json.dumps(args, sort_keys=True)
                    tried[sig] = tried.get(sig, 0) + 1
                    if tried[sig] > 1 and (name in BROWSER_TOOLS or name == "submit_spec"):  # same call twice: no new information
                        out = ("Enviaste exactamente la misma spec: cámbiala según el error anterior." if name == "submit_spec"
                               else "Ya ejecutaste exactamente esto y no aporta nada nuevo. Usa lo que ya tienes y llama a submit_spec.")
                        note("tool", {"name": name, "args": args, "result": out})
                        messages.append({"role": "tool", "tool_call_id": c.get("id", name), "content": out})
                        continue
                    out = dispatch(name, args, sb, mcp, repo, app, base_url, result, state)
                    if name == "finish":
                        problems = runner.run_checks(repo, app, base_url) if state["rendered"] else ["aún no se ha generado ningún archivo: llama a submit_spec con una spec válida"]
                        if problems and refusals < FINISH_REFUSALS:
                            refusals += 1
                            out = "No puedes terminar todavía; corrige:\n" + "\n".join(problems[:15])
                            result["finished"] = False
                        else:
                            result["finished"], result["problems"] = True, problems
                out = out if len(out) <= RESULT_CHARS else out[:RESULT_CHARS] + "\n...[truncado]"
                note("tool", {"name": name, "args": args, "result": out[:1500]})
                messages.append({"role": "tool", "tool_call_id": c.get("id", name), "content": out})
            if result["finished"]:
                break
        if not result["finished"] and not result["summary"]:
            result["summary"] = f"stopped without finishing: {result['turns']} turns, {USAGE['tokens'] - start_tokens} tokens, no valid spec"
    finally:
        if log:
            log.close()
        if own_mcp:
            mcp.close()
    return result


def dispatch(name, args, sb, mcp, repo, app, base_url, result, state):
    try:
        if name in BROWSER_TOOLS:
            text, is_error = mcp.call(name, args)
            return ("ERROR: " if is_error else "") + text
        if name == "list_files":
            return sb.list_files(args.get("path", "."))
        if name == "read_file":
            return sb.read_file(args["path"])
        if name == "submit_spec":
            return submit_spec(args.get("spec"), repo, app, state)
        if name == "verify_locators":
            if not state["spec"]:
                return "primero envía una spec con submit_spec"
            goto = next((s["do"]["path"] for s in state["spec"]["steps"] if s.get("do", {}).get("action") == "goto"), "/")
            return runner.verify_locators(repo, app, state["spec"]["page"], base_url.rstrip("/") + goto)
        if name == "run_test":
            return runner.run_test(repo, app, args["file"], base_url)
        if name == "finish":
            result["summary"] = args.get("summary", "")
            return "ok"
        return f"herramienta desconocida: {name}"
    except PermissionError as e:
        return f"PROHIBIDO: {e}"
    except Exception as e:  # keep the agent alive; it can read the error and adapt
        return f"ERROR: {type(e).__name__}: {e}"


def submit_spec(spec, repo, app, state):
    if isinstance(spec, str):  # some models send the object as a JSON string
        try:
            spec = json.loads(spec)
        except json.JSONDecodeError as e:  # say WHERE it breaks: "not valid JSON" gave the model nothing to fix (it resent it 4 times)
            near = spec[max(0, e.pos - 50):e.pos + 30].replace("\n", " ")
            return (f"la spec no es JSON válido: {e.msg} en línea {e.lineno}, columna {e.colno}. Cerca de: ...{near}...\n"
                    "Corrige ese punto y envíala de nuevo (envía el objeto JSON, sin texto adicional).")
    page = (spec or {}).get("page", "") if isinstance(spec, dict) else ""
    existing = render.read_locators(os.path.join(repo, "apps", app, "locators", f"{page}_locators.py")) if page else {}
    errors = specmod.validate(spec, state["case"], existing_locators=existing)
    if errors:
        return "La spec tiene errores; corrígela y vuelve a enviarla:\n- " + "\n- ".join(errors)
    res = render.render(repo, app, state["case"], spec)
    if res["errors"]:
        return "No se generó nada:\n- " + "\n- ".join(res["errors"])
    state["spec"], state["rendered"] = spec, True
    problems = structure_check.check(repo, app)  # should be empty by construction; a safety net
    out = "Generado:\n" + "\n".join("  " + f for f in res["files"])
    if res["warnings"]:
        out += "\nAvisos:\n- " + "\n- ".join(res["warnings"])
    if problems:
        out += "\nREVISAR (esto no debería pasar):\n- " + "\n- ".join(problems)
    return out + "\nSiguiente: verify_locators, luego finish."


def loaded_context(lm_url, model):
    try:
        base = lm_url.rsplit("/v1", 1)[0]
        for m in json.load(urllib.request.urlopen(base + "/api/v0/models", timeout=10))["data"]:
            if m["id"] == model and m.get("state") == "loaded":
                return m.get("loaded_context_length")
    except Exception:
        return None


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--case", required=True, help="normalized case JSON (tools/jira output)")
    ap.add_argument("--repo", required=True)
    ap.add_argument("--app", required=True)
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--model", default="prism-ml/bonsai-27b")
    ap.add_argument("--lm-url", default="http://127.0.0.1:1234/v1")
    ap.add_argument("--max-turns", type=int, default=40)
    ap.add_argument("--effort", default="none")
    ap.add_argument("--map", default="", help="appmap/<App>.crawl.json: the crawl, given to the model so it does not re-explore")
    ap.add_argument("--max-tokens", type=int, default=0, help="hard cap on tokens for this case (0 = none)")
    ap.add_argument("--notes", default="", help="knowledge/<App>.md: peculiarities of the app, given to the model")
    a = ap.parse_args()
    ctx = loaded_context(a.lm_url, a.model)
    if ctx and ctx < 24000:
        sys.exit(f"{a.model} está cargado con contexto {ctx}: el agente necesita al menos 24000 (recomendado 32768). Recárgalo en LM Studio.")
    with open(a.case, encoding="utf-8") as f:
        case = json.load(f)
    os.makedirs(os.path.join(HERE, "runs"), exist_ok=True)
    log = os.path.join(HERE, "runs", f"{case['id']}-{time.strftime('%Y%m%d-%H%M%S')}.jsonl")
    res = run(case, os.path.expanduser(a.repo), a.app, a.base_url, a.model, a.lm_url, a.max_turns, a.effort, ctx_tokens=ctx or 32000, log_path=log,
              notes=open(a.notes, encoding="utf-8").read() if a.notes else "",
              app_map=map_summary(a.map) if a.map else "", max_tokens=a.max_tokens)
    print(json.dumps({**res, "tokens": USAGE["tokens"], "log": log}, indent=2, ensure_ascii=False))
    sys.exit(0 if res["finished"] and not res["problems"] else 1)
