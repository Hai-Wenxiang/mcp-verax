// Package leaderboard renders registry pulse JSON files into a static,
// self-contained HTML leaderboard (no JS dependencies, dark-on-light,
// sortable client-side via tiny inline script is intentionally omitted —
// plain tables sorted by score server-side).
package leaderboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Pulse mirrors cmd/verax registry-pulse output.
type Pulse struct {
	Date     string `json:"date"`
	Entries  int    `json:"entries_pulled"`
	NpmTotal int    `json:"npm_packaged"`
	Scanned  int    `json:"scanned"`
	Rows     []struct {
		Name     string      `json:"name"`
		Package  string      `json:"package"`
		Score    int         `json:"score"`
		Findings interface{} `json:"findings"`
		Error    string      `json:"error"`
	} `json:"rows"`
}

// LoadPulse reads one pulse file.
func LoadPulse(path string) (*Pulse, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Pulse
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Render builds the leaderboard HTML from pulses (any order; latest first
// by date). The page is fully self-contained inline CSS.
func Render(pulses []*Pulse) string {
	sort.Slice(pulses, func(i, j int) bool { return pulses[i].Date > pulses[j].Date })

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Verax Registry Leaderboard</title>
<style>
  :root { --pass:#1a7f37; --mid:#9a6700; --low:#cf222e; --ink:#1f2328; --dim:#656d76; --bg:#f6f8fa; }
  body { font:14px/1.5 -apple-system,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif; color:var(--ink); margin:0; background:var(--bg); }
  header { background:#24292f; color:#fff; padding:22px 28px; }
  header h1 { margin:0 0 4px; font-size:18px; }
  header p { margin:0; color:#afb8c1; font-size:12px; }
  main { max-width:900px; margin:0 auto; padding:20px; }
  section { background:#fff; border:1px solid #d0d7de; border-radius:8px; padding:14px 18px; margin-bottom:18px; }
  h2 { font-size:14px; margin:0 0 8px; color:var(--dim); }
  table { width:100%; border-collapse:collapse; font-size:13px; }
  th, td { text-align:left; padding:6px 8px; border-bottom:1px solid #eaeef2; }
  th { color:var(--dim); font-weight:600; font-size:12px; }
  .score { font-weight:700; }
  .s-hi { color:var(--pass); } .s-mid { color:var(--mid); } .s-lo { color:var(--low); }
  code { background:var(--bg); padding:1px 5px; border-radius:4px; font-size:12px; }
  .err { color:var(--dim); font-style:italic; }
  footer { text-align:center; color:var(--dim); font-size:12px; padding:14px; }
</style>
</head>
<body>
<header>
  <h1>Verax Registry Leaderboard / 注册表榜单</h1>
  <p>verax scan scores for npm-packaged servers on the official MCP registry · updated nightly</p>
</header>
<main>
`)

	// trend data: package -> [(date, score)]; pulses are date-desc, so
	// reverse each series to render oldest → newest
	trend := map[string][][2]interface{}{}
	for _, p := range pulses {
		for _, r := range p.Rows {
			if r.Error == "" {
				trend[r.Package] = append(trend[r.Package], [2]interface{}{p.Date, r.Score})
			}
		}
	}
	for pkg, pts := range trend {
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
		trend[pkg] = pts
	}

	for _, p := range pulses {
		rows := append([]struct {
			Name     string      `json:"name"`
			Package  string      `json:"package"`
			Score    int         `json:"score"`
			Findings interface{} `json:"findings"`
			Error    string      `json:"error"`
		}(nil), p.Rows...)
		sort.SliceStable(rows, func(i, j int) bool {
			if (rows[i].Error != "") != (rows[j].Error != "") {
				return rows[i].Error == ""
			}
			return rows[i].Score > rows[j].Score
		})
		fmt.Fprintf(&b, "<section>\n<h2>%s — %d registry entries · %d npm-packaged · %d scanned</h2>\n", p.Date, p.Entries, p.NpmTotal, p.Scanned)
		b.WriteString("<table><tr><th>#</th><th>Server</th><th>Package</th><th>Score</th></tr>\n")
		for i, r := range rows {
			scoreHTML := `<span class="err">error</span>`
			cls := ""
			if r.Error == "" {
				switch {
				case r.Score >= 90:
					cls = "s-hi"
				case r.Score >= 60:
					cls = "s-mid"
				default:
					cls = "s-lo"
				}
				scoreHTML = fmt.Sprintf(`<span class="score %s">%d</span>`, cls, r.Score)
			}
			trendHTML := ""
			if pts := trend[r.Package]; len(pts) > 1 {
				parts := make([]string, len(pts))
				for k, pt := range pts {
					d, _ := pt[0].(string)
					if len(d) > 5 {
						d = d[5:] // "2026-11-08" → "11-08"
					}
					parts[k] = fmt.Sprintf("%s:%v", d, pt[1])
				}
				trendHTML = fmt.Sprintf(`<br><span class="trend">%s</span>`, strings.Join(parts, " → "))
			}
			fmt.Fprintf(&b, "<tr><td>%d</td><td>%s</td><td><code>%s</code>%s</td><td>%s</td></tr>\n",
				i+1, r.Name, r.Package, trendHTML, scoreHTML)
		}
		b.WriteString("</table>\n</section>\n")
	}

	b.WriteString(`</main>
<footer>generated by <code>verax leaderboard</code> · scores by <code>verax scan</code> · data: nightly registry pulse</footer>
</body>
</html>`)
	return b.String()
}

// GenerateDir loads every pulse-*.json in dir (latest N by name) and writes
// index.html next to them.
func GenerateDir(dir string, latest int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "pulse-") && strings.HasSuffix(e.Name(), ".json") && e.Name() != "pulse-latest.json" {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) > latest {
		names = names[:latest]
	}
	var pulses []*Pulse
	for _, n := range names {
		p, err := LoadPulse(filepath.Join(dir, n))
		if err != nil {
			continue
		}
		pulses = append(pulses, p)
	}
	if len(pulses) == 0 {
		return nil
	}
	return os.WriteFile(filepath.Join(dir, "index.html"), []byte(Render(pulses)), 0o644)
}
