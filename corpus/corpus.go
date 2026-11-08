//go:build !corpus_dev

// Package corpus embeds the community fuzz payloads (corpus/fuzz/*.txt).
// Release builds use the embedded copy; contributors edit the txt files.
package corpus

import (
	"embed"
	"strings"
)

//go:embed all:fuzz
var embedded embed.FS

// FuzzPayloads loads every payload line from all corpus/fuzz/*.txt files
// (blank lines and # comments ignored), deduplicated.
func FuzzPayloads() []string {
	var out []string
	seen := map[string]bool{}
	entries, err := embedded.ReadDir("fuzz")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		raw, err := embedded.ReadFile("fuzz/" + e.Name())
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimRight(line, "\r")
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if !seen[line] {
				seen[line] = true
				out = append(out, line)
			}
		}
	}
	return out
}
