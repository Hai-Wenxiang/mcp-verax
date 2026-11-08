package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/assert"
	"github.com/Hai-Wenxiang/mcp-verax/internal/dsl"
	"github.com/Hai-Wenxiang/mcp-verax/internal/fuzz"
)

// runFuzzCampaign fuzzes the server's tools with schema-driven hostile
// arguments. Tool-level error results are tolerated; transport/RPC errors
// count as crashes, budget overruns as hangs.
func runFuzzCampaign(cli mcpConn, f dsl.Fuzz) (fuzz.Result, error) {
	listCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tools, err := cli.ListTools(listCtx)
	if err != nil {
		return fuzz.Result{}, fmt.Errorf("tools/list for fuzzing: %w", err)
	}
	specs := make([]fuzz.ToolSpec, 0, len(tools.Tools))
	for _, t := range tools.Tools {
		specs = append(specs, fuzz.ToolSpec{Name: t.Name, Schema: t.InputSchema})
	}
	if len(specs) == 0 {
		return fuzz.Result{}, fmt.Errorf("server exposes no tools to fuzz")
	}

	return fuzz.Run(fuzz.Config{Runs: f.Runs, Seed: f.Seed, HangMS: f.HangMS, Tools: f.Tools}, specs,
		func(ctx context.Context, tool string, args map[string]any) fuzz.Outcome {
			res, err := cli.CallTool(ctx, tool, args)
			switch {
			case err == nil && res != nil && !res.IsError:
				return fuzz.OutcomeOK
			case err == nil:
				return fuzz.OutcomeTolerated
			case errors.Is(err, context.DeadlineExceeded):
				return fuzz.OutcomeHang
			default:
				return fuzz.OutcomeCrash
			}
		}), nil
}

// runFuzzBlock runs the fuzz campaign in respawn mode with a dedicated server.
func runFuzzBlock(s *dsl.Suite, opts Options, report *SuiteReport) {
	cli, err := spawnGrace(s, nil, opts.CloseGraceMS)
	if err != nil {
		report.Cases = append(report.Cases, CaseResult{
			Name: "fuzz", Status: StatusError, Err: "start server: " + err.Error(),
		})
		return
	}
	defer func() { _ = cli.Close() }()
	fres, err := runFuzzCampaign(cli, *s.Fuzz)
	report.Cases = append(report.Cases, fuzzCaseResult(fres, err))
}

func fuzzCaseResult(res fuzz.Result, err error) CaseResult {
	if err != nil {
		return CaseResult{Name: "fuzz", Status: StatusError, Err: err.Error()}
	}
	vs := res.Violations()
	toolNames := make([]string, 0, len(res.Stats))
	for n := range res.Stats {
		toolNames = append(toolNames, n)
	}
	sortStrings(toolNames)
	check := assert.CheckResult{
		Desc:   fmt.Sprintf("fuzz: %d runs across %d tools (%s): no crash, no hang", res.TotalRuns, len(res.Stats), strings.Join(toolNames, ",")),
		Pass:   len(vs) == 0,
		Detail: strings.Join(vs, "; "),
	}
	return CaseResult{
		Name:   "fuzz",
		Status: statusFromChecks([]assert.CheckResult{check}),
		Checks: []assert.CheckResult{check},
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
