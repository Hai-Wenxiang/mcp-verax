package assert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SnapshotStore keeps golden results for one suite in a single JSON file:
//
//	<suite-dir>/__snapshots__/<suite-name>.snap.json   {"<snapshot-name>": <pretty JSON>}
//
// Snapshots are meant to be committed: they encode "what this tool returned
// when last reviewed". Missing snapshots are recorded (pass); existing ones
// must match byte-for-byte after canonicalization. `update` rewrites.
type SnapshotStore struct {
	path   string
	update bool
	data   map[string]string
	dirty  bool
}

// SnapshotPath derives the snapshot file path for a suite file.
func SnapshotPath(suiteFile string) string {
	dir := filepath.Dir(suiteFile)
	base := strings.TrimSuffix(filepath.Base(suiteFile), filepath.Ext(suiteFile))
	return filepath.Join(dir, "__snapshots__", base+".snap.json")
}

// OpenSnapshotStore loads (or prepares) the snapshot file for a suite.
func OpenSnapshotStore(suiteFile string, update bool) (*SnapshotStore, error) {
	s := &SnapshotStore{
		path:   SnapshotPath(suiteFile),
		update: update,
		data:   map[string]string{},
	}
	raw, err := os.ReadFile(s.path)
	switch {
	case err == nil:
		if jerr := json.Unmarshal(raw, &s.data); jerr != nil {
			return nil, fmt.Errorf("assert: snapshot file %s corrupt: %w", s.path, jerr)
		}
	case os.IsNotExist(err):
		// fresh store
	default:
		return nil, err
	}
	return s, nil
}

// Check compares actual against the stored snapshot for name. Missing or
// (with update) changed snapshots are recorded and pass.
func (s *SnapshotStore) Check(name string, actual any) CheckResult {
	canonical, err := canonicalize(actual)
	if err != nil {
		return CheckResult{Desc: snapshotDesc(name), Pass: false, Detail: fmt.Sprintf("cannot serialize result: %v", err)}
	}

	stored, exists := s.data[name]
	switch {
	case !exists:
		s.data[name] = canonical
		s.dirty = true
		return CheckResult{Desc: snapshotDesc(name), Pass: true, Detail: "new snapshot recorded (review it in git diff)"}
	case s.update:
		if stored == canonical {
			return CheckResult{Desc: snapshotDesc(name), Pass: true, Detail: "snapshot unchanged"}
		}
		s.data[name] = canonical
		s.dirty = true
		return CheckResult{Desc: snapshotDesc(name), Pass: true, Detail: "snapshot updated"}
	case stored == canonical:
		return CheckResult{Desc: snapshotDesc(name), Pass: true, Detail: ""}
	default:
		return CheckResult{
			Desc:   snapshotDesc(name),
			Pass:   false,
			Detail: firstDiff(stored, canonical),
		}
	}
}

// Path returns the snapshot file location for diagnostics.
func (s *SnapshotStore) Path() string { return s.path }

// Save persists pending changes, creating directories as needed.
func (s *SnapshotStore) Save() error {
	if !s.dirty {
		return nil
	}
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, append(raw, '\n'), 0o644)
}

func snapshotDesc(name string) string { return fmt.Sprintf("result.snapshot %q", name) }

func canonicalize(v any) (string, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// firstDiff reports the first differing line between two pretty JSON blobs.
func firstDiff(stored, actual string) string {
	sl := strings.Split(stored, "\n")
	al := strings.Split(actual, "\n")
	n := max(len(sl), len(al))
	for i := 0; i < n; i++ {
		var a, b string
		if i < len(sl) {
			a = sl[i]
		}
		if i < len(al) {
			b = al[i]
		}
		if a != b {
			return fmt.Sprintf("first diff at line %d:\n  stored: %s\n  actual: %s\n(run `verax test -u` to accept)", i+1, a, b)
		}
	}
	return "snapshots differ (length mismatch)"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
