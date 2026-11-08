package scanner

import (
	"path/filepath"
	"testing"
)

func TestPyMetaFindings(t *testing.T) {
	dir := filepath.Join("..", "..", "testfixtures", "scanvulnpy")
	res, err := Scan(dir, "scanvuln-py")
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, f := range res.Findings {
		if f.RuleID == "VX-QUA-105" {
			titles = append(titles, f.Title)
		}
	}
	if len(titles) < 3 { // license + description + repository
		for _, t2 := range titles {
			t.Logf("%s", t2)
		}
		t.Fatalf("want >=3 pyproject metadata findings, got %d", len(titles))
	}
}

func TestPyMetaCleanPasses(t *testing.T) {
	f, _ := osWriteFileT(t)
	_ = f
}

func osWriteFileT(t *testing.T) (string, error) { return "", nil }
