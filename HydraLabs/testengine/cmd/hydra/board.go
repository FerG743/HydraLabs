package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// hydra board: one page per project with every view side by side, for demos and onboarding: the cases and whether
// each is stale, the pipeline's live view (poll/deliver in any terminal or in CI on this machine), the last crawl,
// and the case editor. It only mounts the existing pages and reads what the runs already wrote.

type BoardCase struct {
	Key     string   `json:"key"`
	Name    string   `json:"name"`
	Status  string   `json:"status"` // the last run's status; "" = never run
	Tier    string   `json:"tier"`
	Updated string   `json:"updated"`
	Runs    int      `json:"runs"`
	RunsOK  int      `json:"runsPassed"`
	Stale   []string `json:"stale"` // why the last result may no longer hold
	Next    string   `json:"next"`  // the one action its row offers: "automate" until it has a test for its current text, then "execute"
}

type Board struct {
	ProjectStatus
	BaseURL    string      `json:"baseUrl"`
	Source     string      `json:"source"`
	SignIn     bool        `json:"signIn"`
	CrawlAt    string      `json:"crawlAt"`
	CrawlPages int         `json:"crawlPages"`
	Board      []BoardCase `json:"board"`
}

func (c Config) board() Board {
	b := Board{ProjectStatus: c.Status(), BaseURL: c.PortalURL, Source: "jira", SignIn: c.LoginFile != ""}
	if c.localCases() {
		b.Source = "local"
	}
	var crawlTime time.Time
	if fi, err := os.Stat(c.crawlPath()); err == nil {
		crawlTime = fi.ModTime()
		b.CrawlAt = crawlTime.Format("2006-01-02 15:04")
		var m struct {
			Pages []json.RawMessage `json:"pages"`
		}
		if raw, err := os.ReadFile(c.crawlPath()); err == nil && json.Unmarshal(raw, &m) == nil {
			b.CrawlPages = len(m.Pages)
		}
	}
	byKey := map[string]*BoardCase{}
	get := func(k string) *BoardCase {
		if byKey[k] == nil {
			byKey[k] = &BoardCase{Key: k, Stale: []string{}}
		}
		return byKey[k]
	}
	cases, _ := c.loadCases()
	for _, cs := range cases {
		get(cs.ID).Name = cs.Name
	}
	for _, s := range b.Cases {
		bc := get(s.Key)
		bc.Status, bc.Tier, bc.Updated, bc.Runs, bc.RunsOK = s.Status, s.Tier, s.Updated, s.Runs, s.RunsOK
		if fi, err := os.Stat(filepath.Join(c.Out, "runs", s.Key+".json")); err == nil && crawlTime.After(fi.ModTime()) {
			bc.Stale = append(bc.Stale, "the app was crawled again after this run")
		}
	}
	if c.Home != "" { // a case file (any project): an edit since the last run means the result is for older text
		seen := loadState(&c)
		issues, _ := localIssues(c.localCaseDir(), "")
		for _, is := range issues {
			bc := get(is.Key)
			bc.Name = is.Summary
			switch stamp, ok := seen[is.Key]; {
			case !ok:
				bc.Stale = append(bc.Stale, "not run yet")
			case strings.SplitN(stamp, "|", 2)[0] != caseStamp(is): // the text only: Run executes a built case (stamp |execute)
				bc.Stale = append(bc.Stale, "edited after its last run")
			}
		}
	}
	for _, bc := range byKey {
		bc.Next = "execute"
		if !c.built(bc.Key) || slices.Contains(bc.Stale, "edited after its last run") {
			bc.Next = "automate"
		}
		b.Board = append(b.Board, *bc)
	}
	sort.Slice(b.Board, func(i, j int) bool { return keyLess(b.Board[i].Key, b.Board[j].Key) })
	return b
}

// BoardUI serves the board until interrupted.
func (c Config) BoardUI(addr string) error {
	if err := c.requireHome(); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, strings.NewReplacer("{{APP}}", c.App, "{{PROJECT}}", c.Project).Replace(boardPage))
	})
	mux.HandleFunc("/api/board", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c.board())
	})
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if c.Registry == nil {
		return fmt.Errorf("the board needs the registry: use --project")
	}
	registry, run := filepath.Join(c.Registry.dir, "projects.json"), &runner{}
	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(run.state())
			return
		}
		// JSON only: a page on another site cannot send it here without a preflight this server never answers
		if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			w.WriteHeader(405)
			return
		}
		var req struct{ Action, Key string }
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if req.Action == "stop" {
			json.NewEncoder(w).Encode(map[string]bool{"stopped": run.stop()})
			return
		}
		args, err := boardArgs(req.Action, c.Project, registry, req.Key)
		if err == nil {
			label := req.Action
			if req.Key != "" {
				label += " " + req.Key
			}
			err = run.start(label, exe, engineDir(), args)
		}
		if err != nil {
			w.WriteHeader(409)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(run.state())
	})
	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			key := r.URL.Query().Get("key")
			if !caseKey.MatchString(key) {
				w.WriteHeader(400)
				return
			}
			json.NewEncoder(w).Encode(c.caseData(key))
			return
		}
		if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			w.WriteHeader(405)
			return
		}
		var req struct {
			Key    string
			Values map[string]DataValue
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if err := c.saveCaseData(req.Key, req.Values); err != nil {
			w.WriteHeader(409)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(c.caseData(req.Key))
	})
	mux.Handle("/live/", http.StripPrefix("/live", c.liveView()))
	mux.Handle("/crawl/", http.StripPrefix("/crawl", crawlView(filepath.Join(c.Out, "crawl", "events.jsonl"))))
	mux.Handle("/cases/", http.StripPrefix("/cases", c.casesView()))
	mux.Handle("/help/", http.StripPrefix("/help", helpView()))
	mux.HandleFunc("/playlists/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, playlistsPage)
	})
	mux.HandleFunc("/api/playlists", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				w.WriteHeader(405)
				return
			}
			var req struct {
				Name  string
				Steps []string
			}
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			if err := c.savePlaylist(req.Name, req.Steps); err != nil {
				w.WriteHeader(409)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
		}
		json.NewEncoder(w).Encode(c.playlistNames())
	})
	mux.HandleFunc("/api/playlist-cases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cases, _ := c.loadCases()
		out := []map[string]string{}
		for _, x := range cases {
			out = append(out, map[string]string{"id": x.ID, "name": x.Name})
		}
		json.NewEncoder(w).Encode(out)
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot serve on %s (%v): pick another port with --addr 127.0.0.1:8091", addr, err)
	}
	fmt.Printf("board of %s: http://%s  (Ctrl-C to stop)\n", c.App, ln.Addr())
	return http.Serve(ln, localOnly(mux))
}

const boardPage = `<!doctype html><meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1">
<title>Hydra board: {{APP}}</title>
<style>
:root{color-scheme:light dark;--bg:#f3f3f0;--fg:#1a1a1a;--mute:#6b6b6b;--card:#fff;--line:#dcdcd6;--on:#2563eb;--ok:#16a34a;--bad:#dc2626;--warn:#b45309}
@media (prefers-color-scheme:dark){:root{--bg:#0f0f0f;--fg:#ececec;--mute:#9a9a9a;--card:#1a1a1a;--line:#2e2e2e;--warn:#f59e0b}}
*{box-sizing:border-box}html,body{height:100%}body{margin:0;background:var(--bg);color:var(--fg);font:14px/1.4 system-ui,sans-serif;display:flex;flex-direction:column}
header{display:flex;flex-wrap:wrap;gap:8px 20px;align-items:baseline;padding:12px 16px;border-bottom:1px solid var(--line);background:var(--card)}
header h1{font-size:17px;margin:0}header .url{color:var(--mute)}.stats{display:flex;flex-wrap:wrap;gap:6px 16px;margin-left:auto}
.stat b{font-size:16px;font-variant-numeric:tabular-nums}.stat span{color:var(--mute);margin-left:4px}
main{flex:1;display:grid;grid-template-columns:1fr 1fr;grid-template-rows:1fr;gap:12px;padding:12px;min-height:0}
.tile{background:var(--card);border:1px solid var(--line);border-radius:10px;display:flex;flex-direction:column;min-height:0;overflow:hidden}
.tile h2{font-size:13px;margin:0;padding:8px 12px;border-bottom:1px solid var(--line);display:flex;gap:8px;align-items:center;text-transform:uppercase;letter-spacing:.04em;color:var(--mute)}
.tile h2 .sp{flex:1}.tile h2 button,.tile h2 a{font:12px system-ui;text-transform:none;letter-spacing:0;color:var(--fg);background:none;border:1px solid var(--line);border-radius:5px;padding:2px 8px;cursor:pointer;text-decoration:none}
.tile iframe{flex:1;border:0;width:100%;min-height:0;background:var(--card)}.body{flex:1;overflow:auto}
main.one{grid-template-columns:1fr;grid-template-rows:1fr}main.one .tile:not(.big){display:none}
.tabs{display:flex;gap:2px;margin-left:4px}.tile h2 .tabs button{border:0;border-bottom:2px solid transparent;border-radius:0;padding:2px 10px;color:var(--mute);font-weight:600}
.tile h2 .tabs button[aria-selected=true]{color:var(--fg);border-bottom-color:var(--on)}
table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:7px 12px;border-bottom:1px solid var(--line);vertical-align:top}
th{font-weight:600;color:var(--mute);font-size:12px;position:sticky;top:0;background:var(--card)}td small{color:var(--mute);display:block}
.pill{display:inline-block;padding:1px 8px;border-radius:999px;font-size:12px;font-weight:600;white-space:nowrap}
.passed{background:color-mix(in srgb,var(--ok) 18%,transparent);color:var(--ok)}.failed,.env-down{background:color-mix(in srgb,var(--bad) 18%,transparent);color:var(--bad)}
.other{background:color-mix(in srgb,var(--warn) 18%,transparent);color:var(--warn)}.none{background:var(--line);color:var(--mute)}
.stale{color:var(--warn);font-size:12px}.empty{padding:24px;color:var(--mute)}
.actions{display:flex;flex-wrap:wrap;gap:8px;align-items:center;padding:8px 16px;border-bottom:1px solid var(--line);background:var(--card)}
.actions button{font:13px system-ui;padding:6px 14px;border-radius:6px;border:1px solid var(--line);background:var(--bg);color:var(--fg);cursor:pointer}
.actions button.go{background:var(--on);border-color:var(--on);color:#fff;font-weight:600}.actions button:disabled{opacity:.45;cursor:not-allowed}
.actions button#stop{border-color:var(--bad);color:var(--bad)}#runst{color:var(--mute);font-size:13px}#runst b{color:var(--fg)}
#runlog{display:none;margin:0;max-height:32vh;overflow:auto;padding:10px 16px;background:#111;color:#ddd;font:12px/1.45 ui-monospace,monospace;white-space:pre-wrap;border-bottom:1px solid var(--line)}
#runlog.open{display:block}td .run,td .data{font:12px system-ui;padding:2px 8px;border-radius:5px;border:1px solid var(--line);background:none;color:var(--fg);cursor:pointer}
button:focus-visible{outline:2px solid var(--on);outline-offset:1px}
dialog{border:1px solid var(--line);border-radius:10px;background:var(--card);color:var(--fg);padding:0;width:min(640px,calc(100vw - 32px));max-height:85vh}
dialog::backdrop{background:#0008}dialog header{padding:14px 18px;border-bottom:1px solid var(--line);background:none}dialog h3{margin:0;font-size:16px}
dialog .steps{padding:8px 18px;overflow:auto;max-height:55vh}.ds{display:grid;grid-template-columns:28px 1fr;gap:4px 8px;padding:8px 0;border-bottom:1px solid var(--line)}
.ds b{color:var(--mute);text-align:right}.ds.uses{background:color-mix(in srgb,var(--on) 7%,transparent);border-radius:6px;padding:8px 6px}
.ds label{grid-column:2;font-size:12px;color:var(--mute);display:flex;gap:6px;align-items:center}.ds input[type=text]{grid-column:2;width:100%;font:inherit;padding:7px 9px;border:1px solid var(--line);border-radius:6px;background:var(--bg);color:inherit}
.ds .col{grid-column:2;font:12px ui-monospace,monospace;color:var(--on)}dialog footer{display:flex;gap:10px;align-items:center;padding:12px 18px;border-top:1px solid var(--line)}
dialog footer button{font:inherit;padding:7px 14px;border-radius:6px;border:1px solid var(--line);background:var(--bg);color:inherit;cursor:pointer}dialog footer button.go{background:var(--on);border-color:var(--on);color:#fff;font-weight:600}
#dmsg{flex:1;font-size:13px}#dmsg.bad{color:var(--bad)}#dmsg.ok{color:var(--ok)}.dnote{padding:10px 18px;color:var(--mute);font-size:13px;margin:0}
@media (max-width:900px){main{grid-template-columns:1fr;grid-template-rows:repeat(2,80vh)}}
</style>
<header><h1>{{APP}} <small style="font-weight:400;color:var(--mute)">{{PROJECT}}</small></h1><span class=url id=url></span><div class=stats id=stats></div></header>
<div class=actions role=toolbar aria-label="Run">
<button type=button class=go data-a=poll title="Pick up new or edited cases, build their tests and verify them">▶ Poll</button>
<button type=button class=go data-a=crawl title="Map the app again (read-only)">▶ Crawl</button>
<button type=button class=go data-a=deliver title="Run the suite in the real framework; passing tests go to a review branch">▶ Deliver</button>
<button type=button id=stop disabled>■ Stop</button>
<span id=runst aria-live=polite>idle</span><span style="flex:1"></span><button type=button id=logbtn aria-expanded=false>Show output</button>
</div>
<pre id=runlog tabindex=0 aria-label="Command output"></pre>
<dialog id=data aria-labelledby=dtitle><header><h3 id=dtitle>Data</h3><p class=dnote id=dfile style="padding:4px 0 0"></p></header>
<p class=dnote id=dnote hidden></p><div class=steps id=dsteps></div>
<footer><span id=dmsg role=status></span><button type=button onclick="$('data').close()">Close</button><button type=button class=go id=dsave>Save</button></footer></dialog>
<main id=grid>
<section class="tile cases"><h2>Cases<span class=tabs role=tablist aria-label="Cases"><button type=button role=tab id=t-status aria-controls=cases aria-selected=true>Status</button><button type=button role=tab id=t-write aria-controls=p-write aria-selected=false tabindex=-1>Write a case</button></span><span class=sp></span><span id=srcinfo style="text-transform:none;letter-spacing:0"></span><a id=writeopen href="cases/" target=_blank hidden>Open ↗</a><button type=button class=exp onclick="big(this)">Expand</button></h2><div class=body id=cases role=tabpanel aria-labelledby=t-status></div><iframe id=p-write role=tabpanel aria-labelledby=t-write title="Write a case" hidden></iframe></section>
<section class=tile id=side><h2><span class=tabs role=tablist aria-label="Pipeline and tools"><button type=button role=tab id=s-live aria-controls=sp-live aria-selected=true>Live pipeline</button><button type=button role=tab id=s-crawl aria-controls=sp-crawl aria-selected=false tabindex=-1>Last crawl</button><button type=button role=tab id=s-play aria-controls=sp-play aria-selected=false tabindex=-1>Playlists</button><button type=button role=tab id=s-help aria-controls=sp-help aria-selected=false tabindex=-1>Commands</button></span><span class=sp></span><a id=sideopen href="live/" target=_blank>Open ↗</a><button type=button class=exp onclick="big(this)">Expand</button></h2><iframe id=sp-live role=tabpanel aria-labelledby=s-live title="Live pipeline"></iframe><iframe id=sp-crawl role=tabpanel aria-labelledby=s-crawl title="Last crawl" hidden></iframe><iframe id=sp-play role=tabpanel aria-labelledby=s-play title="Playlists" hidden></iframe><iframe id=sp-help role=tabpanel aria-labelledby=s-help title="Command reference" hidden></iframe></section>
</main>
<script>
const $=id=>document.getElementById(id),esc=s=>String(s??'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));
function big(btn){const t=btn.closest('.tile'),g=$('grid'),on=!t.classList.contains('big');document.querySelectorAll('.tile').forEach(x=>x.classList.remove('big'));
g.classList.toggle('one',on);if(on)t.classList.add('big');document.querySelectorAll('.tile h2 button.exp').forEach(b=>b.textContent=b.closest('.tile').classList.contains('big')?'Back to grid':'Expand')}
function pill(s){const cls=!s?'none':(s=='passed'?'passed':(s=='failed'||s=='env-down'?s:'other'));return '<span class="pill '+cls+'">'+esc(s||'not run')+'</span>'}
async function load(){try{const b=await (await fetch('api/board')).json();
$('url').textContent=(b.baseUrl||'no base URL')+(b.signIn?' · signs in first':' · no sign-in');
$('srcinfo').textContent='from '+(b.source=='local'?'cases/ files':'Jira + cases/ files');
const st=[[b.board.length,'cases'],[b.passing,'passing'],[b.needsPerson,'need a person'],[b.board.filter(c=>c.stale.length).length,'stale'],[b.testsOnBase,'tests on '+(b.base||'main')],[(b.awaitingReview||[]).length,'awaiting review']];
$('stats').innerHTML=st.map(([n,l])=>'<span class=stat><b>'+n+'</b><span>'+l+'</span></span>').join('')+'<span class=stat><span>crawl '+(b.crawlAt?esc(b.crawlAt)+' · '+b.crawlPages+' pages':'never')+'</span></span>';
$('cases').innerHTML=b.board.length?'<table><thead><tr><th>Case</th><th>Last result</th><th>History</th><th></th></tr></thead><tbody>'+b.board.map(c=>
'<tr><td><b>'+esc(c.key)+'</b> '+esc(c.name)+(c.stale.length?'<small class=stale>⚠ '+c.stale.map(esc).join(' · ')+'</small>':'')+'</td><td>'+pill(c.status)+'<small>'+esc(c.tier)+' '+esc(c.updated)+'</small></td><td>'+(c.runs?c.runsPassed+' / '+c.runs+' passed':'—')+'</td><td style="white-space:nowrap"><button type=button class=data data-key="'+esc(c.key)+'" aria-label="Data of '+esc(c.key)+'">✎ Data</button> '+(c.next==='execute'?'<button type=button class=run data-a=execute data-key="'+esc(c.key)+'" aria-label="Execute '+esc(c.key)+'" title="Run its test as it is, with the test repo\'s data"'+(busy?' disabled':'')+'>▶ Execute</button>'+(c.status&&c.status!=='passed'?' <button type=button class=run data-a=automate data-key="'+esc(c.key)+'" aria-label="Rebuild '+esc(c.key)+'" title="Its last run did not pass: build its test again from the case text"'+(busy?' disabled':'')+'>↻ Rebuild</button>':''):'<button type=button class=run data-a=automate data-key="'+esc(c.key)+'" aria-label="Automate '+esc(c.key)+'" title="Build its test from the case text (walks the case on the app if a locator is missing)"'+(busy?' disabled':'')+'>⚙ Automate</button>')+'</td></tr>').join('')+'</tbody></table>'
:'<p class=empty>No cases yet.'+' Write one in the Write a case tab.'+'</p>'}catch(e){}setTimeout(load,3000)}
load()
const tabs=[$('t-status'),$('t-write')];
function tab(t){tabs.forEach(x=>{const on=x===t;x.setAttribute('aria-selected',on);x.tabIndex=on?0:-1;$(x.getAttribute('aria-controls')).hidden=!on});
  if(t.id=='t-write'&&!$('p-write').src)$('p-write').src='cases/';$('writeopen').hidden=t.id!='t-write'}
tabs.forEach((t,i)=>{t.onclick=()=>tab(t);t.onkeydown=e=>{if(e.key=='ArrowRight'||e.key=='ArrowLeft'){const n=tabs[(i+(e.key=='ArrowRight'?1:tabs.length-1))%tabs.length];tab(n);n.focus()}}});
// the right tile: one tab per page; a page loads the first time its tab opens
const sideTabs=[['s-live','sp-live','live/'],['s-crawl','sp-crawl','crawl/'],['s-play','sp-play','playlists/'],['s-help','sp-help','help/']];
function sideTab(id){sideTabs.forEach(([t,p,path])=>{const on=t==id,tb=$(t),pn=$(p);tb.setAttribute('aria-selected',on);tb.tabIndex=on?0:-1;pn.hidden=!on;
  if(on){if(!pn.getAttribute('src'))pn.src=path;$('sideopen').href=path}})}
sideTabs.forEach(([t],i)=>{const tb=$(t);tb.onclick=()=>sideTab(t);tb.onkeydown=e=>{if(e.key=='ArrowRight'||e.key=='ArrowLeft'){const n=sideTabs[(i+(e.key=='ArrowRight'?1:sideTabs.length-1))%sideTabs.length][0];sideTab(n);$(n).focus()}}});
sideTab('s-live');
let busy=false,logOpen=false;
async function run(action,key){
  if(action=='execute'&&!key)return;if(action=='deliver'&&!confirm('Deliver runs the whole suite in the real framework and puts the tests that pass on a review branch in the test repo (nothing is pushed). Go?'))return;
  const r=await fetch('api/run',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action,key:key||''})});
  const j=await r.json();if(!r.ok){alert(j.error);return}if(!logOpen)$('logbtn').click();runState()}
document.querySelectorAll('.actions [data-a]').forEach(b=>b.onclick=()=>run(b.dataset.a));
$('cases').addEventListener('click',e=>{const b=e.target.closest('button.run');if(b)run(b.dataset.a,b.dataset.key);const d=e.target.closest('button.data');if(d)openData(d.dataset.key)});
let dkey='';
async function openData(key){dkey=key;$('dmsg').textContent='';$('dmsg').className='';const d=await (await fetch('api/data?key='+encodeURIComponent(key))).json();showData(d);$('data').showModal()}
function showData(d){$('dtitle').textContent=d.key+' · test data';$('dfile').textContent=d.delivered?'Saved in the test repo: '+d.file+' (committed), used by every run':'';
  $('dnote').hidden=!d.note;$('dnote').textContent=d.note||'';$('dsave').hidden=!d.delivered||!d.steps.some(s=>s.column);
  $('dsteps').innerHTML=d.steps.map(s=>{const id='dv-'+esc(s.column);if(!s.column)return '<div class=ds><b>'+esc(s.n)+'.</b><span>'+esc(s.text)+'</span></div>';
    const env=!!s.env;return '<div class="ds uses"><b>'+esc(s.n)+'.</b><span>'+esc(s.text.replace(/\s\S+?\.?$/,' '))+'<i>‹value›</i></span><span class=col>uses: '+esc(s.column)+'</span>'+
    '<input type=text id="'+id+'" data-col="'+esc(s.column)+'" aria-label="Value for step '+esc(s.n)+'" value="'+esc(s.value)+'" placeholder="'+(env?(s.envSet?'kept in ~/.hydra.env as '+esc(s.env)+' (type to replace)':'not set on this machine: type it'):'')+'"'+(d.delivered?'':' disabled')+'>'+
    '<label><input type=checkbox data-env="'+esc(s.column)+'"'+(env?' checked':'')+(d.delivered?'':' disabled')+'> keep it out of the repo (stored in ~/.hydra.env)</label></div>'}).join('')||'<p class=dnote>No steps.</p>'}
$('dsave').onclick=async()=>{const values={};document.querySelectorAll('#dsteps input[data-col]').forEach(i=>{const env=document.querySelector('#dsteps input[data-env="'+i.dataset.col+'"]').checked;values[i.dataset.col]={value:i.value.trim(),env}});
  const r=await fetch('api/data',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({key:dkey,values})});const j=await r.json();
  if(!r.ok){$('dmsg').className='bad';$('dmsg').textContent=j.error;return}showData(j);$('dmsg').className='ok';$('dmsg').textContent='Saved. Run it to use the new values.'};
$('stop').onclick=()=>fetch('api/run',{method:'POST',headers:{'Content-Type':'application/json'},body:'{"action":"stop"}'}).then(runState);
$('logbtn').onclick=()=>{logOpen=!logOpen;$('runlog').classList.toggle('open',logOpen);$('logbtn').textContent=logOpen?'Hide output':'Show output';$('logbtn').setAttribute('aria-expanded',logOpen)};
async function runState(){try{const s=await (await fetch('api/run')).json();busy=s.running;
  document.querySelectorAll('.actions [data-a], td button.run').forEach(b=>b.disabled=busy);$('stop').disabled=!busy;
  const t=s.seconds>=60?Math.floor(s.seconds/60)+'m '+s.seconds%60+'s':s.seconds+'s';
  $('runst').innerHTML=busy?'running <b>'+esc(s.label)+'</b> · '+t:(s.label?'last: <b>'+esc(s.label)+'</b> · '+esc(s.exit=='ok'?'done':s.exit)+' · '+t:'idle');
  const l=$('runlog'),end=l.scrollTop+l.clientHeight>=l.scrollHeight-20;l.textContent=s.log||'';if(end)l.scrollTop=l.scrollHeight}catch(e){}}
setInterval(runState,1000);runState()
</script>`
