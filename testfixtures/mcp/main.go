// Command mcp is a minimal but well-behaved MCP server used as a test target
// by Verax's own tests. It speaks stdio by default; with MCP_HTTP_PORT set it
// serves Streamable-HTTP-style POST /mcp instead (JSON or SSE response).
//
// Fault/env switches:
//
//	MCP_FAKE_VERSION    respond this protocolVersion instead of the default
//	MCP_SLOW_MS         "slow" tool sleeps this long (default 0)
//	MCP_NOTIFY          when "1", emit notifications/message after initialized
//	MCP_DROP_HANDSHAKE  when "1", never respond to initialize (hang tests)
//	MCP_NO_PROMPTS      when "1", prompts capability/list absent
//	MCP_IGNORE_EOF      when "1", ignore stdin EOF (leak tests, stdio mode)
//	MCP_HTTP_PORT       serve HTTP on this port instead of stdio
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultVersion = "2025-06-18"

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type envelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// handleMessage processes one incoming message; emit is called zero or more
// times with complete response/notification objects.
func handleMessage(msg []byte, emit func(map[string]any)) {
	var e envelope
	if err := json.Unmarshal(msg, &e); err != nil {
		return
	}
	isNotification := len(e.ID) == 0 || string(e.ID) == "null"

	switch e.Method {
	case "initialize":
		if isNotification {
			return
		}
		if os.Getenv("MCP_DROP_HANDSHAKE") == "1" {
			fmt.Fprintln(os.Stderr, "fixture: handshake dropped (MCP_DROP_HANDSHAKE=1)")
			return
		}
		caps := map[string]any{
			"tools":     map[string]any{"listChanged": true},
			"resources": map[string]any{"subscribe": true},
			"prompts":   map[string]any{"listChanged": true},
			"logging":   map[string]any{},
		}
		if os.Getenv("MCP_NO_PROMPTS") == "1" {
			delete(caps, "prompts")
		}
		emit(map[string]any{
			"jsonrpc": "2.0",
			"id":      e.ID,
			"result": map[string]any{
				"protocolVersion": env("MCP_FAKE_VERSION", defaultVersion),
				"capabilities":    caps,
				"serverInfo":      map[string]any{"name": "mcp-fixture", "version": "0.1.0"},
				"instructions":    "Fixture server for Verax self-tests.",
			},
		})

	case "notifications/initialized":
		if os.Getenv("MCP_NOTIFY") == "1" {
			emit(map[string]any{
				"jsonrpc": "2.0",
				"method":  "notifications/message",
				"params":  map[string]any{"level": "info", "data": "fixture initialized"},
			})
		}

	case "ping":
		if !isNotification {
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{}})
		}

	case "tools/list":
		if isNotification {
			return
		}
		emit(map[string]any{
			"jsonrpc": "2.0",
			"id":      e.ID,
			"result": map[string]any{"tools": []any{
				map[string]any{
					"name":        "echo",
					"description": "Echo back its arguments.",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"msg": map[string]any{"type": "string"},
						},
						"required": []string{"msg"},
					},
				},
				map[string]any{
					"name":        "fail",
					"description": "Always returns a tool error.",
					"inputSchema": map[string]any{"type": "object"},
				},
				map[string]any{
					"name":        "slow",
					"description": "Sleeps then echoes.",
					"inputSchema": map[string]any{"type": "object"},
				},
				map[string]any{
					"name":        "env",
					"description": "Returns VERAX_PROBE (tests per-case env injection).",
					"inputSchema": map[string]any{"type": "object"},
				},
				map[string]any{
					"name":        "fetch_url",
					"description": "Fetches a URL server-side (SSRF probe target).",
					"inputSchema": map[string]any{
						"type":       "object",
						"properties": map[string]any{"url": map[string]any{"type": "string"}},
						"required":   []string{"url"},
					},
				},
			}},
		})

	case "tools/call":
		if isNotification {
			return
		}
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		_ = json.Unmarshal(e.Params, &p)
		switch p.Name {
		case "echo":
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
				"content": []any{map[string]any{"type": "text", "text": fmt.Sprint(p.Arguments["msg"])}},
			}})
		case "env":
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
				"content": []any{map[string]any{"type": "text", "text": "VERAX_PROBE=" + os.Getenv("VERAX_PROBE")}},
			}})
		case "fetch_url":
			var fu struct {
				Arguments struct {
					URL string `json:"url"`
				} `json:"arguments"`
			}
			_ = json.Unmarshal(e.Params, &fu)
			client := http.Client{Timeout: 3 * time.Second}
			resp, ferr := client.Get(fu.Arguments.URL)
			if ferr != nil {
				emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
					"content": []any{map[string]any{"type": "text", "text": "fetch error: " + ferr.Error()}},
					"isError": true,
				}})
				return
			}
			defer resp.Body.Close()
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
				"content": []any{map[string]any{"type": "text", "text": fmt.Sprintf("status %s", resp.Status)}},
			}})

		case "fail":
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
				"content": []any{map[string]any{"type": "text", "text": "tool failed on purpose"}},
				"isError": true,
			}})
		case "slow":
			if d, err := time.ParseDuration(env("MCP_SLOW_MS", "0") + "ms"); err == nil && d > 0 {
				time.Sleep(d)
			}
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
				"content": []any{map[string]any{"type": "text", "text": "slow done"}},
			}})
		default:
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "error": map[string]any{
				"code": -32631, "message": "tool not found: " + p.Name,
			}})
		}

	case "resources/list":
		if isNotification {
			return
		}
		emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
			"resources": []any{
				map[string]any{"uri": "fixture://readme", "name": "readme", "mimeType": "text/plain"},
			},
		}})

	case "prompts/list":
		if isNotification {
			return
		}
		if os.Getenv("MCP_NO_PROMPTS") == "1" {
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "error": map[string]any{
				"code": -32601, "message": "prompts not supported by this fixture",
			}})
			return
		}
		emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "result": map[string]any{
			"prompts": []any{map[string]any{"name": "greet", "description": "Say hi"}},
		}})

	default:
		if !isNotification {
			emit(map[string]any{"jsonrpc": "2.0", "id": e.ID, "error": map[string]any{
				"code": -32601, "message": "method not found: " + e.Method,
			}})
		}
	}
}

func main() {
	if port := os.Getenv("MCP_HTTP_PORT"); port != "" {
		serveHTTP(":" + port)
		return
	}
	serveStdio()
}

func serveStdio() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for sc.Scan() {
		handleMessage(sc.Bytes(), func(obj map[string]any) {
			out, err := json.Marshal(obj)
			if err != nil {
				return
			}
			w.Write(out)
			w.WriteByte('\n')
			w.Flush()
		})
	}
	// stdin closed. A well-behaved MCP server exits here; the leak switch
	// simulates servers that hang around (process-leak detection target).
	if os.Getenv("MCP_IGNORE_EOF") == "1" {
		for {
			time.Sleep(time.Hour)
		}
	}
}

// serveHTTP implements the POST /mcp surface of Streamable HTTP: one JSON-RPC
// message in, one JSON object (or SSE event) out. Notifications return 202.
func serveHTTP(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var msgs []json.RawMessage
		single := json.RawMessage{}
		if json.Unmarshal(body, &single) == nil && len(single) > 0 && single[0] == '{' {
			msgs = []json.RawMessage{single}
		} else if json.Unmarshal(body, &msgs) != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Session semantics (MCP_SESSION=1): issue a session id on the first
		// request and require it echoed afterwards — exercises the client's
		// Mcp-Session-Id handling.
		session := "verax-fixture-session-1"
		sent := r.Header.Get("Mcp-Session-Id")
		if os.Getenv("MCP_SESSION") == "1" {
			if sent == "" {
				w.Header().Set("Mcp-Session-Id", session)
			} else if sent != session {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}

		var responses []map[string]any
		for _, m := range msgs {
			handleMessage(m, func(obj map[string]any) { responses = append(responses, obj) })
		}
		if len(responses) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if len(responses) == 1 {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(responses[0])
			return
		}
		// Multiple messages (response + notifications): SSE framing.
		w.Header().Set("Content-Type", "text/event-stream")
		for _, resp := range responses {
			out, _ := json.Marshal(resp)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", out)
		}
	})
	_ = http.ListenAndServe(addr, mux)
}
