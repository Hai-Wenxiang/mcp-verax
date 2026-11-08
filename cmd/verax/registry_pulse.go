package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/scanner"
)

// registry-pulse: pull the official MCP registry, scan a sample of its
// npm-packaged entries with verax's own scanner, and write a JSON + Markdown
// pulse report. Registry Intelligence v0.

type regEntry struct {
	Server struct {
		Name     string `json:"name"`
		Packages []struct {
			RegistryType string `json:"registryType"`
			Identifier   string `json:"identifier"`
			Version      string `json:"version"`
		} `json:"packages"`
	} `json:"server"`
}

type regPage struct {
	Servers  []regEntry `json:"servers"`
	Metadata struct {
		NextCursor string `json:"nextCursor"`
		Count      int    `json:"count"`
	} `json:"metadata"`
}

var npmNameRe = regexp.MustCompile(`^(@[a-z0-9-~][a-z0-9-._~]*/)?[a-z0-9-~][a-z0-9-._~]*$`)

// runRegistryPulse implements
// `verax registry-pulse [--pages N] [--sample N] [--out dir] [--registry URL]`.
func runRegistryPulse(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verax registry-pulse", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pages := fs.Int("pages", 3, "registry pages to pull (30 entries each)")
	sample := fs.Int("sample", 10, "max npm entries to scan")
	outDir := fs.String("out", "docs/registry", "output directory for pulse reports")
	registry := fs.String("registry", "https://registry.modelcontextprotocol.io", "registry base URL")
	jobs := fs.Int("j", 4, "concurrent package scans")
	all := fs.Bool("all", false, "pull every registry page (bounded by --max-pages)")
	maxPages := fs.Int("max-pages", 100, "hard page cap even with --all")
	pageDelay := fs.Duration("page-delay", 500*time.Millisecond, "throttle between registry pages")
	timeout := fs.Duration("http-timeout", 30*time.Second, "registry HTTP timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	client := &http.Client{Timeout: *timeout}

	// 1) pull pages
	var entries []regEntry
	cursor := ""
	pageLimit := *pages
	if *all {
		pageLimit = *maxPages
	}
	for p := 0; p < pageLimit; p++ {
		url := *registry + "/v0/servers?cursor=" + cursor
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(stderr, "verax registry-pulse: page %d: %v\n", p+1, err)
			return 1
		}
		var page regPage
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			fmt.Fprintf(stderr, "verax registry-pulse: decode page %d: %v\n", p+1, err)
			return 1
		}
		entries = append(entries, page.Servers...)
		if page.Metadata.NextCursor == "" || page.Metadata.Count == 0 {
			break
		}
		cursor = page.Metadata.NextCursor
		if *all {
			// be a polite crawler when walking the whole registry
			time.Sleep(*pageDelay)
		}
	}
	if *all {
		fmt.Fprintf(stdout, "registry: full walk pulled %d entries\n", len(entries))
	}

	// 2) pick npm-packaged entries (deterministic: name order)
	var npmEntries []regEntry
	for _, e := range entries {
		for _, pkg := range e.Server.Packages {
			if pkg.RegistryType == "npm" && npmNameRe.MatchString(pkg.Identifier) {
				npmEntries = append(npmEntries, e)
				break
			}
		}
	}
	sort.Slice(npmEntries, func(i, j int) bool { return npmEntries[i].Server.Name < npmEntries[j].Server.Name })
	// dedup by package identifier, keeping the highest published version
	// (the registry lists multiple versions of the same package as entries)
	dedup := npmEntries[:0]
	best := map[string]regEntry{}
	order := []string{}
	for _, e := range npmEntries {
		id := e.Server.Packages[0].Identifier
		cur, ok := best[id]
		if !ok {
			best[id] = e
			order = append(order, id)
			continue
		}
		if e.Server.Packages[0].Version > cur.Server.Packages[0].Version {
			best[id] = e
		}
	}
	for _, id := range order {
		dedup = append(dedup, best[id])
	}
	npmEntries = dedup
	if len(npmEntries) > *sample {
		npmEntries = npmEntries[:*sample]
	}

	fmt.Fprintf(stdout, "registry: %d entries pulled, %d npm-packaged, scanning %d\n", len(entries), len(npmEntries), len(npmEntries))

	// 3) scan each npm entry (published artifact)
	type row struct {
		Name     string            `json:"name"`
		Package  string            `json:"package"`
		Score    int               `json:"score"`
		Files    int               `json:"files"`
		Findings []scanner.Finding `json:"findings,omitempty"`
		Error    string            `json:"error,omitempty"`
	}
	// concurrent scans, results kept in input order
	rows := make([]row, len(npmEntries))
	sem := make(chan struct{}, maxInt(1, *jobs))
	var wg sync.WaitGroup
	for i, e := range npmEntries {
		pkg := e.Server.Packages[0]
		spec := pkg.Identifier
		if pkg.Version != "" {
			spec += "@" + pkg.Version
		}
		rows[i] = row{Name: e.Server.Name, Package: spec}
		wg.Add(1)
		go func(i int, name, spec string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			r := rows[i]
			root, cleanup, err := scanner.FetchNPMPackage(spec)
			if err != nil {
				r.Error = err.Error()
				rows[i] = r
				fmt.Fprintf(stdout, "  %-40s FETCH-ERR %v\n", name, err)
				return
			}
			res, err := scanner.Scan(root, spec)
			cleanup()
			if err != nil {
				r.Error = err.Error()
				rows[i] = r
				return
			}
			r.Score = res.Score()
			r.Files = res.Files
			r.Findings = res.Findings
			rows[i] = r
			fmt.Fprintf(stdout, "  %-40s %3d/100 (%d findings, %d files)\n", name, r.Score, len(r.Findings), r.Files)
		}(i, e.Server.Name, spec)
	}
	wg.Wait()

	// 4) write reports
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "verax registry-pulse: %v\n", err)
		return 1
	}
	date := time.Now().UTC().Format("2006-01-02")
	report := struct {
		Date     string `json:"date"`
		Entries  int    `json:"entries_pulled"`
		NpmTotal int    `json:"npm_packaged"`
		Scanned  int    `json:"scanned"`
		Rows     []row  `json:"rows"`
	}{Date: date, Entries: len(entries), NpmTotal: countNpm(entries), Scanned: len(rows), Rows: rows}
	raw, _ := json.MarshalIndent(report, "", "  ")
	jsonPath := *outDir + "/pulse-" + date + ".json"
	if err := os.WriteFile(jsonPath, raw, 0o644); err != nil {
		fmt.Fprintf(stderr, "verax registry-pulse: %v\n", err)
		return 1
	}

	mdPath := *outDir + "/pulse-" + date + ".md"
	var b strings.Builder
	fmt.Fprintf(&b, "# Registry Pulse — %s\n\n", date)
	fmt.Fprintf(&b, "Official registry entries pulled: **%d** · npm-packaged: **%d** · scanned: **%d**\n\n", report.Entries, report.NpmTotal, report.Scanned)
	fmt.Fprintf(&b, "| Server | Package | Score | Findings |\n|---|---|---|---|\n")
	for _, r := range rows {
		score := fmt.Sprintf("%d", r.Score)
		if r.Error != "" {
			score = "—"
		}
		fmt.Fprintf(&b, "| %s | `%s` | %s | %d |\n", r.Name, r.Package, score, len(r.Findings))
	}
	fmt.Fprintf(&b, "\n_Generated by `verax registry-pulse`; findings detail in the sibling JSON._\n")
	if err := os.WriteFile(mdPath, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintf(stderr, "verax registry-pulse: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s and %s\n", jsonPath, mdPath)
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func countNpm(entries []regEntry) int {
	n := 0
	for _, e := range entries {
		for _, pkg := range e.Server.Packages {
			if pkg.RegistryType == "npm" {
				n++
				break
			}
		}
	}
	return n
}
