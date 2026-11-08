package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/Hai-Wenxiang/mcp-verax/apps/leaderboard"
)

// runLeaderboardCmd: `verax leaderboard [--dir docs/registry] [--latest 14]`.
func runLeaderboardCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verax leaderboard", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", "docs/registry", "directory containing pulse-*.json files")
	latest := fs.Int("latest", 14, "how many recent pulses to include")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if err := leaderboard.GenerateDir(*dir, *latest); err != nil {
		fmt.Fprintf(stderr, "verax leaderboard: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s/index.html\n", *dir)
	return 0
}
