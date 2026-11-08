package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchStateDetectsChange(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.suite.yaml")
	if err := os.WriteFile(f, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	ws := newWatchState([]string{f})
	if got := ws.changed([]string{f}); len(got) != 0 {
		t.Fatalf("no change expected right after snapshot, got %v", got)
	}

	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(f, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ws.changed([]string{f})
	if len(got) != 1 || got[0] != f {
		t.Fatalf("change not detected: %v", got)
	}
	if again := ws.changed([]string{f}); len(again) != 0 {
		t.Fatalf("change should latch once, got %v", again)
	}
}

func TestWatchStateHandlesRemoval(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "gone.yaml")
	_ = os.WriteFile(f, []byte("x"), 0o644)
	ws := newWatchState([]string{f})
	_ = os.Remove(f)
	if got := ws.changed([]string{f}); got != nil {
		t.Fatalf("removal should not crash or report: %v", got)
	}
}
