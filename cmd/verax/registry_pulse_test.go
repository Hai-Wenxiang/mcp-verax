package main

import (
	"encoding/json"
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
