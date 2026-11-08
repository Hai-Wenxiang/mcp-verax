// Package runner executes parsed suites against real MCP servers and
// produces structured reports.
//
// Execution model: one long-lived server per suite by default; when any case
// declares per-case env, each case gets a freshly spawned server (isolation
// for fault injection). Discovery catalogues are fetched per server instance.
package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/assert"
	"github.com/Hai-Wenxiang/mcp-verax/internal/dsl"
	"github.com/Hai-Wenxiang/mcp-verax/internal/mcpclient"
)

// Status of a single case.
type Status string

const (
	StatusPass  Status = "pass"
	StatusFail  Status = "fail"
	StatusError Status = "error"
)

// Options tune a run.
type Options struct {
	UpdateSnapshots bool
	// Lookup resolves ${VAR} expansion (default: os.LookupEnv).
	Lookup dsl.LookupFunc
	// DefaultCallMS bounds a call without explicit timeout_ms (default 60000).
	DefaultCallMS int
}

func (o Options) lookup() dsl.LookupFunc {
	if o.Lookup != nil {
		return o.Lookup
	}
	return os.LookupEnv
}

// CaseResult is the outcome of one case.
type CaseResult struct {
	Name       string
	Status     Status
	Checks     []assert.CheckResult
	Err        string // suite/case infrastructure error (StatusError)
	DurationMS int64
	StderrTail []string
}

// SuiteReport aggregates a suite run.
type SuiteReport struct {
	Suite        string
	Source       string
	Cases        []CaseResult
	DurationMS   int64
	SnapshotFile string
}

// Pass reports whether every case passed.
func (r *SuiteReport) Pass() bool {
	for _, c := range r.Cases {
		if c.Status != StatusPass {
			return false
		}
	}
	return true
}

// RunSuiteFile loads, expands, validates and runs one suite file. Parse or
// validation problems produce a failed report (CLI exit 1), not a crash.
func RunSuiteFile(path string, opts Options) *SuiteReport {
	raw, err := os.ReadFile(path)
	if err != nil {
		return errorReport(suiteNameFallback(path), path, fmt.Sprintf("read suite: %v", err))
	}
	s, err := dsl.Parse(raw, path)
	if err != nil {
		return errorReport(suiteNameFallback(path), path, err.Error())
	}
	if s, err = s.ExpandEnv(opts.lookup()); err != nil {
		return errorReport(s.Suite, path, err.Error())
	}
	return RunSuite(s, opts)
}

// RunSuite executes a parsed (and expanded) suite.
func RunSuite(s *dsl.Suite, opts Options) *SuiteReport {
	report := &SuiteReport{Suite: s.Suite, Source: s.Source}
	start := time.Now()
	defer func() { report.DurationMS = time.Since(start).Milliseconds() }()

	store, err := assert.OpenSnapshotStore(s.Source, opts.UpdateSnapshots)
	if err != nil {
		report.Cases = []CaseResult{{Name: "(suite)", Status: StatusError, Err: err.Error()}}
		return report
	}
	report.SnapshotFile = store.Path()
	defer func() { _ = store.Save() }()

	if anyCaseEnv(s) {
		// Respawn mode: fresh server per case (per-case env isolation).
		for _, c := range s.Cases {
			cli, err := spawn(s, c.Env)
			if err != nil {
				report.Cases = append(report.Cases, CaseResult{
					Name: c.Name, Status: StatusError, Err: "start server: " + err.Error(),
				})
				continue
			}
			report.Cases = append(report.Cases, runCase(cli, c, store, opts))
			_ = cli.Close()
		}
		return report
	}

	// Shared mode: one server for the whole suite.
	shared, err := spawn(s, nil)
	if err != nil {
		report.Cases = []CaseResult{{Name: "(suite)", Status: StatusError, Err: "start server: " + err.Error()}}
		return report
	}
	defer shared.Close()
	for _, c := range s.Cases {
		res := runCase(shared, c, store, opts)
		if res.Status != StatusPass && len(shared.Stderr()) > 0 {
			res.StderrTail = tail(shared.Stderr(), 10)
		}
		report.Cases = append(report.Cases, res)
	}
	return report
}

func spawn(s *dsl.Suite, extraEnv map[string]string) (*mcpclient.Client, error) {
	parts, err := s.Target.CommandParts()
	if err != nil {
		return nil, err
	}
	envMap := map[string]string{}
	for k, v := range s.Target.Env {
		envMap[k] = v
	}
	for k, v := range extraEnv {
		envMap[k] = v
	}
	var envList []string
	for k, v := range envMap {
		envList = append(envList, k+"="+v)
	}

	timeout := 15 * time.Second
	if s.Target.StartupTimeoutMS > 0 {
		timeout = time.Duration(s.Target.StartupTimeoutMS) * time.Millisecond
	}
	return mcpclient.Connect(context.Background(), mcpclient.StdioConfig{
		Command: parts[0],
		Args:    parts[1:],
		Env:     envList,
	}, mcpclient.Options{HandshakeTimeout: timeout})
}

func runCase(cli *mcpclient.Client, c dsl.Case, store *assert.SnapshotStore, opts Options) CaseResult {
	res := CaseResult{Name: c.Name}
	start := time.Now()
	defer func() { res.DurationMS = time.Since(start).Milliseconds() }()

	callBudget := orDefault(opts.DefaultCallMS, 60000)
	if c.TimeoutMS > 0 {
		callBudget = c.TimeoutMS
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(callBudget)*time.Millisecond)
	defer cancel()

	if !c.IsCall() {
		disc, err := discover(ctx, cli)
		if err != nil {
			res.Status = StatusError
			res.Err = "discovery: " + err.Error()
			return res
		}
		res.Checks = assert.EvaluateDiscovery(c.Expect, disc)
		res.Status = statusFromChecks(res.Checks)
		return res
	}

	callRes, err := cli.CallTool(ctx, c.Call, c.Args)
	outcome := assert.CallOutcome{Result: callRes, Err: err}
	res.Checks = assert.EvaluateCall(c.Expect, outcome)
	if c.Expect.ResultSnapshot != "" && callRes != nil {
		res.Checks = append(res.Checks, store.Check(c.Expect.ResultSnapshot, callRes))
	}
	res.Status = statusFromChecks(res.Checks)
	if outcome.Err != nil && len(res.Checks) == 0 {
		// No assertion applied but the call itself blew up: surface it.
		res.Status = StatusError
		res.Err = outcome.Err.Error()
	}
	return res
}

func discover(ctx context.Context, cli *mcpclient.Client) (assert.Discovery, error) {
	tools, err := cli.ListTools(ctx)
	if err != nil {
		return assert.Discovery{}, err
	}
	resources, err := cli.ListResources(ctx)
	if err != nil {
		return assert.Discovery{}, err
	}
	prompts, err := cli.ListPrompts(ctx)
	if err != nil {
		return assert.Discovery{}, err
	}
	return assert.Discovery{
		HandshakeOK: true,
		Tools:       tools.Tools,
		Resources:   resources.Resources,
		Prompts:     prompts.Prompts,
	}, nil
}

func statusFromChecks(checks []assert.CheckResult) Status {
	for _, c := range checks {
		if !c.Pass {
			return StatusFail
		}
	}
	return StatusPass
}

func anyCaseEnv(s *dsl.Suite) bool {
	for _, c := range s.Cases {
		if len(c.Env) > 0 {
			return true
		}
	}
	return false
}

func tail(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

func orDefault(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func errorReport(suite, source, msg string) *SuiteReport {
	return &SuiteReport{
		Suite:  suite,
		Source: source,
		Cases:  []CaseResult{{Name: "(suite)", Status: StatusError, Err: msg}},
	}
}

func suiteNameFallback(path string) string {
	base := path
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		base = path[i+1:]
	}
	return base
}
