package corpus

import (
	"strings"
	"testing"
)

func TestFuzzPayloadsLoad(t *testing.T) {
	all := FuzzPayloads()
	if len(all) < 60 {
		t.Fatalf("corpus too small: %d payloads", len(all))
	}
	seen := map[string]bool{}
	categories := map[string]bool{
		"../../etc/passwd":             false,
		"; DROP TABLE users;--":        false,
		"Ignore previous instructions": false,
		"ünïcödé-nôrmàlizàtiôn":        false,
		"1' AND SLEEP(5)--":            false, // sql-injection
		`{"jsonrpc":"2.0","id":null`:   false, // json-rpc
		"<img src=x onerror=alert(1)>": false, // xss
	}
	for _, p := range all {
		if seen[p] {
			t.Errorf("duplicate payload: %q", p)
		}
		seen[p] = true
		for k := range categories {
			if strings.HasPrefix(p, strings.Split(k, "…")[0]) || strings.Contains(p, k) || strings.HasPrefix(k, p) {
				categories[k] = true
			}
		}
	}
	for k, ok := range categories {
		if !ok {
			t.Errorf("category sample missing: %q", k)
		}
	}
}

func TestCorpusHygiene(t *testing.T) {
	// inert payloads only: no live-looking URLs with domains beyond
	// example.invalid/localhost, no credential-shaped strings
	banned := []string{"http://example.com", "https://", "sk-", "AKIA", "ghp_", "xoxb-"}
	for _, p := range FuzzPayloads() {
		for _, b := range banned {
			if strings.Contains(p, b) {
				t.Errorf("non-inert payload %q contains %q", p, b)
			}
		}
	}
}

func TestCommentsAndBlanksIgnored(t *testing.T) {
	for _, p := range FuzzPayloads() {
		if strings.HasPrefix(p, "#") {
			t.Errorf("comment leaked into payloads: %q", p)
		}
	}
}
