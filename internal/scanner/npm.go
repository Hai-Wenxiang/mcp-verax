package scanner

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// FetchNPMPackage downloads the published tarball for an npm spec via
// `npm pack` and extracts it into a temp dir. Scanning the packed artifact
// (not a git checkout) means we audit exactly what consumers install.
// The returned cleanup removes the temp dir.
func FetchNPMPackage(spec string) (root string, cleanup func(), err error) {
	tmp, err := os.MkdirTemp("", "verax-scan-")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(tmp) }

	packOut, err := exec.Command("npm", "pack", spec, "--pack-destination", tmp, "--json").Output()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("scanner: npm pack %s: %w", spec, err)
	}
	filename := firstPackFilename(packOut)
	if filename == "" {
		cleanup()
		return "", nil, fmt.Errorf("scanner: npm pack output not understood: %s", string(packOut))
	}
	// npm may emit the bare filename; join with the destination ourselves.
	tgz := filename
	if !filepath.IsAbs(tgz) {
		tgz = filepath.Join(tmp, tgz)
	}

	root = filepath.Join(tmp, "extracted")
	if err := extractTarGz(tgz, root); err != nil {
		cleanup()
		return "", nil, err
	}
	return root, cleanup, nil
}

// firstPackFilename digs the tarball filename out of `npm pack --json`
// output, which is an array on older npm and an object keyed by package
// name on npm >= 9-ish.
func firstPackFilename(out []byte) string {
	type packInfo struct {
		Filename string `json:"filename"`
	}
	var arr []packInfo
	if err := json.Unmarshal(out, &arr); err == nil {
		for _, p := range arr {
			if p.Filename != "" {
				return p.Filename
			}
		}
		return ""
	}
	var obj map[string]packInfo
	if err := json.Unmarshal(out, &obj); err == nil {
		for _, p := range obj {
			if p.Filename != "" {
				return p.Filename
			}
		}
	}
	return ""
}

// extractTarGz unpacks a gzipped tarball, stripping any leading "package/"
// component (npm layout) and rejecting path escapes.
func extractTarGz(tgzPath, dest string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("scanner: gunzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("scanner: tar read: %w", err)
		}
		name := path.Clean(hdr.Name)
		if name == "package" || name == "." || name == "/" {
			continue // top-level layout directory
		}
		name = strings.TrimPrefix(name, "package/")
		if name == "" || name == "." {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("scanner: tar entry escapes destination: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			// symlinks/other types are skipped: scanner reads source text only
		}
	}
}
