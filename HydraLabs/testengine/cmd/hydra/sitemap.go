package main

import (
	"os"
)

// The site map: the app map (the crawl's pages, links and elements, <App>.crawl.json) drawn for people, as a graph of
// pages joined by the links the crawl found. The board's Site map tab shows it; the page reads it from /api/sitemap.

// sitemapData is the crawl the app map was built from, or an empty map when no crawl has run yet.
func (c Config) sitemapData() []byte {
	b, err := os.ReadFile(c.crawlPath())
	if err != nil {
		return []byte(`{"pages":[]}`)
	}
	return b
}

const sitemapPage = `<!doctype html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Site map</title>
<style>
:root { --bg:#f3f3f0; --fg:#1a1a1a; --mute:#6b6b6b; --card:#fff; --line:#dcdcd6; --on:#2563eb; --ok:#16a34a; --warn:#b45309; color-scheme: light }
@media (prefers-color-scheme: dark) { :root { --bg:#0f0f0f; --fg:#ececec; --mute:#9a9a9a; --card:#1a1a1a; --line:#2e2e2e; --warn:#f59e0b; color-scheme: dark } }
* { box-sizing: border-box }
body { margin:0; background:var(--bg); color:var(--fg); font:14px/1.4 system-ui,sans-serif; height:100vh; display:flex; flex-direction:column }
header { padding:8px 12px; border-bottom:1px solid var(--line); background:var(--card); color:var(--mute); font-size:13px }
.wrap { flex:1; display:flex; min-height:0 }
#canvas { flex:1; overflow:auto; padding:12px; min-width:0 }
#detail { width:340px; border-left:1px solid var(--line); background:var(--card); overflow:auto; padding:12px 14px; font-size:13px }
svg text { fill:var(--fg); font:12px system-ui,sans-serif; pointer-events:none }
svg .path { fill:var(--mute); font-family:ui-monospace,Menlo,monospace; font-size:11px }
svg .node rect { fill:var(--card); stroke:var(--line); stroke-width:1.5 }
svg .node { cursor:pointer }
svg .node:hover rect { stroke:var(--mute) }
svg .node.sel rect { stroke:var(--on); stroke-width:2.5 }
svg .edge { stroke:var(--mute); stroke-width:1.4; fill:none; opacity:.7 }
svg .edge.dim, svg .node.dim { opacity:.18 }
svg .edge.hot { stroke:var(--on); opacity:1; stroke-width:2 }
#detail h2 { font-size:16px; margin:0 0 4px }
#detail .url { color:var(--mute); word-break:break-all; font-family:ui-monospace,Menlo,monospace; font-size:12px }
#detail h3 { font-size:12px; text-transform:uppercase; letter-spacing:.04em; color:var(--mute); margin:16px 0 6px }
#detail ul { list-style:none; padding:0; margin:0 }
#detail li { padding:4px 0; border-bottom:1px solid var(--line) }
#detail button.link { font:inherit; color:var(--on); background:none; border:0; padding:0; cursor:pointer; text-align:left }
.el { display:grid; grid-template-columns:auto 1fr; gap:2px 8px; padding:6px 0; border-bottom:1px solid var(--line) }
.el small { color:var(--mute) }
.el code { font:11px ui-monospace,Menlo,monospace; word-break:break-all; color:var(--fg) }
.tag { font-size:11px; padding:0 6px; border-radius:999px; border:1px solid var(--line); color:var(--mute); white-space:nowrap; align-self:start }
.tag.ok { color:var(--ok); border-color:var(--ok) }
.tag.warn { color:var(--warn); border-color:var(--warn) }
.empty { padding:24px; color:var(--mute) }
@media (max-width:700px) { .wrap { flex-direction:column } #detail { width:auto; border-left:0; border-top:1px solid var(--line); max-height:50vh } }
</style>
<header id="meta">Loading the site map…</header>
<div class="wrap">
  <div id="canvas"></div>
  <aside id="detail"></aside>
</div>
<script>
var NS = "http://www.w3.org/2000/svg", W = 190, H = 58, COL = 250, ROW = 84, PAD = 24;
function el(tag, text, cls) { var e = document.createElement(tag); if (text != null) e.textContent = text; if (cls) e.className = cls; return e; }
function svg(tag, attrs) { var e = document.createElementNS(NS, tag); for (var k in attrs) e.setAttribute(k, attrs[k]); return e; }
function norm(u) { var x = new URL(u); x.hash = ""; return x.href.replace(/\/$/, ""); }
function label(e) { return e.name || e.placeholder || e.byLabel || e.testid || e.id || e.tag || "element"; }
function pathOf(u) { try { var x = new URL(u); return x.pathname + x.search || "/"; } catch (err) { return u; } }

function draw(data) {
  var pages = data.pages || [], meta = document.getElementById("meta");
  if (!pages.length) { document.getElementById("canvas").appendChild(el("p", "No crawl yet. Run hydra crawl, then this map fills in.", "empty")); meta.textContent = "Site map"; return; }
  var byUrl = {}, seen = {}, links = 0;
  pages.forEach(function (p, i) { p.i = i; p.out = []; p.inn = []; p.elements = p.elements || []; byUrl[norm(p.url)] = p; });
  pages.forEach(function (p) {
    p.elements.forEach(function (e) {
      if (!e.href) return;
      var t; try { t = new URL(e.href, p.url); } catch (err) { return; }
      var q = byUrl[norm(t.href)];
      if (!q || q === p || seen[p.i + ">" + q.i]) return;
      seen[p.i + ">" + q.i] = true; links++;
      p.out.push(q); q.inn.push(p);
    });
  });
  meta.textContent = pages.length + " pages · " + links + " links between them" + (data.crawledAt ? " · crawled " + data.crawledAt : "") + " · click a page for its elements";

  // a page's column is its distance from the start page; pages the start page cannot reach go in a last column
  var start = byUrl[norm(data.baseUrl || pages[0].url)] || pages[0], depth = {}, queue = [start], maxD = 0;
  depth[start.i] = 0;
  while (queue.length) { var p = queue.shift(); p.out.forEach(function (q) { if (depth[q.i] === undefined) { depth[q.i] = depth[p.i] + 1; maxD = Math.max(maxD, depth[q.i]); queue.push(q); } }); }
  pages.forEach(function (p) { if (depth[p.i] === undefined) depth[p.i] = maxD + 1; });
  var cols = [];
  pages.forEach(function (p) { (cols[depth[p.i]] = cols[depth[p.i]] || []).push(p); });
  var rows = 0;
  cols.forEach(function (c, d) { c.forEach(function (p, idx) { p.x = PAD + d * COL; p.y = PAD + idx * ROW; }); rows = Math.max(rows, c.length); });

  var s = svg("svg", { width: PAD * 2 + (cols.length - 1) * COL + W, height: PAD * 2 + (rows - 1) * ROW + H });
  var defs = svg("defs", {}), mk = svg("marker", { id: "arr", viewBox: "0 0 10 10", refX: 9, refY: 5, markerWidth: 7, markerHeight: 7, orient: "auto" });
  mk.appendChild(svg("path", { d: "M0,0 L10,5 L0,10 z", fill: "var(--mute)" })); defs.appendChild(mk); s.appendChild(defs);

  var edges = [], nodes = [], selected = null;
  pages.forEach(function (p) {
    p.out.forEach(function (q) {
      var x1 = p.x + W, y1 = p.y + H / 2, x2 = q.x, y2 = q.y + H / 2, d = "M" + x1 + "," + y1 + " C" + (x1 + 50) + "," + y1 + " " + (x2 - 50) + "," + y2 + " " + x2 + "," + y2;
      var line = svg("path", { d: d, class: "edge", "marker-end": "url(#arr)" });
      line.from = p; line.to = q; edges.push(line); s.appendChild(line);
    });
  });
  pages.forEach(function (p) {
    var g = svg("g", { class: "node", transform: "translate(" + p.x + "," + p.y + ")", tabindex: 0, role: "button" });
    g.appendChild(svg("rect", { width: W, height: H, rx: 8 }));
    // the path names the page: a single-page app can give every page the same title
    var t = svg("text", { x: 10, y: 24 }); t.textContent = pathOf(p.url).slice(0, 28); g.appendChild(t);
    var u = svg("text", { x: 10, y: 43, class: "path" }); u.textContent = (p.title || "").slice(0, 32); g.appendChild(u);
    g.onclick = function () { select(p); };
    g.onkeydown = function (ev) { if (ev.key === "Enter" || ev.key === " ") { ev.preventDefault(); select(p); } };
    g.page = p; nodes.push(g); s.appendChild(g);
  });
  document.getElementById("canvas").appendChild(s);

  function select(p) {
    selected = p;
    nodes.forEach(function (g) { var on = g.page === p; g.classList.toggle("sel", on); g.classList.toggle("dim", !on && p.out.indexOf(g.page) < 0 && p.inn.indexOf(g.page) < 0); });
    edges.forEach(function (e) { var hot = e.from === p || e.to === p; e.classList.toggle("hot", hot); e.classList.toggle("dim", !hot); });
    showDetail(p);
  }
  function linkList(title, list) {
    var h = el("h3", title + " (" + list.length + ")"), ul = el("ul");
    list.forEach(function (q) { var li = el("li"), b = el("button", pathOf(q.url), "link"); b.type = "button"; b.onclick = function () { select(q); }; li.appendChild(b); ul.appendChild(li); });
    if (!list.length) ul.appendChild(el("li", "none", "")).style.color = "var(--mute)";
    return [h, ul];
  }
  function showDetail(p) {
    var box = document.getElementById("detail");
    box.innerHTML = "";
    box.appendChild(el("h2", pathOf(p.url)));
    box.appendChild(el("div", p.title || "(no title)", "url"));
    box.appendChild(el("div", p.url, "url"));
    linkList("Links to", p.out).forEach(function (n) { box.appendChild(n); });
    linkList("Linked from", p.inn).forEach(function (n) { box.appendChild(n); });
    box.appendChild(el("h3", "Elements (" + p.elements.length + ")"));
    p.elements.slice(0, 300).forEach(function (e) {
      var row = el("div", null, "el");
      row.appendChild(el("span", e.role || e.tag || "", "tag " + (e.stable ? "ok" : "warn")));
      var info = el("div"); info.appendChild(el("div", label(e)));
      info.appendChild(el("code", e.locator || "no locator"));
      row.appendChild(info); box.appendChild(row);
    });
    if (p.elements.length > 300) box.appendChild(el("p", "Showing 300 of " + p.elements.length + " elements.", "empty"));
  }
  select(start);
}

fetch("../api/sitemap").then(function (r) { return r.json(); }).then(draw).catch(function () {
  document.getElementById("meta").textContent = "Could not load the site map.";
});
</script>
`
