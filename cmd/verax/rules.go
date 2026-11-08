package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/Hai-Wenxiang/mcp-verax/internal/scanner"
)

// runRules implements `verax rules`: the scanner rule catalog.
func runRules(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verax rules", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	fmt.Fprintf(stdout, "%-14s %-9s %-8s %-6s %s\n", "RULE", "SEVERITY", "LANG", "KIND", "TITLE")
	for _, r := range scanner.AllRules() {
		fmt.Fprintf(stdout, "%-14s %-9s %-8s %-6s %s\n", r.ID, r.Severity, r.Lang, r.Kind, r.Title)
	}
	fmt.Fprintf(stdout, "\nseverity weights: critical 25 / high 10 / medium 4 / low 1 (score floors at 0)\n")
	return 0
}
