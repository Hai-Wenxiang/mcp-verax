package scanner

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePyPISpec(t *testing.T) {
	for _, c := range []struct {
		spec    string
		name    string
		ver     string
		wantErr bool
	}{
		{"pypi:mcp-server-fetch", "mcp-server-fetch", "", false},
		{"pypi:Foo.Bar_Baz@1.2.3", "Foo.Bar_Baz", "1.2.3", false},
		{"pypi:../evil", "", "", true},
		{"pypi:ok@../evil", "", "", true},
		{"pypi:", "", "", true},
	} {
		name, ver, err := parsePyPISpec(c.spec)
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

func TestNormalizePyPIName(t *testing.T) {
	for in, want := range map[string]string{
		"mcp-server-fetch": "mcp-server-fetch",
		"Foo.Bar_Baz":      "foo-bar-baz",
		"A__B..C":          "a-b-c",
		"Simple":           "simple",
	} {
		if got := normalizePyPIName(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

// fakePyPI serves the JSON API and one sdist archive (format-flavored).
func fakePyPI(t *testing.T, archiveFn func(t *testing.T, path string), filename string) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	sdist := filepath.Join(dir, "sdist")
	var archiveOnce func()
	archiveOnce = func() { archiveFn(t, sdist) }
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/json"):
			suffix := "tar.gz"
			if strings.HasSuffix(filename, ".zip") {
				suffix = "zip"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"info": map[string]string{"version": "0.9.2"},
				"urls": []map[string]string{
					{"url": "http://" + r.Host + "/files/pkg." + suffix, "filename": filename, "packagetype": "sdist"},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/files/"):
			archiveOnce()
			http.ServeFile(w, r, sdist)
		default:
			http.NotFound(w, r)
		}
	}))
}

func buildPyPITgz(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	root := "tinypkg-0.9.2"
	files := map[string]string{
		"pyproject.toml":  "[project]\nname = \"tinypkg\"\nversion = \"0.9.2\"\n",
		"pkg/__init__.py": "\"\"\"doc\"\"\"\n",
	}
	for rel, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: root + "/" + rel, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}

func buildPyPIZip(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	root := "tinypkg-0.9.2"
	for rel, body := range map[string]string{
		"pyproject.toml":  "[project]\nname = \"tinypkg\"\nversion = \"0.9.2\"\n",
		"pkg/__init__.py": "\"\"\"doc\"\"\"\n",
	} {
		w, err := zw.Create(root + "/" + rel)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFetchPyPITarGzEndToEnd(t *testing.T) {
	srv := fakePyPI(t, buildPyPITgz, "tinypkg-0.9.2.tar.gz")
	defer srv.Close()
	root, cleanup, err := fetchPyPI(srv.URL, "pypi:tinypkg")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	for _, rel := range []string{"pyproject.toml", "pkg/__init__.py"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("extracted %s missing: %v", rel, err)
		}
	}
}

func TestFetchPyPIZipEndToEnd(t *testing.T) {
	srv := fakePyPI(t, buildPyPIZip, "tinypkg-0.9.2.zip")
	defer srv.Close()
	root, cleanup, err := fetchPyPI(srv.URL, "pypi:tinypkg")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err != nil {
		t.Errorf("zip-extracted pyproject.toml missing: %v", err)
	}
}

func TestFetchPyPINotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	if _, _, err := fetchPyPI(srv.URL, "pypi:nope-does-not-exist"); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestResolveTargetPyPI(t *testing.T) {
	kind, err := ResolveTarget("pypi:mcp-server-fetch")
	if err != nil || kind != TargetPyPI {
		t.Fatalf("pypi: prefix should resolve to TargetPyPI, got %v %v", kind, err)
	}
}
