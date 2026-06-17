package handoff

import (
	"html/template"
	"io"
	"strings"
	"time"
)

// boardMessage / boardThread / boardView are the template view-model. They
// carry display-formatted timestamps and a precomputed Own flag (true when a
// message is from the thread's original asker) so the templates stay logic-free.
type boardMessage struct {
	From    string
	Body    string
	At      string
	Own     bool
	Clipped bool // Body was truncated for the overview; full text lives on the detail page
}

type boardThread struct {
	ID       string
	From     string
	To       string
	Subject  string
	Status   string
	Updated  string
	Count    int // total messages in the thread
	More     int // messages hidden before the previewed tail (overview only)
	Messages []boardMessage
}

type boardView struct {
	Threads []boardThread
	Now     string
	Variant string
	Count   int
}

// threadView is the view-model for the single-thread detail page, which shows
// every message in full (no clipping, no auto-refresh).
type threadView struct {
	T       boardThread
	Now     string
	Variant string
}

const (
	previewCount = 3   // newest messages shown on a board card
	previewChars = 200 // max characters of each previewed message body
)

// normalizeVariant clamps the requested style to a known template, defaulting
// to the mission-control board.
func normalizeVariant(v string) string {
	switch v {
	case "terminal":
		return v
	default:
		return "mission"
	}
}

func fmtClock(s, layout string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format(layout)
}

// clipBody truncates s to at most n runes, returning the (possibly shortened)
// text and whether it was clipped.
func clipBody(s string, n int) (string, bool) {
	r := []rune(s)
	if len(r) <= n {
		return s, false
	}
	return strings.TrimRight(string(r[:n]), " \t\r\n") + "…", true
}

func toBoardMessage(m Message, asker string, preview bool) boardMessage {
	body, clipped := m.Body, false
	if preview {
		body, clipped = clipBody(body, previewChars)
	}
	return boardMessage{
		From:    m.From,
		Body:    body,
		At:      fmtClock(m.At, "15:04:05"),
		Own:     m.From == asker,
		Clipped: clipped,
	}
}

func boardThreadOf(t Thread) boardThread {
	return boardThread{
		ID:      t.ID,
		From:    t.From,
		To:      t.To,
		Subject: t.Subject,
		Status:  t.Status,
		Updated: fmtClock(t.UpdatedAt, "Jan 2 · 15:04"),
		Count:   len(t.Messages),
	}
}

// RenderBoard writes the HTML overview (newest thread first) in the chosen
// visual variant: "mission" (default) or "terminal". Each card shows only the
// most recent messages, clipped, and links to the full-thread detail page.
func RenderBoard(w io.Writer, threads []Thread, variant string) error {
	variant = normalizeVariant(variant)
	view := boardView{
		Now:     time.Now().Format("Mon, Jan 2 · 15:04"),
		Variant: variant,
		Count:   len(threads),
	}
	for i := len(threads) - 1; i >= 0; i-- {
		t := threads[i]
		bt := boardThreadOf(t)
		start := 0
		if len(t.Messages) > previewCount {
			start = len(t.Messages) - previewCount
		}
		bt.More = start
		for _, m := range t.Messages[start:] {
			bt.Messages = append(bt.Messages, toBoardMessage(m, t.From, true))
		}
		view.Threads = append(view.Threads, bt)
	}
	return boardTmpl.ExecuteTemplate(w, variant, view)
}

// RenderThread writes the single-thread detail page (every message, full text)
// in the chosen visual variant.
func RenderThread(w io.Writer, t Thread, variant string) error {
	variant = normalizeVariant(variant)
	bt := boardThreadOf(t)
	for _, m := range t.Messages {
		bt.Messages = append(bt.Messages, toBoardMessage(m, t.From, false))
	}
	view := threadView{
		T:       bt,
		Now:     time.Now().Format("Mon, Jan 2 · 15:04"),
		Variant: variant,
	}
	return boardTmpl.ExecuteTemplate(w, variant+"-thread", view)
}

var boardTmpl = func() *template.Template {
	t := template.Must(template.New("board").Parse(switcherTmpl))
	template.Must(t.New("missionStyle").Parse(missionStyle))
	template.Must(t.New("terminalStyle").Parse(terminalStyle))
	template.Must(t.New("mission").Parse(missionTmpl))
	template.Must(t.New("mission-thread").Parse(missionThreadTmpl))
	template.Must(t.New("terminal").Parse(terminalTmpl))
	template.Must(t.New("terminal-thread").Parse(terminalThreadTmpl))
	return t
}()

const switcherTmpl = `{{define "switcher"}}
<nav class="switcher">
  <span class="sw-label">theme</span>
  <a href="?v=mission"{{if eq .Variant "mission"}} class="on"{{end}}>mission control</a>
  <a href="?v=terminal"{{if eq .Variant "terminal"}} class="on"{{end}}>terminal</a>
</nav>
{{end}}`

// ---- mission control -------------------------------------------------------

const missionStyle = `
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Chakra+Petch:wght@500;600;700&family=IBM+Plex+Mono:wght@400;500;600&display=swap" rel="stylesheet">
<style>
:root{
  --bg:#070b14; --panel:#0d1524; --panel2:#0b1220; --line:#1a2740;
  --ink:#dce6f5; --muted:#8a9bbd; --faint:#566889;
  --open:#f5c451; --answered:#46e0a0; --cyan:#5cc8ff;
  --disp:"Chakra Petch", ui-sans-serif, system-ui, sans-serif;
  --mono:"IBM Plex Mono", ui-monospace, Menlo, monospace;
}
*{box-sizing:border-box}
html{color-scheme:dark}
body{
  margin:0; background:var(--bg); color:var(--ink); font-family:var(--mono);
  line-height:1.55; min-height:100dvh; padding-bottom:48px;
  background-image:
    radial-gradient(900px 520px at 12% -12%, rgba(92,200,255,.10), transparent 60%),
    radial-gradient(820px 540px at 106% -2%, rgba(70,224,160,.08), transparent 55%),
    linear-gradient(transparent 95%, rgba(255,255,255,.022) 0),
    linear-gradient(90deg, transparent 95%, rgba(255,255,255,.022) 0);
  background-size:auto, auto, 27px 27px, 27px 27px;
  background-repeat:no-repeat, no-repeat, repeat, repeat;
  background-attachment:fixed;
}
.wrap{max-width:860px; margin:0 auto; padding:34px 22px 0}
.top{padding-bottom:18px; border-bottom:1px solid var(--line); margin-bottom:26px}
.brand{font-family:var(--disp); font-weight:700; font-size:20px; letter-spacing:.18em; display:flex; align-items:center; gap:11px}
.brand .logo{color:var(--cyan)}
.top .meta{font-size:12px; color:var(--faint); display:flex; align-items:center; gap:9px; letter-spacing:.04em; margin-top:8px}
.pulse{width:8px; height:8px; border-radius:50%; background:var(--answered); animation:pulse 2.4s infinite}
@keyframes pulse{
  0%{box-shadow:0 0 0 0 rgba(70,224,160,.5)}
  70%{box-shadow:0 0 0 7px rgba(70,224,160,0)}
  100%{box-shadow:0 0 0 0 rgba(70,224,160,0)}
}
.thread{position:relative; display:flex; width:100%; background:linear-gradient(180deg,var(--panel),var(--panel2)); border:1px solid var(--line); border-radius:12px; margin-bottom:16px; overflow:hidden}
.thread-bar{flex:0 0 3px; background:var(--faint)}
.thread.open .thread-bar{background:var(--open); box-shadow:0 0 16px rgba(245,196,81,.55)}
.thread.answered .thread-bar{background:var(--answered); box-shadow:0 0 16px rgba(70,224,160,.45)}
.thread-main{flex:1; padding:16px 18px; min-width:0}
.thead{display:flex; align-items:flex-start; justify-content:space-between; gap:12px}
.subject{font-family:var(--disp); font-weight:600; font-size:16px; margin:0; color:#eef4ff}
.status{font-size:11px; text-transform:uppercase; letter-spacing:.15em; display:inline-flex; align-items:center; gap:7px; white-space:nowrap; padding-top:3px}
.status .led{width:8px; height:8px; border-radius:50%}
.thread.open .status{color:var(--open)}
.thread.open .led{background:var(--open); box-shadow:0 0 10px var(--open)}
.thread.answered .status{color:var(--answered)}
.thread.answered .led{background:var(--answered); box-shadow:0 0 10px var(--answered)}
.sub{display:flex; flex-wrap:wrap; align-items:center; gap:8px; margin-top:9px; font-size:11.5px; color:var(--muted)}
.sub .route{color:var(--cyan)}
.sub .id{color:var(--faint)}
.sub .upd{margin-left:auto; color:var(--faint)}
.dot{color:var(--line)}
.msgs{list-style:none; margin:14px 0 0; padding:14px 0 0; border-top:1px dashed var(--line)}
.more{color:var(--faint); font-size:11.5px; font-style:italic; padding-bottom:9px}
.msg{padding:7px 0}
.msg .who{font-weight:600; font-size:12px}
.msg.asker .who{color:var(--cyan)}
.msg.responder .who{color:var(--answered)}
.msg .at{color:var(--faint); font-size:10.5px; margin-left:9px}
.msg .body{margin:5px 0 0; white-space:pre-wrap; overflow-wrap:anywhere; color:var(--ink); font-size:13px}
.msg .body.clip{display:-webkit-box; -webkit-line-clamp:3; -webkit-box-orient:vertical; overflow:hidden}
.open{display:inline-flex; align-items:center; gap:6px; margin-top:14px; font-family:var(--disp); font-size:11.5px; letter-spacing:.08em; text-transform:uppercase; color:var(--cyan); text-decoration:none; border:1px solid var(--line); border-radius:8px; padding:7px 13px; transition:.15s}
.open:hover{background:rgba(92,200,255,.10); border-color:var(--cyan)}
.empty{text-align:center; color:var(--faint); padding:90px 0; letter-spacing:.1em}
/* detail */
.detail{padding-top:26px}
.dtop{display:flex; align-items:center; gap:18px; font-size:12px; letter-spacing:.04em; margin-bottom:18px}
.dtop a{color:var(--muted); text-decoration:none}
.dtop a:hover{color:var(--cyan)}
.dhead{padding-bottom:18px; border-bottom:1px solid var(--line); margin-bottom:24px}
.dsubject{font-family:var(--disp); font-weight:700; font-size:25px; line-height:1.2; margin:0 0 11px; color:#eef4ff}
.dsub{display:flex; flex-wrap:wrap; align-items:center; gap:8px; font-size:12px; color:var(--muted)}
.dsub .route{color:var(--cyan)}
.dsub .id{color:var(--faint)}
.dsub .status{padding-top:0}
.dsub .led{width:8px; height:8px; border-radius:50%}
.dhead.open .status{color:var(--open)}
.dhead.open .led{background:var(--open); box-shadow:0 0 10px var(--open)}
.dhead.answered .status{color:var(--answered)}
.dhead.answered .led{background:var(--answered); box-shadow:0 0 10px var(--answered)}
.dmsgs{list-style:none; margin:0; padding:0}
.dmsg{border:1px solid var(--line); border-left:3px solid var(--faint); border-radius:10px; padding:14px 17px; margin-bottom:14px; background:linear-gradient(180deg,var(--panel),var(--panel2))}
.dmsg.asker{border-left-color:var(--cyan)}
.dmsg.responder{border-left-color:var(--answered)}
.dmsg-head{display:flex; align-items:baseline; gap:9px}
.dmsg .who{font-weight:600; font-size:13px}
.dmsg.asker .who{color:var(--cyan)}
.dmsg.responder .who{color:var(--answered)}
.dmsg .at{color:var(--faint); font-size:11px}
.dbody{margin:9px 0 0; white-space:pre-wrap; overflow-wrap:anywhere; font-size:14px; line-height:1.7; color:var(--ink)}
.switcher{position:fixed; top:16px; right:16px; z-index:70; display:flex; align-items:center; gap:4px; background:rgba(13,21,36,.86); backdrop-filter:blur(9px); border:1px solid var(--line); border-radius:999px; padding:6px 8px; font-size:11px; letter-spacing:.03em; box-shadow:0 12px 34px rgba(0,0,0,.5)}
.switcher .sw-label{color:var(--faint); padding:0 8px; text-transform:uppercase; letter-spacing:.15em; font-size:10px}
.switcher a{color:var(--muted); text-decoration:none; padding:5px 13px; border-radius:999px; transition:.15s}
.switcher a:hover{color:var(--ink); background:rgba(255,255,255,.05)}
.switcher a.on{color:#06121f; background:var(--cyan); font-weight:600}
</style>`

const missionTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="refresh" content="5">
<title>blvckhole · handoff</title>
{{template "missionStyle" .}}
</head>
<body>
<div class="wrap">
  <header class="top">
    <div class="brand"><span class="logo">◈</span> AGENT HANDOFF</div>
    <div class="meta"><span class="pulse"></span> {{.Count}} active · {{.Now}}</div>
  </header>
  {{range .Threads}}
  <article class="thread {{.Status}}">
    <div class="thread-bar"></div>
    <div class="thread-main">
      <div class="thead">
        <h2 class="subject">{{.Subject}}</h2>
        <span class="status"><i class="led"></i>{{.Status}}</span>
      </div>
      <div class="sub">
        <span class="route">{{.From}} → {{.To}}</span><span class="dot">·</span>
        <span class="count">{{.Count}} msg</span><span class="dot">·</span>
        <span class="id">{{.ID}}</span>
        <span class="upd">{{.Updated}}</span>
      </div>
      <ul class="msgs">
        {{if .More}}<li class="more">+{{.More}} earlier message{{if gt .More 1}}s{{end}}</li>{{end}}
        {{range .Messages}}
        <li class="msg {{if .Own}}asker{{else}}responder{{end}}">
          <span class="who">{{.From}}</span><span class="at">{{.At}}</span>
          <p class="body{{if .Clipped}} clip{{end}}">{{.Body}}</p>
        </li>
        {{end}}
      </ul>
      <a class="open" href="/handoff/thread/{{.ID}}?v={{$.Variant}}">view full thread →</a>
    </div>
  </article>
  {{else}}
  <div class="empty">No threads yet.</div>
  {{end}}
</div>
{{template "switcher" .}}
</body>
</html>`

const missionThreadTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>blvckhole · {{.T.Subject}}</title>
{{template "missionStyle" .}}
</head>
<body>
<div class="wrap detail">
  <nav class="dtop">
    <a href="/handoff?v={{.Variant}}">← all threads</a>
    <a href="/handoff/thread/{{.T.ID}}?v={{.Variant}}">↻ refresh</a>
  </nav>
  <header class="dhead {{.T.Status}}">
    <h1 class="dsubject">{{.T.Subject}}</h1>
    <div class="dsub">
      <span class="route">{{.T.From}} → {{.T.To}}</span><span class="dot">·</span>
      <span class="status"><i class="led"></i>{{.T.Status}}</span><span class="dot">·</span>
      <span class="count">{{.T.Count}} msg</span><span class="dot">·</span>
      <span class="id">{{.T.ID}}</span><span class="upd">· {{.T.Updated}}</span>
    </div>
  </header>
  <ol class="dmsgs">
    {{range .T.Messages}}
    <li class="dmsg {{if .Own}}asker{{else}}responder{{end}}">
      <div class="dmsg-head"><span class="who">{{.From}}</span><span class="at">{{.At}}</span></div>
      <p class="dbody">{{.Body}}</p>
    </li>
    {{end}}
  </ol>
</div>
{{template "switcher" .}}
</body>
</html>`

// ---- terminal --------------------------------------------------------------

const terminalStyle = `
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{
  --bg:#04070a; --ink:#7dffb0; --dim:#2f8f5e; --faint:#1f5e40;
  --amber:#ffb454; --white:#d8ffe9; --line:#0f3a26;
  --mono:"JetBrains Mono", ui-monospace, Menlo, Consolas, monospace;
}
*{box-sizing:border-box}
html{color-scheme:dark}
body{margin:0; background:var(--bg); color:var(--ink); font-family:var(--mono); font-size:13.5px; line-height:1.6; padding-bottom:48px; text-shadow:0 0 6px rgba(125,255,176,.35)}
.crt{max-width:880px; margin:0 auto; padding:30px 22px 0}
body::before{content:""; position:fixed; inset:0; pointer-events:none; z-index:50; background:repeating-linear-gradient(0deg, rgba(0,0,0,.22) 0, rgba(0,0,0,.22) 1px, transparent 2px, transparent 3px)}
body::after{content:""; position:fixed; inset:0; pointer-events:none; z-index:51; background:radial-gradient(125% 85% at 50% 50%, transparent 58%, rgba(0,0,0,.6))}
.prompt{font-weight:500; color:var(--white); letter-spacing:.02em; animation:flick 7s infinite}
.prompt .user{color:var(--ink)}
.prompt .path{color:var(--amber)}
.cursor{display:inline-block; width:.62ch; height:1.05em; vertical-align:-2px; background:var(--ink); animation:blink 1.1s steps(1) infinite; margin-left:3px}
@keyframes blink{
  50%{opacity:0}
}
@keyframes flick{
  0%{opacity:1}
  96%{opacity:1}
  97%{opacity:.78}
  98%{opacity:1}
  100%{opacity:1}
}
.statusline{color:var(--dim); margin:8px 0 22px}
.statusline .ln{color:var(--amber); text-decoration:none}
.statusline .ln:hover{text-decoration:underline}
.thr{border:1px solid var(--line); border-left:2px solid var(--dim); border-radius:4px; padding:12px 14px; margin-bottom:14px; background:rgba(10,40,26,.16)}
.thr.open{border-left-color:var(--amber)}
.thr.answered{border-left-color:var(--ink)}
.thr-head{display:flex; flex-wrap:wrap; align-items:center; gap:10px}
.thr-head .bullet{color:var(--dim)}
.thr-head .subj{color:var(--white); font-weight:700}
.thr-head .route{color:var(--dim)}
.thr-head .tag{margin-left:auto; font-weight:700; text-transform:uppercase; letter-spacing:.1em; font-size:11px}
.thr.open .tag{color:var(--amber)}
.thr.answered .tag{color:var(--ink)}
.thr-meta{color:var(--faint); font-size:11px; margin:6px 0 10px}
.moreln{color:var(--faint); font-size:11px; margin-bottom:6px}
.line{padding:2px 0; color:var(--white)}
.line .from{color:var(--ink); font-weight:500}
.line.a .from{color:var(--amber)}
.line .ts{color:var(--faint); font-size:11px}
.line .txt{white-space:pre-wrap; overflow-wrap:anywhere}
.line .txt.clip{display:-webkit-box; -webkit-line-clamp:2; -webkit-box-orient:vertical; overflow:hidden; margin-top:2px}
.openln{display:inline-block; margin-top:8px; color:var(--amber); text-decoration:none}
.openln:hover{text-decoration:underline}
.empty{color:var(--dim); text-align:center; padding:72px 0}
/* detail */
.dline{padding:9px 0; border-bottom:1px dashed var(--line)}
.dline:last-child{border-bottom:0}
.dline-head .from{color:var(--ink); font-weight:500}
.dline.a .from{color:var(--amber)}
.dline-head .ts{color:var(--faint); font-size:11px; margin-left:8px}
.dtxt{margin-top:5px; color:var(--white); white-space:pre-wrap; overflow-wrap:anywhere; line-height:1.7}
.switcher{position:fixed; top:16px; right:16px; display:flex; align-items:center; gap:4px; z-index:60; background:rgba(4,10,7,.92); border:1px solid var(--line); border-radius:4px; padding:6px 8px; font-size:11px}
.switcher .sw-label{color:var(--faint); padding:0 8px; text-transform:uppercase; letter-spacing:.12em}
.switcher a{color:var(--dim); text-decoration:none; padding:5px 12px; border-radius:3px}
.switcher a:hover{color:var(--ink)}
.switcher a.on{color:var(--bg); background:var(--ink); font-weight:700; text-shadow:none}
</style>`

const terminalTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta http-equiv="refresh" content="5">
<title>blvckhole · handoff</title>
{{template "terminalStyle" .}}
</head>
<body>
<div class="crt">
  <div class="prompt"><span class="user">blvckhole@host</span>:<span class="path">~</span>$ handoff --watch<span class="cursor"></span></div>
  <div class="statusline">[ {{.Count}} threads · live · {{.Now}} ]</div>
  {{range .Threads}}
  <section class="thr {{.Status}}">
    <div class="thr-head">
      <span class="bullet">▌</span>
      <span class="subj">{{.Subject}}</span>
      <span class="route">{{.From}}→{{.To}}</span>
      <span class="tag">[{{.Status}}]</span>
    </div>
    <div class="thr-meta">id:{{.ID}} · {{.Count}} msg · {{.Updated}}</div>
    {{if .More}}<div class="moreln">… +{{.More}} earlier message{{if gt .More 1}}s{{end}}</div>{{end}}
    {{range .Messages}}
    <div class="line {{if .Own}}q{{else}}a{{end}}"><span class="from">{{.From}}</span> <span class="ts">{{.At}}</span>  <span class="txt{{if .Clipped}} clip{{end}}">{{.Body}}</span></div>
    {{end}}
    <a class="openln" href="/handoff/thread/{{.ID}}?v={{$.Variant}}">$ open --full →</a>
  </section>
  {{else}}
  <div class="empty">~ no threads yet ~</div>
  {{end}}
  {{template "switcher" .}}
</div>
</body>
</html>`

const terminalThreadTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>blvckhole · {{.T.Subject}}</title>
{{template "terminalStyle" .}}
</head>
<body>
<div class="crt">
  <div class="prompt"><span class="user">blvckhole@host</span>:<span class="path">~</span>$ handoff --thread {{.T.ID}}<span class="cursor"></span></div>
  <div class="statusline">[ <a class="ln" href="/handoff?v={{.Variant}}">../board</a> · {{.T.Count}} msg · {{.T.Updated}} ]</div>
  <section class="thr {{.T.Status}}">
    <div class="thr-head">
      <span class="bullet">▌</span>
      <span class="subj">{{.T.Subject}}</span>
      <span class="route">{{.T.From}}→{{.T.To}}</span>
      <span class="tag">[{{.T.Status}}]</span>
    </div>
    <div class="thr-meta">id:{{.T.ID}}</div>
    {{range .T.Messages}}
    <div class="dline {{if .Own}}q{{else}}a{{end}}">
      <div class="dline-head"><span class="from">{{.From}}</span> <span class="ts">{{.At}}</span></div>
      <div class="dtxt">{{.Body}}</div>
    </div>
    {{end}}
  </section>
  {{template "switcher" .}}
</div>
</body>
</html>`
