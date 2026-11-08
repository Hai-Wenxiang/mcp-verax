package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Hai-Wenxiang/mcp-verax/internal/assert"
	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

func sampleReports() []*runner.SuiteReport {
	return []*runner.SuiteReport{
		{
			Suite: "demo", Source: "demo.suite.yaml", DurationMS: 42,
			Cases: []runner.CaseResult{
				{Name: "good", Status: runner.StatusPass, DurationMS: 5,
					Checks: []assert.CheckResult{{Desc: "tools.list.includes", Pass: true}}},
				{Name: "bad", Status: runner.StatusFail, DurationMS: 7,
					Checks: []assert.CheckResult{
						{Desc: "ok", Pass: false, Detail: "expected successful call; result isError=true: boom"},
					}},
				{Name: "broken", Status: runner.StatusError, DurationMS: 0, Err: "start server: exec: not found"},
			},
		},
	}
}

func TestRenderTerminalContainsVerdicts(t *testing.T) {
	var buf bytes.Buffer
	RenderTerminal(&buf, sampleReports(), false)
	out := buf.String()

	for _, want := range []string{
		"== demo (demo.suite.yaml) ==",
		"[PASS] good",
		"[FAIL] bad",
		"[ERR ] broken",
		"expected successful call",
		"start server: exec: not found",
		"1/3 cases passed",
		"verax: FAILED — 1/3 cases passed across 1 suite(s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "tools.list.includes") {
		t.Error("passing check details should be hidden without -v")
	}
}

func TestRenderTerminalVerboseShowsPassingChecks(t *testing.T) {
	var buf bytes.Buffer
	RenderTerminal(&buf, sampleReports(), true)
	if !strings.Contains(buf.String(), "tools.list.includes") {
		t.Error("verbose mode should list passing checks")
	}
}

func TestRenderTerminalAllGreen(t *testing.T) {
	reports := []*runner.SuiteReport{{
		Suite: "ok", Source: "ok.suite.yaml",
		Cases: []runner.CaseResult{
			{Name: "a", Status: runner.StatusPass},
			{Name: "b", Status: runner.StatusPass},
		},
	}}
	var buf bytes.Buffer
	RenderTerminal(&buf, reports, false)
	if !strings.Contains(buf.String(), "verax: OK — 2/2 cases passed") {
		t.Errorf("verdict line wrong:\n%s", buf.String())
	}
}
