package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/assert"
	"github.com/Hai-Wenxiang/mcp-verax/internal/dsl"
)

// honeypotToken is substituted in SideEffectProbe.Args values with the
// per-run honeypot directory path.
const honeypotToken = "${VERAX_HONEYROOT}"

// fileTree snapshots path -> content hash of every regular file under root.
func fileTree(root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		sum := sha256.Sum256(data)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:8])
		return nil
	})
	return out
}

func treeDiff(before, after map[string]string) []string {
	var out []string
	for p, h := range after {
		old, existed := before[p]
		switch {
		case !existed:
			out = append(out, "created "+p)
		case old != h:
			out = append(out, "modified "+p)
		}
	}
	for p := range before {
		if _, still := after[p]; !still {
			out = append(out, "deleted "+p)
		}
	}
	sort.Strings(out)
	return out
}

// runSideEffectProbe calls every tool with arguments built from the
// template (honeypot path substituted into string values), diffing the
// honeypot tree around each call: writes by allegedly read-only tools are
// the classic lying-server failure this catches.
func runSideEffectProbe(cli mcpConn, cfg dsl.SideEffectProbe) assert.CheckResult {
	hang := cfg.HangMS
	if hang <= 0 {
		hang = 8000
	}

	root, err := os.MkdirTemp("", "verax-honey-")
	if err != nil {
		return assert.CheckResult{Desc: "probe(side_effect)", Pass: false, Detail: "honeypot: " + err.Error()}
	}
	defer func() { _ = os.RemoveAll(root) }()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	tools, err := cli.ListTools(ctx)
	if err != nil {
		return assert.CheckResult{Desc: "probe(side_effect)", Pass: false, Detail: "tools/list: " + err.Error()}
	}

	argsFor := func(schema []byte) map[string]any {
		props := stringProperties(schema)
		if len(props) == 0 {
			props = []string{"input"}
		}
		out := map[string]any{}
		for _, p := range props {
			if v, ok := cfg.Args[p]; ok {
				out[p] = strings.ReplaceAll(v, honeypotToken, root)
				continue
			}
			out[p] = filepath.Join(root, "probe.txt")
		}
		return out
	}

	var offenders []string
	probed := 0
	for _, tool := range tools.Tools {
		if probed >= 20 {
			break
		}
		probed++
		before := fileTree(root)
		cctx, ccancel := context.WithTimeout(context.Background(), time.Duration(hang)*time.Millisecond)
		_, _ = cli.CallTool(cctx, tool.Name, argsFor(tool.InputSchema))
		ccancel()
		if diff := treeDiff(before, fileTree(root)); len(diff) > 0 {
			offenders = append(offenders, fmt.Sprintf("%s (%s)", tool.Name, strings.Join(diff[:min(3, len(diff))], ", ")))
		}
	}

	res := assert.CheckResult{
		Desc: fmt.Sprintf("probe(side_effect): %d tools probed, %d wrote to honeypot", probed, len(offenders)),
		Pass: len(offenders) == 0,
	}
	if len(offenders) > 0 {
		res.Detail = fmt.Sprintf("tools with side effects: %v — declare them or make writes explicit", offenders)
	}
	return res
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
