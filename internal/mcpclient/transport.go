// Package mcpclient implements the MCP client side used by the Verax runner:
// a stdio transport that speaks line-delimited JSON-RPC 2.0 to a spawned
// server process, and (in client.go) the MCP handshake and primitives.
package mcpclient

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/jsonrpc"
)

// StdioConfig describes how to launch an MCP server over stdio.
type StdioConfig struct {
	Command string
	Args    []string
	Env     []string // extra env vars (merged over os.Environ); VERAX_* reserved
	Dir     string   // working directory ("" = inherit)
}

// StdioTransport is a line-delimited JSON-RPC pipe to a child process.
// It is deliberately dumb: Send writes one message, Read blocks for the next
// incoming message (response or server-initiated request/notification).
// Concurrency: Send is goroutine-safe; Read must be called from a single
// reader goroutine.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	enc    *jsonrpc.Encoder
	dec    *jsonrpc.Decoder

	mu       sync.Mutex
	closed   bool
	stderrCh chan string
	stderr   []string
	waitErr  error
	waitDone chan struct{}
}

// StartStdio spawns the server process and wires up its stdio.
func StartStdio(ctx context.Context, cfg StdioConfig) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	if len(cfg.Env) > 0 {
		cmd.Env = append(cmd.Environ(), cfg.Env...)
	}
	cmd.Dir = cfg.Dir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcpclient: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcpclient: stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("mcpclient: stderr pipe: %w", err)
	}

	t := &StdioTransport{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		enc:      jsonrpc.NewEncoder(stdin),
		dec:      jsonrpc.NewDecoder(stdout),
		stderrCh: make(chan string, 64),
		waitDone: make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcpclient: start %s: %w", cfg.Command, err)
	}
	go t.drainStderr(stderrPipe)
	go func() {
		t.waitErr = cmd.Wait()
		close(t.waitDone)
	}()
	return t, nil
}

// Send writes one JSON-RPC message.
func (t *StdioTransport) Send(msg *jsonrpc.Request) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("mcpclient: transport closed")
	}
	if err := t.enc.Write(msg); err != nil {
		return fmt.Errorf("mcpclient: send %s: %w", msg.Method, err)
	}
	return nil
}

// Read blocks for the next incoming message from the server.
// It returns an error when the stream ends (process exit, close, context).
func (t *StdioTransport) Read(ctx context.Context) (*jsonrpc.Envelope, error) {
	type result struct {
		env *jsonrpc.Envelope
		err error
	}
	ch := make(chan result, 1)
	go func() {
		env, err := t.dec.Read()
		ch <- result{env, err}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("mcpclient: read: %w", r.err)
		}
		return r.env, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.waitDone:
		// Process died; drain any final buffered message before failing.
		select {
		case r := <-ch:
			if r.err != nil {
				return nil, fmt.Errorf("mcpclient: server exited: %w", t.waitErr)
			}
			return r.env, nil
		default:
			return nil, fmt.Errorf("mcpclient: server exited: %w", t.waitErr)
		}
	}
}

// Stderr returns the server's captured stderr lines so far.
func (t *StdioTransport) Stderr() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.stderr))
	copy(out, t.stderr)
	return out
}

// Close shuts stdin (an MCP-friendly graceful signal), waits briefly for the
// process to exit, then kills it. Always safe to call twice.
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	t.stdin.Close()
	select {
	case <-t.waitDone:
	case <-time.After(3 * time.Second):
		if err := t.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("mcpclient: kill after grace: %w", err)
		}
		<-t.waitDone
	}
	if t.waitErr != nil && t.waitErr.Error() != "wait: no child processes" {
		// Non-zero exit is surfaced, but Close itself is not an error for
		// servers that exit non-zero; the runner decides severity.
		_ = t.waitErr
	}
	return nil
}

// ExitErr returns the process wait error (nil if clean exit 0), only valid
// after the process has exited.
func (t *StdioTransport) ExitErr() error {
	select {
	case <-t.waitDone:
		return t.waitErr
	default:
		return nil
	}
}

func (t *StdioTransport) drainStderr(r io.Reader) {
	sc := newLineScanner(r)
	for sc.Scan() {
		line := sc.Text()
		t.mu.Lock()
		t.stderr = append(t.stderr, line)
		t.mu.Unlock()
		select {
		case t.stderrCh <- line:
		default:
		}
	}
}
