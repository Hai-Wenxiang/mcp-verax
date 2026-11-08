package scanner

import (
	"archive/tar"
	"compress/gzip"
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

// GitHub targets: `gh:owner/repo[@ref]` (ref = tag or branch; default =
// latest release tag, falling back to the default branch). The published
// archive is fetched from codeload.github.com — public repos only.

// ghRepoRe: owner (alnum start, alnum+hyphen, ≤39) / repo (alnum/_ start,
// alnum + ._- ). Leading dots are rejected so "../" tricks never parse.
var ghRepoRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9_][A-Za-z0-9_.-]{0,99}$`)
var ghRefRe = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,100}$`)

func ghUserAgent() string {
	return fmt.Sprintf("verax-scanner/%s (+https://github.com/Hai-Wenxiang/mcp-verax)", version.Version)
}

// FetchGithubPackage downloads the repository archive for a
// `gh:owner/repo[@ref]` spec and extracts it into a temp dir. The returned
// cleanup removes the temp dir.
func FetchGithubPackage(spec string) (string, func(), error) {
	return fetchGithub("https://api.github.com", "https://codeload.github.com", spec)
}

// parseGithubSpec splits `gh:owner/repo[@ref]`.
func parseGithubSpec(spec string) (repo, ref string, err error) {
	s := strings.TrimPrefix(spec, "gh:")
	if i := strings.Index(s, "@"); i >= 0 {
		repo, ref = s[:i], s[i+1:]
	} else {
		repo = s
	}
	if !ghRepoRe.MatchString(repo) {
		return "", "", fmt.Errorf("scanner: invalid owner/repo in %q", spec)
	}
	if ref != "" && !ghRefRe.MatchString(ref) {
		return "", "", fmt.Errorf("scanner: invalid ref in %q", spec)
	}
	return repo, ref, nil
}

func fetchGithub(apiBase, codeloadBase, spec string) (string, func(), error) {
	repo, ref, err := parseGithubSpec(spec)
	if err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp("", "verax-scan-gh-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }

	client := &http.Client{Timeout: 120 * time.Second}

	kind := "" // "tag" | "branch"
	if ref == "" {
		ref, kind, err = githubDefaultRef(client, apiBase, repo)
		if err != nil {
			cleanup()
			return "", nil, err
		}
	} else {
		// an explicit ref is used as a tag first, then as a branch
		kind = "tag"
	}

	archive := filepath.Join(tmp, "repo.tar.gz")
	downloaded := false
	if kind == "tag" {
		downloaded = githubDownload(client, codeloadBase, repo, "refs/tags/"+ref, archive) == nil
	}
	if !downloaded {
		// fall back to treating the ref as a branch (also covers the
		// releases-latest kind, which returns a tag)
		if err := githubDownload(client, codeloadBase, repo, ref, archive); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("scanner: github %s@%s: no tag or branch archive: %w", repo, ref, err)
		}
	}

	root := filepath.Join(tmp, "extracted")
	// GitHub archive roots are "{repo}-{mangled-ref}/" with non-alphanumeric
	// ref characters mangled; detect the actual top directory instead of
	// guessing the mangling.
	if err := extractTarGzAuto(archive, root); err != nil {
		cleanup()
		return "", nil, err
	}
	return root, cleanup, nil
}

// githubDefaultRef prefers the latest release tag; repos without releases
// fall back to the default branch (returned kind tells the caller).
func githubDefaultRef(client *http.Client, apiBase, repo string) (ref, kind string, err error) {
	req, _ := http.NewRequest(http.MethodGet, apiBase+"/repos/"+repo+"/releases/latest", nil)
	req.Header.Set("User-Agent", ghUserAgent())
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("scanner: github api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		var doc struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&doc); err == nil && doc.TagName != "" {
			return doc.TagName, "tag", nil
		}
	}
	// no usable release: resolve the default branch
	req2, _ := http.NewRequest(http.MethodGet, apiBase+"/repos/"+repo, nil)
	req2.Header.Set("User-Agent", ghUserAgent())
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req2.Header.Set("Authorization", "Bearer "+tok)
	}
	resp2, err := client.Do(req2)
	if err != nil {
		return "", "", fmt.Errorf("scanner: github api: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("scanner: github repo %q not found", repo)
	}
	if resp2.StatusCode < 200 || resp2.StatusCode > 299 {
		return "", "", fmt.Errorf("scanner: github api: HTTP %d", resp2.StatusCode)
	}
	var doc struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&doc); err != nil || doc.DefaultBranch == "" {
		return "", "", fmt.Errorf("scanner: github api returned no default branch for %q", repo)
	}
	return doc.DefaultBranch, "branch", nil
}

// githubDownload fetches one codeload archive; false-ish error means try
// the next ref shape.
func githubDownload(client *http.Client, codeloadBase, repo, refPath, dest string) error {
	url := codeloadBase + "/" + repo + "/tar.gz/" + refPath
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", ghUserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: %s", refPath)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, refPath)
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

// extractTarGzAuto extracts a tarball whose entries share one top-level
// directory of unknown name (GitHub archives), auto-detecting it; falls
// back to flat extraction when entries are already at the root.
func extractTarGzAuto(tgzPath, dest string) error {
	top, err := tarTopDir(tgzPath)
	if err != nil {
		return err
	}
	if top == "" {
		return extractTarGzStrip(tgzPath, dest, "")
	}
	return extractTarGzStrip(tgzPath, dest, top+"/")
}

// tarTopDir returns the shared top-level directory of a tarball's file
// entries ("" when entries sit at the root or disagree).
func tarTopDir(tgzPath string) (string, error) {
	f, err := os.Open(tgzPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("scanner: gunzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	top := ""
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return top, nil
		}
		if err != nil {
			return "", fmt.Errorf("scanner: tar read: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue // dirs, symlinks, and pax_global_header carry no signal
		}
		name := path.Clean(hdr.Name)
		i := strings.IndexByte(name, '/')
		if i <= 0 {
			return "", nil // file at root: no shared top directory
		}
		seg := name[:i]
		if top == "" {
			top = seg
		} else if top != seg {
			return "", nil // mixed roots: extract flat
		}
	}
}
