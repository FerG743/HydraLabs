package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// hydra cases: a small local page to write test cases without opening the test repo. Saving writes cases/<KEY>.md
// (the same file a person could write by hand) and commits it on the base branch, so poll picks it up and delivery
// finds a clean repo. Localhost only: the page writes to the repo.

type caseForm struct {
	Key          string   `json:"key"` // empty: a new case, numbered after the last one
	Name         string   `json:"name"`
	Intent       string   `json:"intent"` // automate | execute
	Precondition string   `json:"precondition"`
	Steps        []string `json:"steps"`
	Expected     []string `json:"expected"`
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// markdown is the file a person would write; mdIssue + ToCase read it back.
func (f caseForm) markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\nlabels: %s\n\n", oneLine(f.Name), f.Intent)
	if p := oneLine(f.Precondition); p != "" {
		fmt.Fprintf(&b, "- Preconditions: %s\n", p)
	}
	b.WriteString("- Steps:\n")
	for i, s := range f.Steps {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, oneLine(s))
	}
	b.WriteString("- Expected result:\n")
	for _, e := range f.Expected {
		fmt.Fprintf(&b, "  - %s\n", oneLine(e))
	}
	return b.String()
}

func formOf(is Issue, lang string) (caseForm, *Case) {
	cs, _ := ToCase(is.HTML, is.Key, is.Summary, lang, "", "")
	f := caseForm{Key: is.Key, Name: is.Summary, Intent: Intent(is.Labels), Precondition: cs.Precondition, Steps: []string{}, Expected: []string{}}
	for _, s := range cs.Steps {
		f.Steps = append(f.Steps, s.Action)
	}
	for _, e := range cs.Expected {
		f.Expected = append(f.Expected, e.Text)
	}
	return f, cs
}

// nextKey: <PROJECT>-<highest number in cases/ + 1>.
func nextKey(project string, issues []Issue) string {
	n := 0
	for _, is := range issues {
		if v, err := strconv.Atoi(strings.TrimPrefix(is.Key, project+"-")); err == nil && v > n {
			n = v
		}
	}
	return fmt.Sprintf("%s-%d", project, n+1)
}

// saveCase validates, writes and commits one case. It returns the key and what the pipeline would still ask.
func (c Config) saveCase(f caseForm) (string, []string, error) {
	f.Name = oneLine(f.Name)
	var steps, exp []string
	for _, s := range f.Steps {
		if s = oneLine(s); s != "" {
			steps = append(steps, s)
		}
	}
	for _, e := range f.Expected {
		if e = oneLine(e); e != "" {
			exp = append(exp, e)
		}
	}
	f.Steps, f.Expected = steps, exp
	if f.Intent != "execute" {
		f.Intent = "automate"
	}
	switch {
	case f.Name == "":
		return "", nil, fmt.Errorf("give the case a name")
	case len(f.Steps) == 0:
		return "", nil, fmt.Errorf("add at least one step")
	case len(f.Expected) == 0:
		return "", nil, fmt.Errorf("add at least one expected result")
	}
	repo := c.Home
	if b, _ := git(repo, "branch", "--show-current"); b != c.BaseBranch {
		return "", nil, fmt.Errorf("the test repo is on %q, not %q (a delivery running?): try again when it is back", b, c.BaseBranch)
	}
	issues, err := localIssues(c.localCaseDir(), "")
	if err != nil {
		return "", nil, err
	}
	if f.Key == "" {
		f.Key = nextKey(c.Project, issues)
	} else if f.Key = strings.ToUpper(f.Key); !caseKey.MatchString(f.Key) {
		return "", nil, fmt.Errorf("%q is not a case key (e.g. %s-1)", f.Key, c.Project)
	}
	rel := filepath.Join("cases", f.Key+".md")
	files, _ := filepath.Glob(filepath.Join(c.localCaseDir(), "*.md"))
	for _, p := range files { // editing a hand-written shop-1.md: write over it, not next to it
		if strings.EqualFold(filepath.Base(p), f.Key+".md") {
			rel = filepath.Join("cases", filepath.Base(p))
		}
	}
	md := f.markdown()
	if err := os.MkdirAll(c.localCaseDir(), 0o755); err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(filepath.Join(repo, rel), []byte(md), 0o644); err != nil {
		return "", nil, err
	}
	if _, err := git(repo, "add", rel); err != nil {
		return "", nil, err
	}
	if st, _ := git(repo, "status", "--porcelain", "--", rel); st != "" { // saving an unchanged case is not a commit
		if _, err := git(repo, append(gitIdentity(repo), "commit", "-q", "-m", "case "+f.Key+": "+f.Name, "--", rel)...); err != nil {
			return "", nil, err
		}
	}
	is := mdIssue(md)
	is.Key = f.Key
	_, cs := formOf(is, c.Lang)
	blocking, _ := Maturity(cs)
	return f.Key, blocking, nil
}

// CasesUI serves the page until interrupted.
func (c Config) CasesUI(addr string) error {
	if c.Home == "" || !fileExists(filepath.Join(c.Home, ".git")) {
		return fmt.Errorf("no test repo for %s yet: run hydra init (or poll once) first", c.App)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.NewReplacer("{{APP}}", c.App, "{{PROJECT}}", c.Project).Replace(casesPage))
	})
	mux.HandleFunc("/api/cases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		reply := func(code int, v any) { w.WriteHeader(code); json.NewEncoder(w).Encode(v) }
		if r.Method == http.MethodGet {
			issues, err := localIssues(c.localCaseDir(), "")
			if err != nil {
				reply(400, map[string]string{"error": err.Error()})
				return
			}
			forms := []caseForm{}
			for _, is := range issues {
				f, _ := formOf(is, c.Lang)
				forms = append(forms, f)
			}
			sort.Slice(forms, func(a, b int) bool { return keyLess(forms[a].Key, forms[b].Key) })
			reply(200, forms)
			return
		}
		// a page on another site cannot send JSON here without a preflight this server never answers
		if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			reply(405, map[string]string{"error": "POST JSON only"})
			return
		}
		var f caseForm
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&f); err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		key, ask, err := c.saveCase(f)
		if err != nil {
			reply(400, map[string]string{"error": err.Error()})
			return
		}
		reply(200, map[string]any{"key": key, "questions": ask})
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot serve on %s (%v): pick another port with --addr 127.0.0.1:8101", addr, err)
	}
	fmt.Printf("test cases of %s: http://%s  (Ctrl-C to stop)\n", c.App, ln.Addr())
	return http.Serve(ln, localOnly(mux))
}

// localOnly refuses requests for another host name: a site that points its own name at 127.0.0.1 (DNS rebinding)
// would otherwise be same-origin with this page.
func localOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			http.Error(w, "open this page at http://localhost", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// keyLess: SHOP-2 before SHOP-10.
func keyLess(a, b string) bool {
	pa, na, _ := strings.Cut(a, "-")
	pb, nb, _ := strings.Cut(b, "-")
	ia, ea := strconv.Atoi(na)
	ib, eb := strconv.Atoi(nb)
	if pa != pb || ea != nil || eb != nil {
		return a < b
	}
	return ia < ib
}

const casesPage = `<!doctype html><meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1">
<title>Test cases: {{APP}}</title>
<style>
:root{color-scheme:light dark;--bg:#fff;--fg:#1a1a1a;--mute:#6b6b6b;--card:#f6f6f4;--line:#dcdcd6;--on:#2563eb;--ok:#16a34a;--bad:#dc2626;--warn:#b45309}
@media (prefers-color-scheme:dark){:root{--bg:#141414;--fg:#ececec;--mute:#9a9a9a;--card:#1e1e1e;--line:#333;--warn:#f59e0b}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--fg);font:15px/1.45 system-ui,sans-serif}
main{display:grid;grid-template-columns:260px 1fr;min-height:100vh}
aside{border-right:1px solid var(--line);padding:16px;background:var(--card)}
aside h1{font-size:15px;margin:0 0 12px}aside button.case{display:block;width:100%;text-align:left;background:none;border:0;color:inherit;padding:8px;border-radius:6px;cursor:pointer;font:inherit}
aside button.case:hover,aside button.case[aria-current=true]{background:var(--line)}aside small{color:var(--mute);display:block}
section{padding:24px;max-width:760px}h2{margin:0 0 16px;font-size:20px}
label{display:block;font-weight:600;margin:16px 0 6px}label small{font-weight:400;color:var(--mute)}
input,textarea,select{width:100%;font:inherit;padding:8px 10px;border:1px solid var(--line);border-radius:6px;background:var(--bg);color:inherit}
textarea{min-height:42px;resize:vertical}.row{display:flex;gap:6px;margin:6px 0;align-items:flex-start}.row b{min-width:22px;padding-top:9px;color:var(--mute);text-align:right}
.row button{flex:none}button{font:inherit;padding:8px 14px;border-radius:6px;border:1px solid var(--line);background:var(--card);color:inherit;cursor:pointer}
button.primary{background:var(--on);border-color:var(--on);color:#fff;font-weight:600}button.new{width:100%;margin-bottom:12px}
button:focus-visible,input:focus-visible,textarea:focus-visible,select:focus-visible{outline:2px solid var(--on);outline-offset:1px}
.bar{display:flex;gap:10px;align-items:center;margin-top:24px}#msg{margin-top:16px;padding:10px 12px;border-radius:6px;display:none}
#msg.ok{display:block;background:color-mix(in srgb,var(--ok) 14%,transparent)}#msg.bad{display:block;background:color-mix(in srgb,var(--bad) 14%,transparent)}
#msg ul{margin:6px 0 0;padding-left:20px;color:var(--warn)}.hint{color:var(--mute);font-size:13px;margin:4px 0 0}
@media (max-width:720px){main{grid-template-columns:1fr}aside{border-right:0;border-bottom:1px solid var(--line)}section{padding:16px}}
</style>
<main>
<aside><h1>{{APP}} test cases</h1><button class=new type=button onclick="edit()">+ New case</button><nav id=list aria-label="Cases"></nav></aside>
<section>
<h2 id=title>New case</h2>
<form id=f onsubmit="save(event)">
<label for=name>Name</label><input id=name required placeholder="Sign in and open the orders page">
<label for=intent>What should Hydra do?</label>
<select id=intent><option value=automate>Build the test (automate)</option><option value=execute>Only run its stored test (execute)</option></select>
<label for=pre>Precondition <small>the starting state: signed in? data present? "Nobody is signed in" starts signed out</small></label>
<textarea id=pre placeholder="The QA user is signed in."></textarea>
<label>Steps <small>one action each</small></label><div id=steps></div>
<button type=button onclick="add('steps').querySelector('textarea').focus()">+ Step</button>
<p class=hint>e.g. <i>Navigate to /orders.</i> · <i>Click "New order".</i> · <i>Enter order number ` + "`98`" + `.</i></p>
<label>Expected result</label><div id=expected></div>
<button type=button onclick="add('expected').querySelector('textarea').focus()">+ Expected result</button>
<p class=hint>e.g. <i>The page shows "Orders".</i> · <i>"Procesar" is disabled before the order is added and enabled after.</i></p>
<div class=bar><button class=primary>Save case</button><span id=keyinfo class=hint></span></div>
<div id=msg role=status></div>
</form>
</section>
</main>
<script>
let cases=[],cur=null;
const $=id=>document.getElementById(id);
function add(box,v=''){const d=document.createElement('div');d.className='row';
d.innerHTML='<b></b><textarea rows=1></textarea><button type=button aria-label="Remove">✕</button>';
d.querySelector('textarea').value=v;d.querySelector('button').onclick=()=>{d.remove();num(box)};$(box).append(d);num(box);return d}
function num(box){[...$(box).children].forEach((r,i)=>r.querySelector('b').textContent=box=='steps'?(i+1)+'.':'•')}
function vals(box){return [...$(box).querySelectorAll('textarea')].map(t=>t.value.trim()).filter(Boolean)}
function edit(c){cur=c||null;$('title').textContent=c?c.key+': '+c.name:'New case';$('name').value=c?c.name:'';
$('intent').value=c?c.intent||'automate':'automate';$('pre').value=c?c.precondition:'';
for(const b of ['steps','expected']){$(b).innerHTML='';const v=c?(b=='steps'?c.steps:c.expected):[];(v.length?v:['']).forEach(x=>add(b,x))}
$('keyinfo').textContent=c?'saved as cases/'+c.key+'.md':'it gets the next number: {{PROJECT}}-…';$('msg').className='';
[...$('list').children].forEach(b=>b.setAttribute('aria-current',c&&b.dataset.key==c.key));$('name').focus()}
async function load(){const r=await fetch('/api/cases');const j=await r.json();if(!r.ok){show('bad',j.error);return}cases=j;
$('list').innerHTML='';for(const c of cases){const b=document.createElement('button');b.type='button';b.className='case';b.dataset.key=c.key;
b.innerHTML='<span></span><small></small>';b.firstChild.textContent=c.name||'(no name)';b.lastChild.textContent=c.key+' · '+c.intent;
b.onclick=()=>edit(c);$('list').append(b)}if(!cases.length)$('list').innerHTML='<p class=hint>No cases yet.</p>'}
function show(cls,text,qs){const m=$('msg');m.className=cls;m.textContent=text;if(qs&&qs.length){const u=document.createElement('ul');
qs.forEach(q=>{const li=document.createElement('li');li.textContent=q;u.append(li)});m.append(u)}}
async function save(e){e.preventDefault();const body={key:cur?cur.key:'',name:$('name').value,intent:$('intent').value,precondition:$('pre').value,steps:vals('steps'),expected:vals('expected')};
const r=await fetch('/api/cases',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});const j=await r.json();
if(!r.ok){show('bad',j.error);return}await load();edit(cases.find(c=>c.key==j.key));
show('ok','Saved '+j.key+'. Hydra picks it up on the next poll.'+(j.questions&&j.questions.length?' Before it can be automated, answer:':''),j.questions)}
load().then(()=>edit())
</script>`
