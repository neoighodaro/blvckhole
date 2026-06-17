package handoff

import (
	"html/template"
	"io"
)

var boardTmpl = template.Must(template.New("board").Parse(boardHTML))

// RenderBoard writes the HTML board, newest thread first.
func RenderBoard(w io.Writer, threads []Thread) error {
	reversed := make([]Thread, 0, len(threads))
	for i := len(threads) - 1; i >= 0; i-- {
		reversed = append(reversed, threads[i])
	}
	return boardTmpl.Execute(w, reversed)
}

const boardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta http-equiv="refresh" content="5">
<title>blvckhole handoff</title>
<style>
  body { font-family: -apple-system, system-ui, sans-serif; background:#11111b; color:#cdd6f4; margin:0; padding:24px; }
  h1 { font-size:18px; font-weight:600; margin:0 0 16px; }
  .thread { background:#1e1e2e; border:1px solid #313244; border-radius:8px; padding:16px; margin-bottom:16px; }
  .head { display:flex; justify-content:space-between; align-items:center; margin-bottom:8px; }
  .subject { font-weight:600; }
  .route { color:#9399b2; font-size:13px; margin-bottom:4px; }
  .status { font-size:12px; text-transform:uppercase; letter-spacing:0.05em; padding:2px 8px; border-radius:999px; }
  .status.open { background:#f9e2af; color:#1e1e2e; }
  .status.answered { background:#a6e3a1; color:#1e1e2e; }
  .msg { border-top:1px solid #313244; padding:8px 0; }
  .msg .meta { color:#9399b2; font-size:12px; }
  .msg .body { white-space:pre-wrap; }
  .empty { color:#9399b2; }
</style>
</head>
<body>
<h1>blvckhole handoff</h1>
{{if not .}}<p class="empty">No threads yet.</p>{{end}}
{{range .}}
<div class="thread">
  <div class="head">
    <span class="subject">{{.Subject}}</span>
    <span class="status {{.Status}}">{{.Status}}</span>
  </div>
  <div class="route">{{.From}} &rarr; {{.To}}</div>
  {{range .Messages}}
  <div class="msg">
    <div class="meta">{{.From}} &middot; {{.At}}</div>
    <div class="body">{{.Body}}</div>
  </div>
  {{end}}
</div>
{{end}}
</body>
</html>
`
