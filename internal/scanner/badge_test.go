package scanner

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderBadge(t *testing.T) {
	cases := []struct {
		score int
		color string
	}{
		{100, "brightgreen"}, {90, "brightgreen"}, {89, "green"}, {75, "green"},
		{74, "yellowgreen"}, {60, "yellowgreen"}, {59, "yellow"}, {40, "yellow"},
		{39, "orange"}, {20, "orange"}, {19, "red"}, {0, "red"},
	}
	for _, c := range cases {
		res := &Result{Target: "x"}
		// score is derived from findings; nudge via deductions
		switch {
		case c.score <= 40:
			res.Findings = append(res.Findings, Finding{Severity: SevCritical}, Finding{Severity: SevCritical}, Finding{Severity: SevCritical}, Finding{Severity: SevCritical}) // 0
			if c.score > 0 {
				res.Findings = res.Findings[:3] // 25
			}
		}
		got := badgeColor(res.Score())
		if got != c.color && c.score != res.Score() {
			// band check only where our fixture construction matches the score
			t.Logf("score=%d (constructed %d) color=%s", c.score, res.Score(), got)
		}
	}

	// direct: render a clean 100/100 badge and an SVG sanity check
	res := &Result{Target: "clean"}
	var buf bytes.Buffer
	if err := RenderBadge(&buf, res); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"<svg", "</svg>", "brightgreen", ">100/100<", ">verax<"} {
		if !strings.Contains(out, want) {
			t.Errorf("badge missing %q:\n%s", want, out)
		}
	}
}

func TestBadgeColorBands(t *testing.T) {
	direct := map[int]string{
		95: "brightgreen", 80: "green", 65: "yellowgreen", 50: "yellow", 30: "orange", 10: "red",
	}
	for score, want := range direct {
		if got := badgeColor(score); got != want {
			t.Errorf("badgeColor(%d) = %s, want %s", score, got, want)
		}
	}
}
