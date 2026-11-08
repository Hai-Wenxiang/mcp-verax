// Command verax is the entry point of the Verax toolkit: testing, security
// and quality tooling for MCP (Model Context Protocol) servers.
package main

import (
	"fmt"
	"os"

	"github.com/Hai-Wenxiang/mcp-verax/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Println("verax", version.Version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "verax: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `verax — truth layer for the MCP ecosystem

Usage:
  verax <command> [flags]

Commands:
  version    Print the verax version
  help       Show this help

Upcoming in M1:
  test       Run MCP server test suites (YAML-defined)

Docs: https://github.com/Hai-Wenxiang/mcp-verax
`)
}
