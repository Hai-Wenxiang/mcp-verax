package fuzz

import (
	"context"
	"strings"
	"testing"
)

var testTools = []ToolSpec{
	{Name: "echo", Schema: []byte(`{"type":"object","properties":{"msg":{"type":"string"}},"required":["msg"]}`)},
	{Name: "boom", Schema: []byte(`{"type":"object"}`)},
}

func TestRunAllTolerated(t *testing.T) {
	res := Run(Config{Runs: 40, Seed: 7, HangMS: 1000, Tools: []string{"echo"}},
		testTools,
		func(ctx context.Context, tool string, args map[string]any) Outcome {
			return OutcomeOK // even hostile input politely handled
		})
	if res.AbortedAt != 0 || len(res.Violations()) != 0 {
		t.Errorf("clean campaign must have no violations: %+v", res)
	}
	if res.TotalRuns != 40 {
		t.Errorf("total runs = %d, want 40", res.TotalRuns)
	}
	if res.Stats["echo"].Runs != 40 {
		t.Errorf("echo runs = %d", res.Stats["echo"].Runs)
	}
}

func TestRunCrashAbortsCampaign(t *testing.T) {
	n := 0
	res := Run(Config{Runs: 100, Seed: 7, HangMS: 1000},
		testTools,
		func(ctx context.Context, tool string, args map[string]any) Outcome {
			n++
			if tool == "boom" {
				return OutcomeCrash
			}
			return OutcomeOK
		})
	if res.Stats["boom"].Crashes != 1 {
		t.Errorf("boom crashes = %d, want 1", res.Stats["boom"].Crashes)
	}
	if res.AbortedAt == 0 || res.TotalRuns >= 100 {
		t.Errorf("campaign should abort early: abortedAt=%d total=%d", res.AbortedAt, res.TotalRuns)
	}
	v := strings.Join(res.Violations(), "; ")
	if !strings.Contains(v, "crashed") || !strings.Contains(v, "aborted at run") {
		t.Errorf("violations should mention crash and abort: %s", v)
	}
}

func TestRunHangCounted(t *testing.T) {
	res := Run(Config{Runs: 10, Seed: 9, HangMS: 500, Tools: []string{"echo"}},
		testTools,
		func(ctx context.Context, tool string, args map[string]any) Outcome {
			return OutcomeHang // deterministic: every call hangs
		})
	if res.Stats["echo"].Hangs != 10 || res.Stats["echo"].Runs != 10 {
		t.Errorf("hangs=%d runs=%d, want 10/10", res.Stats["echo"].Hangs, res.Stats["echo"].Runs)
	}
	v := strings.Join(res.Violations(), "; ")
	if !strings.Contains(v, "hung") {
		t.Errorf("violations should mention hang: %s", v)
	}
	if strings.Contains(v, "crashed") {
		t.Error("no crash expected")
	}
}

func TestToolFilter(t *testing.T) {
	res := Run(Config{Runs: 20, Seed: 1, HangMS: 500, Tools: []string{"echo"}},
		testTools, func(context.Context, string, map[string]any) Outcome { return OutcomeOK })
	if _, ok := res.Stats["boom"]; ok {
		t.Error("boom must not run when filtered out")
	}
}
