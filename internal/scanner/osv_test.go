package scanner

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckOSVFindsAdvisories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/querybatch" {
			http.NotFound(w, r)
			return
		}
		// deps sorted: axios (0 hits), lodash (2 hits)
		_, _ = w.Write([]byte(`{"results":[{"vulns":[]},{"vulns":[{"id":"GHSA-1"},{"id":"CVE-2024-1"}]}]}`))
	}))
	defer srv.Close()

	res := &Result{Deps: map[string]string{"lodash": "4.17.15", "axios": "1.0.0"}}
	if err := CheckOSV(res, OSVClient{Endpoint: srv.URL}); err != nil {
		t.Fatal(err)
	}

	var got []Finding
	for _, f := range res.Findings {
		if f.RuleID == "VX-SEC-006" {
			got = append(got, f)
		}
	}
	if len(got) != 1 {
		t.Fatalf("want 1 CVE finding, got %d: %+v", len(got), res.Findings)
	}
	f := got[0]
	if f.Severity != SevHigh || f.File != "package.json" {
		t.Errorf("finding wrong: %+v", f)
	}
	if !contains(f.Snippet, "lodash") || !contains(f.Why, "GHSA-1") || !contains(f.Why, "CVE-2024-1") {
		t.Errorf("finding should name dep and advisories: %+v", f)
	}
}

func TestCheckOSVEmptyDepsNoop(t *testing.T) {
	res := &Result{}
	if err := CheckOSV(res, OSVClient{Endpoint: "http://127.0.0.1:1"}); err != nil {
		t.Fatalf("empty deps must not hit network: %v", err)
	}
	if len(res.Findings) != 0 {
		t.Errorf("no findings expected: %+v", res.Findings)
	}
}

func TestCheckOSVServerErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	res := &Result{Deps: map[string]string{"lodash": "1.0.0"}}
	if err := CheckOSV(res, OSVClient{Endpoint: srv.URL}); err == nil {
		t.Fatal("server error must surface for graceful degradation")
	}
}

func TestJoinTruncated(t *testing.T) {
	if got := joinTruncated([]string{"a", "b"}, 5); got != "a, b" {
		t.Errorf("join = %q", got)
	}
	got := joinTruncated([]string{"a", "b", "c", "d"}, 2)
	if got != "a, b … +2 more" {
		t.Errorf("truncated join = %q", got)
	}
}

func TestScanPopulatesDeps(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "v")
	if err != nil {
		t.Fatal(err)
	}
	if res.Deps == nil || res.Deps["lodahs"] == "" {
		t.Errorf("deps not populated: %+v", res.Deps)
	}
}
