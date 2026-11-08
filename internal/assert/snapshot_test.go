package assert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hai-Wenxiang/mcp-verax/internal/mcpclient"
)

func newStore(t *testing.T, update bool) *SnapshotStore {
	t.Helper()
	suiteFile := filepath.Join(t.TempDir(), "my.suite.yaml")
	s, err := OpenSnapshotStore(suiteFile, update)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Save() })
	return s
}

func echoResult(text string) *mcpclient.CallToolResult {
	return &mcpclient.CallToolResult{Content: []mcpclient.Content{{Type: "text", Text: text}}}
}

func TestSnapshotRecordOnFirstSight(t *testing.T) {
	s := newStore(t, false)
	r := s.Check("echo-basic", echoResult("hello"))
	if !r.Pass {
		t.Fatalf("first sight should record+pass: %+v", r)
	}
	if !strings.Contains(r.Detail, "new snapshot recorded") {
		t.Errorf("detail should say recorded: %s", r.Detail)
	}
}

func TestSnapshotMatchAndMismatch(t *testing.T) {
	s := newStore(t, false)
	s.Check("echo", echoResult("stable value"))

	if r := s.Check("echo", echoResult("stable value")); !r.Pass {
		t.Fatalf("identical result should pass: %+v", r)
	}

	changed := &mcpclient.CallToolResult{Content: []mcpclient.Content{{Type: "text", Text: "DIFFERENT"}}}
	r := s.Check("echo", changed)
	if r.Pass {
		t.Fatal("changed result must fail")
	}
	if !strings.Contains(r.Detail, "first diff at line") || !strings.Contains(r.Detail, "verax test -u") {
		t.Errorf("mismatch detail should locate diff and hint -u: %s", r.Detail)
	}
}

func TestSnapshotUpdateMode(t *testing.T) {
	s := newStore(t, true)
	s.Check("x", echoResult("old"))
	r := s.Check("x", echoResult("new"))
	if !r.Pass || !strings.Contains(r.Detail, "updated") {
		t.Fatalf("update mode should rewrite+pass: %+v", r)
	}
	if r := s.Check("x", echoResult("new")); !r.Pass {
		t.Errorf("post-update check should pass: %+v", r)
	}
}

func TestSnapshotPersistence(t *testing.T) {
	dir := t.TempDir()
	suiteFile := filepath.Join(dir, "persist.suite.yaml")

	s1, err := OpenSnapshotStore(suiteFile, false)
	if err != nil {
		t.Fatal(err)
	}
	s1.Check("k", echoResult("persisted"))
	if err := s1.Save(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(SnapshotPath(suiteFile)); err != nil {
		t.Fatalf("snapshot file not written: %v", err)
	}

	s2, err := OpenSnapshotStore(suiteFile, false)
	if err != nil {
		t.Fatal(err)
	}
	if r := s2.Check("k", echoResult("persisted")); !r.Pass {
		t.Fatalf("snapshot should survive reload: %+v", r)
	}
}

func TestSnapshotPathLayout(t *testing.T) {
	got := SnapshotPath(filepath.Join("suites", "weather.suite.yaml"))
	want := filepath.Join("suites", "__snapshots__", "weather.suite.snap.json")
	if got != want {
		t.Errorf("path = %s, want %s", got, want)
	}
}
