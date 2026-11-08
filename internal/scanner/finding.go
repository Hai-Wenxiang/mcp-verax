// Package scanner statically inspects MCP server packages (a local project
// directory or an npm-published tarball) for security, quality and privacy
// findings, and reduces them to a 0–100 score.
package scanner

import (
	"fmt"
	"strings"
)

// Severity drives both reporting order and score weight.
type Severity string

const (
	SevCritical Severity = "critical"
	SevHigh     Severity = "high"
	SevMedium   Severity = "medium"
	SevLow      Severity = "low"
)

// Weight returns the score deduction for one finding of this severity.
func (s Severity) Weight() int {
	switch s {
	case SevCritical:
		return 25
	case SevHigh:
		return 10
	case SevMedium:
		return 4
	case SevLow:
		return 1
	}
	return 0
}

// Finding is one concrete rule hit.
type Finding struct {
	RuleID   string
	Title    string
	Severity Severity
	File     string // path relative to the scan root
	Line     int    // 1-based; 0 = file-level (e.g. package.json metadata)
	Snippet  string // trimmed offending line ("" allowed)
	Why      string // what an attacker/accident could do
}

// Reason returns a human line for reports.
func (f Finding) Reason() string {
	loc := f.File
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return fmt.Sprintf("%s %s: %s (%s)", f.RuleID, loc, f.Title, f.Severity)
}

// Result is a full scan outcome.
type Result struct {
	Target   string            // resolved target description (dir or pkg@version)
	Root     string            // filesystem root that was scanned
	Files    int               // files inspected
	Deps     map[string]string // package.json dependencies (CVE enrichment input)
	Findings []Finding
}

// Score computes 100 minus accumulated weights, floored at 0.
func (r *Result) Score() int {
	score := 100
	for _, f := range r.Findings {
		score -= f.Severity.Weight()
	}
	if score < 0 {
		score = 0
	}
	return score
}

// CountsPerSeverity returns finding counts per severity level.
func (r *Result) CountsPerSeverity() map[Severity]int {
	m := map[Severity]int{}
	for _, f := range r.Findings {
		m[f.Severity]++
	}
	return m
}

// ParseSeverity maps a CLI severity name onto the scale; empty = low
// (report anything). Unknown names are errors.
func ParseSeverity(name string) (Severity, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "low":
		return SevLow, nil
	case "medium":
		return SevMedium, nil
	case "high":
		return SevHigh, nil
	case "critical":
		return SevCritical, nil
	}
	return "", fmt.Errorf("unknown severity %q (low|medium|high|critical)", name)
}
