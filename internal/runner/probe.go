package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/assert"
	"github.com/Hai-Wenxiang/mcp-verax/internal/dsl"
)

// ssrfHits is a local honeyport: it counts inbound TCP connections made by
// the server under test after we inject the listener URL into tool args.
type ssrfHits struct {
	listener net.Listener
	url      string
	count    atomic.Int64
	stop     chan struct{}
}

func startSSRFListener() (*ssrfHits, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	h := &ssrfHits{listener: l, url: fmt.Sprintf("http://%s/verax-ssrf-probe", l.Addr().String()), stop: make(chan struct{})}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			h.count.Add(1)
			// drain a little so the dialer's request write succeeds cleanly
			go func(c net.Conn) {
				_ = c.SetDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 512)
				_, _ = c.Read(buf)
				_, _ = c.Write([]byte("HTTP/1.1 204 No Content\r\nContent-Length: 0\r\n\r\n"))
				_ = c.Close()
			}(conn)
		}
	}()
	return h, nil
}

func (h *ssrfHits) close() {
	close(h.stop)
	_ = h.listener.Close()
}

// runSSRFProbe injects the honeyport URL into every string property of every
// tool schema (one call per property) and reports tools whose call made the
// server dial the listener. Tool-level errors are fine — the outbound
// attempt itself is the signal.
func runSSRFProbe(cli mcpConn, cfg dsl.SSRFProbe) (assert.CheckResult, []string) {
	hang := cfg.HangMS
	if hang <= 0 {
		hang = 5000
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tools, err := cli.ListTools(ctx)
	if err != nil {
		return assert.CheckResult{Desc: "probe(ssrf)", Pass: false, Detail: "tools/list: " + err.Error()}, nil
	}

	h, err := startSSRFListener()
	if err != nil {
		return assert.CheckResult{Desc: "probe(ssrf)", Pass: false, Detail: "listener: " + err.Error()}, nil
	}
	defer h.close()

	var offenders []string
	for _, tool := range tools.Tools {
		props := stringProperties(tool.InputSchema)
		if len(props) == 0 {
			props = []string{"input"} // schema-less tools still get one shot
		}
		for _, prop := range props {
			args := map[string]any{prop: h.url}
			cctx, ccancel := context.WithTimeout(context.Background(), time.Duration(hang)*time.Millisecond)
			_, _ = cli.CallTool(cctx, tool.Name, args)
			ccancel()
			if h.count.Load() > 0 {
				offenders = append(offenders, fmt.Sprintf("%s.%s", tool.Name, prop))
				break // one hit per tool is enough evidence
			}
		}
		if h.count.Load() > 0 && len(offenders) >= 3 {
			break // cap evidence collection
		}
	}
	_ = h.listener.Close() // stop accepting (h.close's channel close stays with defer)

	hits := h.count.Load()
	res := assert.CheckResult{
		Desc: fmt.Sprintf("probe(ssrf): %d tools probed, %d outbound hits", len(tools.Tools), hits),
		Pass: hits == 0,
	}
	if hits > 0 {
		res.Detail = fmt.Sprintf("server fetched injected URLs via: %v — SSRF surface (validate destinations against an allowlist)", offenders)
	}
	return res, offenders
}

// stringProperties extracts top-level string-typed property names from a
// tool input schema.
func stringProperties(schema []byte) []string {
	if len(schema) == 0 {
		return nil
	}
	var s struct {
		Properties map[string]struct {
			Type string `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		return nil
	}
	var out []string
	for name, p := range s.Properties {
		if p.Type == "string" {
			out = append(out, name)
		}
	}
	return out
}

// runProbeIfConfigured appends probe cases (SSRF, side-effect) as requested.
func runProbeIfConfigured(s *dsl.Suite, cli mcpConn, report *SuiteReport) {
	if s.Probe == nil {
		return
	}
	if s.Probe.SSRF != nil {
		check, _ := runSSRFProbe(cli, *s.Probe.SSRF)
		report.Cases = append(report.Cases, CaseResult{
			Name:   "probe(ssrf)",
			Status: statusFromChecks([]assert.CheckResult{check}),
			Checks: []assert.CheckResult{check},
		})
	}
	if s.Probe.SideEffect != nil {
		check := runSideEffectProbe(cli, *s.Probe.SideEffect)
		report.Cases = append(report.Cases, CaseResult{
			Name:   "probe(side_effect)",
			Status: statusFromChecks([]assert.CheckResult{check}),
			Checks: []assert.CheckResult{check},
		})
	}
}
