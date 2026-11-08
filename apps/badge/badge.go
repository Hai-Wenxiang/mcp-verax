// Verax badge worker — serves SVG score badges from pulse data.
// Deploy target: any static host; data source: the pulse JSON committed by
// the nightly registry-pulse workflow. Zero runtime dependencies.
//
// GET /badge/<urlencoded-package-spec>.svg   → score badge
// GET /healthz                               → liveness
//
// The worker reads docs/registry/pulse-latest.json checked in next to it
// (CI keeps it fresh); keeping data in-repo means the badge service is a
// plain static file server — no database, no build step.
package veraxbadge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

type pulseRow struct {
	Name    string `json:"name"`
	Package string `json:"package"`
	Score   int    `json:"score"`
	Error   string `json:"error,omitempty"`
}

type pulseDoc struct {
	Date string     `json:"date"`
	Rows []pulseRow `json:"rows"`
}

// Store loads pulse data once and serves from memory; Reload swaps it.
type Store struct {
	mu   sync.RWMutex
	rows map[string]int // package spec -> score
	date string
}

func (s *Store) Load(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc pulseDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	m := make(map[string]int, len(doc.Rows))
	for _, r := range doc.Rows {
		if r.Error == "" {
			m[r.Package] = r.Score
		}
	}
	s.mu.Lock()
	s.rows, s.date = m, doc.Date
	s.mu.Unlock()
	return nil
}

func (s *Store) Score(pkg string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.rows[pkg]
	return v, ok
}

func badgeColor(score int) string {
	switch {
	case score >= 90:
		return "brightgreen"
	case score >= 75:
		return "green"
	case score >= 60:
		return "yellowgreen"
	case score >= 40:
		return "yellow"
	case score >= 20:
		return "orange"
	}
	return "red"
}

func RenderBadgeSVG(pkg string, score int) string {
	value := fmt.Sprintf("%d/100", score)
	color := badgeColor(score)
	labelW, valueW := 40+len(pkg)*6, 46
	if labelW > 220 {
		labelW = 220
	}
	width := labelW + valueW
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="verax: %s">
  <clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>
  <g clip-path="url(#r)">
    <rect width="%d" height="20" fill="#555"/>
    <rect x="%d" width="%d" height="20" fill="#%s"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,sans-serif" font-size="10">
    <text x="%d" y="14">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>
`, width, value, width, width, labelW, valueW, color,
		labelW/2+1, pkg, labelW+valueW/2-1, value)
}

// Handler builds the HTTP mux for the badge service.
func Handler(store *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"date":%q}`, store.date)
	})
	mux.HandleFunc("/badge/", func(w http.ResponseWriter, r *http.Request) {
		spec := strings.Trim(strings.TrimPrefix(r.URL.Path, "/badge/"), "/")
		spec = strings.TrimSuffix(spec, ".svg")
		if spec == "" {
			http.Error(w, "missing package spec", http.StatusBadRequest)
			return
		}
		score, ok := store.Score(spec)
		if !ok {
			http.Error(w, "package not in latest pulse", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		fmt.Fprint(w, RenderBadgeSVG(spec, score))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "verax badge service\n\nGET /badge/<package-spec>.svg\nGET /healthz\n")
	})
	return mux
}
