package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// designView is the drafting page and its API, mounted under the board (/design/). Drafting runs the same code as
// hydra design; saving writes only the passing drafts of a run, and only when someone clicks Save.
func (c Config) designView() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, strings.NewReplacer("{{APP}}", c.App).Replace(designPage))
	})
	mux.HandleFunc("/api/draft", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reply := func(code int, v any) { w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
		if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			reply(405, map[string]string{"error": "POST JSON only"})
			return
		}
		var in struct {
			Requirement string `json:"requirement"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		if strings.TrimSpace(in.Requirement) == "" {
			reply(400, map[string]string{"error": "write the requirement first"})
			return
		}
		d, err := c.Design(in.Requirement, "board", 3)
		if err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		reply(200, d)
	})
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reply := func(code int, v any) { w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
		if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			reply(405, map[string]string{"error": "POST JSON only"})
			return
		}
		var in struct {
			Run string `json:"run"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		// only a run in this project's design folder; the base name keeps the path inside it
		run := filepath.Join(c.Out, "design", filepath.Base(in.Run))
		d, err := c.LoadDesign(run)
		if err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		keys, err := c.saveDrafts(&d)
		if err == nil {
			err = writeJSON(filepath.Join(run, "design.json"), d)
		}
		if err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		reply(200, map[string]any{"keys": keys})
	})
	return mux
}

const designPage = `<!doctype html><html lang=en><head><meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1">
<title>Draft cases · {{APP}}</title>
<style>
body{font:14px system-ui;margin:0;padding:14px;color:#1a1a1a;background:#fff}
textarea{width:100%;min-height:110px;box-sizing:border-box;font:14px system-ui;padding:8px;border:1px solid #bbb;border-radius:6px}
button{font:14px system-ui;padding:7px 14px;border-radius:6px;border:1px solid #888;background:#fff;cursor:pointer}
button.go{background:#2857d6;border-color:#2857d6;color:#fff}button:disabled{opacity:.45;cursor:not-allowed}
.card{border:1px solid #ccc;border-radius:8px;padding:10px 12px;margin:10px 0}
.ok{color:#17773b;font-weight:600}.bad{color:#b42318;font-weight:600}
small{color:#666}ol,ul{margin:4px 0 4px 20px;padding:0}
#msg{margin:8px 0;color:#444}
@media (prefers-color-scheme: dark){body{color:#eee;background:#16181d}textarea,button{background:#1f2229;color:#eee;border-color:#555}.card{border-color:#444}small,#msg{color:#aaa}}
</style></head><body>
<p><b>Draft test cases from a requirement</b> <small>The drafts are checked against this app's crawl; nothing is saved until you press Save.</small></p>
<textarea id=req placeholder="Page: /Tarjetas&#10;The Tarjetas page shows the buttons &quot;Agregar&quot; and &quot;Procesar&quot;."></textarea>
<p><button class=go id=draft onclick=draft()>Draft cases</button> <button id=save onclick=save() disabled>Save passing drafts to cases/</button></p>
<div id=msg></div><div id=out></div>
<script>
const $=id=>document.getElementById(id);
const esc=s=>String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let run='';
async function post(path,body){const r=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});const j=await r.json();if(!r.ok)throw new Error(j.error||'failed');return j}
async function draft(){
  const req=$('req').value.trim(); if(!req){$('msg').textContent='Write the requirement first.';return}
  $('draft').disabled=true; $('save').disabled=true; $('out').innerHTML=''; run='';
  $('msg').textContent='Drafting... this runs on the local model and can take a minute.';
  try{
    const d=await post('api/draft',{requirement:req}); run=d.dir;
    const list=(d.drafts||[]).map(r=>'<div class=card><b>'+esc(r.draft.name)+'</b> '+(r.ok?'<span class=ok>passed</span>':'<span class=bad>rejected</span>')+(r.case?' <small>saved as '+esc(r.case)+'</small>':'')+
      '<div><small>Precondition</small> '+esc(r.draft.precondition||'(app default)')+'</div>'+
      '<ol>'+(r.draft.steps||[]).map(s=>'<li>'+esc(s)+'</li>').join('')+'</ol>'+
      '<ul>'+(r.draft.expected||[]).map(e=>'<li>'+esc(e)+'</li>').join('')+'</ul>'+
      (r.problems&&r.problems.length?'<small class=bad>'+r.problems.map(esc).join('<br>')+'</small>':'')+'</div>').join('');
    $('out').innerHTML=list||'<p>No cases came back.</p>';
    $('msg').textContent='Model '+d.model+' · '+d.tokens+' tokens · '+(d.drafts||[]).filter(r=>r.ok).length+' of '+(d.drafts||[]).length+' passed.';
    $('save').disabled=!(d.drafts||[]).some(r=>r.ok&&!r.case);
  }catch(e){$('msg').textContent=e.message}
  $('draft').disabled=false;
}
async function save(){
  if(!run)return; $('save').disabled=true;
  try{const j=await post('api/save',{run:run}); $('msg').textContent=j.keys&&j.keys.length?'Saved to cases/: '+j.keys.join(', ')+'.':'Nothing to save: every passing draft is saved already.'}
  catch(e){$('msg').textContent=e.message; $('save').disabled=false}
}
</script></body></html>`
