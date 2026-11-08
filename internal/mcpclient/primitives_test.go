package mcpclient

import (
	"context"
	"testing"
)

func TestListTools(t *testing.T) {
	c := connectFixture(t)
	res, err := c.ListTools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"echo": false, "fail": false, "slow": false}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
}

func TestCallToolEcho(t *testing.T) {
	c := connectFixture(t)
	res, err := c.CallTool(context.Background(), "echo", map[string]any{"msg": "hello verax"})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("echo unexpectedly failed: %+v", res)
	}
	if len(res.Content) == 0 || res.Content[0].Text != "hello verax" {
		t.Fatalf("echo content mismatch: %+v", res.Content)
	}
}

func TestCallToolErrorIsResultNotError(t *testing.T) {
	c := connectFixture(t)
	res, err := c.CallTool(context.Background(), "fail", nil)
	if err != nil {
		t.Fatalf("tool-level failure must be a result, got transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected isError=true result: %+v", res)
	}
}

func TestCallToolUnknown(t *testing.T) {
	c := connectFixture(t)
	_, err := c.CallTool(context.Background(), "no-such-tool", nil)
	if err == nil {
		t.Fatal("unknown tool must surface a JSON-RPC error")
	}
}

func TestListResourcesAndPrompts(t *testing.T) {
	c := connectFixture(t)

	rr, err := c.ListResources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rr.Resources) != 1 || rr.Resources[0].URI != "fixture://readme" {
		t.Errorf("resources mismatch: %+v", rr.Resources)
	}

	pr, err := c.ListPrompts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(pr.Prompts) != 1 || pr.Prompts[0].Name != "greet" {
		t.Errorf("prompts mismatch: %+v", pr.Prompts)
	}
}
