// Package dsl parses and validates Verax test suite definitions (YAML).
// The DSL is intentionally declarative and non-Turing-complete: every
// construct must survive a git diff review.
package dsl

import (
	"fmt"
	"strings"
)

// Suite is a parsed test suite file.
type Suite struct {
	Suite    string   `yaml:"suite"`
	Target   Target   `yaml:"target"`
	Protocol string   `yaml:"protocol"`
	Before   []string `yaml:"before"`
	Cases    []Case   `yaml:"case"`
	SLO      *SLO     `yaml:"slo"`
	Fuzz     *Fuzz    `yaml:"fuzz"`
	Leak     *Leak    `yaml:"leak"`
	Probe    *Probe   `yaml:"probe"`

	// Source is the file path the suite was loaded from (diagnostics only).
	Source string `yaml:"-"`
}

// Target describes how to launch or reach the server under test.
type Target struct {
	// Command launches a stdio server, e.g. "go run ./testfixtures/mcp".
	Command string `yaml:"command"`
	// URL targets a remote Streamable HTTP server (not yet implemented).
	URL string `yaml:"url"`
	// Headers apply to URL targets.
	Headers map[string]string `yaml:"headers"`
	// Env are extra environment variables for the spawned process.
	Env map[string]string `yaml:"env"`
	// StartupTimeoutMS bounds process start + initialize handshake.
	StartupTimeoutMS int `yaml:"timeout_startup_ms"`
}

// IsStdio reports whether this is a launch (stdio) target.
func (t Target) IsStdio() bool { return t.Command != "" }

// CommandParts splits Command honouring single/double quotes.
func (t Target) CommandParts() ([]string, error) { return splitQuoted(t.Command) }

// Case is one scenario. Cases without Call assert discovery primitives
// (tools.list etc.); cases with Call invoke a tool and assert on its result.
type Case struct {
	Name      string            `yaml:"name"`
	Expect    ExpectBlock       `yaml:"expect"`
	Call      string            `yaml:"call"`
	Args      map[string]any    `yaml:"args"`
	Env       map[string]string `yaml:"env"`
	TimeoutMS int               `yaml:"timeout_ms"`
}

// IsCall reports whether the case invokes a tool.
func (c Case) IsCall() bool { return c.Call != "" }

// ExpectBlock holds all assertions of a case. Discovery keys apply when the
// case has no `call`; result keys apply when it does.
type ExpectBlock struct {
	// Discovery assertions (no call).
	ToolsListIncludes   []string `yaml:"tools.list.includes"`
	ToolsListExact      []string `yaml:"tools.list.exact"`
	ToolsListCount      int      `yaml:"tools.list.count"`
	ToolsListSchemaOK   bool     `yaml:"tools.list.schema_valid"`
	ResourcesListMin    int      `yaml:"resources.list.min"`
	PromptsListIncludes []string `yaml:"prompts.list.includes"`

	// Result assertions (with call).
	OK                bool     `yaml:"ok"`
	IsError           bool     `yaml:"is_error"`
	ErrorContains     string   `yaml:"error.contains"`
	ResultContains    []string `yaml:"result.contains"`
	ResultNotContains []string `yaml:"result.not_contains"`
	ResultSnapshot    string   `yaml:"result.snapshot"`
}

// SLO are latency/stability budgets measured against repeated tool calls.
// Defaults applied by the runner: sample_calls=20, concurrency=1 when unset.
type SLO struct {
	Tool        string         `yaml:"tool"`
	Args        map[string]any `yaml:"args"`
	P95CallMS   int            `yaml:"p95_call_ms"`
	Concurrency int            `yaml:"concurrency"`
	SuccessRate float64        `yaml:"success_rate"`
	SampleCalls int            `yaml:"sample_calls"`
}

// Fuzz configures schema-driven fuzzing.
type Fuzz struct {
	Runs   int      `yaml:"runs"`
	Seed   int64    `yaml:"seed"`
	HangMS int      `yaml:"hang_ms"`
	Tools  []string `yaml:"tools"`
}

// Leak configures process-leak detection.
type Leak struct {
	ProcessExit bool `yaml:"process_exit"`
}

// Probe configures dynamic behaviour probes.
type Probe struct {
	SSRF       *SSRFProbe       `yaml:"ssrf"`
	SideEffect *SideEffectProbe `yaml:"side_effect"`
}

// SideEffectProbe: honeypot directory + argument template; every tool is
// called once and any write into the honeypot is reported per tool.
type SideEffectProbe struct {
	Args   map[string]string `yaml:"args"`    // prop -> value template; ${VERAX_HONEYROOT} substituted
	HangMS int               `yaml:"hang_ms"` // per-call budget (default 8000)
}

// SSRFProbe: inject a listener URL into every string argument of every tool
// and report any server that makes the outbound connection.
type SSRFProbe struct {
	HangMS int `yaml:"hang_ms"` // per-call budget (default 5000)
}

// Validate checks the suite for structural errors and returns all problems
// found (not just the first).
func (s *Suite) Validate() error {
	var problems []string

	if s.Suite == "" {
		problems = append(problems, "suite: name is required")
	}
	switch {
	case s.Target.Command == "" && s.Target.URL == "":
		problems = append(problems, "target: one of command (stdio) or url (http) is required")
	case s.Target.Command != "" && s.Target.URL != "":
		problems = append(problems, "target: command and url are mutually exclusive")
	}
	if s.Target.URL != "" {
		for i, c := range s.Cases {
			if len(c.Env) > 0 {
				problems = append(problems, fmt.Sprintf("case[%d] %q: per-case env requires a stdio command target", i, c.Name))
			}
		}
	}
	if s.Target.Command != "" {
		if _, err := s.Target.CommandParts(); err != nil {
			problems = append(problems, fmt.Sprintf("target.command: %v", err))
		}
	}
	if s.Target.StartupTimeoutMS < 0 {
		problems = append(problems, "target.timeout_startup_ms: must be >= 0")
	}

	if s.Protocol != "" && !supportedProtocol(s.Protocol) {
		problems = append(problems, fmt.Sprintf("protocol: %q is not a supported MCP version", s.Protocol))
	}

	for i, b := range s.Before {
		if b != "initialize.ok" {
			problems = append(problems, fmt.Sprintf("before[%d]: unknown probe %q (only initialize.ok is supported)", i, b))
		}
	}

	if len(s.Cases) == 0 {
		problems = append(problems, "case: at least one case is required")
	}
	for i, c := range s.Cases {
		where := fmt.Sprintf("case[%d]", i)
		if c.Name == "" {
			problems = append(problems, where+": name is required")
		} else {
			where = fmt.Sprintf("case[%d] %q", i, c.Name)
		}
		if !c.IsCall() && c.Expect.isEmptyExpect() {
			problems = append(problems, where+": needs a call or at least one expect")
		}
		if c.IsCall() && c.Expect.hasDiscoveryKeys() {
			problems = append(problems, where+": discovery assertions (tools.list.*) do not apply to call cases")
		}
		if !c.IsCall() && c.Expect.hasResultKeys() {
			problems = append(problems, where+": result assertions (ok/result.*) require a call")
		}
		if c.TimeoutMS < 0 {
			problems = append(problems, where+": timeout_ms must be >= 0")
		}
	}

	if s.SLO != nil {
		if s.SLO.P95CallMS < 0 {
			problems = append(problems, "slo.p95_call_ms: must be >= 0")
		}
		if s.SLO.Concurrency < 0 {
			problems = append(problems, "slo.concurrency: must be >= 0")
		}
		if s.SLO.SuccessRate < 0 || s.SLO.SuccessRate > 1 {
			problems = append(problems, "slo.success_rate: must be within [0,1]")
		}
		if s.SLO.SampleCalls < 0 {
			problems = append(problems, "slo.sample_calls: must be >= 0")
		}
		if s.SLO.Tool == "" && (s.SLO.SampleCalls > 0 || s.SLO.P95CallMS > 0 || s.SLO.SuccessRate > 0) {
			problems = append(problems, "slo.tool: required when any SLO budget is set")
		}
	}

	if s.Fuzz != nil {
		if s.Fuzz.Runs <= 0 {
			problems = append(problems, "fuzz.runs: must be > 0")
		}
		if s.Fuzz.HangMS <= 0 {
			problems = append(problems, "fuzz.hang_ms: must be > 0")
		}
	}

	return joinProblems(problems)
}

func (e ExpectBlock) isEmptyExpect() bool {
	return len(e.ToolsListIncludes) == 0 && len(e.ToolsListExact) == 0 &&
		e.ToolsListCount == 0 && !e.ToolsListSchemaOK &&
		e.ResourcesListMin == 0 && len(e.PromptsListIncludes) == 0 &&
		!e.OK && !e.IsError && e.ErrorContains == "" &&
		len(e.ResultContains) == 0 && len(e.ResultNotContains) == 0 && e.ResultSnapshot == ""
}

func (e ExpectBlock) hasDiscoveryKeys() bool {
	return len(e.ToolsListIncludes) > 0 || len(e.ToolsListExact) > 0 ||
		e.ToolsListCount != 0 || e.ToolsListSchemaOK ||
		e.ResourcesListMin != 0 || len(e.PromptsListIncludes) > 0
}

func (e ExpectBlock) hasResultKeys() bool {
	return e.OK || e.IsError || e.ErrorContains != "" ||
		len(e.ResultContains) > 0 || len(e.ResultNotContains) > 0 || e.ResultSnapshot != ""
}

func joinProblems(ps []string) error {
	if len(ps) == 0 {
		return nil
	}
	return fmt.Errorf("invalid suite:\n  - %s", strings.Join(ps, "\n  - "))
}

// supportedProtocol mirrors mcpclient.SupportedProtocolVersions without an
// import cycle risk (kept literal, cross-checked by tests).
var supportedProtocols = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

func supportedProtocol(v string) bool {
	for _, p := range supportedProtocols {
		if p == v {
			return true
		}
	}
	return false
}
