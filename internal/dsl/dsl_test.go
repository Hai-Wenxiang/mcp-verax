package dsl

import (
	"strings"
	"testing"
)

const validSuite = `
suite: mcp-fixture
protocol: 2025-06-18
target:
  command: go run github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp
  env:
    MCP_SLOW_MS: "10"
  timeout_startup_ms: 15000
before:
  - initialize.ok
case:
  - name: tools list exposes echo
    expect:
      tools.list.includes: [echo, fail, slow]
      tools.list.count: 3
      tools.list.schema_valid: true
  - name: echo round trip
    call: echo
    args:
      msg: "hello verax"
    expect:
      ok: true
      result.contains: ["hello verax"]
      result.snapshot: echo-basic
slo:
  tool: echo
  args: {msg: "slo sample"}
  p95_call_ms: 800
  concurrency: 5
  success_rate: 1.0
  sample_calls: 20
fuzz:
  runs: 50
  seed: 42
  hang_ms: 5000
leak:
  process_exit: true
`

func TestParseValidSuite(t *testing.T) {
	s, err := Parse([]byte(validSuite), "valid.yaml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.Suite != "mcp-fixture" {
		t.Errorf("suite = %q", s.Suite)
	}
	if len(s.Cases) != 2 {
		t.Fatalf("cases = %d, want 2", len(s.Cases))
	}
	if s.Cases[0].IsCall() {
		t.Error("case 0 should be a discovery case")
	}
	if !s.Cases[1].IsCall() || s.Cases[1].Call != "echo" {
		t.Errorf("case 1 should call echo: %+v", s.Cases[1])
	}
	if s.SLO == nil || s.SLO.P95CallMS != 800 || s.SLO.SampleCalls != 20 {
		t.Errorf("slo mismatch: %+v", s.SLO)
	}
	if s.Fuzz == nil || s.Fuzz.Runs != 50 || s.Fuzz.Seed != 42 {
		t.Errorf("fuzz mismatch: %+v", s.Fuzz)
	}
	parts, err := s.Target.CommandParts()
	if err != nil || parts[0] != "go" || parts[1] != "run" {
		t.Errorf("command parts = %v, %v", parts, err)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	bad := strings.Replace(validSuite, "suite: mcp-fixture", "suite: x\ntoolls: typo", 1)
	_, err := Parse([]byte(bad), "typo.yaml")
	if err == nil {
		t.Fatal("unknown field must be rejected")
	}
	if !strings.Contains(err.Error(), "toolls") {
		t.Errorf("error should name the unknown field: %v", err)
	}
}

func TestValidateAccumulatesProblems(t *testing.T) {
	bad := `
suite: ""
target:
  command: ""
case:
  - name: bad mix
    call: echo
    expect:
      tools.list.includes: [echo]
      ok: true
  - name: ""
slo:
  success_rate: 2.0
  p95_call_ms: 100
  sample_calls: 5
`
	_, err := Parse([]byte(bad), "bad.yaml")
	if err == nil {
		t.Fatal("expected validation errors")
	}
	msg := err.Error()
	for _, want := range []string{
		"suite: name is required",
		"target: one of command",
		"discovery assertions",
		"case[1]", // name required reported at index
		"success_rate",
		"slo.tool: required",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q:\n%s", want, msg)
		}
	}
}

func TestValidateProtocol(t *testing.T) {
	bad := strings.Replace(validSuite, "2025-06-18", "1999-01-01", 1)
	if _, err := Parse([]byte(bad), "proto.yaml"); err == nil || !strings.Contains(err.Error(), "not a supported MCP version") {
		t.Errorf("protocol validation failed: %v", err)
	}
}

func TestValidateEmptyCaseExpect(t *testing.T) {
	bad := `
suite: x
target: {command: "go run x"}
case:
  - name: pointless
`
	_, err := Parse([]byte(bad), "empty.yaml")
	if err == nil || !strings.Contains(err.Error(), "needs a call or at least one expect") {
		t.Errorf("empty case not rejected: %v", err)
	}
}

func TestURLTargetAcceptsButGuardsEnv(t *testing.T) {
	ok := `
suite: x
target: {url: "https://example.com/mcp"}
case:
  - name: a
    expect: {tools.list.includes: [x]}
`
	if _, err := Parse([]byte(ok), "url-ok.yaml"); err != nil {
		t.Errorf("plain url target must parse: %v", err)
	}

	bad := strings.Replace(ok, "expect:", "env: {A: \"1\"}\n    expect:", 1)
	_, err := Parse([]byte(bad), "url-env.yaml")
	if err == nil || !strings.Contains(err.Error(), "per-case env requires a stdio command target") {
		t.Errorf("url + per-case env must be rejected: %v", err)
	}
}

func TestBeforeOnlyInitializeOK(t *testing.T) {
	bad := strings.Replace(validSuite, "- initialize.ok", "- everything.ok", 1)
	if _, err := Parse([]byte(bad), "before.yaml"); err == nil || !strings.Contains(err.Error(), "unknown probe") {
		t.Errorf("unknown before probe not rejected: %v", err)
	}
}

func TestSplitQuoted(t *testing.T) {
	cases := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{`npx -y @scope/server`, []string{"npx", "-y", "@scope/server"}, false},
		{`npx -y "some server" --flag`, []string{"npx", "-y", "some server", "--flag"}, false},
		{`go run './x y'`, []string{"go", "run", "./x y"}, false},
		{`cmd "unclosed`, nil, true},
		{`   `, nil, true},
	}
	for _, c := range cases {
		got, err := splitQuoted(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("splitQuoted(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("splitQuoted(%q): %v", c.in, err)
			continue
		}
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("splitQuoted(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
