package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Hai-Wenxiang/mcp-verax/internal/scanner"
)

// runScan implements `verax scan <local-dir | npm-package-spec> [--sarif out.json]`.
// Exit codes: 0 clean or low-only findings · 1 any critical/high finding · 2 usage.
func runScan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verax scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sarifPath := fs.String("sarif", "", "write a SARIF 2.1.0 report to this path")
	noOSV := fs.Bool("no-osv", false, "skip dependency CVE enrichment (osv.dev)")
	minScore := fs.Int("min-score", 0, "exit 1 when the score drops below this threshold (0-100)")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	targets := fs.Args()
	if len(targets) != 1 {
		fmt.Fprintln(stderr, "verax scan: exactly one target required (local directory or npm package spec)")
		fmt.Fprintln(stderr, "usage: verax scan [--sarif out.json] <dir | @scope/pkg | pkg@version>")
		return 2
	}
	target := targets[0]

	kind, err := scanner.ResolveTarget(target)
	if err != nil {
		fmt.Fprintf(stderr, "verax scan: %v\n", err)
		return 2
	}

	var res *scanner.Result
	switch kind {
	case scanner.TargetLocal:
		res, err = scanner.Scan(target, target)
	default: // TargetNpm
		var root string
		var cleanup func()
		root, cleanup, err = scanner.FetchNPMPackage(target)
		if err != nil {
			fmt.Fprintf(stderr, "verax scan: %v\n", err)
			return 1
		}
		defer cleanup()
		res, err = scanner.Scan(root, target)
	}
	if err != nil {
		fmt.Fprintf(stderr, "verax scan: %v\n", err)
		return 1
	}

	if !*noOSV {
		if err := scanner.CheckOSV(res, scanner.NewOSVClient()); err != nil {
			fmt.Fprintf(stderr, "verax scan: osv enrichment unavailable (skipped): %v\n", err)
		}
	}

	scanner.RenderTerminal(stdout, res)

	if *sarifPath != "" {
		f, ferr := os.Create(*sarifPath)
		if ferr != nil {
			fmt.Fprintf(stderr, "verax scan: sarif: %v\n", ferr)
			return 1
		}
		if ferr := scanner.RenderSARIF(f, res); ferr != nil {
			f.Close()
			fmt.Fprintf(stderr, "verax scan: sarif: %v\n", ferr)
			return 1
		}
		f.Close()
		fmt.Fprintf(stdout, "sarif report: %s\n", *sarifPath)
	}

	counts := res.CountsPerSeverity()
	if counts[scanner.SevCritical] > 0 || counts[scanner.SevHigh] > 0 {
		return 1
	}
	if *minScore > 0 && res.Score() < *minScore {
		fmt.Fprintf(stderr, "verax scan: score %d below --min-score %d\n", res.Score(), *minScore)
		return 1
	}
	return 0
}
