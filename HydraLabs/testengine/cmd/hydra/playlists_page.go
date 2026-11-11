package main

// playlistsPage is the board's Playlists tile: the flows in hydra/playlists, each with Check and Run. It talks to the
// board's own /api/run and /api/playlists, and shows the output of the last action.
const playlistsPage = `<!doctype html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Playlists</title>
<style>
:root { --bg:#fff; --fg:#1d1f23; --muted:#5d636d; --line:#d9dbe0; --accent:#1f6feb; --code:#f2f3f5; color-scheme: light }
@media (prefers-color-scheme: dark) { :root { --bg:#15171b; --fg:#e8eaed; --muted:#9aa1ab; --line:#2c3038; --accent:#6aa5ff; --code:#1d2026; color-scheme: dark } }
body { margin:0; background:var(--bg); color:var(--fg); font:14px/1.45 system-ui,-apple-system,"Segoe UI",sans-serif; padding:12px 16px }
.row { display:flex; flex-wrap:wrap; align-items:center; gap:8px; padding:8px 0; border-bottom:1px solid var(--line); min-width:0 }
.name { font-family:ui-monospace,Menlo,monospace; flex:1 1 180px; overflow-wrap:anywhere }
button { font:inherit; color:var(--accent); background:transparent; border:1px solid var(--line); border-radius:6px; padding:4px 10px; cursor:pointer }
button:disabled { color:var(--muted); cursor:default }
button:focus-visible { outline:2px solid var(--accent); outline-offset:2px }
p.muted { color:var(--muted); margin:8px 0 }
pre { background:var(--code); border-radius:6px; padding:8px 10px; white-space:pre-wrap; overflow-wrap:anywhere; margin:12px 0 0; max-height:40vh; overflow:auto; font:12px/1.5 ui-monospace,Menlo,monospace }
h3 { font-size:14px; margin:20px 0 8px }
.build { display:flex; flex-wrap:wrap; gap:8px; align-items:center }
input, select { font:inherit; color:var(--fg); background:var(--bg); border:1px solid var(--line); border-radius:6px; padding:4px 8px; min-width:0 }
input { flex:1 1 180px }
select { flex:1 1 220px }
ol { margin:8px 0; padding-left:0; list-style:none }
ol li { display:flex; align-items:center; gap:6px; padding:4px 0; border-bottom:1px solid var(--line) }
ol li span { flex:1; font-family:ui-monospace,Menlo,monospace; overflow-wrap:anywhere }
</style>
<p class="muted">Flows in hydra/playlists of this app. Check looks at the flow without opening a browser. Run executes it in order against the app.</p>
<div id="list"></div>
<h3>New flow</h3>
<div class="build">
  <input id="pname" placeholder="name, e.g. order-lifecycle" autocomplete="off">
  <select id="pcase"></select>
  <button id="padd" type="button">Add step</button>
</div>
<ol id="steps"></ol>
<button id="psave" type="button">Save flow</button>
<p class="muted">Saving writes hydra/playlists/&lt;name&gt;.json and commits it on the test repo's base branch. Captures and value passing are added by hand in the file.</p>
<pre id="out" hidden></pre>
<script>
var list = document.getElementById("list"), out = document.getElementById("out");
var pname = document.getElementById("pname"), pcase = document.getElementById("pcase");
var stepsEl = document.getElementById("steps"), steps = [], cases = {};
function show(s) {
  out.hidden = false;
  out.textContent = (s.label ? s.label + "\n" : "") + (s.log || "") +
    (s.running ? "\n(running " + s.seconds + "s)" : (s.exit ? "\nexit: " + s.exit : ""));
}
function poll() {
  fetch("../api/run").then(function (r) { return r.json(); }).then(function (s) {
    if (s.running || s.log) { show(s); }
    if (s.running) { setTimeout(poll, 1000); } else { enable(true); }
  }).catch(function () { enable(true); });
}
function enable(on) {
  Array.prototype.forEach.call(list.querySelectorAll("button"), function (b) { b.disabled = !on; });
}
function act(action, name) {
  enable(false);
  fetch("../api/run", { method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ Action: action, Key: name }) }).then(function (r) {
    if (!r.ok) { return r.json().then(function (e) { out.hidden = false; out.textContent = e.error; enable(true); }); }
    poll();
  });
}
function renderList(names) {
  list.innerHTML = "";
  if (!names || names.length === 0) {
    list.innerHTML = '<p class="muted">No flows yet. Build one below.</p>';
    return;
  }
  names.forEach(function (n) {
    var row = document.createElement("div");
    row.className = "row";
    var name = document.createElement("span");
    name.className = "name";
    name.textContent = n;
    row.appendChild(name);
    [["Check", "playlist-check"], ["Run", "playlist-run"]].forEach(function (b) {
      var btn = document.createElement("button");
      btn.type = "button";
      btn.textContent = b[0];
      btn.onclick = function () { act(b[1], n); };
      row.appendChild(btn);
    });
    list.appendChild(row);
  });
}
function loadFlows() {
  return fetch("../api/playlists").then(function (r) { return r.json(); }).then(renderList)
    .catch(function () { list.textContent = "Could not load the flows."; });
}
function drawSteps() {
  stepsEl.innerHTML = "";
  steps.forEach(function (id, i) {
    var li = document.createElement("li"), label = document.createElement("span");
    label.textContent = id + (cases[id] ? "  " + cases[id] : "");
    li.appendChild(label);
    [["↑", -1], ["↓", 1], ["✕", 0]].forEach(function (b) {
      var btn = document.createElement("button");
      btn.type = "button";
      btn.textContent = b[0];
      btn.setAttribute("aria-label", b[1] === 0 ? "remove step" : b[1] < 0 ? "move step up" : "move step down");
      btn.onclick = function () {
        if (b[1] === 0) { steps.splice(i, 1); }
        else if (i + b[1] >= 0 && i + b[1] < steps.length) {
          var t = steps[i]; steps[i] = steps[i + b[1]]; steps[i + b[1]] = t;
        }
        drawSteps();
      };
      li.appendChild(btn);
    });
    stepsEl.appendChild(li);
  });
}
document.getElementById("padd").onclick = function () {
  if (pcase.value) { steps.push(pcase.value); drawSteps(); }
};
document.getElementById("psave").onclick = function () {
  fetch("../api/playlists", { method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ Name: pname.value.trim(), Steps: steps }) }).then(function (r) {
    return r.json().then(function (j) { return { ok: r.ok, j: j }; });
  }).then(function (x) {
    out.hidden = false;
    if (!x.ok) { out.textContent = x.j.error; return; }
    out.textContent = "Saved " + pname.value.trim() + " (hydra/playlists) and committed it.";
    pname.value = ""; steps = []; drawSteps();
    renderList(x.j);
  }).catch(function () { out.hidden = false; out.textContent = "Could not save the flow."; });
};
fetch("../api/playlist-cases").then(function (r) { return r.json(); }).then(function (cs) {
  cs.forEach(function (c) {
    cases[c.id] = c.name;
    var o = document.createElement("option");
    o.value = c.id;
    o.textContent = c.id + "  " + c.name;
    pcase.appendChild(o);
  });
  if (cs.length === 0) { pcase.innerHTML = '<option value="">no cases yet</option>'; }
}).catch(function () { pcase.innerHTML = '<option value="">could not load cases</option>'; });
loadFlows().then(poll);
</script>
`
