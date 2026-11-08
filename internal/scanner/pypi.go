package scanner

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/version"
)

// PyPI targets: `pypi:name` or `pypi:name@version`. Scanning the published
// sdist means we audit the source consumers install from, same policy as
// npm pack and crates.io.

var pypiNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,100}$`)
var pypiVersionRe = regexp.MustCompile(`^[0-9A-Za-z.+-]{1,64}$`)

func pypiUserAgent() string {
	return fmt.Sprintf("verax-scanner/%s (+https://github.com/Hai-Wenxiang/mcp-verax)", version.Version)
}

// FetchPyPIPackage downloads the published sdist for a `pypi:name[@version]`
// spec from PyPI and extracts it into a temp dir. The returned cleanup
// removes the temp dir.
func FetchPyPIPackage(spec string) (string, func(), error) {
	return fetchPyPI("https://pypi.org", spec)
}

// parsePyPISpec splits `pypi:name[@version]`; empty version means latest.
func parsePyPISpec(spec string) (name, ver string, err error) {
	s := strings.TrimPrefix(spec, "pypi:")
	if i := strings.Index(s, "@"); i >= 0 {
		name, ver = s[:i], s[i+1:]
	} else {
		name = s
	}
	if !pypiNameRe.MatchString(name) {
		return "", "", fmt.Errorf("scanner: invalid pypi name in %q", spec)
	}
	if ver != "" && !pypiVersionRe.MatchString(ver) {
		return "", "", fmt.Errorf("scanner: invalid pypi version in %q", spec)
	}
	return name, ver, nil
}

type pypiFile struct {
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	Packagetype string `json:"packagetype"`
}

func fetchPyPI(base, spec string) (string, func(), error) {
	name, ver, err := parsePyPISpec(spec)
	if err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp("", "verax-scan-pypi-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	client := &http.Client{Timeout: 90 * time.Second}
	files, resolved, err := pypiFiles(client, base, name, ver)
	if err != nil {
		cleanup()
		return "", nil, err
	}

	sdist := pickSdist(files)
	if sdist == nil {
		cleanup()
		return "", nil, fmt.Errorf("scanner: pypi %s has no sdist (source) distribution to audit", name)
	}

	archive := filepath.Join(tmp, "pkg")
	if err := pypiDownload(client, sdist.URL, archive); err != nil {
		cleanup()
		return "", nil, err
	}

	root := filepath.Join(tmp, "extracted")
	strip := fmt.Sprintf("%s-%s/", normalizePyPIName(name), resolved)
	switch {
	case strings.HasSuffix(sdist.Filename, ".tar.gz") || strings.HasSuffix(sdist.Filename, ".tgz"):
		if err := extractTarGzStrip(archive, root, strip); err != nil {
			cleanup()
			return "", nil, err
		}
	case strings.HasSuffix(sdist.Filename, ".zip"):
		if err := extractZipStrip(archive, root, strip); err != nil {
			cleanup()
			return "", nil, err
		}
	default:
		cleanup()
		return "", nil, fmt.Errorf("scanner: unsupported sdist format: %s", sdist.Filename)
	}
	return root, cleanup, nil
}

// pypiFiles asks the PyPI JSON API for a version's distribution files.
func pypiFiles(client *http.Client, base, name, ver string) ([]pypiFile, string, error) {
	url := base + "/pypi/" + normalizePyPIName(name) + "/json"
	if ver != "" {
		url = base + "/pypi/" + normalizePyPIName(name) + "/" + ver + "/json"
	}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", pypiUserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("scanner: pypi api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("scanner: pypi package %q not found", name)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", fmt.Errorf("scanner: pypi api: HTTP %d", resp.StatusCode)
	}
	var doc struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
		URLs []pypiFile `json:"urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, "", fmt.Errorf("scanner: pypi api decode: %w", err)
	}
	if doc.Info.Version == "" {
		return nil, "", fmt.Errorf("scanner: pypi api returned no version for %q", name)
	}
	return doc.URLs, doc.Info.Version, nil
}

// pickSdist prefers .tar.gz over .zip (text-friendlier layout).
func pickSdist(files []pypiFile) *pypiFile {
	var zipFile *pypiFile
	for i := range files {
		f := &files[i]
		if f.Packagetype != "sdist" {
			continue
		}
		if strings.HasSuffix(f.Filename, ".tar.gz") || strings.HasSuffix(f.Filename, ".tgz") {
			return f
		}
		if zipFile == nil && strings.HasSuffix(f.Filename, ".zip") {
			zipFile = f
		}
	}
	return zipFile
}

func pypiDownload(client *http.Client, url, dest string) error {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", pypiUserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("scanner: pypi download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("scanner: pypi download: HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// normalizePyPIName applies PEP 503 normalization (lowercase, runs of
// -_. collapse to -): sdist archive roots use the normalized name.
func normalizePyPIName(name string) string {
	lower := strings.ToLower(name)
	var b strings.Builder
	dash := false
	for _, r := range lower {
		if r == '-' || r == '_' || r == '.' {
			dash = true
			continue
		}
		if dash && b.Len() > 0 {
			b.WriteByte('-')
		}
		dash = false
		b.WriteRune(r)
	}
	return b.String()
}

// extractZipStrip unpacks a zip with the same escape-proofing as the tar
// reader, stripping the given leading directory.
func extractZipStrip(zipPath, dest, strip string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("scanner: zip open: %w", err)
	}
	defer zr.Close()

	topDir := strings.TrimSuffix(strip, "/")
	for _, f := range zr.File {
		name := path.Clean(f.Name)
		if name == topDir || name == "." || name == "/" {
			continue
		}
		name = strings.TrimPrefix(name, strip)
		if name == "" || name == "." {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("scanner: zip entry escapes destination: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
