package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// Live view of a crawl, for showing other people: the crawler appends events to a file, this serves them with the
// latest screenshot to a browser page. Read-only; nothing here can change the app under test.

const viewPage = `<!doctype html><meta charset=utf-8><title>HydraLabs crawl</title>
<style>:root{color-scheme:light dark}body{font:14px system-ui;margin:0;display:grid;grid-template-columns:420px 1fr;height:100vh}
#l{overflow:auto;padding:12px;border-right:1px solid #8884}#r{padding:12px;overflow:auto}
.e{padding:6px 8px;margin:4px 0;border-radius:6px;background:#8881}.page{border-left:3px solid #2a7}.skip,.note{border-left:3px solid #ca3}.error{border-left:3px solid #d44}.done{border-left:3px solid #38f;font-weight:600}
h1{font-size:16px;margin:0 0 8px}img{max-width:100%;border:1px solid #8884;border-radius:6px}small{opacity:.7}</style>
<div id=l><h1>Crawling <small id=st>…</small></h1><div id=ev></div></div><div id=r><div id=cap></div><img id=im></div>
<script>let n=0,pages=0,els=0;const ev=document.getElementById('ev');
async function tick(){try{const t=await (await fetch('/events?from='+n)).text();
for(const line of t.split('\n').filter(Boolean)){n++;const e=JSON.parse(line);const d=document.createElement('div');d.className='e '+e.kind;
if(e.kind=='page'){pages++;els+=e.elements;d.innerHTML='<b>'+e.title+'</b> <small>'+e.url+'</small><br>'+e.elements+' elements '+JSON.stringify(e.strategies);
im.src='/shots/'+e.shot+'?'+n;cap.innerHTML='<b>'+e.url+'</b>'}
else if(e.kind=='done'){d.textContent='Done: '+e.pages+' pages, '+e.elements+' elements, '+e.weak+' weak locators'}
else d.textContent=e.kind+': '+(e.text||e.url||e.action||'');
ev.prepend(d);st.textContent=pages+' pages, '+els+' elements'}}catch(x){}setTimeout(tick,700)}tick()</script>`

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
	cmd := exec.Command(c.Python, c.crawlArgs(events, out, headed)...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}
	fmt.Println("app map:", out)
	return nil
}
