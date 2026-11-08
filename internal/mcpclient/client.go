package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/jsonrpc"
)

// HandshakeError distinguishes initialize-phase failures (bad version, no
// response) from runtime call failures.
type HandshakeError struct {
	Msg string
}

func (e *HandshakeError) Error() string { return "mcpclient: handshake: " + e.Msg }

// Notification is a server-initiated message (no ID): e.g. logging or
// progress notifications.
type Notification struct {
	Method string
	Params json.RawMessage
}

// Client is a coroutine-safe MCP client over a stdio transport: it performs
// request/response correlation, exposes Call/Notify and a notification
// callback, and drives the initialize handshake.
type Client struct {
	tr   *StdioTransport
	opts Options

	nextID   atomic.Int64
	mu       sync.Mutex
	pending  map[int64]chan *jsonrpc.Response
	notifyFn func(Notification)

	readDone chan struct{}
	readErr  error

	info InitializeResult

	closeOnce sync.Once
	closeErr  error
}

// InitializeResult returns the server's handshake payload (protocol version,
// capabilities, serverInfo). Valid after Connect succeeds.
func (c *Client) InitializeResult() InitializeResult { return c.info }

// Options tunes Client behaviour.
type Options struct {
	// ProtocolVersion overrides the proposed version (default: latest).
	ProtocolVersion string
	// HandshakeTimeout bounds the initialize exchange (default 15s).
	HandshakeTimeout time.Duration
	// CallTimeout bounds each Call when ctx has no deadline (default 60s).
	CallTimeout time.Duration
	// OnNotification is invoked (from the reader goroutine) for
	// server-initiated notifications. Nil drops them.
	OnNotification func(Notification)
}

// Connect starts the server, performs the MCP initialize handshake and
// returns a ready-to-use client.
func Connect(ctx context.Context, cfg StdioConfig, opts Options) (*Client, error) {
	if opts.ProtocolVersion == "" {
		opts.ProtocolVersion = LatestProtocolVersion()
	}
	if opts.HandshakeTimeout <= 0 {
		opts.HandshakeTimeout = 15 * time.Second
	}
	if opts.CallTimeout <= 0 {
		opts.CallTimeout = 60 * time.Second
	}

	tr, err := StartStdio(ctx, cfg)
	if err != nil {
		return nil, err
	}
	c := &Client{
		tr:       tr,
		opts:     opts,
		pending:  make(map[int64]chan *jsonrpc.Response),
		notifyFn: opts.OnNotification,
		readDone: make(chan struct{}),
	}
	go c.readLoop()

	if err := c.handshake(ctx); err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) handshake(ctx context.Context) error {
	hsCtx, cancel := context.WithTimeout(ctx, c.opts.HandshakeTimeout)
	defer cancel()

	var params InitializeParams
	params.ProtocolVersion = c.opts.ProtocolVersion
	params.Capabilities.Roots.ListChanged = false
	params.ClientInfo = ClientInfo

	resp, err := c.Call(hsCtx, "initialize", params)
	if err != nil {
		return &HandshakeError{Msg: err.Error()}
	}
	if resp.Error != nil {
		return &HandshakeError{Msg: fmt.Sprintf("initialize rejected: %v", resp.Error)}
	}
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return &HandshakeError{Msg: fmt.Sprintf("decode initialize result: %v", err)}
	}
	c.info = result
	if !c.versionSupported(result.ProtocolVersion) {
		return &HandshakeError{Msg: fmt.Sprintf(
			"server protocol version %q not supported by verax (supported: %v)",
			result.ProtocolVersion, SupportedProtocolVersions)}
	}
	if err := c.Notify(context.Background(), "notifications/initialized", nil); err != nil {
		return &HandshakeError{Msg: err.Error()}
	}
	return nil
}

func (c *Client) versionSupported(v string) bool {
	for _, s := range SupportedProtocolVersions {
		if s == v {
			return true
		}
	}
	return false
}

// readLoop routes incoming messages to pending calls or the notification
// callback. Runs until the transport stream ends.
func (c *Client) readLoop() {
	defer close(c.readDone)
	for {
		env, err := c.tr.Read(context.Background())
		if err != nil {
			c.readErr = err
			c.failAllPending(err)
			return
		}
		if env.IsRequest() {
			if c.notifyFn != nil {
				c.notifyFn(Notification{Method: env.Method, Params: env.Params})
			}
			continue
		}
		if env.ID == nil {
			continue // malformed response without id: drop
		}
		c.mu.Lock()
		ch, ok := c.pending[env.ID.Num]
		if ok {
			delete(c.pending, env.ID.Num)
		}
		c.mu.Unlock()
		if ok {
			ch <- env.AsResponse()
		}
	}
}

func (c *Client) failAllPending(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.pending {
		delete(c.pending, id)
		resp := jsonrpc.NewErrorResponse(jsonrpc.NewNumID(id), jsonrpc.CodeInternalError, err.Error(), nil)
		ch <- resp
	}
}

// Call sends a request and waits for its correlated response.
func (c *Client) Call(ctx context.Context, method string, params any) (*jsonrpc.Response, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.opts.CallTimeout)
		defer cancel()
	}

	id := jsonrpc.NewNumID(c.nextID.Add(1))
	req, err := jsonrpc.NewRequest(id, method, params)
	if err != nil {
		return nil, fmt.Errorf("mcpclient: build %s: %w", method, err)
	}

	ch := make(chan *jsonrpc.Response, 1)
	c.mu.Lock()
	c.pending[id.Num] = ch
	c.mu.Unlock()

	if err := c.tr.Send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id.Num)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id.Num)
		c.mu.Unlock()
		return nil, fmt.Errorf("mcpclient: %s: %w", method, ctx.Err())
	}
}

// Notify sends a notification (no response expected).
func (c *Client) Notify(ctx context.Context, method string, params any) error {
	note, err := jsonrpc.NewNotification(method, params)
	if err != nil {
		return err
	}
	return c.tr.Send(note)
}

// Close tears down the transport exactly once.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.closeErr = c.tr.Close()
	})
	return c.closeErr
}

// Stderr exposes the server's captured stderr lines.
func (c *Client) Stderr() []string { return c.tr.Stderr() }

// CleanExit reports whether the server process exited by itself after stdin
// close (valid after Close; false also before Close).
func (c *Client) CleanExit() bool { return c.tr.CleanExit() }
