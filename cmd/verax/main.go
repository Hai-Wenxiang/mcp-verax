// Command verax is the entry point of the Verax toolkit: testing, security
// and quality tooling for MCP (Model Context Protocol) servers.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Hai-Wenxiang/mcp-verax/internal/report"
	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
	"github.com/Hai-Wenxiang/mcp-verax/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable core of the CLI; returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, "verax", version.Version)
		return 0
	case "help", "--help", "-h":
		usage(stdout)
		return 0
	case "test":
		return runTest(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "verax: unknown command %q\n\n", args[0])
		usage(stderr)
		return 2
	}
}

func runTest(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verax test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	update := fs.Bool("u", false, "accept changed/new snapshots (goldens rewritten)")
	verbose := fs.Bool("v", false, "show passing checks too")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(stderr, "verax test: at least one suite file is required")
		fmt.Fprintln(stderr, "usage: verax test [-u] [-v] <suite.yaml> [...]")
		return 2
	}

	var reports []*runner.SuiteReport
	for _, f := range files {
		reports = append(reports, runner.RunSuiteFile(f, runner.Options{
			UpdateSnapshots: *update,
		}))
	}
	report.RenderTerminal(stdout, reports, *verbose)

	for _, r := range reports {
		if !r.Pass() {
			return 1
		}
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprint(w, `verax — truth layer for the MCP ecosystem

Usage:
  verax <command> [flags]

Commands:
  test       Run MCP server test suites (YAML-defined)
               verax test [-u] [-v] <suite.yaml> [...]
               -u  accept snapshot changes (rewrites goldens)
               -v  verbose: show passing checks
  version    Print the verax version
  help       Show this help

Upcoming:
  scan       Security/quality rule packs (M2)

Docs: https://github.com/Hai-Wenxiang/mcp-verax
`)
}
