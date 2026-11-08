package scanner

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	res, err := Scan(fixtureDir("scanclean"), "c")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := RenderJSON(&buf, res); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Target   string         `json:"target"`
		Score    int            `json:"score"`
		Counts   map[string]int `json:"counts"`
		Findings []Finding      `json:"findings"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("not valid json: %v\n%s", err, buf.String())
	}
	if doc.Score != 100 || doc.Findings == nil {
		t.Errorf("clean scan json wrong: score=%d findings=%v", doc.Score, doc.Findings)
	}
}
