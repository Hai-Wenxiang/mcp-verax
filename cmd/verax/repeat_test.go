package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

func mkReports(pass bool) []*runner.SuiteReport {
	status := runner.StatusPass
	if !pass {
		status = runner.StatusFail
	}
	return []*runner.SuiteReport{{
		Suite: "s",
		Cases: []runner.CaseResult{
			{Name: "stable-ok", Status: runner.StatusPass},
			{Name: "flaky", Status: status},
		},
	}}
}

func TestStabilityDetectsFlaky(t *testing.T) {
	st := newStability(4)
	st.record(mkReports(true))
	st.record(mkReports(false))
	st.record(mkReports(true))
	st.record(mkReports(true))

	if !st.failed {
		t.Error("one failing run must mark failure")
	}
	if st.passed["s/flaky"] != 3 || st.passed["s/stable-ok"] != 4 {
		t.Errorf("counts wrong: %+v", st.passed)
	}

	var buf bytes.Buffer
	st.render(&buf)
	out := buf.String()
	for _, want := range []string{"[OK  ] s/stable-ok  4/4", "[FLAKY] s/flaky  3/4"} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "FLAKY") || strings.Index(out, "FLAKY") > strings.Index(out, "[OK") {
		t.Errorf("flaky cases should render first:\n%s", out)
	}
}

func TestStabilityAllFail(t *testing.T) {
	st := newStability(2)
	allFail := []*runner.SuiteReport{{
		Suite: "s",
		Cases: []runner.CaseResult{
			{Name: "stable-ok", Status: runner.StatusFail},
			{Name: "flaky", Status: runner.StatusFail},
		},
	}}
	st.record(allFail)
	st.record(allFail)
	var buf bytes.Buffer
	st.render(&buf)
	if !strings.Contains(buf.String(), "[FAIL] s/stable-ok  0/2") {
		t.Errorf("all-fail case wrong:\n%s", buf.String())
	}
}
