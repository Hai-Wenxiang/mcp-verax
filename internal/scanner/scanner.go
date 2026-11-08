package scanner

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxScanBytes bounds a single file we are willing to read.
const maxScanBytes = 4 << 20 // 4 MiB

// sourceExts are file types inspected line-by-line by source rules.
var sourceExts = map[string]bool{
	".js": true, ".cjs": true, ".mjs": true, ".ts": true, ".cts": true, ".mts": true,
	".py": true,
}

// ResolveTarget decides whether target is a local directory (with or without
// package.json) or something else (npm spec handled by npm.go later).
type TargetKind int

const (
	TargetLocal TargetKind = iota
	TargetNpm              // npm package spec — fetched via `npm pack`
)

func ResolveTarget(target string) (TargetKind, error) {
	if st, err := os.Stat(target); err == nil && st.IsDir() {
		return TargetLocal, nil
	}
	// npm scoped specs always contain '/', so the @-prefix wins before any
	// path-likeness heuristic.
	if strings.HasPrefix(target, "@") {
		return TargetNpm, nil
	}
	if strings.ContainsAny(target, "/\\") {
		return 0, fmt.Errorf("scanner: %s looks like a path but is not a directory", target)
	}
	return TargetNpm, nil
}

// Scan walks root and applies every registered rule.
func Scan(root string, target string) (*Result, error) {
	res := &Result{Target: target, Root: root}

	var pkgPath string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped, not fatal
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		name := d.Name()
		switch {
		case d.IsDir():
			if name == "node_modules" || name == ".git" || name == ".venv" || name == "__pycache__" {
				return fs.SkipDir
			}
			return nil
		case (name == "package.json" || name == "pyproject.toml") && (pkgPath == "" || len(rel) < len(pkgPath)):
			pkgPath = rel
		case sourceExts[strings.ToLower(filepath.Ext(name))]:
			data, rerr := os.ReadFile(path)
			if rerr != nil || len(data) > maxScanBytes {
				return nil
			}
			res.Files++
			lang := langOf(rel)
			for _, rule := range sourceRules {
				if rule.Lang != "" && rule.Lang != lang {
					continue
				}
				res.Findings = append(res.Findings, rule.Check(rel, string(data))...)
			}
		}
		return nil
	})

	if pkgPath != "" {
		if data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pkgPath))); err == nil {
			res.Files++
			if pkg, ok := parsePackageJSON(data); ok {
				res.Deps = pkg.Dependencies
			}
			for _, rule := range packageRules {
				res.Findings = append(res.Findings, rule.Check(pkgPath, data)...)
			}
			_ = 0
		}
	}
	if pkgPath == "" && res.Files > 0 {
		// source without package.json: still record QUA-105 with a synthetic root
		for _, rule := range packageRules {
			res.Findings = append(res.Findings, rule.Check("package.json", nil)...)
		}
	}

	// VX-QUA-106: README presence — the cheapest trust signal for a listing.
	if pkgPath != "" && !hasREADME(root) {
		res.Findings = append(res.Findings, Finding{
			RuleID: "VX-QUA-106", Title: "no README", Severity: SevLow,
			File: ".", Why: "published package ships without usage or intent documentation",
		})
	}
	return res, nil
}

// hasREADME reports whether a README exists at the scan root.
func hasREADME(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(strings.ToLower(e.Name()), "readme") {
			return true
		}
	}
	return false
}

// pkgSummary is the subset of package.json rules consume.
type pkgSummary struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	License      any               `json:"license"`
	Repository   any               `json:"repository"`
	Scripts      map[string]string `json:"scripts"`
	Dependencies map[string]string `json:"dependencies"`
}

func parsePackageJSON(data []byte) (pkgSummary, bool) {
	if len(data) == 0 {
		return pkgSummary{}, false
	}
	var p pkgSummary
	if err := json.Unmarshal(data, &p); err != nil {
		return pkgSummary{}, false
	}
	return p, true
}
