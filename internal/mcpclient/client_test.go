package mcpclient

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func connectFixture(t *testing.T, env ...string) *Client {
	t.Helper()
	cfg := StdioConfig{
		Command: "go",
		Args:    []string{"run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp"},
		Env:     env,
	}
	c, err := Connect(context.Background(), cfg, Options{HandshakeTimeout: 60 * time.Second})
	if err != nil {
		t.Fatalf("connect fixture: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestConnectHandshake(t *testing.T) {
	c := connectFixture(t)
	info := c.InitializeResult()
	if info.ServerInfo.Name != "mcp-fixture" {
		t.Errorf("serverInfo.name = %q", info.ServerInfo.Name)
	}
	if info.ProtocolVersion != LatestProtocolVersion() {
		t.Errorf("negotiated version = %q, want %q", info.ProtocolVersion, LatestProtocolVersion())
	}
	if info.Capabilities.Tools == nil {
		t.Error("tools capability not reported")
	}
}

func TestHandshakeRejectsUnknownVersion(t *testing.T) {
	cfg := StdioConfig{
		Command: "go",
		Args:    []string{"run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp"},
		Env:     []string{"MCP_FAKE_VERSION=1999-01-01"},
	}
	_, err := Connect(context.Background(), cfg, Options{HandshakeTimeout: 60 * time.Second})
	if err == nil {
		t.Fatal("expected handshake failure for bogus version")
	}
	var he *HandshakeError
	if !asHandshakeError(err, &he) {
		t.Fatalf("want HandshakeError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention unsupported version: %v", err)
	}
}

func TestHandshakeTimeoutOnDrop(t *testing.T) {
	cfg := StdioConfig{
		Command: "go",
		Args:    []string{"run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp"},
		Env:     []string{"MCP_DROP_HANDSHAKE=1"},
	}
	start := time.Now()
	_, err := Connect(context.Background(), cfg, Options{HandshakeTimeout: 2 * time.Second})
	if err == nil {
		t.Fatal("expected handshake timeout")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
	if !strings.Contains(err.Error(), "handshake") {
		t.Errorf("error should be a handshake error: %v", err)
	}
}

func TestCallCorrelationUnderConcurrency(t *testing.T) {
	c := connectFixture(t)
	const n = 16
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := c.Call(context.Background(), "ping", nil)
			if err != nil {
				errs <- err
				return
			}
			if resp.Error != nil {
				errs <- resp.Error
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent call failed: %v", err)
	}
}

func TestUnknownMethodReturnsRPCError(t *testing.T) {
	c := connectFixture(t)
	resp, err := c.Call(context.Background(), "bogus/method", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil {
		t.Fatalf("expected JSON-RPC error, got result %s", resp.Result)
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

func TestServerNotificationDelivered(t *testing.T) {
	notifications := make(chan Notification, 8)
	cfg := StdioConfig{
		Command: "go",
		Args:    []string{"run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp"},
		Env:     []string{"MCP_NOTIFY=1"},
	}
	c, err := Connect(context.Background(), cfg, Options{
		HandshakeTimeout: 60 * time.Second,
		OnNotification:   func(n Notification) { notifications <- n },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	select {
	case n := <-notifications:
		if n.Method != "notifications/message" {
			t.Errorf("notification method = %q", n.Method)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no server notification received within 10s")
	}
}

func asHandshakeError(err error, target **HandshakeError) bool {
	he, ok := err.(*HandshakeError)
	if ok {
		*target = he
	}
	return ok
}
