// Package assert evaluates DSL expectation blocks against observed outcomes
// and manages golden snapshots for tool-call results.
package assert

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Hai-Wenxiang/mcp-verax/internal/dsl"
	"github.com/Hai-Wenxiang/mcp-verax/internal/mcpclient"
)

// CheckResult is a single assertion outcome.
type CheckResult struct {
	Desc   string
	Pass   bool
	Detail string
}

// Discovery is what a suite's discovery phase observed: handshake outcome
// and the server's primitives catalogues.
type Discovery struct {
	HandshakeOK bool
	Tools       []mcpclient.Tool
	Resources   []mcpclient.Resource
	Prompts     []mcpclient.Prompt
}

// EvaluateDiscovery applies discovery assertions (tools.list.* etc.).
// Only keys the user set are evaluated; zero-value keys are skipped.
func EvaluateDiscovery(expect dsl.ExpectBlock, got Discovery) []CheckResult {
	var out []CheckResult

	if len(expect.ToolsListIncludes) > 0 {
		have := toolNames(got.Tools)
		var missing []string
		for _, want := range expect.ToolsListIncludes {
			if !contains(have, want) {
				missing = append(missing, want)
			}
		}
		out = append(out, CheckResult{
			Desc:   "tools.list.includes",
			Pass:   len(missing) == 0,
			Detail: detailMissing("tools", have, missing),
		})
	}

	if len(expect.ToolsListExact) > 0 {
		have := toolNames(got.Tools)
		want := sortedCopy(expect.ToolsListExact)
		pass := equalSlices(have, want)
		out = append(out, CheckResult{
			Desc:   "tools.list.exact",
			Pass:   pass,
			Detail: fmt.Sprintf("want exactly %v, have %v", want, have),
		})
	}

	if expect.ToolsListCount > 0 {
		pass := len(got.Tools) == expect.ToolsListCount
		out = append(out, CheckResult{
			Desc:   fmt.Sprintf("tools.list.count == %d", expect.ToolsListCount),
			Pass:   pass,
			Detail: fmt.Sprintf("have %d tools", len(got.Tools)),
		})
	}

	if expect.ToolsListSchemaOK {
		var bad []string
		for _, tool := range got.Tools {
			if !schemaIsObject(tool.InputSchema) {
				bad = append(bad, tool.Name)
			}
		}
		out = append(out, CheckResult{
			Desc:   "tools.list.schema_valid",
			Pass:   len(bad) == 0,
			Detail: detailBadSchemas(bad),
		})
	}

	if expect.ResourcesListMin > 0 {
		pass := len(got.Resources) >= expect.ResourcesListMin
		out = append(out, CheckResult{
			Desc:   fmt.Sprintf("resources.list >= %d", expect.ResourcesListMin),
			Pass:   pass,
			Detail: fmt.Sprintf("have %d resources", len(got.Resources)),
		})
	}

	if len(expect.PromptsListIncludes) > 0 {
		have := promptNames(got.Prompts)
		var missing []string
		for _, want := range expect.PromptsListIncludes {
			if !contains(have, want) {
				missing = append(missing, want)
			}
		}
		out = append(out, CheckResult{
			Desc:   "prompts.list.includes",
			Pass:   len(missing) == 0,
			Detail: detailMissing("prompts", have, missing),
		})
	}

	return out
}

// CallOutcome is the result of one tools/call attempt: either a result
// (possibly isError=true) or an error (RPC error, transport failure, timeout).
type CallOutcome struct {
	Result *mcpclient.CallToolResult
	Err    error
}

// Text returns human-readable output: concatenated content text, or the
// error string. Used by contains-style assertions.
func (o CallOutcome) Text() string {
	if o.Err != nil {
		return o.Err.Error()
	}
	if o.Result == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range o.Result.Content {
		sb.WriteString(c.Text)
		sb.WriteString("\n")
	}
	return sb.String()
}

// EvaluateCall applies result assertions. The `result.snapshot` key is
// intentionally not handled here — the runner routes it through the
// SnapshotStore (see snapshot.go).
func EvaluateCall(expect dsl.ExpectBlock, got CallOutcome) []CheckResult {
	var out []CheckResult

	if expect.OK {
		pass := got.Err == nil && got.Result != nil && !got.Result.IsError
		out = append(out, CheckResult{
			Desc:   "ok",
			Pass:   pass,
			Detail: callDetail("expected successful call", got),
		})
	}

	if expect.IsError {
		pass := got.Result != nil && got.Result.IsError
		out = append(out, CheckResult{
			Desc:   "is_error",
			Pass:   pass,
			Detail: callDetail("expected tool-level error result", got),
		})
	}

	if expect.ErrorContains != "" {
		pass := strings.Contains(got.Text(), expect.ErrorContains)
		out = append(out, CheckResult{
			Desc:   fmt.Sprintf("error.contains %q", expect.ErrorContains),
			Pass:   pass,
			Detail: truncate(got.Text(), 200),
		})
	}

	if len(expect.ResultContains) > 0 {
		text := ""
		if got.Result != nil {
			text = got.Text()
		}
		var missing []string
		for _, want := range expect.ResultContains {
			if !strings.Contains(text, want) {
				missing = append(missing, want)
			}
		}
		out = append(out, CheckResult{
			Desc:   "result.contains",
			Pass:   len(missing) == 0 && got.Result != nil,
			Detail: fmt.Sprintf("missing %v in %q", missing, truncate(text, 200)),
		})
	}

	return out
}

// schemaIsObject checks that a tool inputSchema is a JSON object with
// type=object — the minimum MCP requires.
func schemaIsObject(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var schema struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return false
	}
	return schema.Type == "object"
}

func toolNames(tools []mcpclient.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Name)
	}
	sortStrings(out)
	return out
}

func promptNames(ps []mcpclient.Prompt) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Name)
	}
	sortStrings(out)
	return out
}

func detailMissing(kind string, have, missing []string) string {
	if len(missing) == 0 {
		return fmt.Sprintf("%s present: %v", kind, have)
	}
	return fmt.Sprintf("missing %v; %s exposes %v", missing, kind, have)
}

func detailBadSchemas(bad []string) string {
	if len(bad) == 0 {
		return "all tool schemas are objects"
	}
	return fmt.Sprintf("tools without object inputSchema: %v", bad)
}

func callDetail(msg string, got CallOutcome) string {
	switch {
	case got.Err != nil:
		return fmt.Sprintf("%s; call errored: %v", msg, got.Err)
	case got.Result == nil:
		return fmt.Sprintf("%s; no result and no error (protocol violation)", msg)
	case got.Result.IsError:
		return fmt.Sprintf("%s; result isError=true: %s", msg, truncate(got.Text(), 200))
	default:
		return fmt.Sprintf("%s; result succeeded: %s", msg, truncate(got.Text(), 200))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
