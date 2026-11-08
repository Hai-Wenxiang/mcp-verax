package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

// osvVulnDetail is the severity-relevant subset of an OSV record.
type osvVulnDetail struct {
	ID       string `json:"id"`
	Severity []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
}

// classifyOSVSeverity maps an OSV record onto our scale: GHSA CRITICAL/HIGH
// → high; MODERATE → high (upgrade-now material); LOW → medium; CVSS
// baseScore ≥7 high / ≥4 medium / else low; unknown → high (conservative).
func classifyOSVSeverity(d osvVulnDetail) Severity {
	switch d.DatabaseSpecific.Severity {
	case "CRITICAL", "HIGH", "MODERATE", "MEDIUM":
		return SevHigh
	case "LOW":
		return SevMedium
	}
	for _, s := range d.Severity {
		if s.Type != "CVSS_V3" {
			continue
		}
		var cvss struct {
			BaseScore float64 `json:"baseScore"`
		}
		if err := json.Unmarshal([]byte(s.Score), &cvss); err == nil && cvss.BaseScore > 0 {
			switch {
			case cvss.BaseScore >= 7:
				return SevHigh
			case cvss.BaseScore >= 4:
				return SevMedium
			default:
				return SevLow
			}
		}
	}
	return SevHigh
}

var advisoryIDRe = regexp.MustCompile(`\b(?:GHSA|CVE)[A-Za-z0-9_\-]{6,}\b`)

// enrichOSVSeverity refines VX-SEC-006 finding severities by fetching
// per-advisory details (bounded). Best-effort: failures leave findings as-is.
func EnrichOSVSeverity(res *Result, client OSVClient, maxDetails int) {
	if client.Endpoint == "" {
		client.Endpoint = DefaultOSVEndpoint
	}
	if client.HTTP == nil {
		client.HTTP = &http.Client{Timeout: 15 * time.Second}
	}

	fetched := 0
	for i := range res.Findings {
		if fetched >= maxDetails {
			return
		}
		f := &res.Findings[i]
		if f.RuleID != "VX-SEC-006" {
			continue
		}
		ids := advisoryIDRe.FindAllString(f.Why, -1)
		if len(ids) == 0 {
			continue
		}
		worst := SevLow
		for _, id := range ids {
			if fetched >= maxDetails {
				break
			}
			fetched++
			d, err := fetchOSVDetail(client, id)
			if err != nil {
				continue
			}
			if s := classifyOSVSeverity(d); s.Weight() > worst.Weight() {
				worst = s
			}
		}
		if worst != SevLow { // all fetches failed: keep the conservative default
			f.Severity = worst
		}
	}
}

func fetchOSVDetail(client OSVClient, id string) (osvVulnDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.Endpoint+"/v1/vulns/"+id, nil)
	if err != nil {
		return osvVulnDetail{}, err
	}
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return osvVulnDetail{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return osvVulnDetail{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	var d osvVulnDetail
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return osvVulnDetail{}, err
	}
	return d, nil
}
