package apitest

// reportTemplate renders a sequence diagram report as a single, self-contained HTML page. It loads nothing from
// the network: styles and scripts are inline, and the diagram is an inline SVG that works without JavaScript.
const reportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="generator" content="apitest">
<title>{{ .Title }}{{ if .SubTitle }} · {{ .SubTitle }}{{ end }}</title>
<style>
:root {
  color-scheme: light;
  --bg: #fbf9f5;
  --fg: #1f272e;
  --muted: #5d6874;
  --card: #ffffff;
  --card-2: #f6f2eb;
  --border: rgba(96, 112, 132, 0.22);
  --primary: #c8650c;
  --primary-soft: rgba(248, 176, 94, 0.18);
  --lifeline: rgba(96, 112, 132, 0.35);
  --ok: #15803d;
  --ok-soft: rgba(22, 163, 74, 0.12);
  --warn: #b45309;
  --warn-soft: rgba(217, 119, 6, 0.14);
  --error: #dc2626;
  --error-soft: rgba(220, 38, 38, 0.12);
  --j-key: #9a3412;
  --j-str: #0f766e;
  --j-num: #1d4ed8;
  --j-lit: #7c3aed;
  --shadow: 0 1px 2px rgba(31, 39, 46, 0.06), 0 8px 24px rgba(31, 39, 46, 0.06);
  --sans: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Inter, Roboto, "Helvetica Neue", Arial, sans-serif;
  --mono: ui-monospace, SFMono-Regular, "JetBrains Mono", Menlo, Consolas, "Liberation Mono", monospace;
}
:root[data-theme="dark"] {
  color-scheme: dark;
  --bg: #0e1216;
  --fg: #ebe7e0;
  --muted: #9aa5b1;
  --card: #141a20;
  --card-2: #1a2129;
  --border: rgba(126, 141, 160, 0.2);
  --primary: #f8b05e;
  --primary-soft: rgba(248, 176, 94, 0.14);
  --lifeline: rgba(126, 141, 160, 0.38);
  --ok: #4ade80;
  --ok-soft: rgba(74, 222, 128, 0.12);
  --warn: #fbbf24;
  --warn-soft: rgba(251, 191, 36, 0.12);
  --error: #f87171;
  --error-soft: rgba(248, 113, 113, 0.13);
  --j-key: #fdba74;
  --j-str: #5eead4;
  --j-num: #93c5fd;
  --j-lit: #c4b5fd;
  --shadow: 0 1px 2px rgba(0, 0, 0, 0.3), 0 8px 24px rgba(0, 0, 0, 0.25);
}
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    color-scheme: dark;
    --bg: #0e1216;
    --fg: #ebe7e0;
    --muted: #9aa5b1;
    --card: #141a20;
    --card-2: #1a2129;
    --border: rgba(126, 141, 160, 0.2);
    --primary: #f8b05e;
    --primary-soft: rgba(248, 176, 94, 0.14);
    --lifeline: rgba(126, 141, 160, 0.38);
    --ok: #4ade80;
    --ok-soft: rgba(74, 222, 128, 0.12);
    --warn: #fbbf24;
    --warn-soft: rgba(251, 191, 36, 0.12);
    --error: #f87171;
    --error-soft: rgba(248, 113, 113, 0.13);
    --j-key: #fdba74;
    --j-str: #5eead4;
    --j-num: #93c5fd;
    --j-lit: #c4b5fd;
    --shadow: 0 1px 2px rgba(0, 0, 0, 0.3), 0 8px 24px rgba(0, 0, 0, 0.25);
  }
}
* { box-sizing: border-box; }
[hidden] { display: none !important; }
html { -webkit-text-size-adjust: 100%; }
body { margin: 0; background: var(--bg); color: var(--fg); font: 14px/1.55 var(--sans); -webkit-font-smoothing: antialiased; }
code, pre, kbd { font-family: var(--mono); font-size: 12.5px; }
a { color: inherit; }
button { font: inherit; color: inherit; }
.wrap { max-width: 1440px; margin: 0 auto; padding: 0 24px; }

.topbar { position: sticky; top: 0; z-index: 10; background: color-mix(in srgb, var(--bg) 88%, transparent); backdrop-filter: blur(10px); border-bottom: 1px solid var(--border); }
.topbar .wrap { display: flex; align-items: center; gap: 12px; height: 52px; }
.brand { display: flex; align-items: center; gap: 9px; font-weight: 650; letter-spacing: -0.01em; text-decoration: none; }
.brand .mark { width: 18px; height: 18px; border-radius: 5px; background: conic-gradient(var(--fg) 25%, var(--primary) 0 50%, var(--fg) 0 75%, var(--primary) 0); background-size: 9px 9px; }
.brand small { color: var(--muted); font-weight: 500; }
.spacer { flex: 1; }
.btn { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--border); background: var(--card); border-radius: 9px; padding: 5px 10px; font-size: 12.5px; cursor: pointer; transition: border-color .15s, background .15s; }
.btn:hover { border-color: var(--primary); }
.btn:focus-visible, .event-head:focus-visible, .seq a:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px; }
.btn svg { width: 14px; height: 14px; }
.icon-btn { padding: 6px; }

.hero { padding: 28px 0 8px; }
.hero h1 { margin: 0; font: 650 22px/1.3 var(--mono); letter-spacing: -0.01em; word-break: break-all; }
.hero .sub { margin: 6px 0 0; color: var(--muted); font-size: 15px; }
.stats { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 16px; }
.chip { display: inline-flex; align-items: center; gap: 6px; border: 1px solid var(--border); background: var(--card); border-radius: 999px; padding: 3px 11px; font-size: 12.5px; color: var(--muted); }
.chip b { color: var(--fg); font-weight: 600; }
.status { display: inline-flex; align-items: center; gap: 6px; border-radius: 999px; padding: 3px 11px; font: 600 12.5px var(--sans); }
.status::before { content: ""; width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
.status.ok { color: var(--ok); background: var(--ok-soft); }
.status.warn { color: var(--warn); background: var(--warn-soft); }
.status.error { color: var(--error); background: var(--error-soft); }
.status.none { color: var(--muted); background: var(--card-2); }

.grid { display: grid; grid-template-columns: minmax(0, 1fr); gap: 20px; padding: 20px 0 40px; }
@media (min-width: 1100px) {
  .grid { grid-template-columns: minmax(0, 1.05fr) minmax(0, 1fr); align-items: start; }
  .diagram { position: sticky; top: 68px; max-height: calc(100vh - 88px); display: flex; flex-direction: column; }
  .diagram .scroll { overflow: auto; }
}
.panel { background: var(--card); border: 1px solid var(--border); border-radius: 16px; box-shadow: var(--shadow); min-width: 0; }
.panel-head { display: flex; align-items: center; gap: 10px; padding: 12px 16px; border-bottom: 1px solid var(--border); }
.panel-head h2 { margin: 0; font-size: 13px; font-weight: 650; letter-spacing: 0.02em; text-transform: uppercase; color: var(--muted); }
.legend { display: flex; gap: 14px; margin-left: auto; font-size: 12px; color: var(--muted); }
.legend span { display: inline-flex; align-items: center; gap: 6px; }
.legend i { display: inline-block; width: 22px; border-top: 2px solid var(--fg); }
.legend i.dashed { border-top-style: dashed; }
.scroll { overflow-x: auto; padding: 8px; }

.seq { display: block; margin: 0 auto; font-family: var(--mono); max-width: 100%; height: auto; }
.seq.natural { max-width: none; }
.seq .lifeline { stroke: var(--lifeline); stroke-width: 1.5; stroke-dasharray: 2 5; stroke-linecap: round; }
.seq .box { fill: var(--card-2); stroke: var(--border); stroke-width: 1.2; }
.seq .name { fill: var(--fg); font-size: 13px; font-weight: 600; }
.seq .participant { cursor: pointer; }
.seq .participant:hover .box, .seq .participant.focus .box { stroke: var(--primary); }
.seq .participant.focus .box { fill: var(--primary-soft); }
.seq .msg { color: var(--fg); text-decoration: none; cursor: pointer; }
.seq .msg.s-ok { color: var(--ok); }
.seq .msg.s-warn { color: var(--warn); }
.seq .msg.s-error { color: var(--error); }
.seq .msg.unanswered { color: var(--warn); }
.seq .no-reply circle { fill: var(--warn-soft); stroke: var(--warn); stroke-width: 1.2; }
.seq .no-reply path { stroke: var(--warn); stroke-width: 1.6; stroke-linecap: round; }
.seq .line { stroke: currentColor; stroke-width: 1.6; fill: none; }
.seq .response .line { stroke-dasharray: 6 5; }
.seq .head { fill: currentColor; stroke: currentColor; stroke-width: 1.2; stroke-linejoin: round; }
.seq .head.open { fill: none; stroke-width: 1.6; stroke-linecap: round; }
.seq .hit { stroke: transparent; stroke-width: 18; fill: none; }
.seq .label-bg { fill: var(--card); }
.seq .label { fill: var(--fg); font-size: 12px; }
.seq .label .num { fill: var(--primary); font-weight: 700; }
.seq .msg:hover .label-bg, .seq .msg.hover .label-bg { fill: var(--card-2); }
.seq .msg.active .label-bg { fill: var(--primary-soft); stroke: var(--primary); stroke-width: 1; }
.seq .msg.active .line { stroke-width: 2.4; }
.seq.filtering .msg:not(.match), .seq.filtering .participant:not(.match) { opacity: 0.18; }

.tools { display: flex; align-items: center; gap: 8px; margin-left: auto; }
.search { position: relative; }
.search input { width: 220px; max-width: 42vw; border: 1px solid var(--border); background: var(--bg); color: var(--fg); border-radius: 9px; padding: 5px 10px 5px 28px; font: 12.5px var(--sans); outline: none; }
.search input:focus { border-color: var(--primary); }
.search svg { position: absolute; left: 8px; top: 50%; width: 13px; height: 13px; transform: translateY(-50%); color: var(--muted); }

.events { list-style: none; margin: 0; padding: 8px; display: flex; flex-direction: column; gap: 8px; }
.event { border: 1px solid var(--border); border-radius: 12px; background: var(--card); scroll-margin-top: 76px; transition: border-color .15s, box-shadow .15s; }
.event.selected { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-soft); }
.event.hover:not(.selected) { border-color: color-mix(in srgb, var(--primary) 50%, var(--border)); }
.event[hidden] { display: none; }
.event-head { display: grid; grid-template-columns: auto auto minmax(0, 1fr) auto auto; align-items: center; gap: 10px; width: 100%; padding: 10px 12px; border: 0; background: none; text-align: left; cursor: pointer; border-radius: 12px; }
.badge { display: inline-flex; align-items: center; justify-content: center; min-width: 24px; height: 24px; padding: 0 6px; border-radius: 7px; background: var(--primary-soft); color: var(--primary); font: 700 12px var(--mono); }
.route { font-size: 12px; color: var(--muted); white-space: nowrap; }
.route b { color: var(--fg); font-weight: 600; }
.route .arrow { margin: 0 4px; }
.summary { font-family: var(--mono); font-size: 12.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.offset { font: 11.5px var(--mono); color: var(--muted); white-space: nowrap; }
.end { display: inline-flex; align-items: center; gap: 8px; }
.chev { width: 16px; height: 16px; color: var(--muted); transition: transform .2s; }
.event.collapsed .chev { transform: rotate(-90deg); }
.event.collapsed .event-body { display: none; }
.event-body { padding: 0 12px 12px; display: flex; flex-direction: column; gap: 10px; }
.start-line { font: 600 12.5px var(--mono); padding: 8px 10px; background: var(--card-2); border-radius: 8px; overflow-x: auto; white-space: pre; }
.headers { display: grid; grid-template-columns: max-content minmax(0, 1fr); gap: 2px 14px; margin: 0; font: 12.5px var(--mono); padding: 0 2px; }
.headers dt { color: var(--muted); }
.headers dd { margin: 0; overflow-wrap: anywhere; }
.code { position: relative; border: 1px solid var(--border); border-radius: 10px; background: var(--bg); overflow: hidden; }
.code-head { display: flex; align-items: center; gap: 8px; padding: 6px 8px 6px 12px; border-bottom: 1px solid var(--border); font-size: 11.5px; color: var(--muted); }
.code pre { margin: 0; padding: 10px 12px; overflow: auto; max-height: 420px; line-height: 1.55; }
.code.expanded pre { max-height: none; }
.j-k { color: var(--j-key); }
.j-s { color: var(--j-str); }
.j-n { color: var(--j-num); }
.j-b, .j-z { color: var(--j-lit); }
.j-p { color: var(--muted); }
.actions { display: flex; flex-wrap: wrap; gap: 6px; }
.btn.copied { border-color: var(--ok); color: var(--ok); }
.empty { padding: 28px; text-align: center; color: var(--muted); }

.meta { display: flex; flex-wrap: wrap; gap: 6px; }
footer { border-top: 1px solid var(--border); color: var(--muted); font-size: 12.5px; }
footer .wrap { display: flex; flex-wrap: wrap; gap: 8px 16px; justify-content: space-between; padding-top: 16px; padding-bottom: 24px; }
footer a { color: var(--primary); text-decoration: none; }
kbd { border: 1px solid var(--border); border-bottom-width: 2px; border-radius: 5px; padding: 0 5px; font-size: 11px; background: var(--card); }
@media (max-width: 640px) {
  .wrap { padding: 0 14px; }
  .event-head { grid-template-columns: auto minmax(0, 1fr) auto auto; }
  .route, .offset { display: none; }
  .event-head { gap: 8px; }
  .search input { width: 150px; }
  .legend { display: none; }
}
@media print {
  .topbar, .tools, .actions, .chev { display: none !important; }
  body { background: #fff; }
  .panel, .event { box-shadow: none; }
  .event.collapsed .event-body { display: flex; }
  .code pre { max-height: none; }
  .diagram { position: static; max-height: none; }
}
</style>
<script>
(function () {
  try {
    var theme = localStorage.getItem("apitest-report-theme");
    if (theme === "light" || theme === "dark") document.documentElement.setAttribute("data-theme", theme);
  } catch (e) {}
})();
</script>
</head>
<body>
<!-- Generated by apitest. https://apitest.dev -->
<header class="topbar">
  <div class="wrap">
    <a class="brand" href="https://apitest.dev" target="_blank" rel="noopener"><span class="mark" aria-hidden="true"></span>apitest <small>report</small></a>
    <span class="spacer"></span>
    <button class="btn" type="button" id="toggle-all" title="Expand or collapse every event">Collapse all</button>
    <button class="btn icon-btn" type="button" id="theme" title="Toggle dark mode" aria-label="Toggle dark mode">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/></svg>
    </button>
  </div>
</header>

<div class="wrap">
  <section class="hero">
    <h1>{{ .Title }}</h1>
    {{ if .SubTitle }}<p class="sub">{{ .SubTitle }}</p>{{ end }}
    <div class="stats">
      {{ if gt .StatusCode 0 }}<span class="status {{ .StatusClass }}">{{ .StatusCode }} {{ .StatusText }}</span>{{ end }}
      {{ if .Duration }}<span class="chip">Duration <b>{{ .Duration }}</b></span>{{ end }}
      <span class="chip">Events <b>{{ len .Events }}</b></span>
      <span class="chip">Participants <b>{{ len .Participants }}</b></span>
      {{ if .StartedAt }}<span class="chip">{{ .StartedAt }}</span>{{ end }}
      {{ range .Meta }}<span class="chip">{{ .Key }} <b>{{ .Value }}</b></span>{{ end }}
    </div>
  </section>

  <main class="grid">
    <section class="panel diagram" aria-labelledby="diagram-title">
      <div class="panel-head">
        <h2 id="diagram-title">Sequence</h2>
        <div class="legend"><span><i></i>request</span><span><i class="dashed"></i>response</span></div>
      </div>
      <div class="scroll">{{ .Diagram }}</div>
    </section>

    <section class="panel" aria-labelledby="events-title">
      <div class="panel-head">
        <h2 id="events-title">Events</h2>
        <div class="tools">
          <label class="search">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>
            <input type="search" id="filter" placeholder="Filter events" aria-label="Filter events" autocomplete="off">
          </label>
        </div>
      </div>
      <ol class="events" id="events">
        {{ range .Events }}
        <li class="event {{ .Kind }}{{ if .StatusClass }} s-{{ .StatusClass }}{{ end }}" id="event-{{ .Number }}" data-event="{{ .Number }}" data-source="{{ .Source }}" data-target="{{ .Target }}">
          <button class="event-head" type="button" aria-expanded="true" aria-controls="event-body-{{ .Number }}">
            <span class="badge">{{ .Number }}</span>
            <span class="route"><b>{{ .Source }}</b><span class="arrow">{{ if eq .Kind "request" }}→{{ else }}⇢{{ end }}</span><b>{{ .Target }}</b></span>
            <span class="summary" title="{{ .Label }}">{{ .Label }}</span>
            <span class="end">{{ if .StatusCode }}<span class="status {{ .StatusClass }}">{{ .StatusCode }}</span>{{ end }}{{ if .Unanswered }}<span class="status warn" title="No response was recorded. For a mocked call, no mock matched the request.">no response</span>{{ end }}<span class="offset">{{ .Offset }}</span></span>
            <svg class="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6"/></svg>
          </button>
          <div class="event-body" id="event-body-{{ .Number }}">
            <div class="start-line">{{ .StartLine }}</div>
            {{ if .Headers }}<dl class="headers">{{ range .Headers }}<dt>{{ .Name }}</dt><dd>{{ .Value }}</dd>{{ end }}</dl>{{ end }}
            {{ if .Body }}
            <div class="code">
              <div class="code-head"><span>{{ if eq .BodyKind "json" }}JSON{{ else }}Body{{ end }} · {{ .BodySize }}</span><span class="spacer"></span><button class="btn" type="button" data-expand hidden>Show all</button><button class="btn" type="button" data-copy="{{ .Body }}">Copy</button></div>
              <pre><code>{{ .BodyHTML }}</code></pre>
            </div>
            {{ end }}
            <div class="actions">
              <button class="btn" type="button" data-copy="{{ .Raw }}">Copy raw</button>
              {{ if .Curl }}<button class="btn" type="button" data-copy="{{ .Curl }}">Copy as curl</button>{{ end }}
            </div>
          </div>
        </li>
        {{ end }}
      </ol>
      <p class="empty" id="no-match" hidden>No events match the filter.</p>
    </section>
  </main>
</div>

<footer>
  <div class="wrap">
    <span>Generated by <a href="https://apitest.dev" target="_blank" rel="noopener">apitest</a></span>
    <span><kbd>j</kbd> <kbd>k</kbd> move between events · <kbd>/</kbd> filter · click a participant to focus on it</span>
  </div>
</footer>

{{ if .MetaJSON }}<script type="application/json" id="metaJson">{{ .MetaJSON }}</script>{{ end }}
<script>
(function () {
  var root = document.documentElement;
  var svg = document.querySelector(".seq");
  var events = Array.prototype.slice.call(document.querySelectorAll(".event"));
  var arrows = svg ? Array.prototype.slice.call(svg.querySelectorAll(".msg")) : [];
  var participants = svg ? Array.prototype.slice.call(svg.querySelectorAll(".participant")) : [];
  var filter = document.getElementById("filter");
  var noMatch = document.getElementById("no-match");
  var focused = null;
  var selected = null;

  function arrowFor(n) { return svg ? svg.querySelector('.msg[data-event="' + n + '"]') : null; }
  function eventFor(n) { return document.getElementById("event-" + n); }

  function setExpanded(el, open) {
    el.classList.toggle("collapsed", !open);
    el.querySelector(".event-head").setAttribute("aria-expanded", open ? "true" : "false");
  }

  function select(n, scroll) {
    var el = eventFor(n);
    if (!el) return;
    events.forEach(function (e) { e.classList.remove("selected"); });
    arrows.forEach(function (a) { a.classList.remove("active"); });
    el.classList.add("selected");
    setExpanded(el, true);
    var arrow = arrowFor(n);
    if (arrow) arrow.classList.add("active");
    selected = n;
    if (scroll) el.scrollIntoView({ block: "nearest", behavior: "smooth" });
    if (history.replaceState) history.replaceState(null, "", "#event-" + n);
  }

  arrows.forEach(function (a) {
    var n = a.getAttribute("data-event");
    a.addEventListener("click", function (ev) { ev.preventDefault(); select(n, true); });
    a.addEventListener("mouseenter", function () { var e = eventFor(n); if (e) e.classList.add("hover"); });
    a.addEventListener("mouseleave", function () { var e = eventFor(n); if (e) e.classList.remove("hover"); });
  });

  events.forEach(function (el) {
    var n = el.getAttribute("data-event");
    el.querySelector(".event-head").addEventListener("click", function () {
      if (selected === n) { setExpanded(el, el.classList.contains("collapsed")); return; }
      select(n, false);
    });
    el.addEventListener("mouseenter", function () { var a = arrowFor(n); if (a) a.classList.add("hover"); });
    el.addEventListener("mouseleave", function () { var a = arrowFor(n); if (a) a.classList.remove("hover"); });
  });

  participants.forEach(function (p) {
    p.addEventListener("click", function () {
      var name = p.getAttribute("data-participant");
      focused = focused === name ? null : name;
      applyFilter();
    });
  });

  function applyFilter() {
    var q = filter ? filter.value.trim().toLowerCase() : "";
    var visible = 0;
    events.forEach(function (el) {
      var src = el.getAttribute("data-source"), dst = el.getAttribute("data-target");
      var show = (!focused || src === focused || dst === focused) && (!q || el.textContent.toLowerCase().indexOf(q) >= 0);
      el.hidden = !show;
      var a = arrowFor(el.getAttribute("data-event"));
      if (a) a.classList.toggle("match", show);
      if (show) visible++;
    });
    participants.forEach(function (p) {
      var name = p.getAttribute("data-participant");
      p.classList.toggle("focus", name === focused);
      p.classList.toggle("match", !focused || name === focused || arrows.some(function (a) {
        return a.classList.contains("match") && (a.getAttribute("data-source") === name || a.getAttribute("data-target") === name);
      }));
    });
    if (svg) svg.classList.toggle("filtering", !!(q || focused));
    if (noMatch) noMatch.hidden = visible > 0;
  }
  if (filter) filter.addEventListener("input", applyFilter);

  document.addEventListener("keydown", function (ev) {
    var tag = (ev.target && ev.target.tagName) || "";
    if (tag === "INPUT" || tag === "TEXTAREA" || ev.metaKey || ev.ctrlKey || ev.altKey) {
      if (ev.key === "Escape" && ev.target === filter) { filter.value = ""; applyFilter(); filter.blur(); }
      return;
    }
    if (ev.key === "/") { ev.preventDefault(); if (filter) filter.focus(); return; }
    if (ev.key === "Escape") { focused = null; if (filter) filter.value = ""; applyFilter(); return; }
    var shown = events.filter(function (e) { return !e.hidden; });
    if (!shown.length) return;
    var idx = shown.findIndex(function (e) { return e.getAttribute("data-event") === selected; });
    if (ev.key === "j" || ev.key === "ArrowDown") { ev.preventDefault(); select(shown[Math.min(idx + 1, shown.length - 1)].getAttribute("data-event"), true); }
    if (ev.key === "k" || ev.key === "ArrowUp") { ev.preventDefault(); select(shown[Math.max(idx - 1, 0)].getAttribute("data-event"), true); }
  });

  var toggleAll = document.getElementById("toggle-all");
  if (toggleAll) toggleAll.addEventListener("click", function () {
    var collapse = toggleAll.textContent === "Collapse all";
    events.forEach(function (el) { setExpanded(el, !collapse); });
    toggleAll.textContent = collapse ? "Expand all" : "Collapse all";
  });

  var themeBtn = document.getElementById("theme");
  if (themeBtn) themeBtn.addEventListener("click", function () {
    var dark = root.getAttribute("data-theme") ? root.getAttribute("data-theme") === "dark" : matchMedia("(prefers-color-scheme: dark)").matches;
    var next = dark ? "light" : "dark";
    root.setAttribute("data-theme", next);
    try { localStorage.setItem("apitest-report-theme", next); } catch (e) {}
  });

  function copy(text, btn) {
    function done() {
      var label = btn.textContent;
      btn.classList.add("copied");
      btn.textContent = "Copied";
      setTimeout(function () { btn.classList.remove("copied"); btn.textContent = label; }, 1200);
    }
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).then(done, function () { fallback(text); done(); });
    } else {
      fallback(text);
      done();
    }
  }
  function fallback(text) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand("copy"); } catch (e) {}
    document.body.removeChild(ta);
  }
  document.querySelectorAll("[data-copy]").forEach(function (btn) {
    btn.addEventListener("click", function () { copy(btn.getAttribute("data-copy"), btn); });
  });

  document.querySelectorAll(".code").forEach(function (block) {
    var pre = block.querySelector("pre");
    var btn = block.querySelector("[data-expand]");
    if (!pre || !btn || pre.scrollHeight <= pre.clientHeight + 4) return;
    btn.hidden = false;
    btn.addEventListener("click", function () {
      var open = block.classList.toggle("expanded");
      btn.textContent = open ? "Show less" : "Show all";
    });
  });

  // scale a wide diagram down to fit its panel, but scroll instead once the text would get too small
  function fitDiagram() {
    if (!svg) return;
    var natural = parseFloat(svg.getAttribute("width"));
    var available = svg.parentNode.clientWidth - 16;
    svg.classList.toggle("natural", natural > 0 && available / natural < 0.72);
  }
  fitDiagram();
  window.addEventListener("resize", fitDiagram);

  var initial = location.hash.match(/^#event-(\d+)$/);
  if (initial) select(initial[1], true);
})();
</script>
</body>
</html>`
