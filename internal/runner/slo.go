package runner

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/assert"
	"github.com/Hai-Wenxiang/mcp-verax/internal/dsl"
	"github.com/Hai-Wenxiang/mcp-verax/internal/mcpclient"
)

// SLOResult carries latency/stability measurements for one SLO block.
type SLOResult struct {
	Tool        string
	Samples     int
	Concurrency int
	MeanMS      float64
	P95MS       float64
	MaxMS       float64
	SuccessRate float64
	Violations  []string
}

// Checks converts the SLO outcome into uniform CheckResults.
func (r SLOResult) Checks() []assert.CheckResult {
	var out []assert.CheckResult
	if r.Samples > 0 {
		out = append(out, assert.CheckResult{
			Desc:   fmt.Sprintf("slo %s: %d samples, p95=%.3fms mean=%.3fms max=%.3fms success=%.1f%%", r.Tool, r.Samples, r.P95MS, r.MeanMS, r.MaxMS, r.SuccessRate*100),
			Pass:   len(r.Violations) == 0,
			Detail: joinViolations(r.Violations),
		})
	}
	return out
}

// runSLO hammers slo.Tool with samples calls across concurrency workers and
// evaluates the budgets. Per-call ceiling is 30s regardless of budget.
func runSLO(cli *mcpclient.Client, slo dsl.SLO) SLOResult {
	samples := orDefault(slo.SampleCalls, 20)
	conc := orDefault(slo.Concurrency, 1)
	res := SLOResult{Tool: slo.Tool, Samples: samples, Concurrency: conc}

	var (
		mu      sync.Mutex
		durs    []float64
		success int
		next    atomic.Int64
		perCall = 30 * time.Second
	)

	var wg sync.WaitGroup
	for w := 0; w < conc; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := next.Add(1)
				if i > int64(samples) {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), perCall)
				start := time.Now()
				out, err := cli.CallTool(ctx, slo.Tool, slo.Args)
				ms := float64(time.Since(start).Nanoseconds()) / 1e6
				cancel()
				ok := err == nil && out != nil && !out.IsError
				mu.Lock()
				durs = append(durs, ms)
				if ok {
					success++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	sort.Float64s(durs)
	if len(durs) > 0 {
		var sum float64
		for _, d := range durs {
			sum += d
		}
		res.MeanMS = sum / float64(len(durs))
		res.MaxMS = durs[len(durs)-1]
		res.P95MS = durs[p95Index(len(durs))]
		res.SuccessRate = float64(success) / float64(len(durs))
	}

	if slo.P95CallMS > 0 && res.P95MS > float64(slo.P95CallMS) {
		res.Violations = append(res.Violations,
			fmt.Sprintf("p95 %.0fms exceeds budget %dms", res.P95MS, slo.P95CallMS))
	}
	if slo.SuccessRate > 0 && res.SuccessRate < slo.SuccessRate {
		res.Violations = append(res.Violations,
			fmt.Sprintf("success rate %.1f%% below budget %.1f%%", res.SuccessRate*100, slo.SuccessRate*100))
	}
	return res
}

func p95Index(n int) int {
	if n == 1 {
		return 0
	}
	idx := int(0.95 * float64(n))
	if idx >= n {
		idx = n - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}

func joinViolations(vs []string) string {
	out := ""
	for i, v := range vs {
		if i > 0 {
			out += "; "
		}
		out += v
	}
	return out
}
