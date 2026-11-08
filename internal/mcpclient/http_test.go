package mcpclient

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"testing"
	"time"
)

// startHTTPFixture launches the fixture in HTTP mode on a free port and
// returns its base URL; the process is killed on cleanup.
func startHTTPFixture(t *testing.T, extraEnv ...string) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	cmd := exec.Command("go", "run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp")
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("MCP_HTTP_PORT=%d", port))
	cmd.Env = append(cmd.Env, extraEnv...)
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// readiness: wait for the port to accept connections
	deadline := time.Now().Add(60 * time.Second)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
			_ = c.Close()
			return fmt.Sprintf("http://%s", addr)
		}
		select {
		case <-time.After(200 * time.Millisecond):
		}
	}
	buf := make([]byte, 4096)
	n, _ := stderr.Read(buf)
	t.Fatalf("fixture http not ready in 60s; stderr: %s", buf[:n])
	return ""
}

func TestHTTPHandshakeAndPrimitives(t *testing.T) {
	base := startHTTPFixture(t)
	ctx := context.Background()

	c, err := ConnectHTTP(ctx, HTTPConfig{BaseURL: base, HandshakeTimeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	defer c.Close()

	info := c.InitializeResult()
	if info.ServerInfo.Name != "mcp-fixture" || info.ProtocolVersion != LatestProtocolVersion() {
		t.Errorf("handshake info wrong: %+v", info)
	}

	tools, err := c.ListTools(ctx)
	if err != nil || len(tools.Tools) == 0 {
		t.Fatalf("tools/list: %v %+v", err, tools)
	}
	res, err := c.CallTool(ctx, "echo", map[string]any{"msg": "over http"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Content) == 0 || res.Content[0].Text != "over http" {
		t.Fatalf("echo content: %+v", res)
	}

	prompts, err := c.ListPrompts(ctx)
	if err != nil || len(prompts.Prompts) == 0 {
		t.Errorf("prompts/list: %v %+v", err, prompts)
	}
}

func TestHTTPHandshakeRejectsBogusVersion(t *testing.T) {
	base := startHTTPFixture(t, "MCP_FAKE_VERSION=1999-01-01")
	_, err := ConnectHTTP(context.Background(), HTTPConfig{BaseURL: base, HandshakeTimeout: 30 * time.Second})
	if err == nil || !containsStr(err.Error(), "not supported") {
		t.Fatalf("bogus version must fail handshake: %v", err)
	}
}

func TestParseSSEResponse(t *testing.T) {
	body := `event: message
data: {"jsonrpc":"2.0","method":"notifications/message","params":{"x":1}}

event: message
data: {"jsonrpc":"2.0","id":7,"result":{"ok":true}}

`
	resp, err := parseSSEResponse([]byte(body), 7)
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Result) != `{"ok":true}` {
		t.Errorf("sse result wrong: %s", resp.Result)
	}
	if _, err := parseSSEResponse([]byte(body), 99); err == nil {
		t.Error("missing id must error")
	}
}

// quick helper to avoid importing strings just for one check
func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
