package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const okPage = `{"servers":[{"server":{"name":"x","packages":[
  {"registryType":"pypi","identifier":"x","version":"1.0.0"}]}}],
  "metadata":{"nextCursor":"","count":1}}`

func TestFetchRegistryPageRetries(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			http.Error(w, "boom", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okPage))
	}))
	defer srv.Close()

	start := time.Now()
	page, err := fetchRegistryPage(srv.Client(), srv.URL, 3, time.Millisecond, os.Stderr)
	if err != nil {
		t.Fatalf("expected success on 3rd attempt, got %v", err)
	}
	if hits != 3 {
		t.Fatalf("expected 3 requests, got %d", hits)
	}
	if len(page.Servers) != 1 || page.Metadata.Count != 1 {
		t.Fatalf("unexpected page payload: %+v", page)
	}
	if elapsed := time.Since(start); elapsed < 3*time.Millisecond {
		t.Fatalf("linear backoff (1x+2x steps) not applied, elapsed %v", elapsed)
	}
}

func TestFetchRegistryPageExhaustsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	if _, err := fetchRegistryPage(srv.Client(), srv.URL, 2, time.Millisecond, os.Stderr); err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func mkEntry(t *testing.T, name, rt, id, ver string) regEntry {
	t.Helper()
	var e regEntry
	doc := fmt.Sprintf(`{"server":{"name":%q,"packages":[{"registryType":%q,"identifier":%q,"version":%q}]}}`, name, rt, id, ver)
	if err := json.Unmarshal([]byte(doc), &e); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestDedupeHighestVersion(t *testing.T) {
	entries := []regEntry{
		mkEntry(t, "bb", "crates.io", "rmcp", "0.1.0"),
		mkEntry(t, "aa", "crates.io", "rmcp", "0.3.2"),
		mkEntry(t, "aa", "crates.io", "rmcp", "0.2.0"),
		mkEntry(t, "cc", "npm", "x", "1.0.0"),
	}
	got := dedupeHighestVersion(entries)
	if len(got) != 2 {
		t.Fatalf("dedup = %d entries, want 2: %+v", len(got), got)
	}
	if got[0].Server.Name != "aa" || got[0].Server.Packages[0].Version != "0.3.2" {
		t.Errorf("aa should keep 0.3.2, got %+v", got[0].Server.Packages[0])
	}
	if got[1].Server.Name != "cc" {
		t.Errorf("name sort broken: %+v", got)
	}
}

func TestCountCrate(t *testing.T) {
	entries := []regEntry{
		mkEntry(t, "a", "crates.io", "rmcp", "1.0.0"),
		mkEntry(t, "b", "crate", "legacy-shape", "1.0.0"),
		mkEntry(t, "c", "npm", "x", "1.0.0"),
		mkEntry(t, "d", "pypi", "y", "1.0.0"),
	}
	if n := countCrate(entries); n != 2 {
		t.Fatalf("countCrate = %d, want 2", n)
	}
	if n := countNpm(entries); n != 1 {
		t.Fatalf("countNpm = %d, want 1", n)
	}
	if n := countPyPI(entries); n != 1 {
		t.Fatalf("countPyPI = %d, want 1", n)
	}
}

func TestRunRegistryPulseWritesLatestForBadge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okPage)) // pypi-only: no npm fetches, stays offline
	}))
	defer srv.Close()

	out := t.TempDir()
	var stdout, stderr strings.Builder
	code := runRegistryPulse([]string{
		"--registry", srv.URL,
		"--out", out,
		"--retries", "1",
		"--sample-crates", "0", // stay offline: crate fetches hit real crates.io
		"--sample-pypi", "0", // ...and pypi fetches hit real PyPI
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr:\n%s", code, stderr.String())
	}

	raw, err := os.ReadFile(filepath.Join(out, "pulse-latest.json"))
	if err != nil {
		t.Fatalf("pulse-latest.json not written: %v", err)
	}
	var doc struct {
		Date    string `json:"date"`
		Scanned int    `json:"scanned"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("pulse-latest.json not valid JSON: %v", err)
	}
	if doc.Date == "" || doc.Scanned != 0 {
		t.Fatalf("unexpected pulse-latest payload: %+v", doc)
	}
	// dated artifacts alongside
	if _, err := os.Stat(filepath.Join(out, "pulse-"+doc.Date+".md")); err != nil {
		t.Fatalf("dated markdown report missing: %v", err)
	}
}
