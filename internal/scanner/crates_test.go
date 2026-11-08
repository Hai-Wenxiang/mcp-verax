package scanner

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildCrateTgz writes a .crate-style tarball: entries rooted under
// "{name}-{version}/".
func buildCrateTgz(t *testing.T, tgzPath, name, ver string, files map[string]string) {
	t.Helper()
	f, err := os.Create(tgzPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	root := name + "-" + ver
	hdrDir := &tar.Header{Name: root, Typeflag: tar.TypeDir, Mode: 0o755}
	if err := tw.WriteHeader(hdrDir); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		body := content
		if err := tw.WriteHeader(&tar.Header{
			Name: root + "/" + rel, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestParseCrateSpec(t *testing.T) {
	for _, c := range []struct {
		spec    string
		name    string
		ver     string
		wantErr bool
	}{
		{"crate:anyhow", "anyhow", "", false},
		{"crate:serde@1.0.200", "serde", "1.0.200", false},
		{"crate:a-b_c", "a-b_c", "", false},
		{"crate:../evil", "", "", true},
		{"crate:ok@../evil", "", "", true},
		{"crate:", "", "", true},
	} {
		name, ver, err := parseCrateSpec(c.spec)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: expected error", c.spec)
			}
			continue
		}
		if err != nil || name != c.name || ver != c.ver {
			t.Errorf("%q → (%q,%q,%v), want (%q,%q,nil)", c.spec, name, ver, err, c.name, c.ver)
		}
	}
}

func TestFetchCrateEndToEnd(t *testing.T) {
	var sawUA, sawDownloadPath string
	apiHit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawUA = r.Header.Get("User-Agent")
		switch {
		case r.URL.Path == "/api/v1/crates/tinycrate":
			apiHit = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"crate": map[string]string{"max_stable_version": "0.2.1"},
			})
		case r.URL.Path == "/api/v1/crates/tinycrate/0.2.1/download":
			sawDownloadPath = r.URL.Path
			w.Header().Set("Content-Type", "application/octet-stream")
			tgz := filepath.Join(t.TempDir(), "crate")
			f, ferr := os.Create(tgz)
			if ferr != nil {
				t.Fatal(ferr)
			}
			buildCrateTgz(t, tgz, "tinycrate", "0.2.1", map[string]string{
				"Cargo.toml": "[package]\nname = \"tinycrate\"\nversion = \"0.2.1\"\nlicense = \"MIT\"\nrepository = \"x\"\ndescription = \"d\"\n",
				"src/lib.rs": "pub fn f() {}\n",
			})
			f.Close()
			http.ServeFile(w, r, tgz)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	root, cleanup, err := fetchCrate(srv.URL, "crate:tinycrate")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if !apiHit {
		t.Fatal("version resolution API was not hit")
	}
	if !strings.HasPrefix(sawUA, "verax-scanner/") {
		t.Errorf("crates.io requires an identifying User-Agent, got %q", sawUA)
	}
	if sawDownloadPath == "" {
		t.Fatal("download endpoint was not hit")
	}
	for _, rel := range []string{"Cargo.toml", "src/lib.rs"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("extracted %s missing: %v", rel, err)
		}
	}
	// full scan pipeline over the extracted artifact
	res, err := Scan(root, "crate:tinycrate")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Findings {
		// complete Cargo.toml metadata must stay quiet; the bare "no LICENSE
		// file" check applies to any non-npm manifest and is expected here
		if f.RuleID == "VX-QUA-105" && f.File == "Cargo.toml" {
			t.Errorf("complete Cargo.toml should stay quiet: %+v", f)
		}
	}
}

func TestFetchCrateNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	if _, _, err := fetchCrate(srv.URL, "crate:does-not-exist-xyz"); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestResolveTargetCrate(t *testing.T) {
	kind, err := ResolveTarget("crate:anyhow")
	if err != nil || kind != TargetCrate {
		t.Fatalf("crate: prefix should resolve to TargetCrate, got %v %v", kind, err)
	}
	kind, err = ResolveTarget("crate:anyhow")
	if err != nil || kind != TargetCrate {
		t.Fatalf("second resolve failed: %v %v", kind, err)
	}
}
