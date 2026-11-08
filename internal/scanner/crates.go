package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Hai-Wenxiang/mcp-verax/internal/version"
)

// crates.io targets: `crate:name` or `crate:name@version`. Scanning the
// published .crate artifact means we audit exactly what consumers build
// against, same policy as npm pack.

var crateNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
var crateVersionRe = regexp.MustCompile(`^[0-9A-Za-z.+-]{1,64}$`)

// crateUserAgent: crates.io rejects default HTTP agents; identify honestly.
func crateUserAgent() string {
	return fmt.Sprintf("verax-scanner/%s (+https://github.com/Hai-Wenxiang/mcp-verax)", version.Version)
}

// FetchCratePackage downloads the published .crate tarball for a
// `crate:name[@version]` spec from crates.io and extracts it into a temp
// dir. The returned cleanup removes the temp dir.
func FetchCratePackage(spec string) (string, func(), error) {
	return fetchCrate("https://crates.io", spec)
}

// parseCrateSpec splits `crate:name[@version]`; empty version means latest.
func parseCrateSpec(spec string) (name, ver string, err error) {
	s := strings.TrimPrefix(spec, "crate:")
	if i := strings.Index(s, "@"); i >= 0 {
		name, ver = s[:i], s[i+1:]
	} else {
		name = s
	}
	if !crateNameRe.MatchString(name) {
		return "", "", fmt.Errorf("scanner: invalid crate name in %q", spec)
	}
	if ver != "" && !crateVersionRe.MatchString(ver) {
		return "", "", fmt.Errorf("scanner: invalid crate version in %q", spec)
	}
	return name, ver, nil
}

func fetchCrate(base, spec string) (string, func(), error) {
	name, ver, err := parseCrateSpec(spec)
	if err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp("", "verax-scan-crate-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	client := &http.Client{Timeout: 60 * time.Second}
	if ver == "" {
		ver, err = latestCrateVersion(client, base, name)
		if err != nil {
			cleanup()
			return "", nil, err
		}
	}

	tgz := filepath.Join(tmp, "pkg.crate")
	if err := downloadCrate(client, base, name, ver, tgz); err != nil {
		cleanup()
		return "", nil, err
	}

	root := filepath.Join(tmp, "extracted")
	// crate tarballs root everything under "{name}-{version}/"
	if err := extractTarGzStrip(tgz, root, fmt.Sprintf("%s-%s/", name, ver)); err != nil {
		cleanup()
		return "", nil, err
	}
	return root, cleanup, nil
}

// latestCrateVersion asks the crates.io API for the newest stable release.
func latestCrateVersion(client *http.Client, base, name string) (string, error) {
	url := base + "/api/v1/crates/" + name
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", crateUserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("scanner: crates.io api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("scanner: crate %q not found on crates.io", name)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("scanner: crates.io api: HTTP %d", resp.StatusCode)
	}
	var doc struct {
		Crate struct {
			MaxStableVersion string `json:"max_stable_version"`
			MaxVersion       string `json:"max_version"`
			NewestVersion    string `json:"newest_version"`
		} `json:"crate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("scanner: crates.io api decode: %w", err)
	}
	if doc.Crate.MaxStableVersion != "" {
		return doc.Crate.MaxStableVersion, nil
	}
	if doc.Crate.MaxVersion != "" {
		return doc.Crate.MaxVersion, nil
	}
	return "", fmt.Errorf("scanner: crates.io api returned no version for %q", name)
}

// downloadCrate fetches the .crate tarball (the API endpoint redirects to
// static.crates.io; the client follows).
func downloadCrate(client *http.Client, base, name, ver, dest string) error {
	url := base + "/api/v1/crates/" + name + "/" + ver + "/download"
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", crateUserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("scanner: crate download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("scanner: crate download %s: HTTP %d", name+"@"+ver, resp.StatusCode)
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
