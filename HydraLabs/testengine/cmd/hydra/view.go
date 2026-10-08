package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
)

// Live view of a crawl, for showing other people: the crawler appends events to a file, this serves them with the
// latest screenshot to a browser page. Read-only; nothing here can change the app under test.

const viewPage = `<!doctype html><meta charset=utf-8><title>HydraLabs crawl</title>
<style>:root{color-scheme:light dark}body{font:14px system-ui;margin:0;display:grid;grid-template-columns:440px 1fr;height:100vh}
#l{overflow:auto;padding:12px;border-right:1px solid #8884}#r{padding:12px;overflow:auto}
.e{padding:6px 8px;margin:4px 0;border-radius:6px;background:#8881}.page{border-left:3px solid #2a7}.skip,.note{border-left:3px solid #ca3}.error{border-left:3px solid #d44}.done{border-left:3px solid #38f;font-weight:600}
h1{font-size:16px;margin:0 0 8px}img{max-width:100%;border:1px solid #8884;border-radius:6px}small{opacity:.7}
.lg{display:flex;gap:10px;flex-wrap:wrap;margin:6px 0 10px;font-size:12px}.lg i{display:inline-block;width:10px;height:10px;border-radius:2px;margin-right:4px;vertical-align:-1px}
#live{font:12px ui-monospace,monospace;margin:6px 0;min-height:20px}#live div{padding:1px 0;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
#bar{height:4px;background:#8883;border-radius:2px;margin:4px 0}#bar b{display:block;height:4px;background:#38f;border-radius:2px;width:0}</style>
<div id=l><h1>Crawling <small id=st>starting...</small></h1>
<div class=lg><span><i style=background:#16a34a></i>stable id / testid</span><span><i style=background:#2563eb></i>role + name</span><span><i style=background:#ea580c></i>weak or not unique</span></div>
<div id=bar><b id=pb></b></div><div id=live></div><div id=ev></div></div><div id=r><div id=cap></div><img id=im></div>
<script>let n=0,pages=0,els=0;const ev=document.getElementById('ev'),live=document.getElementById('live');
function row(e){const d=document.createElement('div');d.className='e '+e.kind;
if(e.kind=='page'){pages++;els+=e.elements;d.innerHTML='<b>'+e.title+'</b> <small>'+e.url+'</small><br>'+e.elements+' elements '+JSON.stringify(e.strategies);
im.src='/shots/'+e.shot+'?'+n;cap.innerHTML='<b>'+e.url+'</b>';live.innerHTML='';pb.style.width='0'}
else if(e.kind=='done'){d.textContent='Done: '+e.pages+' pages, '+e.elements+' elements, '+e.weak+' weak locators'}
else d.textContent=e.kind+': '+(e.text||e.url||e.action||'');return d}
async function tick(){try{const t=await (await fetch('/events?from='+n)).text();
for(const line of t.split('\n').filter(Boolean)){n++;const e=JSON.parse(line);
if(e.kind=='element'){const d=document.createElement('div');d.style.color=e.color;d.textContent=e.role+' '+JSON.stringify(e.name)+'  ->  '+e.locator;
live.prepend(d);while(live.children.length>14)live.lastChild.remove();pb.style.width=(100*e.n/e.of)+'%';
if(e.live){im.src='/shots/live.png?'+n;cap.innerHTML='<b>'+e.url+'</b> <small>'+e.n+' / '+e.of+' elements</small>'}}
else{ev.prepend(row(e));st.textContent=pages+' pages, '+els+' elements'}}}catch(x){}setTimeout(tick,500)}tick()</script>`

// Serve blocks; run it in a goroutine next to the crawl.
func Serve(addr, eventsFile string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, viewPage) })
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		from, _ := strconv.Atoi(r.URL.Query().Get("from"))
		f, err := os.Open(eventsFile)
		if err != nil {
			return
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for i := 0; sc.Scan(); i++ {
			if i >= from {
				fmt.Fprintln(w, sc.Text())
			}
		}
	})
	mux.Handle("/shots/", http.StripPrefix("/shots/", http.FileServer(http.Dir(filepath.Join(filepath.Dir(eventsFile), "shots")))))
	return http.ListenAndServe(addr, mux)
}

func (c Config) crawlArgs(events, out string, headed bool) []string {
	args := []string{c.Crawler, "--base-url", c.PortalURL, "--out", out, "--events", events, "--max-pages", strconv.Itoa(c.MaxPages)}
	if len(c.Paths) > 0 {
		args = append(args, "--paths", join(c.Paths))
	}
	if c.SafeClicks {
		args = append(args, "--safe-clicks")
	}
	if headed {
		args = append(args, "--headed")
	}
	if c.SlowMS > 0 {
		args = append(args, "--slow", strconv.Itoa(c.SlowMS))
	}
	if c.LoginFile != "" {
		args = append(args, "--login", c.LoginFile, "--login-path", c.LoginPath)
	}
	if c.EnvFile != "" {
		args = append(args, "--env-file", c.EnvFile)
	}
	return args
}

func join(s []string) string {
	out := ""
	for i, x := range s {
		if i > 0 {
			out += ","
		}
		out += x
	}
	return out
}

// Crawl runs the read-only crawler; the result is appmap/<App>.crawl.json.
func (c Config) Crawl(watch string, headed bool) error {
	dir := filepath.Join(c.Out, "crawl")
	os.MkdirAll(filepath.Join(dir, "shots"), 0o755)
	events, out := filepath.Join(dir, "events.jsonl"), filepath.Join(c.AppMap, c.App+".crawl.json")
	os.WriteFile(events, nil, 0o644)
	if watch != "" {
		go func() { fmt.Fprintln(os.Stderr, "viewer stopped:", Serve(watch, events)) }()
		fmt.Printf("watch the crawl live: http://localhost%s\n", watch)
	}
	// Ctrl-C or a kill of hydra must stop the crawler too: an orphan keeps writing to the same events file
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cmd := exec.CommandContext(ctx, c.Python, c.crawlArgs(events, out, headed)...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}
	fmt.Println("app map:", out)
	return nil
}
