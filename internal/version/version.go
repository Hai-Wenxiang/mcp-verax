// Package version holds build metadata for the verax CLI.
package version

import "regexp"

// Version is the semantic version of the verax CLI. It is overridden at
// build time via -ldflags "-X github.com/Hai-Wenxiang/mcp-verax/internal/version.Version=x.y.z".
var Version = "0.5.0"

// semverRe matches loose semantic versions (major.minor.patch with optional
// pre-release/build suffixes).
var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// Valid reports whether v looks like a semantic version.
func Valid(v string) bool {
	return semverRe.MatchString(v)
}
