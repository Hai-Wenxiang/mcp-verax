package scanner

import "sort"

// RuleMeta is a catalog entry for `verax rules` and docs generation.
type RuleMeta struct {
	ID       string
	Title    string
	Severity Severity
	Lang     string // js / py / pkg / "" (agnostic)
	Kind     string // source / manifest / enrichment
}

// AllRules returns the full rule catalog (stable ID order). The same rule
// implemented for multiple languages collapses into one entry (js+py).
func AllRules() []RuleMeta {
	index := map[string]*RuleMeta{}
	add := func(m RuleMeta) {
		if existing, ok := index[m.ID]; ok {
			if existing.Lang != m.Lang && !containsLang(existing.Lang, m.Lang) {
				existing.Lang = mergeLangs(existing.Lang, m.Lang)
			}
			return
		}
		index[m.ID] = &m
	}
	for _, r := range sourceRules {
		lang := r.Lang
		if lang == "" {
			lang = "any"
		}
		add(RuleMeta{ID: r.ID, Title: r.Title, Severity: r.Sev, Lang: lang, Kind: "source"})
	}
	for _, r := range packageRules {
		add(RuleMeta{ID: r.ID, Title: r.Title, Severity: r.Sev, Lang: "pkg", Kind: "manifest"})
	}
	// Non-registry rules implemented inline:
	add(RuleMeta{ID: "VX-SEC-006", Title: "dependency with known vulnerabilities", Severity: SevHigh, Lang: "pkg", Kind: "enrichment"})
	add(RuleMeta{ID: "VX-QUA-106", Title: "no README", Severity: SevLow, Lang: "pkg", Kind: "manifest"})

	out := make([]RuleMeta, 0, len(index))
	for _, m := range index {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func containsLang(merged, lang string) bool {
	for _, part := range splitLangs(merged) {
		if part == lang {
			return true
		}
	}
	return false
}

func mergeLangs(a, b string) string { return a + "+" + b }

func splitLangs(s string) []string {
	var out []string
	cur := ""
	for _, c := range s {
		if c == '+' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(c)
	}
	return append(out, cur)
}
