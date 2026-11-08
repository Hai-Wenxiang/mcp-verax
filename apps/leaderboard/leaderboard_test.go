package leaderboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderSortsAndColors(t *testing.T) {
	pulses := []*Pulse{
		{Date: "2026-11-07", Entries: 30, NpmTotal: 2, Scanned: 2, Rows: []struct {
			Name     string      `json:"name"`
			Package  string      `json:"package"`
			Score    int         `json:"score"`
			Findings interface{} `json:"findings"`
			Error    string      `json:"error"`
		}{
			{Name: "low-one", Package: "low@1.0.0", Score: 30},
			{Name: "high-one", Package: "high@1.0.0", Score: 95},
			{Name: "broken", Package: "broken@1.0.0", Error: "fetch failed"},
		}},
	}
	html := Render(pulses)
	for _, want := range []string{
		"Verax Registry Leaderboard",
		"2026-11-07",
		"s-hi\">95", "s-lo\">30",
		"error",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
	// high must rank before low
	if strings.Index(html, "high-one") > strings.Index(html, "low-one") {
		t.Error("rows must be sorted by score desc")
	}
}

func TestGenerateDir(t *testing.T) {
	dir := t.TempDir()
	doc := `{"date":"2026-11-08","entries_pulled":10,"npm_packaged":1,"scanned":1,"rows":[{"name":"a","package":"a@1.0.0","score":88}]}`
	if err := os.WriteFile(filepath.Join(dir, "pulse-2026-11-08.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	// latest-copy must be ignored
	if err := os.WriteFile(filepath.Join(dir, "pulse-latest.json"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateDir(dir, 14); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "a@1.0.0") || !strings.Contains(string(raw), "88") {
		t.Errorf("index.html wrong:\n%s", raw)
	}
	// appears exactly once (latest copy not double-counted)
	if strings.Count(string(raw), "2026-11-08 —") != 1 {
		t.Errorf("pulse duplicated: %d", strings.Count(string(raw), "2026-11-08 —"))
	}
}
