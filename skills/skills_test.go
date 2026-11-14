package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestVeraxSkillShape keeps the marketplace-uploadable SKILL.md honest:
// valid frontmatter, install-with-verification present, command routing
// complete. Anything the skill promises must still be true.
func TestVeraxSkillShape(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "skills", "verax", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)

	if !strings.HasPrefix(s, "---\n") {
		t.Fatal("SKILL.md must start with YAML frontmatter")
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		t.Fatal("frontmatter not terminated")
	}
	fm := s[4 : 4+end]
	if !regexp.MustCompile(`(?m)^name: verax$`).MatchString(fm) {
		t.Error("frontmatter must declare name: verax")
	}
	if !strings.Contains(fm, "description:") || len(fm) < 200 {
		t.Error("frontmatter needs a substantive description (agents route on it)")
	}
	// description must name the trigger phrases agents match on
	for _, want := range []string{"MCP server", "scan", "SARIF"} {
		if !strings.Contains(fm, want) {
			t.Errorf("description missing trigger keyword %q", want)
		}
	}

	body := s[4+end+4:]
	// install must pin + verify, not curl-and-run
	if !strings.Contains(body, "SHA256SUMS") || !strings.Contains(body, "sha256sum --check") {
		t.Error("install instructions must include checksum verification")
	}
	if strings.Contains(body, "curl … | sh") || strings.Contains(body, "curl -fsSL https://github.com/Hai-Wenxiang/mcp-verax/releases/download/${V}/verax | sh") {
		t.Error("skill must never pipe binaries straight into a shell")
	}
	// command routing must cover the CLI surface
	for _, cmd := range []string{"verax init", "verax test", "verax scan", "verax new", "verax rules"} {
		if !strings.Contains(body, cmd) {
			t.Errorf("skill body missing %s", cmd)
		}
	}
	// every registry target type the scanner supports must be mentioned
	for _, tgt := range []string{"crate:", "pypi:", "gh:"} {
		if !strings.Contains(body, tgt) {
			t.Errorf("skill body missing target prefix %s", tgt)
		}
	}
	// gates + machine outputs
	for _, flag := range []string{"--min-score", "--fail-level", "--sarif", "--repeat"} {
		if !strings.Contains(body, flag) {
			t.Errorf("skill body missing flag %s", flag)
		}
	}
}
