package scanner

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClassifyOSVSeverity(t *testing.T) {
	cases := []struct {
		name string
		d    osvVulnDetail
		want Severity
	}{
		{"ghsa-high", osvVulnDetail{DatabaseSpecific: struct {
			Severity string `json:"severity"`
		}{Severity: "HIGH"}}, SevHigh},
		{"ghsa-low", osvVulnDetail{DatabaseSpecific: struct {
			Severity string `json:"severity"`
		}{Severity: "LOW"}}, SevMedium},
		{"cvss-9", osvVulnDetail{Severity: []struct {
			Type  string `json:"type"`
			Score string `json:"score"`
		}{{Type: "CVSS_V3", Score: `{"baseScore":9.8}`}}}, SevHigh},
		{"cvss-5", osvVulnDetail{Severity: []struct {
			Type  string `json:"type"`
			Score string `json:"score"`
		}{{Type: "CVSS_V3", Score: `{"baseScore":5.3}`}}}, SevMedium},
		{"cvss-2", osvVulnDetail{Severity: []struct {
			Type  string `json:"type"`
			Score string `json:"score"`
		}{{Type: "CVSS_V3", Score: `{"baseScore":2.1}`}}}, SevLow},
		{"unknown", osvVulnDetail{}, SevHigh},
	}
	for _, c := range cases {
		if got := classifyOSVSeverity(c.d); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestEnrichOSVSeverityRefinesFindings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/vulns/") {
			http.NotFound(w, r)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/v1/vulns/")
		sev := "HIGH"
		if strings.Contains(id, "LOW") {
			sev = "LOW"
		}
		_, _ = w.Write([]byte(`{"id":"` + id + `","database_specific":{"severity":"` + sev + `"}}`))
	}))
	defer srv.Close()

	res := &Result{Findings: []Finding{{
		RuleID:   "VX-SEC-006",
		Severity: SevHigh,
		Snippet:  "lodash: 4.17.15 — 2 advisories",
		Why:      "known advisories: GHSA-AAAA-1111-2222, GHSA-LOWO-3333-4444 (osv.dev)",
	}}}
	EnrichOSVSeverity(res, OSVClient{Endpoint: srv.URL}, 5)

	// worst of (HIGH, LOW) = high — unchanged but exercised the fetch path
	if res.Findings[0].Severity != SevHigh {
		t.Errorf("severity = %s, want high (worst of HIGH+LOW)", res.Findings[0].Severity)
	}

	// all-LOW advisories downgrade to medium
	res2 := &Result{Findings: []Finding{{
		RuleID:   "VX-SEC-006",
		Severity: SevHigh,
		Why:      "known advisories: GHSA-LOWO-5555-6666 (osv.dev)",
	}}}
	EnrichOSVSeverity(res2, OSVClient{Endpoint: srv.URL}, 5)
	if res2.Findings[0].Severity != SevMedium {
		t.Errorf("low-only advisory should downgrade to medium, got %s", res2.Findings[0].Severity)
	}
}

func TestEnrichOSVSeverityFailureKeepsDefault(t *testing.T) {
	res := &Result{Findings: []Finding{{
		RuleID:   "VX-SEC-006",
		Severity: SevHigh,
		Why:      "known advisories: GHSA-XXXX-YYYY-ZZZZ (osv.dev)",
	}}}
	EnrichOSVSeverity(res, OSVClient{Endpoint: "http://127.0.0.1:1"}, 5)
	if res.Findings[0].Severity != SevHigh {
		t.Errorf("failed fetches must keep conservative high, got %s", res.Findings[0].Severity)
	}
}
