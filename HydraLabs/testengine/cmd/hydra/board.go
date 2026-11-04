package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	if c.localCases() { // the file is the case: an edit since the last run means the result is for older text
		seen := loadState(&c)
		issues, _ := localIssues(c.localCaseDir(), "")
		for _, is := range issues {
			bc := get(is.Key)
			bc.Name = is.Summary
			switch stamp, ok := seen[is.Key]; {
			case !ok:
				bc.Stale = append(bc.Stale, "not run yet")
			case stamp != caseStamp(is)+"|"+Intent(is.Labels):
				bc.Stale = append(bc.Stale, "edited after its last run")
			}
		}
	}
	for _, bc := range byKey {
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
	mux.Handle("/live/", http.StripPrefix("/live", c.liveView()))
	mux.Handle("/crawl/", http.StripPrefix("/crawl", crawlView(filepath.Join(c.Out, "crawl", "events.jsonl"))))
	mux.Handle("/cases/", http.StripPrefix("/cases", c.casesView()))
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
main{flex:1;display:grid;grid-template-columns:1fr 1fr;grid-template-rows:1fr 1fr;gap:12px;padding:12px;min-height:0}
.tile{background:var(--card);border:1px solid var(--line);border-radius:10px;display:flex;flex-direction:column;min-height:0;overflow:hidden}
.tile h2{font-size:13px;margin:0;padding:8px 12px;border-bottom:1px solid var(--line);display:flex;gap:8px;align-items:center;text-transform:uppercase;letter-spacing:.04em;color:var(--mute)}
.tile h2 .sp{flex:1}.tile h2 button,.tile h2 a{font:12px system-ui;text-transform:none;letter-spacing:0;color:var(--fg);background:none;border:1px solid var(--line);border-radius:5px;padding:2px 8px;cursor:pointer;text-decoration:none}
.tile iframe{flex:1;border:0;width:100%;min-height:0;background:var(--card)}.body{flex:1;overflow:auto}
main.one{grid-template-columns:1fr;grid-template-rows:1fr}main.one .tile:not(.big){display:none}
table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:7px 12px;border-bottom:1px solid var(--line);vertical-align:top}
th{font-weight:600;color:var(--mute);font-size:12px;position:sticky;top:0;background:var(--card)}td small{color:var(--mute);display:block}
.pill{display:inline-block;padding:1px 8px;border-radius:999px;font-size:12px;font-weight:600;white-space:nowrap}
.passed{background:color-mix(in srgb,var(--ok) 18%,transparent);color:var(--ok)}.failed,.env-down{background:color-mix(in srgb,var(--bad) 18%,transparent);color:var(--bad)}
.other{background:color-mix(in srgb,var(--warn) 18%,transparent);color:var(--warn)}.none{background:var(--line);color:var(--mute)}
.stale{color:var(--warn);font-size:12px}.empty{padding:24px;color:var(--mute)}
@media (max-width:900px){main{grid-template-columns:1fr;grid-template-rows:repeat(4,80vh)}}
</style>
<header><h1>{{APP}} <small style="font-weight:400;color:var(--mute)">{{PROJECT}}</small></h1><span class=url id=url></span><div class=stats id=stats></div></header>
<main id=grid>
<section class=tile><h2>Cases<span class=sp></span><span id=srcinfo style="text-transform:none;letter-spacing:0"></span><button type=button onclick="big(this)">Expand</button></h2><div class=body id=cases></div></section>
<section class=tile><h2>Live pipeline<span class=sp></span><a href="live/" target=_blank>Open ↗</a><button type=button onclick="big(this)">Expand</button></h2><iframe src="live/" title="Live pipeline"></iframe></section>
<section class=tile><h2>Last crawl<span class=sp></span><a href="crawl/" target=_blank>Open ↗</a><button type=button onclick="big(this)">Expand</button></h2><iframe src="crawl/" title="Last crawl"></iframe></section>
<section class=tile id=writer><h2>Write a case<span class=sp></span><a href="cases/" target=_blank>Open ↗</a><button type=button onclick="big(this)">Expand</button></h2><iframe src="cases/" title="Write a case"></iframe></section>
</main>
<script>
const $=id=>document.getElementById(id),esc=s=>String(s??'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));
function big(btn){const t=btn.closest('.tile'),g=$('grid'),on=!t.classList.contains('big');document.querySelectorAll('.tile').forEach(x=>x.classList.remove('big'));
g.classList.toggle('one',on);if(on)t.classList.add('big');document.querySelectorAll('.tile h2 button').forEach(b=>b.textContent=b.closest('.tile').classList.contains('big')?'Back to grid':'Expand')}
function pill(s){const cls=!s?'none':(s=='passed'?'passed':(s=='failed'||s=='env-down'?s:'other'));return '<span class="pill '+cls+'">'+esc(s||'not run')+'</span>'}
async function load(){try{const b=await (await fetch('api/board')).json();
$('url').textContent=(b.baseUrl||'no base URL')+(b.signIn?' · signs in first':' · no sign-in');
$('srcinfo').textContent='from '+(b.source=='local'?'cases/ files':'Jira');if(b.source!='local')$('writer').style.display='none';
const st=[[b.board.length,'cases'],[b.passing,'passing'],[b.needsPerson,'need a person'],[b.board.filter(c=>c.stale.length).length,'stale'],[b.testsOnBase,'tests on '+(b.base||'main')],[(b.awaitingReview||[]).length,'awaiting review']];
$('stats').innerHTML=st.map(([n,l])=>'<span class=stat><b>'+n+'</b><span>'+l+'</span></span>').join('')+'<span class=stat><span>crawl '+(b.crawlAt?esc(b.crawlAt)+' · '+b.crawlPages+' pages':'never')+'</span></span>';
$('cases').innerHTML=b.board.length?'<table><thead><tr><th>Case</th><th>Last result</th><th>History</th></tr></thead><tbody>'+b.board.map(c=>
'<tr><td><b>'+esc(c.key)+'</b> '+esc(c.name)+(c.stale.length?'<small class=stale>⚠ '+c.stale.map(esc).join(' · ')+'</small>':'')+'</td><td>'+pill(c.status)+'<small>'+esc(c.tier)+' '+esc(c.updated)+'</small></td><td>'+(c.runs?c.runsPassed+' / '+c.runs+' passed':'—')+'</td></tr>').join('')+'</tbody></table>'
:'<p class=empty>No cases yet.'+(b.source=='local'?' Write one in the Write a case tile.':'')+'</p>'}catch(e){}setTimeout(load,3000)}
load()
</script>`
