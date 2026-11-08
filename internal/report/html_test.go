package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/assert"
	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

func mustCaseWithDetail(detail string) runner.CaseResult {
	return runner.CaseResult{
		Name:   "hostile",
		Status: runner.StatusFail,
		Checks: []assert.CheckResult{{Desc: "hostile check", Pass: false, Detail: detail}},
	}
}

func TestRenderHTMLContainsStructure(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, sampleReports(), time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	for _, want := range []string{
		"<!DOCTYPE html>",
		"Verax report",
		`<span class="verdict v-bad">FAILURES</span>`,
		"class=\"badge b-pass\">PASS",
		"class=\"badge b-fail\">FAIL",
		"class=\"badge b-err\">ERR",
		"1/3 cases passed",
		"expected successful call",
		"stderr",
		"single-file report",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if strings.Count(out, "<html") != 1 || strings.Count(out, "</html>") != 1 {
		t.Error("must be a single complete html document")
	}
	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Error("doctype must lead the file")
	}
}

func TestRenderHTMLAllPassVerdict(t *testing.T) {
	reports := sampleReports()[:1]
	reports[0].Cases = reports[0].Cases[:1] // keep only the passing case
	var buf bytes.Buffer
	if err := RenderHTML(&buf, reports, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "v-ok\">ALL PASS") {
		t.Error("all-pass suite should render ALL PASS verdict")
	}
}

func TestRenderHTMLEscapesHostileContent(t *testing.T) {
	hostile := sampleReports()
	hostile[0].Cases = append(hostile[0].Cases, mustCaseWithDetail(`<script>alert(1)</script>`))
	var buf bytes.Buffer
	if err := RenderHTML(&buf, hostile, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "<script>") {
		t.Error("hostile detail leaked as markup")
	}
}
