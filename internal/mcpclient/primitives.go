package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// ListTools fetches the server's tool catalogue.
func (c *Client) ListTools(ctx context.Context) (*ToolsResult, error) {
	resp, err := c.Call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out ToolsResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, fmt.Errorf("mcpclient: decode tools/list: %w", err)
	}
	return &out, nil
}

// CallTool invokes a tool by name with structured arguments and returns the
// raw result. A tool-level failure (isError=true) is returned as a result,
// not an error — assertions decide severity.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (*CallToolResult, error) {
	resp, err := c.Call(ctx, "tools/call", CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out CallToolResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, fmt.Errorf("mcpclient: decode tools/call: %w", err)
	}
	return &out, nil
}

// ListResources fetches the server's static resource catalogue.
func (c *Client) ListResources(ctx context.Context) (*ResourcesResult, error) {
	resp, err := c.Call(ctx, "resources/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out ResourcesResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, fmt.Errorf("mcpclient: decode resources/list: %w", err)
	}
	return &out, nil
}

// ListPrompts fetches the server's prompt catalogue.
func (c *Client) ListPrompts(ctx context.Context) (*PromptsResult, error) {
	resp, err := c.Call(ctx, "prompts/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out PromptsResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, fmt.Errorf("mcpclient: decode prompts/list: %w", err)
	}
	return &out, nil
}
