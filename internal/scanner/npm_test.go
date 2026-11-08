package scanner

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// buildTgz packs srcDir into a tarball with the npm "package/" prefix layout.
func buildTgz(t *testing.T, srcDir, tgzPath string) {
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

	_ = filepath.Walk(srcDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, p)
		data, _ := os.ReadFile(p)
		hdr := &tar.Header{
			Name: "package/" + filepath.ToSlash(rel),
			Mode: 0o644,
			Size: int64(len(data)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write(data)
		return nil
	})
}

func TestExtractTarGzStripsPackagePrefix(t *testing.T) {
	tgz := filepath.Join(t.TempDir(), "fixture.tgz")
	buildTgz(t, fixtureDir("scanclean"), tgz)

	dest := filepath.Join(t.TempDir(), "extracted")
	if err := extractTarGz(tgz, dest); err != nil {
		t.Fatal(err)
	}
	// files must sit directly under dest (prefix stripped)
	if _, err := os.Stat(filepath.Join(dest, "package.json")); err != nil {
		t.Fatalf("package.json not at root after strip: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "index.js")); err != nil {
		t.Fatalf("index.js missing: %v", err)
	}

	// scanning the extracted tree behaves like a local scan
	res, err := Scan(dest, "extracted-fixture")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 0 || res.Score() != 100 {
		t.Errorf("clean fixture through tarball: %d findings, score %d", len(res.Findings), res.Score())
	}
}

func TestExtractTarGzRejectsEscape(t *testing.T) {
	tgz := filepath.Join(t.TempDir(), "evil.tgz")
	f, _ := os.Create(tgz)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "../../evil.txt", Mode: 0o644, Size: 3})
	_, _ = tw.Write([]byte("bad"))
	tw.Close()
	gz.Close()
	f.Close()

	dest := filepath.Join(t.TempDir(), "dest")
	if err := extractTarGz(tgz, dest); err == nil {
		t.Fatal("path escape must be rejected")
	}
}
