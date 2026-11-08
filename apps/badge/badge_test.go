package veraxbadge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePulse(t *testing.T, rows string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pulse.json")
	doc := `{"date":"2026-11-08","rows":[` + rows + `]}`
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBadgeService(t *testing.T) {
	store := &Store{}
	path := writePulse(t,
		`{"name":"a","package":"a-pkg@1.0.0","score":88},`+
			`{"name":"b","package":"b-pkg@2.0.0","score":100},`+
			`{"name":"e","package":"err-pkg@1.0.0","score":0,"error":"fetch failed"}`)
	if err := store.Load(path); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(Handler(store))
	defer srv.Close()

	// known package → SVG with score and color
	resp, err := http.Get(srv.URL + "/badge/a-pkg@1.0.0.svg")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "svg") {
		t.Fatalf("status=%d ct=%s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	svg := string(body[:n])
	if !strings.Contains(svg, "88/100") || !strings.Contains(svg, "green") || !strings.Contains(svg, "a-pkg@1.0.0") {
		t.Errorf("badge wrong:\n%s", svg)
	}

	// 100 → brightgreen
	resp2, _ := http.Get(srv.URL + "/badge/b-pkg@2.0.0.svg")
	n2, _ := resp2.Body.Read(body)
	resp2.Body.Close()
	if !strings.Contains(string(body[:n2]), "brightgreen") {
		t.Errorf("100 score must be brightgreen")
	}

	// unknown → 404
	resp3, _ := http.Get(srv.URL + "/badge/nope@1.0.0.svg")
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("unknown package = %d, want 404", resp3.StatusCode)
	}

	// errored scan row is excluded
	resp4, _ := http.Get(srv.URL + "/badge/err-pkg@1.0.0.svg")
	resp4.Body.Close()
	if resp4.StatusCode != http.StatusNotFound {
		t.Errorf("errored row must 404, got %d", resp4.StatusCode)
	}

	// healthz
	resp5, _ := http.Get(srv.URL + "/healthz")
	n5, _ := resp5.Body.Read(body)
	resp5.Body.Close()
	var h struct {
		OK   bool   `json:"ok"`
		Date string `json:"date"`
	}
	if err := json.Unmarshal(body[:n5], &h); err != nil || !h.OK || h.Date != "2026-11-08" {
		t.Errorf("healthz wrong: %s", body[:n5])
	}
}
