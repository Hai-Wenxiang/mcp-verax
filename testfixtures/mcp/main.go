// Command mcp is a minimal but well-behaved MCP server over stdio used as a
// test target by Verax's own tests. It implements the initialize handshake,
// tools/resources/prompts primitives, and fault switches via environment:
//
//	MCP_FAKE_VERSION   respond this protocolVersion instead of the default
//	MCP_SLOW_MS        "slow" tool sleeps this long (default 0)
//	MCP_NOTIFY         when "1", server emits notifications/message after initialized
//	MCP_DROP_HANDSHAKE when "1", never responds to initialize (hang tests)
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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

func main() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	initialized := false
	for sc.Scan() {
		var env_ envelope
		if err := json.Unmarshal(sc.Bytes(), &env_); err != nil {
			continue
		}
		isNotification := len(env_.ID) == 0 || string(env_.ID) == "null"

		switch env_.Method {
		case "initialize":
			if isNotification {
				continue
			}
			if os.Getenv("MCP_DROP_HANDSHAKE") == "1" {
				continue // swallow: handshake hangs
			}
			respond(w, env_.ID, map[string]any{
				"protocolVersion": env("MCP_FAKE_VERSION", defaultVersion),
				"capabilities": map[string]any{
					"tools":     map[string]any{"listChanged": true},
					"resources": map[string]any{"subscribe": true},
					"prompts":   map[string]any{"listChanged": true},
					"logging":   map[string]any{},
				},
				"serverInfo":   map[string]any{"name": "mcp-fixture", "version": "0.1.0"},
				"instructions": "Fixture server for Verax self-tests.",
			})
		case "notifications/initialized":
			initialized = true
			if os.Getenv("MCP_NOTIFY") == "1" {
				notify(w, "notifications/message", map[string]any{
					"level": "info", "data": "fixture initialized",
				})
			}
		case "ping":
			if !isNotification {
				respond(w, env_.ID, map[string]any{})
			}
		case "tools/list":
			if isNotification {
				continue
			}
			respond(w, env_.ID, map[string]any{
				"tools": []any{
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
				},
			})
		case "tools/call":
			if isNotification {
				continue
			}
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			_ = json.Unmarshal(env_.Params, &p)
			switch p.Name {
			case "echo":
				respond(w, env_.ID, map[string]any{
					"content": []any{map[string]any{"type": "text", "text": fmt.Sprint(p.Arguments["msg"])}},
				})
			case "fail":
				respond(w, env_.ID, map[string]any{
					"content": []any{map[string]any{"type": "text", "text": "tool failed on purpose"}},
					"isError": true,
				})
			case "slow":
				if d, err := time.ParseDuration(env("MCP_SLOW_MS", "0") + "ms"); err == nil && d > 0 {
					time.Sleep(d)
				}
				respond(w, env_.ID, map[string]any{
					"content": []any{map[string]any{"type": "text", "text": "slow done"}},
				})
			default:
				respondErr(w, env_.ID, -32631, "tool not found: "+p.Name, nil)
			}
		case "resources/list":
			if isNotification {
				continue
			}
			respond(w, env_.ID, map[string]any{
				"resources": []any{
					map[string]any{"uri": "fixture://readme", "name": "readme", "mimeType": "text/plain"},
				},
			})
		case "prompts/list":
			if isNotification {
				continue
			}
			respond(w, env_.ID, map[string]any{
				"prompts": []any{
					map[string]any{"name": "greet", "description": "Say hi"},
				},
			})
		default:
			if !isNotification {
				respondErr(w, env_.ID, -32601, "method not found: "+env_.Method, nil)
			}
		}
	}
	_ = initialized
}

func respond(w *bufio.Writer, id json.RawMessage, result any) {
	write(w, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func respondErr(w *bufio.Writer, id json.RawMessage, code int, msg string, data any) {
	write(w, map[string]any{
		"jsonrpc": "2.0", "id": id,
		"error": map[string]any{"code": code, "message": msg, "data": data},
	})
}

func notify(w *bufio.Writer, method string, params any) {
	write(w, map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func write(w *bufio.Writer, v any) {
	out, err := json.Marshal(v)
	if err != nil {
		return
	}
	w.Write(out)
	w.WriteByte('\n')
	w.Flush()
}
