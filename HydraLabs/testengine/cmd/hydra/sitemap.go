package main

import (
	"os"
)

// The site map: the app map (the crawl's pages, links and elements, <App>.crawl.json) drawn for people as a graph in the
// style of Obsidian's: a dot per page, a line per link the crawl found, pulled into shape by a small force layout. The
// board's Site map tab shows it; the page reads the data from /api/sitemap. Each playlist is a square, linked to the
// pages it reaches, and coloured by whether it exercises every element of them (out/coverage.json, from /api/coverage).

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
:root { --bg:#fbfbfa; --fg:#1d1d1f; --mute:#8a8a8f; --card:#ffffff; --line:#e4e4e8; --on:#7c3aed; --warn:#b45309; --ok:#15803d; color-scheme: light }
@media (prefers-color-scheme: dark) { :root { --bg:#1b1b1d; --fg:#dcdcdf; --mute:#7c7c84; --card:#242427; --line:#353539; --on:#a78bfa; --warn:#f59e0b; --ok:#4ade80; color-scheme: dark } }
* { box-sizing:border-box }
body { margin:0; background:var(--bg); color:var(--fg); font:14px/1.4 system-ui,-apple-system,"Segoe UI",sans-serif; height:100vh; display:flex; flex-direction:column }
header { padding:8px 12px; border-bottom:1px solid var(--line); background:var(--card); color:var(--mute); font-size:13px; display:flex; gap:12px; flex-wrap:wrap; align-items:baseline }
header .hint { margin-left:auto; font-size:12px }
.wrap { flex:1; display:flex; min-height:0 }
#stage { flex:1; position:relative; min-width:0; overflow:hidden }
#graph { position:absolute; inset:0; width:100%; height:100%; cursor:grab; touch-action:none }
#empty { padding:24px; color:var(--mute) }
#state { position:absolute; top:12px; left:12px; z-index:1; background:var(--card); border:1px solid var(--line); border-radius:12px; padding:14px 18px; max-width:calc(100% - 24px); box-shadow:0 4px 18px rgba(0,0,0,.12) }
#state .row { display:flex; gap:28px; flex-wrap:wrap }
#state .big { font-size:40px; font-weight:700; line-height:1.05; color:var(--warn); font-variant-numeric:tabular-nums }
#state .big.ok { color:var(--ok) }
#state .lbl { color:var(--mute); font-size:12px; margin-top:6px }
#state .stat { min-width:150px }
#detail { width:340px; border-left:1px solid var(--line); background:var(--card); overflow:auto; padding:14px 16px; font-size:13px }
#detail h2 { font-size:17px; margin:0 0 2px; word-break:break-all }
#detail .sub { color:var(--mute); font-size:13px }
#detail .url { color:var(--mute); word-break:break-all; font-family:ui-monospace,Menlo,monospace; font-size:11px; margin-top:2px }
#detail h3 { font-size:11px; text-transform:uppercase; letter-spacing:.05em; color:var(--mute); margin:18px 0 6px; font-weight:600 }
#detail img.page { display:block; width:100%; max-height:220px; object-fit:cover; object-position:top; border:1px solid var(--line); border-radius:8px; margin:10px 0 2px; background:#fff }
.chips { display:flex; flex-wrap:wrap; gap:6px }
.chip { font:12px ui-monospace,Menlo,monospace; color:var(--on); background:none; border:1px solid var(--line); border-radius:999px; padding:2px 9px; cursor:pointer }
.chip:hover { border-color:var(--on) }
.none { color:var(--mute); font-size:12px }
.pagelist { list-style:none; padding:0; margin:0 }
.pagelist li { border-bottom:1px solid var(--line) }
.pagelist button { font:12px ui-monospace,Menlo,monospace; color:var(--fg); background:none; border:0; padding:6px 0; width:100%; text-align:left; cursor:pointer; overflow-wrap:anywhere }
.pagelist button:hover { color:var(--on) }
.el { display:grid; grid-template-columns:1fr; gap:2px; padding:8px 0; border-bottom:1px solid var(--line) }
.el code { font:11px ui-monospace,Menlo,monospace; word-break:break-all }
.el small { color:var(--mute); font-size:11px }
.el img.pic { display:block; max-width:100%; max-height:72px; object-fit:contain; background:#fff; border:1px solid var(--line); border-radius:6px; margin-bottom:4px }
.tag { display:block; font-size:11px; color:var(--mute); margin-bottom:2px }
.tag.gap { color:var(--warn) }
.warnline { color:var(--warn); font-size:12px; margin-top:6px }
@media (max-width:700px) { .wrap { flex-direction:column } #detail { width:auto; border-left:0; border-top:1px solid var(--line); max-height:50vh } }
</style>
<header><span id="meta">Loading the site map…</span><span class="hint">drag a dot · scroll to zoom · drag the background to pan · click a page</span></header>
<div class="wrap">
  <div id="stage"><canvas id="graph" aria-label="Graph of pages and the links between them"></canvas><p id="empty" hidden></p><div id="state" hidden></div></div>
  <aside id="detail"></aside>
</div>
<script>
var canvas = document.getElementById("graph"), ctx = canvas.getContext("2d"), dpr = window.devicePixelRatio || 1;
var C = {}, G = null, running = false, REP = 1400, LEN = 120;
function el(tag, text, cls) { var e = document.createElement(tag); if (text != null) e.textContent = text; if (cls) e.className = cls; return e; }
function norm(u) { var x = new URL(u); x.hash = ""; return x.href.replace(/\/$/, ""); }
function label(e) { return e.name || e.placeholder || e.byLabel || e.testid || e.id || e.tag || "element"; }
function pathOf(u) { try { var x = new URL(u); return x.pathname + x.search || "/"; } catch (err) { return u; } }
function nodeR(n) { return 5 + Math.min(n.deg, 10) * 0.8; }
// mix blends two #rrggbb colours: t=0 is a, t=1 is b
function mix(a, b, t) {
  var out = [1, 3, 5].map(function (i) {
    var x = parseInt(a.substr(i, 2), 16), y = parseInt(b.substr(i, 2), 16);
    return Math.round(x + (y - x) * t);
  });
  return "rgb(" + out.join(",") + ")";
}
function readColors() { var s = getComputedStyle(document.documentElement); ["fg", "mute", "on", "warn", "ok"].forEach(function (k) { C[k] = s.getPropertyValue("--" + k).trim(); }); }

function resize() {
  var r = canvas.parentNode.getBoundingClientRect();
  canvas.width = r.width * dpr; canvas.height = r.height * dpr;
  canvas.style.width = r.width + "px"; canvas.style.height = r.height + "px";
  draw();
}

function build(data) {
  var pages = data.pages || [], byUrl = {}, seen = {}, edges = [];
  pages.forEach(function (p, i) {
    p.i = i; p.out = []; p.inn = []; p.elements = p.elements || []; byUrl[norm(p.url)] = p;
    var ang = i * 2.399, rad = 40 * Math.sqrt(i + 1); // a sunflower spread: a start that the force layout then settles
    p.node = { p: p, x: Math.cos(ang) * rad, y: Math.sin(ang) * rad, vx: 0, vy: 0, deg: 0, nbr: new Set() };
  });
  pages.forEach(function (p) {
    p.elements.forEach(function (e) {
      if (!e.href) return;
      var t; try { t = new URL(e.href, p.url); } catch (err) { return; }
      var q = byUrl[norm(t.href)];
      if (!q || q === p || seen[p.i + ">" + q.i]) return;
      seen[p.i + ">" + q.i] = true;
      p.out.push(q); q.inn.push(p); edges.push({ a: p.node, b: q.node });
      p.node.deg++; q.node.deg++; p.node.nbr.add(q.node); q.node.nbr.add(p.node);
    });
  });
  G = { pages: pages, byUrl: byUrl, nodes: pages.map(function (p) { return p.node; }), edges: edges, links: edges.length,
    s: 1, tx: canvas.clientWidth / 2, ty: canvas.clientHeight / 2, alpha: 1, sel: null, hover: null, drag: null, pan: null, moved: false,
    start: byUrl[norm(data.baseUrl || pages[0].url)] || pages[0] };
}

function tick() {
  var N = G.nodes, a = G.alpha, i, j, n;
  for (i = 0; i < N.length; i++) for (j = i + 1; j < N.length; j++) { // every pair pushes apart
    var A = N[i], B = N[j], dx = B.x - A.x, dy = B.y - A.y, d2 = dx * dx + dy * dy + 0.5, d = Math.sqrt(d2), f = REP * a / d2;
    A.vx -= dx / d * f; A.vy -= dy / d * f; B.vx += dx / d * f; B.vy += dy / d * f;
  }
  G.edges.forEach(function (e) { // a link pulls its two ends to about LEN apart
    var dx = e.b.x - e.a.x, dy = e.b.y - e.a.y, d = Math.sqrt(dx * dx + dy * dy) + 0.01, f = (d - LEN) * 0.04 * a;
    e.a.vx += dx / d * f; e.a.vy += dy / d * f; e.b.vx -= dx / d * f; e.b.vy -= dy / d * f;
  });
  for (n of N) {
    n.vx -= n.x * 0.012 * a; n.vy -= n.y * 0.012 * a; // a weak pull to the middle keeps the graph on screen
    if (n === G.start.node && n !== G.drag) { n.x = n.y = n.vx = n.vy = 0; continue; } // the root sits at the centre
    if (n === G.drag) { n.vx = n.vy = 0; continue; }
    n.vx *= 0.7; n.vy *= 0.7; n.x += n.vx; n.y += n.vy;
  }
  G.alpha *= 0.985;
}

function loop() {
  var active = G.alpha > 0.01 || G.drag;
  if (active) tick();
  draw();
  if (active) requestAnimationFrame(loop); else running = false;
}
function kick() { if (G.alpha < 0.3) G.alpha = 0.3; if (!running) { running = true; requestAnimationFrame(loop); } }

function draw() {
  if (!G) return;
  var cw = canvas.clientWidth, ch = canvas.clientHeight, s = G.s;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0); ctx.clearRect(0, 0, cw, ch);
  readColors();
  var focus = G.hover || G.sel, nb = focus ? focus.nbr : null;
  ctx.save(); ctx.translate(G.tx, G.ty); ctx.scale(s, s);
  G.edges.forEach(function (e) {
    var hot = focus && (e.a === focus || e.b === focus), known = e.cov !== undefined;
    ctx.globalAlpha = focus ? (hot ? 0.95 : 0.1) : (known ? 0.7 : 0.35);
    ctx.strokeStyle = known ? (e.cov ? C.ok : C.warn) : (hot ? C.on : C.mute); ctx.lineWidth = (hot ? 2.2 : 1.2) / s;
    ctx.setLineDash(e.flow ? [5 / s, 4 / s] : []); // a playlist's link is dashed
    ctx.beginPath(); ctx.moveTo(e.a.x, e.a.y); ctx.lineTo(e.b.x, e.b.y); ctx.stroke();
  });
  G.nodes.forEach(function (n) {
    var isF = n === focus, hot = nb && nb.has(n);
    ctx.globalAlpha = !focus || isF || hot ? 1 : 0.22;
    if (n.flow) { // a playlist is a square: green when it exercises every element of the pages it reaches
      var r = nodeR(n);
      ctx.fillStyle = n === G.sel ? C.on : (n.gap ? C.warn : C.ok);
      ctx.fillRect(n.x - r, n.y - r, 2 * r, 2 * r);
      return;
    }
    ctx.fillStyle = n === G.sel ? C.on : (isF ? C.fg : (n.share === undefined ? C.mute : mix(C.warn, C.ok, n.share)));
    ctx.beginPath(); ctx.arc(n.x, n.y, nodeR(n), 0, 2 * Math.PI); ctx.fill();
  });
  ctx.restore();
  ctx.font = "12px system-ui,sans-serif"; ctx.fillStyle = C.fg;
  G.nodes.forEach(function (n) { // labels stay in screen space, so they do not grow when you zoom
    var isF = n === focus, hot = nb && nb.has(n);
    if (!(s > 0.6 || n.flow || isF || hot || n === G.sel)) return;
    ctx.globalAlpha = !focus || isF || hot ? 1 : 0.3;
    ctx.fillText(n.flow ? "▸ " + n.flow.name : pathOf(n.p.url), n.x * s + G.tx + nodeR(n) * s + 5, n.y * s + G.ty + 4);
  });
  ctx.globalAlpha = 1;
}

function local(e) { var r = canvas.getBoundingClientRect(); return { x: e.clientX - r.left, y: e.clientY - r.top }; }
function nodeAt(mx, my) {
  var best = null, bd = Infinity;
  G.nodes.forEach(function (n) {
    var d = Math.hypot(n.x * G.s + G.tx - mx, n.y * G.s + G.ty - my);
    if (d <= nodeR(n) * G.s + 5 && d < bd) { bd = d; best = n; }
  });
  return best;
}

canvas.onpointerdown = function (e) {
  if (!G) return;
  var m = local(e), n = nodeAt(m.x, m.y);
  G.moved = false; G.down = m;
  if (n) G.drag = n; else G.pan = { x: m.x, y: m.y, tx: G.tx, ty: G.ty };
  canvas.setPointerCapture(e.pointerId); kick();
};
canvas.onpointermove = function (e) {
  if (!G) return;
  var m = local(e);
  if (G.drag) {
    if (Math.hypot(m.x - G.down.x, m.y - G.down.y) > 4) G.moved = true;
    G.drag.x = (m.x - G.tx) / G.s; G.drag.y = (m.y - G.ty) / G.s; kick();
  } else if (G.pan) {
    G.tx = G.pan.tx + m.x - G.pan.x; G.ty = G.pan.ty + m.y - G.pan.y; draw();
  } else {
    var h = nodeAt(m.x, m.y);
    canvas.style.cursor = h ? "pointer" : "grab";
    if (h !== G.hover) { G.hover = h; draw(); }
  }
};
canvas.onpointerup = function () {
  if (!G) return;
  if (G.drag && !G.moved) select(G.drag);
  G.drag = null; G.pan = null; kick();
};
canvas.onpointerleave = function () { if (G && G.hover && !G.drag) { G.hover = null; draw(); } };
canvas.addEventListener("wheel", function (e) {
  if (!G) return;
  e.preventDefault();
  var m = local(e), f = e.deltaY < 0 ? 1.12 : 1 / 1.12, ns = Math.min(3, Math.max(0.25, G.s * f));
  var wx = (m.x - G.tx) / G.s, wy = (m.y - G.ty) / G.s;
  G.s = ns; G.tx = m.x - wx * ns; G.ty = m.y - wy * ns; draw();
}, { passive: false });

function select(n) { G.sel = n; if (n && n.flow) showFlow(n); else showDetail(n ? n.p : null); draw(); }

function showFlow(n) {
  var box = document.getElementById("detail"), f = n.flow, pages = [];
  box.innerHTML = "";
  n.nbr.forEach(function (q) { pages.push(q.p); });
  pages.sort(function (a, b) { return a.url < b.url ? -1 : 1; });
  box.appendChild(el("h2", f.name));
  box.appendChild(el("div", "Playlist · " + f.tests + " tests", "sub"));
  box.appendChild(el("div", pages.length ? n.covered + " of " + n.elements + " elements exercised on the " + pages.length + " pages it reaches" : "Reaches no page on the map",
    n.gap ? "warnline" : "sub"));
  chips(box, "Pages it reaches", pages);
}

function chips(box, title, list) {
  box.appendChild(el("h3", title + " (" + list.length + ")"));
  var wrap = el("div", null, "chips");
  list.forEach(function (q) { var b = el("button", pathOf(q.url), "chip"); b.type = "button"; b.onclick = function () { select(q.node); }; wrap.appendChild(b); });
  if (!list.length) wrap.appendChild(el("span", "none", "none"));
  box.appendChild(wrap);
}
function showDetail(p) {
  var box = document.getElementById("detail");
  box.innerHTML = "";
  if (!p) { // no page picked: every page, so the map can also be used without the graph
    box.appendChild(el("h2", "All pages"));
    box.appendChild(el("div", G.pages.length + " pages, click one", "sub"));
    var ul = el("ul", null, "pagelist");
    G.pages.forEach(function (q) { var li = el("li"), b = el("button", pathOf(q.url)); b.type = "button"; b.onclick = function () { select(q.node); }; li.appendChild(b); ul.appendChild(li); });
    box.appendChild(ul);
    return;
  }
  box.appendChild(el("h2", pathOf(p.url)));
  box.appendChild(el("div", p.title || "(no title)", "sub"));
  box.appendChild(el("div", p.url, "url"));
  if (p.node.total !== undefined) box.appendChild(el("div", p.node.reached + " of " + p.node.total + " elements reached by a built test", p.node.gap ? "warnline" : "sub"));
  if (p.node.gap) box.appendChild(el("div", "Not reached by any built test", "warnline"));
  if (p.shot) { // the top of the page; the link opens the whole page
    var a = el("a"), pic = el("img", null, "page");
    a.href = "../shots/" + p.shot; a.target = "_blank"; a.rel = "noopener";
    pic.src = a.href; pic.alt = "Screenshot of " + pathOf(p.url); pic.onerror = function () { a.remove(); };
    a.appendChild(pic); box.appendChild(a);
  }
  chips(box, "Links to", p.out);
  chips(box, "Linked from", p.inn);
  box.appendChild(el("h3", "Elements (" + p.elements.length + ")"));
  p.elements.slice(0, 300).forEach(function (e) {
    var row = el("div", null, "el"), info = el("div");
    if (e.pic) { var im = el("img", null, "pic"); im.src = "../shots/" + e.pic; im.alt = label(e); im.loading = "lazy"; im.onerror = function () { im.remove(); }; info.appendChild(im); }
    info.appendChild(el("div", label(e)));
    var gap = !!(p.gaps && p.gaps.has(e.locator));
    info.appendChild(el("span", (e.role || e.tag || "") + (e.stable ? "" : " · fragile locator") + (gap ? " · not covered" : ""), gap ? "tag gap" : "tag"));
    info.appendChild(el("code", e.locator || "no locator"));
    row.appendChild(info); box.appendChild(row);
  });
  if (p.elements.length > 300) box.appendChild(el("p", "Showing 300 of " + p.elements.length + " elements.", "none"));
}

// applyCoverage marks the pages and elements that no built test reaches (out/coverage.json, written by poll)
function applyCoverage(cov) {
  var rec = {}, pages = (cov && cov.pages) || [];
  pages.forEach(function (r) { rec[norm(r.url)] = r; });
  G.gapPages = 0; G.gapEls = 0; G.totalEls = 0;
  G.pages.forEach(function (p) {
    var r = rec[norm(p.url)];
    G.totalEls += p.elements.length;
    if (!r) return;
    p.gaps = new Set(r.gaps || []);
    p.node.gap = !r.covered;
    p.node.reached = r.covered_elements; p.node.total = r.elements; // the dot's colour is the share of these reached
    p.node.share = r.elements ? r.covered_elements / r.elements : (r.covered ? 1 : 0);
    if (p.node.gap) G.gapPages++;
    G.gapEls += p.elements.filter(function (e) { return p.gaps.has(e.locator); }).length; // per element, as the total is
  });
  G.hasCoverage = pages.length > 0;
  // a link is covered when the page it leads to is reached by a built test
  G.edges.forEach(function (e) { e.cov = e.b.gap === undefined ? undefined : !e.b.gap; });
  addFlows(cov);
}

// addFlows adds each playlist as a square with a dashed link to every page it reaches. A link is green when the playlist
// exercises all of that page's elements, amber when some are left out; the square is green when all its links are.
function addFlows(cov) {
  G.flows = ((cov && cov.flows) || []).map(function (f, i) {
    var ang = i * 2.399 + 1, rad = 60 * Math.sqrt(i + 1);
    var n = { flow: f, x: Math.cos(ang) * rad, y: Math.sin(ang) * rad, vx: 0, vy: 0, deg: 0, nbr: new Set(), elements: 0, covered: 0 };
    (f.pages || []).forEach(function (r) {
      var q = G.byUrl[norm(r.url)];
      if (!q) return;
      n.elements += r.elements; n.covered += r.covered_elements;
      G.edges.push({ a: n, b: q.node, flow: true, cov: r.covered_elements === r.elements });
      n.deg++; q.node.deg++; n.nbr.add(q.node); q.node.nbr.add(n);
    });
    n.gap = n.nbr.size === 0 || n.covered < n.elements;
    return n;
  });
  G.nodes = G.nodes.concat(G.flows);
}

// renderState is the coverage card on the graph: the numbers that say how much of the app no built test reaches.
function renderState() {
  var box = document.getElementById("state");
  if (!G.hasCoverage) { box.hidden = true; return; }
  box.hidden = false; box.innerHTML = "";
  function stat(big, label) {
    var d = el("div", null, "stat");
    d.appendChild(el("div", big, "big" + (big.split(" / ")[0] === "0" ? " ok" : "")));
    d.appendChild(el("div", label, "lbl"));
    return d;
  }
  var row = el("div", null, "row");
  row.appendChild(stat(G.gapEls + " / " + G.totalEls, "elements not reached by a built test"));
  row.appendChild(stat(G.gapPages + " / " + G.pages.length, "pages not reached"));
  box.appendChild(row);
  if (G.flows.length) {
    var full = G.flows.filter(function (n) { return !n.gap; }).length;
    box.appendChild(el("div", full + " of " + G.flows.length + " playlists exercise every element they reach", "lbl"));
  }
  box.appendChild(el("div", "dots: amber to green by share of their elements reached · links: green to a page reached", "lbl"));
}

function start(data, cov) {
  var pages = data.pages || [];
  if (!pages.length) {
    document.getElementById("empty").textContent = "No crawl yet. Run hydra crawl, then this map fills in.";
    document.getElementById("empty").hidden = false;
    document.getElementById("meta").textContent = "Site map";
    return;
  }
  resize();
  build(data);
  applyCoverage(cov);
  var meta = document.getElementById("meta");
  meta.textContent = pages.length + " pages · " + G.links + " links" + (data.crawledAt ? " · crawled " + data.crawledAt : "");
  renderState();
  select(G.start.node);
  kick();
}
window.addEventListener("resize", resize);
Promise.all([
  fetch("../api/sitemap").then(function (r) { return r.json(); }),
  fetch("../api/coverage").then(function (r) { return r.json(); }).catch(function () { return { pages: [] }; })
]).then(function (v) { start(v[0], v[1]); }).catch(function () {
  document.getElementById("meta").textContent = "Could not load the site map.";
});
</script>
`
