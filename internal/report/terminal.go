// Package report renders run results for humans (terminal) and machines
// (JUnit/HTML arrive in F7).
package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

// RenderTerminal writes a plain-ASCII report (no ANSI, portable everywhere).
// Failed checks always show details; passing checks only in verbose mode.
func RenderTerminal(w io.Writer, reports []*runner.SuiteReport, verbose bool) {
	totalCases, passCases := 0, 0
	for _, r := range reports {
		fmt.Fprintf(w, "== %s (%s) ==\n", r.Suite, r.Source)
		suitePass := 0
		for _, c := range r.Cases {
			totalCases++
			marker := "PASS"
			switch c.Status {
			case runner.StatusPass:
				passCases++
				suitePass++
			case runner.StatusFail:
				marker = "FAIL"
			case runner.StatusError:
				marker = "ERR "
			}
			fmt.Fprintf(w, "  [%s] %s (%dms)\n", marker, c.Name, c.DurationMS)

			if c.Err != "" {
				fmt.Fprintf(w, "        error: %s\n", indent(c.Err, 8+7))
			}
			for _, ck := range c.Checks {
				if ck.Pass && !verbose {
					continue
				}
				icon := "✓"
				if !ck.Pass {
					icon = "✗"
				}
				fmt.Fprintf(w, "        %s %s\n", icon, ck.Desc)
				if ck.Detail != "" {
					fmt.Fprintf(w, "          %s\n", indent(ck.Detail, 10))
				}
			}
			for _, line := range c.StderrTail {
				fmt.Fprintf(w, "        stderr: %s\n", line)
			}
		}
		fmt.Fprintf(w, "  %d/%d cases passed (%dms)\n\n", suitePass, len(r.Cases), r.DurationMS)
	}

	verdict := "OK"
	if passCases != totalCases {
		verdict = "FAILED"
	}
	fmt.Fprintf(w, "verax: %s — %d/%d cases passed across %d suite(s)\n",
		verdict, passCases, totalCases, len(reports))
}

func indent(s string, pad int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.Repeat(" ", 0) + l
	}
	return strings.Join(lines, "\n"+strings.Repeat(" ", pad))
}
