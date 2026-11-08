package scanner

import (
	"fmt"
	"io"
)

// badgeColors maps score bands to shield colors (shields.io convention).
func badgeColor(score int) string {
	switch {
	case score >= 90:
		return "brightgreen"
	case score >= 75:
		return "green"
	case score >= 60:
		return "yellowgreen"
	case score >= 40:
		return "yellow"
	case score >= 20:
		return "orange"
	}
	return "red"
}

// RenderBadge writes a self-contained shields-style SVG score badge.
func RenderBadge(w io.Writer, res *Result) error {
	label := "verax"
	value := fmt.Sprintf("%d/100", res.Score())
	color := badgeColor(res.Score())
	const (
		labelW = 44
		valueW = 52
	)
	width := labelW + valueW
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="verax: %s">
  <title>verax scan score: %d/100 (%d findings)</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>
  <g clip-path="url(#r)">
    <rect width="%d" height="20" fill="#555"/>
    <rect x="%d" width="%d" height="20" fill="#%s"/>
    <rect width="%d" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
    <text x="%d" y="14">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>
`, width, value, res.Score(), len(res.Findings),
		width,
		width, labelW, valueW, color,
		width,
		labelW/2+1, label,
		labelW+valueW/2-1, value)
	_, err := io.WriteString(w, svg)
	return err
}
