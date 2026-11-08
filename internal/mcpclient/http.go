package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/jsonrpc"
)

// HTTPConfig describes a remote MCP server over Streamable HTTP.
type HTTPConfig struct {
	// BaseURL is the endpoint root; "/mcp" is appended when missing.
	BaseURL string
	Headers map[string]string
	// HandshakeTimeout bounds initialize (default 15s).
	HandshakeTimeout time.Duration
	// CallTimeout bounds each call without its own deadline (default 60s).
	CallTimeout time.Duration
}

// HTTPClient speaks the POST-only subset of MCP Streamable HTTP: one request
// per POST, response as JSON or a short SSE stream; Mcp-Session-Id echoed
// when the server issues one. It mirrors the Client surface the runner uses.
type HTTPClient struct {
	cfg    HTTPConfig
	http   *http.Client
	info   InitializeResult
	nextID int64
}

// ConnectHTTP performs the initialize handshake against the remote server.
func ConnectHTTP(ctx context.Context, cfg HTTPConfig) (*HTTPClient, error) {
	if cfg.HandshakeTimeout <= 0 {
		cfg.HandshakeTimeout = 15 * time.Second
	}
	if cfg.CallTimeout <= 0 {
		cfg.CallTimeout = 60 * time.Second
	}
	c := &HTTPClient{cfg: cfg, http: &http.Client{Timeout: cfg.CallTimeout}}

	hctx, cancel := context.WithTimeout(ctx, cfg.HandshakeTimeout)
	defer cancel()

	var params InitializeParams
	params.ProtocolVersion = LatestProtocolVersion()
	params.ClientInfo = ClientInfo
	resp, err := c.Call(hctx, "initialize", params)
	if err != nil {
		return nil, &HandshakeError{Msg: err.Error()}
	}
	if resp.Error != nil {
		return nil, &HandshakeError{Msg: fmt.Sprintf("initialize rejected: %v", resp.Error)}
	}
	if err := json.Unmarshal(resp.Result, &c.info); err != nil {
		return nil, &HandshakeError{Msg: fmt.Sprintf("decode initialize result: %v", err)}
	}
	if !versionSupported(c.info.ProtocolVersion) {
		return nil, &HandshakeError{Msg: fmt.Sprintf("server protocol version %q not supported by verax", c.info.ProtocolVersion)}
	}
	if err := c.Notify(context.Background(), "notifications/initialized", nil); err != nil {
		return nil, &HandshakeError{Msg: err.Error()}
	}
	return c, nil
}

// InitializeResult returns the handshake payload.
func (c *HTTPClient) InitializeResult() InitializeResult { return c.info }

// Call posts one JSON-RPC request and returns its correlated response.
func (c *HTTPClient) Call(ctx context.Context, method string, params any) (*jsonrpcResponse, error) {
	c.nextID++
	return c.post(ctx, map[string]any{
		"jsonrpc": "2.0",
		"id":      c.nextID,
		"method":  method,
		"params":  params,
	}, true)
}

// Notify posts a notification (no response expected).
func (c *HTTPClient) Notify(ctx context.Context, method string, params any) error {
	_, err := c.post(ctx, map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}, false)
	return err
}

type jsonrpcResponse = jsonrpc.Response

func (c *HTTPClient) post(ctx context.Context, msg map[string]any, wantResponse bool) (*jsonrpc.Response, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	url := c.cfg.BaseURL
	if !strings.HasSuffix(url, "/mcp") && !strings.Contains(url[len(url)-1:], "?") {
		url = strings.TrimRight(url, "/") + "/mcp"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mcpclient(http): post %s: %w", methodOf(msg), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		if wantResponse {
			return nil, fmt.Errorf("mcpclient(http): %s: got 202 for a request", methodOf(msg))
		}
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mcpclient(http): %s: status %d", methodOf(msg), resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if strings.Contains(ct, "text/event-stream") {
		return parseSSEResponse(raw, idOf(msg))
	}
	var out jsonrpc.Response
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("mcpclient(http): decode response: %w", err)
	}
	return &out, nil
}

// parseSSEResponse extracts the JSON-RPC response with the wanted id from an
// SSE body (data: lines), skipping interleaved notifications.
func parseSSEResponse(raw []byte, wantID int64) (*jsonrpc.Response, error) {
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var probe struct {
			ID     *json.Number      `json:"id"`
			Result json.RawMessage   `json:"result"`
			Error  *jsonrpc.RPCError `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &probe); err != nil {
			continue
		}
		if probe.ID == nil {
			continue // notification
		}
		if probe.ID.String() != fmt.Sprint(wantID) {
			continue
		}
		out := &jsonrpc.Response{Result: probe.Result, Error: probe.Error}
		return out, nil
	}
	return nil, fmt.Errorf("mcpclient(http): response for id %d not found in SSE stream", wantID)
}

func methodOf(msg map[string]any) string {
	if m, ok := msg["method"].(string); ok {
		return m
	}
	return "request"
}

func idOf(msg map[string]any) int64 {
	if f, ok := msg["id"].(int64); ok {
		return f
	}
	if f, ok := msg["id"].(float64); ok {
		return int64(f)
	}
	return -1
}

// Close is a no-op for HTTP (kept for interface parity).
func (c *HTTPClient) Close() error { return nil }

// CleanExit is meaningless for remote servers; reported as true.
func (c *HTTPClient) CleanExit() bool { return true }

// Stderr has no equivalent over HTTP.
func (c *HTTPClient) Stderr() []string { return nil }

// Primitives mirror the stdio client's wrappers.

// ListTools fetches the tool catalogue.
func (c *HTTPClient) ListTools(ctx context.Context) (*ToolsResult, error) {
	resp, err := c.Call(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out ToolsResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CallTool invokes a tool.
func (c *HTTPClient) CallTool(ctx context.Context, name string, arguments map[string]any) (*CallToolResult, error) {
	resp, err := c.Call(ctx, "tools/call", CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out CallToolResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListResources fetches static resources.
func (c *HTTPClient) ListResources(ctx context.Context) (*ResourcesResult, error) {
	resp, err := c.Call(ctx, "resources/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out ResourcesResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListPrompts fetches the prompt catalogue.
func (c *HTTPClient) ListPrompts(ctx context.Context) (*PromptsResult, error) {
	resp, err := c.Call(ctx, "prompts/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	var out PromptsResult
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
