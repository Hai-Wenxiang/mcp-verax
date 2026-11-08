package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/Hai-Wenxiang/mcp-verax/internal/runner"
)

// stability tracks per-suite/per-case pass counts across repeated runs.
type stability struct {
	runs   int
	passed map[string]int // "suite/case" -> passes
	order  []string
	failed bool
}

func newStability(runs int) *stability {
	return &stability{runs: runs, passed: map[string]int{}}
}

func (s *stability) record(reports []*runner.SuiteReport) {
	for _, r := range reports {
		if !r.Pass() {
			s.failed = true
		}
		for _, c := range r.Cases {
			key := r.Suite + "/" + c.Name
			if _, seen := s.passed[key]; !seen {
				s.order = append(s.order, key)
				s.passed[key] = 0 // failing cases must occupy the map too
			}
			if c.Status == runner.StatusPass {
				s.passed[key]++
			}
		}
	}
}

// render prints the stability table: pass/total per case, flaky first.
func (s *stability) render(w io.Writer) {
	fmt.Fprintf(w, "\nverax test --repeat %d — stability:\n", s.runs)
	keys := append([]string(nil), s.order...)
	sort.SliceStable(keys, func(i, j int) bool {
		fi, fj := s.flaky(keys[i]), s.flaky(keys[j])
		if fi != fj {
			return fi // flaky cases float to the top
		}
		return false
	})
	for _, k := range keys {
		p := s.passed[k]
		mark := "OK  "
		if p == 0 {
			mark = "FAIL"
		} else if p < s.runs {
			mark = "FLAKY"
		}
		fmt.Fprintf(w, "  [%s] %s  %d/%d\n", mark, k, p, s.runs)
	}
}

func (s *stability) flaky(key string) bool {
	p := s.passed[key]
	return p > 0 && p < s.runs
}
