package mcpclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/jsonrpc"
)

// echoCmd launches the fixture echo server via `go run` (toolchain present in
// dev and CI environments).
func echoCmd(t *testing.T) StdioConfig {
	t.Helper()
	return StdioConfig{Command: "go", Args: []string{"run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/echo"}}
}

func startEcho(t *testing.T) *StdioTransport {
	t.Helper()
	ctx := context.Background()
	tr, err := StartStdio(ctx, echoCmd(t))
	if err != nil {
		t.Fatalf("start echo: %v", err)
	}
	t.Cleanup(func() { _ = tr.Close() })
	return tr
}

func readWithTimeout(t *testing.T, tr *StdioTransport, d time.Duration) *jsonrpc.Envelope {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	env, err := tr.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return env
}

func TestStdioEchoRoundTrip(t *testing.T) {
	tr := startEcho(t)

	req, err := jsonrpc.NewRequest(jsonrpc.NewNumID(1), "tools/list", map[string]any{"cursor": ""})
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Send(req); err != nil {
		t.Fatal(err)
	}

	env := readWithTimeout(t, tr, 30*time.Second) // first call includes go run compile
	if env.IsRequest() {
		t.Fatalf("expected response, got request %+v", env)
	}
	if env.ID == nil || env.ID.Num != 1 {
		t.Fatalf("id mismatch: %+v", env.ID)
	}
	if !strings.Contains(string(env.Result), `"method":"tools/list"`) {
		t.Fatalf("echo payload missing method: %s", env.Result)
	}
	if !strings.Contains(string(env.Result), `"cursor"`) {
		t.Fatalf("echo payload missing params: %s", env.Result)
	}
}

func TestStdioNotificationThenRequest(t *testing.T) {
	tr := startEcho(t)

	note, err := jsonrpc.NewNotification("notifications/initialized", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Send(note); err != nil {
		t.Fatal(err)
	}

	req, _ := jsonrpc.NewRequest(jsonrpc.NewNumID(2), "ping", nil)
	if err := tr.Send(req); err != nil {
		t.Fatal(err)
	}

	// The only incoming message must be the response to id=2: the echo
	// server (like real MCP servers) never answers notifications.
	env := readWithTimeout(t, tr, 30*time.Second)
	if env.IsRequest() || env.ID == nil || env.ID.Num != 2 {
		t.Fatalf("expected response to id=2, got %+v", env)
	}
}

func TestStdioStderrCapture(t *testing.T) {
	tr := startEcho(t)

	req, _ := jsonrpc.NewRequest(jsonrpc.NewNumID(3), "stderr", nil)
	if err := tr.Send(req); err != nil {
		t.Fatal(err)
	}
	env := readWithTimeout(t, tr, 30*time.Second)
	if env.ID == nil || env.ID.Num != 3 {
		t.Fatalf("expected response to id=3: %+v", env)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, line := range tr.Stderr() {
			if strings.Contains(line, "simulated stderr output") {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("stderr not captured, got %v", tr.Stderr())
}

func TestStdioCrashDetection(t *testing.T) {
	tr := startEcho(t)

	req, _ := jsonrpc.NewRequest(jsonrpc.NewNumID(4), "crash", nil)
	if err := tr.Send(req); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := tr.Read(ctx)
	if err == nil {
		t.Fatal("expected read error after server crash")
	}
	if tr.ExitErr() == nil {
		t.Log("note: crash may race exit-err capture; read error is the contract")
	}
}

func TestStdioCloseIsIdempotent(t *testing.T) {
	tr := startEcho(t)
	if err := tr.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if err := tr.Send(nil); err == nil {
		t.Fatal("send after close must fail")
	}
}
