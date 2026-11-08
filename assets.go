// Package mcpverax is the module root package; it embeds the server
// templates so released single-file binaries can scaffold without a
// repository checkout (`verax new`).
package mcpverax

import (
	"embed"
	"io/fs"
)

//go:embed all:templates
var embedded embed.FS

// Templates returns the embedded templates filesystem rooted at templates/.
func Templates() fs.FS {
	sub, err := fs.Sub(embedded, "templates")
	if err != nil {
		panic(err) // embed guarantees the directory exists at build time
	}
	return sub
}

// TemplateIDs lists available template ids (directory names).
func TemplateIDs() []string {
	entries, err := fs.ReadDir(embedded, "templates")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}
