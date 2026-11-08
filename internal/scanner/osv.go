package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// OSVClient queries the OSV.dev batch API for known vulnerabilities of npm
// dependencies. Endpoint is injectable for tests; the default is public OSV.
type OSVClient struct {
	Endpoint string
	HTTP     *http.Client
}

// DefaultOSVEndpoint is the public OSV.dev API.
const DefaultOSVEndpoint = "https://api.osv.dev"

// NewOSVClient builds a client with sane timeouts.
func NewOSVClient() OSVClient {
	return OSVClient{
		Endpoint: DefaultOSVEndpoint,
		HTTP:     &http.Client{Timeout: 20 * time.Second},
	}
}

type osvBatchRequest struct {
	Queries []struct {
		Package struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		} `json:"package"`
	} `json:"queries"`
}

type osvBatchResponse struct {
	Results []struct {
		Vulns []struct {
			ID string `json:"id"`
		} `json:"vulns"`
	} `json:"results"`
}

// CheckOSV enriches a scan result with VX-SEC-006 findings for dependencies
// with known CVEs/GHSAs. It mutates res.Findings. A transport error is
// returned so callers can degrade gracefully (offline scanning stays usable).
func CheckOSV(res *Result, client OSVClient) error {
	if len(res.Deps) == 0 {
		return nil
	}
	if client.Endpoint == "" {
		client.Endpoint = DefaultOSVEndpoint
	}
	if client.HTTP == nil {
		client.HTTP = &http.Client{Timeout: 20 * time.Second}
	}

	names := make([]string, 0, len(res.Deps))
	for name := range res.Deps {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic query order

	var req osvBatchRequest
	for _, name := range names {
		req.Queries = append(req.Queries, struct {
			Package struct {
				Name      string `json:"name"`
				Ecosystem string `json:"ecosystem"`
			} `json:"package"`
		}{Package: struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		}{Name: name, Ecosystem: "npm"}})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := client.HTTP.Post(client.Endpoint+"/v1/querybatch", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("scanner: osv query: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("scanner: osv query: status %d", resp.StatusCode)
	}
	var batch osvBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return fmt.Errorf("scanner: osv decode: %w", err)
	}

	for i, name := range names {
		if i >= len(batch.Results) {
			break
		}
		vulns := batch.Results[i].Vulns
		if len(vulns) == 0 {
			continue
		}
		ids := make([]string, 0, len(vulns))
		for _, v := range vulns {
			ids = append(ids, v.ID)
		}
		res.Findings = append(res.Findings, Finding{
			RuleID: "VX-SEC-006", Title: "dependency with known vulnerabilities",
			Severity: SevHigh, File: "package.json",
			Snippet: name + ": " + res.Deps[name] + " — " + fmt.Sprintf("%d advisories", len(ids)),
			Why:     "known advisories: " + joinTruncated(ids, 5) + " (osv.dev)",
		})
	}
	return nil
}

func joinTruncated(items []string, max int) string {
	if len(items) <= max {
		out := ""
		for i, s := range items {
			if i > 0 {
				out += ", "
			}
			out += s
		}
		return out
	}
	return fmt.Sprintf("%s … +%d more", joinTruncated(items[:max], max), len(items)-max)
}
