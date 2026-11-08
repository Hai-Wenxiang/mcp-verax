package assert

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/Hai-Wenxiang/mcp-verax/internal/dsl"
	"github.com/Hai-Wenxiang/mcp-verax/internal/mcpclient"
)

func discoveryFixture() Discovery {
	objSchema, _ := json.Marshal(map[string]any{"type": "object"})
	return Discovery{
		HandshakeOK: true,
		Tools: []mcpclient.Tool{
			{Name: "echo", InputSchema: objSchema},
			{Name: "fail", InputSchema: objSchema},
			{Name: "slow"}, // no schema on purpose
		},
		Resources: []mcpclient.Resource{{URI: "fixture://readme", Name: "readme"}},
		Prompts:   []mcpclient.Prompt{{Name: "greet"}},
	}
}

func TestEvaluateDiscoveryIncludesAndCount(t *testing.T) {
	expect := dsl.ExpectBlock{
		ToolsListIncludes: []string{"echo", "fail"},
		ToolsListCount:    3,
	}
	results := EvaluateDiscovery(expect, discoveryFixture())
	if len(results) != 2 {
		t.Fatalf("want 2 checks, got %d", len(results))
	}
	for _, r := range results {
		if !r.Pass {
			t.Errorf("%s failed: %s", r.Desc, r.Detail)
		}
	}
}

func TestEvaluateDiscoveryMissingTool(t *testing.T) {
	expect := dsl.ExpectBlock{ToolsListIncludes: []string{"nonexistent"}}
	results := EvaluateDiscovery(expect, discoveryFixture())
	if len(results) != 1 || results[0].Pass {
		t.Fatalf("expected failure, got %+v", results)
	}
	if results[0].Detail == "" || !contains([]string{results[0].Detail}, results[0].Detail) {
		t.Errorf("failure should carry detail")
	}
}

func TestEvaluateDiscoveryExact(t *testing.T) {
	ok := dsl.ExpectBlock{ToolsListExact: []string{"echo", "fail", "slow"}}
	if r := EvaluateDiscovery(ok, discoveryFixture()); len(r) != 1 || !r[0].Pass {
		t.Errorf("exact match should pass: %+v", r)
	}
	bad := dsl.ExpectBlock{ToolsListExact: []string{"echo"}}
	if r := EvaluateDiscovery(bad, discoveryFixture()); len(r) != 1 || r[0].Pass {
		t.Errorf("extra tools should fail exact: %+v", r)
	}
}

func TestEvaluateDiscoverySchemaValid(t *testing.T) {
	expect := dsl.ExpectBlock{ToolsListSchemaOK: true}
	r := EvaluateDiscovery(expect, discoveryFixture())
	if len(r) != 1 || r[0].Pass { // slow tool has no schema
		t.Fatalf("schema_valid should flag 'slow': %+v", r)
	}
	fixed := discoveryFixture()
	for i := range fixed.Tools {
		fixed.Tools[i].InputSchema, _ = json.Marshal(map[string]any{"type": "object"})
	}
	r = EvaluateDiscovery(expect, fixed)
	if len(r) != 1 || !r[0].Pass {
		t.Errorf("all-object schemas should pass: %+v", r)
	}
}

func okResult(text string) *mcpclient.CallToolResult {
	return &mcpclient.CallToolResult{Content: []mcpclient.Content{{Type: "text", Text: text}}}
}

func TestEvaluateCallOK(t *testing.T) {
	expect := dsl.ExpectBlock{OK: true, ResultContains: []string{"hello"}}
	results := EvaluateCall(expect, CallOutcome{Result: okResult("hello verax")})
	for _, r := range results {
		if !r.Pass {
			t.Errorf("%s failed: %s", r.Desc, r.Detail)
		}
	}
}

func TestEvaluateCallOKButErrorResult(t *testing.T) {
	expect := dsl.ExpectBlock{OK: true}
	res := &mcpclient.CallToolResult{Content: []mcpclient.Content{{Type: "text", Text: "boom"}}, IsError: true}
	r := EvaluateCall(expect, CallOutcome{Result: res})
	if len(r) != 1 || r[0].Pass {
		t.Fatalf("isError result must fail ok: %+v", r)
	}
	if r[0].Detail == "" {
		t.Error("failure should explain with detail")
	}
}

func TestEvaluateCallTransportError(t *testing.T) {
	expect := dsl.ExpectBlock{OK: true}
	r := EvaluateCall(expect, CallOutcome{Err: errors.New("spawn: exit 1")})
	if len(r) != 1 || r[0].Pass {
		t.Fatalf("transport error must fail ok: %+v", r)
	}
}

func TestEvaluateCallIsErrorAndContains(t *testing.T) {
	expect := dsl.ExpectBlock{IsError: true, ErrorContains: "on purpose"}
	res := &mcpclient.CallToolResult{Content: []mcpclient.Content{{Type: "text", Text: "tool failed on purpose"}}, IsError: true}
	for _, r := range EvaluateCall(expect, CallOutcome{Result: res}) {
		if !r.Pass {
			t.Errorf("%s failed: %s", r.Desc, r.Detail)
		}
	}
}

func TestEvaluateCallErrorContainsMatchesRPCErrors(t *testing.T) {
	expect := dsl.ExpectBlock{ErrorContains: "tool not found"}
	r := EvaluateCall(expect, CallOutcome{Err: fmt.Errorf("jsonrpc error -32631: tool not found: x")})
	if len(r) != 1 || !r[0].Pass {
		t.Fatalf("error.contains should scan RPC error text: %+v", r)
	}
}

func TestEvaluateCallResultContainsMissing(t *testing.T) {
	expect := dsl.ExpectBlock{ResultContains: []string{"absent"}}
	r := EvaluateCall(expect, CallOutcome{Result: okResult("actual text")})
	if len(r) != 1 || r[0].Pass {
		t.Fatalf("missing substring must fail: %+v", r)
	}
}

func TestCallOutcomeText(t *testing.T) {
	o := CallOutcome{Result: &mcpclient.CallToolResult{Content: []mcpclient.Content{
		{Type: "text", Text: "a"}, {Type: "text", Text: "b"},
	}}}
	if o.Text() != "a\nb\n" {
		t.Errorf("text = %q", o.Text())
	}
	e := CallOutcome{Err: errors.New("boom")}
	if e.Text() != "boom" {
		t.Errorf("err text = %q", e.Text())
	}
}
