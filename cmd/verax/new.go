package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	verax "github.com/Hai-Wenxiang/mcp-verax"
)

// runNew implements `verax new <template> <name> [--dir base]`.
// Scaffolds a server from an embedded template: copies files, renames the
// package, and rewrites embedded suite paths from repo-relative to local.
func runNew(args []string, stdout, stderr io.Writer) int {
	fs_ := flag.NewFlagSet("verax new", flag.ContinueOnError)
	fs_.SetOutput(stderr)
	baseDir := fs_.String("dir", ".", "parent directory for the new project")
	if err := fs_.Parse(reorderFlags(args)); err != nil {
		return 2
	}
	rest := fs_.Args()
	if len(rest) != 2 {
		fmt.Fprintln(stderr, "usage: verax new <template> <name> [--dir base]")
		ids := verax.TemplateIDs()
		if len(ids) > 0 {
			fmt.Fprintf(stderr, "templates: %s\n", strings.Join(ids, ", "))
		}
		return 2
	}
	tmplID, name := rest[0], rest[1]

	if !validName(name) {
		fmt.Fprintf(stderr, "verax new: invalid project name %q (letters, digits, - and _ only)\n", name)
		return 2
	}

	tplFS := verax.Templates()
	if _, err := fs.Stat(tplFS, tmplID); err != nil {
		fmt.Fprintf(stderr, "verax new: unknown template %q; available: %s\n", tmplID, strings.Join(verax.TemplateIDs(), ", "))
		return 2
	}

	dest := filepath.Join(*baseDir, name)
	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(stderr, "verax new: %s already exists\n", dest)
		return 1
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		fmt.Fprintf(stderr, "verax new: %v\n", err)
		return 1
	}

	copied := 0
	err := fs.WalkDir(tplFS, tmplID, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel := strings.TrimPrefix(p, tmplID+"/")
		if rel == "" {
			return nil
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := fs.ReadFile(tplFS, p)
		if rerr != nil {
			return rerr
		}
		if strings.HasSuffix(p, ".yaml") {
			data = localizeSuite(data, tmplID)
		}
		if strings.HasSuffix(p, "package.json") {
			data = renamePackage(data, name)
		}
		if werr := os.MkdirAll(filepath.Dir(target), 0o755); werr != nil {
			return werr
		}
		if werr := os.WriteFile(target, data, 0o644); werr != nil {
			return werr
		}
		if strings.HasSuffix(target, "index.js") {
			_ = os.Chmod(target, 0o755) // shebanged entry
		}
		copied++
		return nil
	})
	if err != nil {
		fmt.Fprintf(stderr, "verax new: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "created %s from template %q (%d files)\n\n", dest, tmplID, copied)
	fmt.Fprintf(stdout, "next:\n  cd %s\n  VERAX_ROOT=./data node index.js        # run it\n  verax test tests/server.suite.yaml      # test it (adjust paths if needed)\n  verax scan .                            # expect 100/100\n", name)
	return 0
}

// localizeSuite rewrites repo-root-relative paths in embedded suites
// (templates/<id>/...) into scaffold-local ones (./...).
func localizeSuite(data []byte, tmplID string) []byte {
	s := string(data)
	s = strings.ReplaceAll(s, "templates/"+tmplID+"/index.js", "index.js")
	s = strings.ReplaceAll(s, "VERAX_ROOT: templates/"+tmplID+"/data", "VERAX_ROOT: ./data")
	return []byte(s)
}

// renamePackage swaps the template's placeholder package name for the project.
func renamePackage(data []byte, name string) []byte {
	s := string(data)
	for _, prefix := range []string{"filesystem-safe-template", "csv-query-template", "notes-store-template"} {
		s = strings.Replace(s, `"name": "`+prefix+`"`, `"name": "`+name+`"`, 1)
	}
	return []byte(s)
}

func validName(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
