package scanner

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGithubSpec(t *testing.T) {
	for _, c := range []struct {
		spec    string
		repo    string
		ref     string
		wantErr bool
	}{
		{"gh:modelcontextprotocol/servers", "modelcontextprotocol/servers", "", false},
		{"gh:o/r@v1.2.3", "o/r", "v1.2.3", false},
		{"gh:o/r@feature/x", "o/r", "feature/x", false},
		{"gh:../evil", "", "", true},
		{"gh:only-owner", "", "", true},
		{"gh:o/r@bad ref!", "", "", true},
		{"gh:", "", "", true},
	} {
		repo, ref, err := parseGithubSpec(c.spec)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: expected error", c.spec)
			}
			continue
		}
		if err != nil || repo != c.repo || ref != c.ref {
			t.Errorf("%q → (%q,%q,%v), want (%q,%q,nil)", c.spec, repo, ref, err, c.repo, c.ref)
		}
	}
}

// buildGithubTgz writes an archive with a GitHub-style mangled root
// ("repo-v1.2.3" for tag v1.2.3 — dots kept, slashes would be dashed).
func buildGithubTgz(t *testing.T, path, root string, files map[string]string) {
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
	for rel, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: root + "/" + rel, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFetchGithubLatestRelease(t *testing.T) {
	var sawAuth bool
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawAuth = true
		}
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v2.1.0"})
			return
		}
		http.NotFound(w, r)
	}))
	defer api.Close()

	tgz := t.TempDir()
	codeload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// codeload path: /{owner}/{repo}/tar.gz/refs/tags/{tag}
		if strings.HasSuffix(r.URL.Path, "/tar.gz/refs/tags/v2.1.0") {
			p := filepath.Join(tgz, "repo.tgz")
			buildGithubTgz(t, p, "myrepo-2.1.0", map[string]string{
				"package.json": `{"name":"myrepo","license":"MIT","repository":"x","description":"d"}`,
				"index.js":     "console.log(1)\n",
			})
			http.ServeFile(w, r, p)
			return
		}
		http.NotFound(w, r)
	}))
	defer codeload.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	root, cleanup, err := fetchGithub(api.URL, codeload.URL, "gh:owner/myrepo")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if !sawAuth {
		t.Error("GITHUB_TOKEN should be sent to the github api")
	}
	for _, rel := range []string{"package.json", "index.js"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("extracted %s missing: %v", rel, err)
		}
	}
}

func TestFetchGithubFallsBackToDefaultBranch(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.NotFound(w, r) // no releases
			return
		}
		if strings.HasSuffix(r.URL.Path, "/repos/o/r") {
			_ = json.NewEncoder(w).Encode(map[string]string{"default_branch": "trunk"})
			return
		}
		http.NotFound(w, r)
	}))
	defer api.Close()

	calls := 0
	codeload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// branch fetch: /o/r/tar.gz/trunk
		if strings.HasSuffix(r.URL.Path, "/tar.gz/trunk") {
			p := filepath.Join(t.TempDir(), "b.tgz")
			buildGithubTgz(t, p, "r-trunk", map[string]string{"README.md": "# r\n"})
			http.ServeFile(w, r, p)
			return
		}
		http.NotFound(w, r)
	}))
	defer codeload.Close()

	root, cleanup, err := fetchGithub(api.URL, codeload.URL, "gh:o/r")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Stat(filepath.Join(root, "README.md")); err != nil {
		t.Errorf("branch archive not extracted: %v", err)
	}
	// kind=branch path must not first try refs/tags (calls counted loosely)
	if calls != 1 {
		t.Errorf("codeload calls = %d, want 1 (branch directly)", calls)
	}
}

func TestFetchGithubRepoNotFound(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer api.Close()
	codeload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer codeload.Close()
	if _, _, err := fetchGithub(api.URL, codeload.URL, "gh:o/nope"); err == nil {
		t.Fatal("expected repo-not-found error")
	}
}

func TestTarTopDir(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.tgz")
	buildGithubTgz(t, p, "weird-root_1.2", map[string]string{"src/x.js": "1", "docs/a.md": "2"})
	if top, err := tarTopDir(p); err != nil || top != "weird-root_1.2" {
		t.Fatalf("tarTopDir = %q %v, want weird-root_1.2", top, err)
	}
}

func TestTarTopDirIgnoresPaxGlobalHeader(t *testing.T) {
	// GitHub tarballs open with a pax_global_header entry; it must not
	// be mistaken for a root-level file (regression: flat-extraction bug).
	// tar.Writer refuses manual TypeXHeader entries, so the header block is
	// encoded by hand.
	p := filepath.Join(t.TempDir(), "gh.tgz")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	rawPaxEntry(gz, "pax_global_header")
	tw := tar.NewWriter(gz)
	for rel, body := range map[string]string{"servers-2.3.0/src/index.js": "1"} {
		if err := tw.WriteHeader(&tar.Header{Name: rel, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()
	f.Close()
	if top, err := tarTopDir(p); err != nil || top != "servers-2.3.0" {
		t.Fatalf("tarTopDir = %q %v, want servers-2.3.0 (pax ignored)", top, err)
	}
}

// rawPaxEntry writes a 512-byte tar header block for a size-0 entry of the
// given type directly to the stream.
func rawPaxEntry(w io.Writer, name string) {
	h := make([]byte, 512)
	copy(h, name)
	copy(h[257:], "ustar\x0000")
	h[156] = 'x' // TypeXHeader
	for i := 148; i < 156; i++ {
		h[i] = ' ' // checksum field is spaces while summing
	}
	var sum int64
	for _, b := range h {
		sum += int64(b)
	}
	copy(h[148:], fmt.Sprintf("%06o\x00 ", sum))
	if _, err := w.Write(h); err != nil {
		panic(err)
	}
}
