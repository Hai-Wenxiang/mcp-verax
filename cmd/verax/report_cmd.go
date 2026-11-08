package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/Hai-Wenxiang/mcp-verax/apps/report"
)

// runReportCmd: `verax report [--dir docs/registry]`.
func runReportCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verax report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "docs/registry", "directory containing pulse-*.json files")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	n, err := report.GenerateDir(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "verax report: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s/report.md and %s/report.html (%d pulses)\n", *dir, *dir, n)
	return 0
}
