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
	mux.HandleFunc("/api/inputs", func(w http.ResponseWriter, r *http.Request) { // the pages and every input the crawl knows
		w.Header().Set("Content-Type", "application/json")
		cr, err := c.loadCrawl()
		if err != nil || cr == nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "no crawl yet: run hydra crawl first"})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"pages": crawlPaths(cr), "inputs": inputsOfHand(cr, readEnvKeys(filepath.Join(c.Home, ".env")), loadHandOptions(c.handOptionsPath())), "sweeps": sweepsFromFile(c.crawlPath())})
	})
	mux.HandleFunc("/api/options", func(w http.ResponseWriter, r *http.Request) { // a person's options for a dropdown the crawl could not open
		w.Header().Set("Content-Type", "application/json")
		reply := func(code int, v any) { w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
		if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			reply(405, map[string]string{"error": "POST JSON only"})
			return
		}
		var in struct {
			Name    string   `json:"name"`
			Options []string `json:"options"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		if err := c.saveHandOption(in.Name, in.Options); err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		reply(200, map[string]string{"saved": in.Name})
	})
	mux.HandleFunc("/api/env", func(w http.ResponseWriter, r *http.Request) { // saves one chosen value; never overwrites a name already set
		w.Header().Set("Content-Type", "application/json")
		reply := func(code int, v any) { w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
		if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			reply(405, map[string]string{"error": "POST JSON only"})
			return
		}
		var in struct{ Name, Value string }
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		if err := c.saveEnvValue(in.Name, in.Value); err != nil {
			reply(409, map[string]string{"error": err.Error()})
			return
		}
		reply(200, map[string]string{"saved": in.Name})
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
<div class=card><b>Page</b> <small>the pages the crawl knows: picking one sets the requirement's Page line</small><div><select id=pageSel onchange=pickPage()></select></div></div>
<div class=card><div id=msgBox></div><b>Inputs</b> <small>what the app shows: pick a value, or type one, then save it to .env (a name already in .env is never overwritten)</small><div id=inputsBody></div></div>
<div id=msg></div><div id=out></div>
<script>
const $=id=>document.getElementById(id);
let allInputs=[], allSweeps=[];
async function loadInputs(){
  try{
    const r=await fetch('api/inputs'); const j=await r.json();
    if(!r.ok){$('msg').textContent=j.error;return}
    allInputs=j.inputs; allSweeps=j.sweeps||[];
    const keep=$('pageSel').value; // a refresh after a save keeps the page that was picked
    $('pageSel').innerHTML='<option value="">(choose a page)</option>'+j.pages.map(p=>'<option>'+esc(p)+'</option>').join('');
    $('pageSel').value=keep;
    renderInputs();
  }catch(e){$('msg').textContent='could not load the pages and inputs: '+e.message}
}
// the inputs of the page picked above: only that page's fields, grouped by tab; its values-only entries folded below
function renderInputs(){
  const page=$('pageSel').value;
  if(!page){$('inputsBody').innerHTML='<p><small>Choose a page above to see its inputs.</small></p>';$('msgBox').innerHTML='';return}
  let mine=allInputs.filter(i=>i.page===page);
  const sw=allSweeps.find(v=>v.page===page);
  const prevMsg=$('msgSel')?$('msgSel').value:'';
  $('msgBox').innerHTML=sw?'<b>'+esc(sw.control)+'</b> <small>pick a message to see only the fields it shows</small><div><select id=msgSel onchange=renderInputs()><option value="">(all fields)</option>'+sw.messages.map(m=>'<option value="'+esc(m.name)+'">'+esc(m.name)+'</option>').join('')+'</select></div>':'';
  if(sw&&$('msgSel'))$('msgSel').value=prevMsg;
  const msg=sw?prevMsg:'';
  if(sw&&msg){
    const want=new Set((sw.messages.find(m=>m.name===msg)||{fields:[]}).fields);
    mine=mine.filter(i=>want.has(i.field+':'+i.name));
  }
  if(!mine.length){$('inputsBody').innerHTML='<p><small>No inputs on '+esc(page)+(msg?' for this message':'')+'.</small></p>';return}
  const groups={}, values={};
  let k=0;
  mine.forEach(i=>{
    const key=i.group||'the page';
    const target=i.labelled?groups:values;
    (target[key]=target[key]||[]).push(inputRow(i,k++));
  });
  let html='';
  Object.keys(groups).sort().forEach(g=>{
    html+='<details open><summary><b>'+esc(g)+'</b> ('+groups[g].length+')</summary>'+groups[g].join('')+'</details>';
  });
  const nValues=Object.values(values).reduce((a,b)=>a+b.length,0);
  if(nValues){
    html+='<details><summary><b>values only, not fields</b> ('+nValues+') · the crawl took these for names; name them in the requirement if you need them</summary>';
    Object.keys(values).sort().forEach(g=>{ html+='<div><small>'+esc(g)+'</small></div>'+values[g].join(''); });
    html+='</details>';
  }
  const nIncomplete=mine.filter(i=>i.incomplete).length;
  $('inputsBody').innerHTML=(nIncomplete?'<p class=bad>'+nIncomplete+' input'+(nIncomplete>1?'s':'')+' on this page need a look (marked ⚠)</p>':'<p><small>every input here has what it needs</small></p>')+html;
}
function optionsEditor(i,k){
  if(i.field!=='combobox')return '';
  const opts=(i.options||[]).join('\n');
  return '<details><summary><small>options'+(i.handEntered?' (typed by hand)':'')+'</small></summary>'
    +'<textarea id="o'+k+'" rows="4" placeholder="one option per line">'+esc(opts)+'</textarea>'
    +'<div><button onclick="saveOptions('+k+',\''+esc(i.name).replace(/\\/g,'\\\\')+'\')">Save options</button></div></details>';
}
async function saveOptions(k,name){
  const lines=$('o'+k).value.split('\n');
  try{
    const r=await post('api/options',{name:name,options:lines});
    $('msg').textContent='saved the options of '+name;
    loadInputs();
  }catch(e){$('msg').textContent=e.message}
}
function inputRow(i,k){
  const opts=(i.options&&i.options.length)?i.options:(i.suggestion?[i.suggestion]:[]);
  const flag=i.incomplete?'<div class=bad>⚠ incomplete: '+esc(i.incomplete)+'</div>':'';
  return '<div class=card>'+flag+'<b>'+esc(i.name)+'</b> <small>'+esc(i.field)+'</small>'+(i.inEnv?' <small>· already in .env</small>':'')+'<div>'
    +'name in .env: <input id="n'+k+'" value="'+esc(i.envName)+'"'+(i.inEnv?' disabled':'')+'> '
    +'<select id="v'+k+'"><option value="">(choose)</option>'+opts.map(o=>'<option>'+esc(o)+'</option>').join('')+'</select> '
    +'<input id="t'+k+'" placeholder="or type a value"> '
    +'<button onclick="saveVal('+k+')"'+(i.inEnv?' disabled':'')+'>Save to .env</button></div>'+optionsEditor(i,k)+'</div>';
}
async function saveVal(k){
  const name=$('n'+k).value.trim();
  const value=$('t'+k).value.trim()||$('v'+k).value;
  if(!name){$('msg').textContent='give the .env name first';return}
  if(!value){$('msg').textContent='choose or type a value first';return}
  if(!confirm('Save '+name+' = '+value+' to the project .env?'))return;
  try{
    const r=await post('api/env',{name:name,value:value});
    $('msg').textContent='saved '+r.saved+' in .env';
    loadInputs();
  }catch(e){$('msg').textContent=e.message}
}
function pickPage(){
  renderInputs();
  const path=$('pageSel').value; if(!path)return;
  const req=$('req'); const line='Page: '+path;
  const lines=req.value.split('\n');
  if(lines.length&&/^Page:/i.test(lines[0].trim())) lines[0]=line; else lines.unshift(line);
  req.value=lines.join('\n');
}
loadInputs();
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
    $('out').innerHTML=needsHTML(d.needs)+(list||'<p>No cases came back.</p>');
    $('msg').textContent='Model '+d.model+' · '+d.tokens+' tokens · '+(d.drafts||[]).filter(r=>r.ok).length+' of '+(d.drafts||[]).length+' passed.';
    $('save').disabled=!(d.drafts||[]).some(r=>r.ok&&!r.case);
  }catch(e){$('msg').textContent=e.message}
  $('draft').disabled=false;
}
function needsHTML(needs){
  if(!needs||!needs.length)return '';
  return '<div class=card><b>What you need to fill in</b><ul>'+needs.map(n=>{
    if(n.kind==='value')return '<li><b>'+esc(n.name)+'</b> · <span class="'+(n.status==='defined in .env'?'ok':'bad')+'">'+esc(n.status)+'</span> <small>'+esc(n.hint)+'</small></li>';
    if(n.kind==='options')return '<li><b>'+esc(n.name)+'</b> · options: '+(n.values||[]).map(esc).join(', ')+'</li>';
    return '<li><b>Table title</b> · '+(n.name?'"'+esc(n.name)+'" (confirm it)':'not named: '+esc(n.hint))+'</li>';
  }).join('')+'</ul></div>';
}
async function save(){
  if(!run)return; $('save').disabled=true;
  try{const j=await post('api/save',{run:run}); $('msg').textContent=j.keys&&j.keys.length?'Saved to cases/: '+j.keys.join(', ')+'.':'Nothing to save: every passing draft is saved already.'}
  catch(e){$('msg').textContent=e.message; $('save').disabled=false}
}
</script></body></html>`
