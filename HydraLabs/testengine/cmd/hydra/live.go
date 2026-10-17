package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The live view of the pipeline, for people watching: every stage a case goes through (case in, gate, map, build,
// verify, stability, review branch) is appended as one JSON line to hydra/out/live/events.jsonl, and `hydra watch`
// serves a page that follows that file. It only reads what the commands already decided: nothing here changes a
// result, and a failure to write an event never stops the pipeline.

// StepMap: one case step and what it was mapped to, as the renderer reports it.
type StepMap struct {
	N         string `json:"n"`
	Text      string `json:"text"`
	Mapped    bool   `json:"mapped"`
	Locator   string `json:"locator,omitempty"`
	Confirmed bool   `json:"confirmed"`
	Color     string `json:"color,omitempty"` // same legend as the crawl: green stable id, blue role + name, orange weak
}

func (c Config) liveDir() string { return filepath.Join(c.Out, "live") }

// live appends one event. Key "kind" says which stage; the rest is that stage's data.
func (c Config) live(kind, key string, data map[string]any) {
	if c.Out == "" {
		return
	}
	ev := map[string]any{"t": time.Now().Format("15:04:05"), "kind": kind, "key": key, "app": c.App}
	for k, v := range data {
		ev[k] = v
	}
	b, _ := json.Marshal(ev)
	os.MkdirAll(c.liveDir(), 0o755)
	if f, err := os.OpenFile(filepath.Join(c.liveDir(), "events.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		f.Write(append(b, '\n'))
		f.Close()
	}
}

// colorOf follows the crawl's legend, from the shape of the selector.
func colorOf(loc string, confirmed bool) string {
	switch {
	case loc == "":
		return ""
	case !confirmed:
		return "#ea580c" // still a guess from the case text
	case strings.HasPrefix(loc, "[data-testid=") || strings.HasPrefix(loc, "#"):
		return "#16a34a"
	case strings.HasPrefix(loc, "role="):
		return "#2563eb"
	}
	return "#ea580c"
}

// Watch serves the live page for one app until interrupted. The page asks for new lines every half second.
func (c Config) Watch(addr string) error {
	events := filepath.Join(c.liveDir(), "events.jsonl")
	os.MkdirAll(c.liveDir(), 0o755)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.ReplaceAll(livePage, "{{APP}}", c.App))
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) { tailFrom(w, events, r.URL.Query().Get("from")) })
	mux.Handle("/shots/", http.StripPrefix("/shots/", http.FileServer(http.Dir(filepath.Join(c.liveDir(), "shots")))))
	mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) { // only a report this app's deliveries wrote
		p, ok := c.reportPath(r.URL.Query().Get("p"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, p)
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil { // the usual cause: a live view is already running, in another terminal tab
		return fmt.Errorf("cannot serve on %s (%v): if a hydra watch is already running, just open http://localhost%s; otherwise pick another port with --addr :8100", addr, err, addr)
	}
	fmt.Printf("live view of %s: http://localhost%s  (Ctrl-C to stop)\n", c.App, addr)
	return http.Serve(ln, mux)
}

// tailFrom writes the lines of file from line number `from` on (the crawl viewer's protocol).
func tailFrom(w http.ResponseWriter, file, from string) {
	b, err := os.ReadFile(file)
	if err != nil {
		return
	}
	n := 0
	fmt.Sscan(from, &n)
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	w.Header().Set("X-Total", fmt.Sprint(len(lines))) // fewer than the page has read: the log was cleared, start over
	for i := n; i < len(lines); i++ {
		if lines[i] != "" {
			fmt.Fprintln(w, lines[i])
		}
	}
}

const livePage = `<!doctype html><meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1">
<title>Hydra live: {{APP}}</title>
<style>
:root{color-scheme:light dark;--bg:#fff;--fg:#1a1a1a;--mute:#6b6b6b;--card:#f6f6f4;--line:#e3e3df;--on:#2563eb;--ok:#16a34a;--bad:#dc2626;--warn:#d97706}
@media (prefers-color-scheme:dark){:root{--bg:#141414;--fg:#ececec;--mute:#9a9a9a;--card:#1e1e1e;--line:#2d2d2d}}
body{margin:0;background:var(--bg);color:var(--fg);font:14px/1.45 system-ui,-apple-system,sans-serif}
header{display:flex;align-items:baseline;gap:12px;padding:16px 24px;border-bottom:1px solid var(--line)}
header h1{font-size:18px;margin:0}header small{color:var(--mute)}
main{display:grid;grid-template-columns:minmax(0,1.25fr) minmax(0,1fr);gap:20px;padding:20px 24px}
@media (max-width:900px){main{grid-template-columns:1fr}}
.case{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:14px 16px;margin-bottom:14px}
.case h2{font-size:15px;margin:0 0 10px;display:flex;justify-content:space-between;gap:8px}
.case h2 span{color:var(--mute);font-weight:400;font-size:13px}
.stages{display:flex;gap:4px;margin:0 0 12px;flex-wrap:wrap}
.st{flex:1;min-width:70px;text-align:center;font-size:12px;padding:6px 4px;border-radius:6px;background:var(--line);color:var(--mute);transition:background .3s,color .3s}
.st.run{background:var(--on);color:#fff;animation:p 1s infinite alternate}.st.ok{background:var(--ok);color:#fff}
.st.bad{background:var(--bad);color:#fff}.st.wait{background:var(--warn);color:#fff}
@keyframes p{to{opacity:.6}}
table{width:100%;border-collapse:collapse;font-size:13px}td{padding:4px 6px;border-top:1px solid var(--line);vertical-align:top}
td.n{color:var(--mute);width:22px}td.loc{font:12px ui-monospace,monospace;word-break:break-all}
.dot{display:inline-block;width:9px;height:9px;border-radius:2px;margin-right:6px}
.why{margin-top:8px;font-size:13px;color:var(--mute);white-space:pre-wrap}
.meta{font-size:12px;color:var(--mute);margin-top:6px}
aside h3{font-size:13px;color:var(--mute);margin:0 0 8px;font-weight:600;text-transform:uppercase;letter-spacing:.04em}
aside img{width:100%;max-height:560px;object-fit:cover;object-position:top;border:1px solid var(--line);border-radius:8px;background:var(--card)}aside img:not([src]){display:none}
#log div{font:12px ui-monospace,monospace;padding:2px 0;color:var(--mute)}
.legend{display:flex;gap:12px;font-size:12px;color:var(--mute);margin-bottom:12px;flex-wrap:wrap}
a{color:var(--on)}
.empty{color:var(--mute);padding:24px 0}
</style>
<header><h1>Hydra · {{APP}}</h1><small id=st>waiting for a run…</small></header>
<main><section>
<div class=legend><span><i class=dot style=background:#16a34a></i>stable id / test id</span><span><i class=dot style=background:#2563eb></i>role + name</span><span><i class=dot style=background:#ea580c></i>guess, not confirmed</span></div>
<div id=cases><div class=empty>Run <code>hydra run</code>, <code>hydra poll</code> or <code>hydra deliver</code> for this app and it shows up here.</div></div>
</section><aside><h3>Last screen</h3><img id=shot alt=""><h3 style="margin-top:18px">Activity</h3><div id=log></div></aside></main>
<script>
const STAGES=['Case in','Gate','Map','Build','Verify','3× stable','Review'];let n=0;const cases={};
const esc=s=>String(s??'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));
function card(k){if(cases[k])return cases[k];document.querySelector('.empty')?.remove();
const el=document.createElement('div');el.className='case';el.innerHTML='<h2><b>'+esc(k)+'</b><span></span></h2><div class=stages>'+
STAGES.map(s=>'<div class=st>'+s+'</div>').join('')+'</div><table></table><div class=why></div><div class=meta></div>';
document.getElementById('cases').prepend(el);return cases[k]={el,st:[...el.querySelectorAll('.st')]}}
function set(c,i,cls){c.st.forEach((s,j)=>{if(j<i&&!s.classList.contains('bad'))s.className='st ok'});c.st[i].className='st '+cls}
function log(e,txt){const d=document.createElement('div');d.textContent=e.t+'  '+(e.key||'')+'  '+txt;const l=document.getElementById('log');l.prepend(d);while(l.children.length>40)l.lastChild.remove()}
const STATUS={passed:'ok','needs-writes':'wait','needs-clarification':'wait','needs-review':'wait','env-down':'bad',failed:'bad',flaky:'bad'};
function on(e){const c=e.key?card(e.key):null;
switch(e.kind){
case'case':c.el.querySelector('h2 span').textContent=e.title||e.intent;set(c,0,'ok');set(c,1,'run');log(e,e.intent);break;
case'gate':if(e.ok){set(c,1,'ok');set(c,2,'run')}else{set(c,1,'wait');c.el.querySelector('.why').textContent='Questions for the author:\n• '+(e.questions||[]).join('\n• ')}log(e,e.ok?'gate passed':'back to the author');break;
case'map':set(c,2,'ok');set(c,3,'ok');set(c,4,'run');c.el.querySelector('table').innerHTML=(e.steps||[]).map(s=>'<tr><td class=n>'+esc(s.n)+'</td><td>'+esc(s.text)+'</td><td class=loc>'+(s.locator?'<i class=dot style=background:'+s.color+'></i>'+esc(s.locator):(s.mapped?'<span style=color:var(--mute)>page</span>':'<span style=color:var(--bad)>not mapped</span>'))+'</td></tr>').join('');
log(e,'mapped '+(e.steps||[]).filter(s=>s.mapped).length+'/'+(e.steps||[]).length+' steps · '+(e.tokens||0)+' tokens');break;
case'verify':set(c,4,'run');log(e,'running in the real app'+(e.show?' (visible browser)':''));break;
case'verified':set(c,4,STATUS[e.status]||'bad');if(e.shot)document.getElementById('shot').src='/shots/'+e.shot+'?'+n;
if(e.status!='passed')c.el.querySelector('.why').textContent=(e.detail||[]).join('\n');log(e,'verify: '+e.status);break;
case'done':c.el.querySelector('.meta').textContent=e.tier+' · '+e.status+' · '+(e.tokens||0)+' tokens · '+(e.ms/1000).toFixed(1)+'s';log(e,'done: '+e.status);break;
case'stability':set(c,5,e.status=='passed'?(e.run<e.of?'run':'ok'):'bad');c.st[5].textContent=e.run+'/'+e.of+(e.status=='passed'?' passed':' '+e.status);log(e,'stability run '+e.run+'/'+e.of+': '+e.status);break;
case'delivered':set(c,6,e.branch?'ok':'wait');if(e.branch)c.st[6].textContent='branch ✓';
if(e.report)c.el.querySelector('.meta').innerHTML+=' · <a target=_blank href="/report?p='+encodeURIComponent(e.report)+'">HTML report</a>';
if(e.gate)c.el.querySelector('.why').textContent=e.gate;log(e,e.branch?'review branch '+e.branch:'not delivered');break;
default:log(e,e.kind)}
document.getElementById('st').textContent='last event '+e.t}
async function tick(){try{const r=await fetch('/events?from='+n);if(+r.headers.get('X-Total')<n){location.reload();return}const t=await r.text();for(const l of t.split('\n').filter(Boolean)){n++;on(JSON.parse(l))}}catch(x){}setTimeout(tick,500)}tick();
</script>`

// caseTitle: the stored case's name, for the live view's card ("" when the case is not stored yet).
func (c Config) caseTitle(key string) string {
	var cs struct {
		Name string `json:"name"`
	}
	if b, err := os.ReadFile(filepath.Join(c.caseDir(), key+".case.json")); err == nil {
		json.Unmarshal(b, &cs)
	}
	return cs.Name
}

// reportPath: the page may open an HTML report, and nothing else: only a .html under this app's deliver folder.
func (c Config) reportPath(p string) (string, bool) {
	p = filepath.Clean(p)
	root := filepath.Join(c.Out, "deliver") + string(filepath.Separator)
	return p, filepath.IsAbs(p) && strings.HasPrefix(p, root) && strings.HasSuffix(p, ".html")
}
