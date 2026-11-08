package report

import (
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

var htmlTmpl = template.Must(template.New("verax").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Verax report — {{.Generated}}</title>
<style>
  :root { --pass:#1a7f37; --fail:#cf222e; --err:#9a6700; --ink:#1f2328; --dim:#656d76; --bg:#f6f8fa; }
  body { font: 14px/1.5 -apple-system,"Segoe UI",Roboto,"PingFang SC","Microsoft YaHei",sans-serif; color: var(--ink); margin: 0; background: var(--bg); }
  header { padding: 20px 28px; background: #24292f; color: #fff; }
  header h1 { margin: 0 0 4px; font-size: 18px; }
  header .meta { color: #afb8c1; font-size: 12px; }
  main { padding: 20px 28px; max-width: 980px; margin: 0 auto; }
  .verdict { display: inline-block; padding: 2px 10px; border-radius: 12px; font-weight: 600; font-size: 13px; }
  .v-ok { background: #dafbe1; color: var(--pass); }
  .v-bad { background: #ffebe9; color: var(--fail); }
  section.suite { background: #fff; border: 1px solid #d0d7de; border-radius: 8px; padding: 14px 18px; margin-bottom: 18px; }
  section.suite > h2 { font-size: 15px; margin: 0 0 8px; }
  .case { border-top: 1px solid #eaeef2; padding: 8px 0; }
  .case:first-of-type { border-top: 0; }
  .case .head { display: flex; gap: 10px; align-items: baseline; }
  .badge { font-weight: 700; font-size: 11px; padding: 1px 8px; border-radius: 10px; }
  .b-pass { background: #dafbe1; color: var(--pass); }
  .b-fail { background: #ffebe9; color: var(--fail); }
  .b-err  { background: #fff8c5; color: var(--err); }
  .case .name { font-weight: 600; }
  .case .dur { color: var(--dim); font-size: 12px; }
  .check { margin: 4px 0 0 18px; font-size: 13px; }
  .check .desc { font-family: ui-monospace,Consolas,monospace; }
  .check.pass .desc { color: var(--pass); }
  .check.fail .desc { color: var(--fail); }
  .check .detail { color: var(--dim); white-space: pre-wrap; margin: 2px 0 0 12px; font-size: 12px; }
  .err { color: var(--err); margin: 4px 0 0 18px; white-space: pre-wrap; font-size: 13px; }
  .stderr { color: var(--dim); margin: 2px 0 0 18px; font-family: ui-monospace,Consolas,monospace; font-size: 12px; }
  footer { color: var(--dim); font-size: 12px; text-align: center; padding: 14px; }
</style>
</head>
<body>
<header>
  <h1>Verax report / Verax 测试报告</h1>
  <div class="meta">generated {{.Generated}} · {{.TotalPassed}}/{{.TotalCases}} cases passed · {{.SuiteCount}} suite(s)</div>
</header>
<main>
  <p><span class="verdict {{if .AllPass}}v-ok{{else}}v-bad{{end}}">{{if .AllPass}}ALL PASS{{else}}FAILURES{{end}}</span></p>
  {{range .Suites}}
  <section class="suite">
    <h2>{{.Suite}} <span style="color:var(--dim);font-weight:400;font-size:12px">({{.Source}} · {{.Passed}}/{{.Total}} · {{.DurationMS}}ms)</span></h2>
    {{range .Cases}}
    <div class="case">
      <div class="head">
        <span class="badge {{.StatusClass}}">{{.Status}}</span>
        <span class="name">{{.Name}}</span>
        <span class="dur">{{.DurationMS}}ms</span>
      </div>
      {{if .Err}}<div class="err">{{.Err}}</div>{{end}}
      {{range .Checks}}
      <div class="check {{if .Pass}}pass{{else}}fail{{end}}">
        <div class="desc">{{if .Pass}}✓{{else}}✗{{end}} {{.Desc}}</div>
        {{if .Detail}}<div class="detail">{{.Detail}}</div>{{end}}
      </div>
      {{end}}
      {{range .StderrTail}}<div class="stderr">stderr: {{.}}</div>{{end}}
    </div>
    {{end}}
  </section>
  {{end}}
</main>
<footer>verax — truth layer for the MCP ecosystem · single-file report, attach anywhere</footer>
</body>
</html>
`))

type htmlCheck struct {
	Pass   bool
	Desc   string
	Detail string
}

type htmlCase struct {
	Status, StatusClass, Name string
	DurationMS                int64
	Err                       string
	Checks                    []htmlCheck
	StderrTail                []string
}

type htmlSuite struct {
	Suite, Source string
	Passed, Total int
	DurationMS    int64
	Cases         []htmlCase
}

type htmlDoc struct {
	Generated   string
	AllPass     bool
	TotalCases  int
	TotalPassed int
	SuiteCount  int
	Suites      []htmlSuite
}

// RenderHTML writes a self-contained single-file HTML report (inline CSS,
// no external assets — attach it to issues or PRs as-is).
func RenderHTML(w io.Writer, reports []*runner.SuiteReport, generatedAt time.Time) error {
	doc := htmlDoc{
		Generated:  generatedAt.Format("2006-01-02 15:04:05 MST"),
		AllPass:    true,
		SuiteCount: len(reports),
	}
	for _, r := range reports {
		hs := htmlSuite{Suite: r.Suite, Source: r.Source, Total: len(r.Cases), DurationMS: r.DurationMS}
		for _, c := range r.Cases {
			doc.TotalCases++
			hc := htmlCase{Name: c.Name, DurationMS: c.DurationMS, Err: c.Err, StderrTail: c.StderrTail}
			hc.Status = strings.ToUpper(string(c.Status))
			switch c.Status {
			case runner.StatusPass:
				hc.StatusClass = "b-pass"
				hs.Passed++
				doc.TotalPassed++
			case runner.StatusFail:
				hc.StatusClass = "b-fail"
				doc.AllPass = false
			case runner.StatusError:
				hc.StatusClass = "b-err"
				doc.AllPass = false
			}
			for _, ck := range c.Checks {
				hc.Checks = append(hc.Checks, htmlCheck{Pass: ck.Pass, Desc: ck.Desc, Detail: ck.Detail})
			}
			hs.Cases = append(hs.Cases, hc)
		}
		doc.Suites = append(doc.Suites, hs)
	}
	return htmlTmpl.Execute(w, doc)
}
