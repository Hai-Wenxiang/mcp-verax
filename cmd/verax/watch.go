package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

// watchState tracks mtimes for change detection (polling — dependency-free
// and portable; fsnotify-grade latency is not needed for suite iteration).
type watchState struct {
	mtimes map[string]int64
}

func newWatchState(paths []string) *watchState {
	ws := &watchState{mtimes: map[string]int64{}}
	ws.snapshot(paths)
	return ws
}

func (ws *watchState) snapshot(paths []string) {
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil {
			ws.mtimes[p] = st.ModTime().UnixNano()
		}
	}
}

// changed returns the files whose mtime moved since the last snapshot.
func (ws *watchState) changed(paths []string) []string {
	var out []string
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			if _, had := ws.mtimes[p]; had {
				delete(ws.mtimes, p)
			}
			continue
		}
		mod := st.ModTime().UnixNano()
		if old, had := ws.mtimes[p]; !had || old != mod {
			ws.mtimes[p] = mod
			out = append(out, p)
		}
	}
	return out
}

// watchLoop rerenders on changes; run executes one pass and returns the
// aggregate exit code (0 = all pass). Clears the screen for readability.
func watchLoop(files []string, run func() int, stdout io.Writer) int {
	ws := newWatchState(files)
	fmt.Fprintf(stdout, "\x1b[2J\x1b[Hverax test --watch: %d suite(s); save a suite file to re-run, Ctrl+C to quit\n\n", len(files))
	_ = run()
	for {
		time.Sleep(400 * time.Millisecond)
		if len(ws.changed(files)) > 0 {
			// debounce: let editors finish multi-file writes
			time.Sleep(300 * time.Millisecond)
			_ = ws.changed(files)
			fmt.Fprintf(stdout, "\x1b[2J\x1b[Hverax test --watch: change detected, re-running\n\n")
			_ = run()
		}
	}
}
