package fuzz

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// Outcome classifies one fuzzed call.
type Outcome int

const (
	// OutcomeOK: call succeeded (result, no error flag).
	OutcomeOK Outcome = iota
	// OutcomeTolerated: tool-level error result (isError=true) — not a fuzz
	// failure; robust servers are allowed to reject hostile input politely.
	OutcomeTolerated
	// OutcomeCrash: transport/RPC error — server died or protocol broke.
	OutcomeCrash
	// OutcomeHang: call exceeded the hang budget.
	OutcomeHang
)

// CallFunc performs one fuzzed tool call under the caller's classification.
type CallFunc func(ctx context.Context, tool string, args map[string]any) Outcome

// ToolSpec is a tool available for fuzzing.
type ToolSpec struct {
	Name   string
	Schema []byte
}

// ToolStats aggregates per-tool fuzz behaviour.
type ToolStats struct {
	Runs    int
	Crashes int
	Hangs   int
}

// Config mirrors the DSL fuzz block.
type Config struct {
	Runs   int
	Seed   int64
	HangMS int
	Tools  []string // empty = all tools
}

// Result summarizes a fuzz campaign.
type Result struct {
	TotalRuns int
	AbortedAt int // 1-based run index where a crash aborted the campaign (0 = none)
	Stats     map[string]ToolStats
}

// Violations renders human-readable failures (empty result = pass).
func (r Result) Violations() []string {
	var out []string
	names := make([]string, 0, len(r.Stats))
	for n := range r.Stats {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		st := r.Stats[n]
		if st.Crashes > 0 {
			out = append(out, fmt.Sprintf("tool %q crashed %d/%d runs", n, st.Crashes, st.Runs))
		}
		if st.Hangs > 0 {
			out = append(out, fmt.Sprintf("tool %q hung (>%dms) on %d/%d runs", n, 0, st.Hangs, st.Runs))
		}
	}
	if r.AbortedAt > 0 {
		out = append(out, fmt.Sprintf("campaign aborted at run %d after first crash", r.AbortedAt))
	}
	return out
}

// Run executes the campaign: deterministic under Seed, mixed benign/edge
// arguments (EdgeFraction 0.5). The first crash aborts further runs — after
// a dead server every call "crashes" and would inflate the numbers.
func Run(cfg Config, tools []ToolSpec, call CallFunc) Result {
	res := Result{Stats: map[string]ToolStats{}}
	pool := filterTools(tools, cfg.Tools)
	if len(pool) == 0 || cfg.Runs <= 0 {
		return res
	}

	rng := rand.New(rand.NewSource(cfg.Seed))
	gen := NewGenerator(cfg.Seed, 0.5)

	for i := 1; i <= cfg.Runs; i++ {
		spec := pool[rng.Intn(len(pool))]
		args, err := gen.Args(spec.Schema)
		if err != nil {
			// Unfuzzable schema: count as tolerated skip, not a failure.
			st := res.Stats[spec.Name]
			st.Runs++
			res.Stats[spec.Name] = st
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.HangMS)*time.Millisecond)
		outcome := call(ctx, spec.Name, args)
		cancel()

		st := res.Stats[spec.Name]
		st.Runs++
		switch outcome {
		case OutcomeCrash:
			st.Crashes++
			res.Stats[spec.Name] = st
			res.TotalRuns++
			res.AbortedAt = i
			return res
		case OutcomeHang:
			st.Hangs++
		}
		res.Stats[spec.Name] = st
		res.TotalRuns++
	}
	return res
}

func filterTools(tools []ToolSpec, want []string) []ToolSpec {
	if len(want) == 0 {
		return tools
	}
	set := make(map[string]bool, len(want))
	for _, w := range want {
		set[w] = true
	}
	var out []ToolSpec
	for _, t := range tools {
		if set[t.Name] {
			out = append(out, t)
		}
	}
	return out
}
