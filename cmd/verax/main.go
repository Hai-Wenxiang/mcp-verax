// Command verax is the entry point of the Verax toolkit: testing, security
// and quality tooling for MCP (Model Context Protocol) servers.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

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
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "rules":
		return runRules(args[1:], stdout, stderr)
	case "new":
		return runNew(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "registry-pulse":
		return runRegistryPulse(args[1:], stdout, stderr)
	case "leaderboard":
		return runLeaderboardCmd(args[1:], stdout, stderr)
	case "report":
		return runReportCmd(args[1:], stdout, stderr)
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
	watch := fs.Bool("watch", false, "re-run suites when their files change (dev mode)")
	repeat := fs.Int("repeat", 1, "run all suites N times and report per-case stability (flake detection)")
	jobs := fs.Int("j", 1, "run suites concurrently (1 = serial)")
	junitPath := fs.String("junit", "", "write a JUnit XML report to this path")
	htmlPath := fs.String("html", "", "write a single-file HTML report to this path")
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(stderr, "verax test: at least one suite file is required")
		fmt.Fprintln(stderr, "usage: verax test [-u] [-v] [--watch] [--junit path.xml] [--html path.html] <suite.yaml> [...]")
		return 2
	}

	runSuites := func() []*runner.SuiteReport {
		n := *jobs
		if n < 1 || n > len(files) {
			n = len(files)
		}
		if n <= 1 {
			var reports []*runner.SuiteReport
			for _, f := range files {
				reports = append(reports, runner.RunSuiteFile(f, runner.Options{UpdateSnapshots: *update}))
			}
			return reports
		}
		reports := make([]*runner.SuiteReport, len(files))
		sem := make(chan struct{}, n)
		var wg sync.WaitGroup
		for i, f := range files {
			wg.Add(1)
			go func(i int, f string) {
				defer wg.Done()
				sem <- struct{}{}
				reports[i] = runner.RunSuiteFile(f, runner.Options{UpdateSnapshots: *update})
				<-sem
			}(i, f)
		}
		wg.Wait()
		return reports
	}
	runOnce := func() int {
		reports := runSuites()
		report.RenderTerminal(stdout, reports, *verbose)

		if *junitPath != "" {
			if err := writeReport(*junitPath, func(f *os.File) error {
				return report.RenderJUnit(f, reports, time.Now())
			}); err != nil {
				fmt.Fprintf(stderr, "verax test: junit report: %v\n", err)
				return 1
			}
			fmt.Fprintf(stdout, "junit report: %s\n", *junitPath)
		}
		if *htmlPath != "" {
			if err := writeReport(*htmlPath, func(f *os.File) error {
				return report.RenderHTML(f, reports, time.Now())
			}); err != nil {
				fmt.Fprintf(stderr, "verax test: html report: %v\n", err)
				return 1
			}
			fmt.Fprintf(stdout, "html report: %s\n", *htmlPath)
		}

		for _, r := range reports {
			if !r.Pass() {
				return 1
			}
		}
		return 0
	}

	if *watch {
		return watchLoop(files, runOnce, stdout)
	}
	if *repeat > 1 {
		st := newStability(*repeat)
		for i := 0; i < *repeat; i++ {
			reports := runSuites()
			report.RenderTerminal(stdout, reports, *verbose)
			st.record(reports)
		}
		st.render(stdout)
		if st.failed {
			return 1
		}
		return 0
	}
	return runOnce()
}

func writeReport(path string, render func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return render(f)
}

// reorderFlags moves flags before positional files so that
// `verax test suite.yaml --junit out.xml` works like flag-idiom users expect
// (Go's flag package stops parsing at the first positional).
func reorderFlags(args []string) []string {
	valueFlags := map[string]bool{
		"--junit": true, "--html": true, "--sarif": true, "--min-score": true, "--dir": true, "--badge": true, "--json": true, "--repeat": true, "--fail-level": true, "-j": true, "--jobs": true,
	}
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		isValueFlag := valueFlags[a]
		isEqFlag := strings.HasPrefix(a, "--junit=") || strings.HasPrefix(a, "--html=") ||
			strings.HasPrefix(a, "--sarif=") || strings.HasPrefix(a, "--min-score=") || strings.HasPrefix(a, "--dir=") || strings.HasPrefix(a, "--badge=") || strings.HasPrefix(a, "--json=") || strings.HasPrefix(a, "--repeat=") || strings.HasPrefix(a, "--fail-level=")
		switch {
		case isValueFlag:
			if i+1 < len(args) {
				flags = append(flags, a, args[i+1])
				i++
			} else {
				flags = append(flags, a)
			}
		case isEqFlag:
			flags = append(flags, a)
		case strings.HasPrefix(a, "-"):
			flags = append(flags, a)
		default:
			positional = append(positional, a)
		}
	}
	return append(flags, positional...)
}

func usage(w io.Writer) {
	fmt.Fprint(w, `verax — truth layer for the MCP ecosystem

Usage:
  verax <command> [flags]

Commands:
  test       Run MCP server test suites (YAML-defined)
               verax test [-u] [-v] [--watch] [--repeat N] [-j N] <suite.yaml> [...]
  scan       Statically scan a server (local dir or npm package) — 16+ rules
               verax scan [--sarif x] [--json x] [--badge x] [--min-score N] <target>
  init       Generate a starter suite from a running server
               verax init <suite.yaml> -- <server command> [args...]
  new        Scaffold a server from a template
               verax new <template> <name> [--dir base]
  rules      List scanner rules
  registry-pulse  Pull the official registry and scan a sample of its npm packages
  leaderboard     Regenerate the leaderboard page from pulse data
  report     Aggregate pulse data into an ecosystem report (md + html)
  version    Print the verax version
  help       Show this help

Docs: https://github.com/Hai-Wenxiang/mcp-verax
`)
}
