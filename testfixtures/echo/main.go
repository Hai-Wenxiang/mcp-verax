// Command echo is a minimal JSON-RPC 2.0 stdio server used as a test target
// by Verax's own tests. It echoes every request back as
// {"echo":{"method":...,"params":...}} and supports fault injection:
//
//	method "crash"  → exit(1) immediately
//	method "hang"   → sleep forever (for timeout/close tests)
//	method "stderr" → write a line to stderr, then echo normally
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func main() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var env struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(line, &env); err != nil {
			continue // malformed input: ignore, keep serving (fuzz-friendly)
		}

		switch env.Method {
		case "crash":
			w.Flush()
			os.Exit(1)
		case "hang":
			w.Flush()
			for {
				time.Sleep(time.Hour)
			}
		case "stderr":
			fmt.Fprintln(os.Stderr, "echo: simulated stderr output")
		}

		if len(env.ID) == 0 || string(env.ID) == "null" {
			continue // notification: no response
		}

		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(env.ID),
			"result": map[string]any{
				"echo": map[string]any{
					"method": env.Method,
					"params": json.RawMessage(orNull(env.Params)),
				},
			},
		}
		out, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		w.Write(out)
		w.WriteByte('\n')
		w.Flush()
	}
}

func orNull(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	return raw
}
