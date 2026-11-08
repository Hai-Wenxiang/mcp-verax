package scanner

import "strings"

func init() {
	packageRules = append(packageRules, secInstallScripts, secTyposquat)
}

// installHooks run code when the CONSUMER installs the package — the classic
// supply-chain execution point.
var installHooks = []string{"preinstall", "install", "postinstall", "prepare"}

// secInstallScripts (VX-SEC-012): lifecycle scripts that execute at install
// time on every consumer machine.
var secInstallScripts = packageRule{
	ID:    "VX-SEC-012",
	Title: "install-time lifecycle script",
	Sev:   SevMedium,
	Check: func(rel string, data []byte) []Finding {
		pkg, ok := parsePackageJSON(data)
		if !ok {
			return nil
		}
		var out []Finding
		for _, hook := range installHooks {
			cmd := strings.TrimSpace(pkg.Scripts[hook])
			if cmd == "" || benignInstallScript(cmd) {
				continue
			}
			out = append(out, Finding{
				RuleID: "VX-SEC-012", Title: "install-time lifecycle script",
				Severity: SevMedium, File: rel,
				Snippet: hook + ": " + cmd,
				Why:     "runs on every consumer's machine at install — review what it executes",
			})
		}
		return out
	},
}

func benignInstallScript(cmd string) bool {
	c := strings.ToLower(cmd)
	return c == "exit 0" || strings.HasPrefix(c, "echo ") && !strings.ContainsAny(c, "|&><$`")
}

// popularPackages is a tiny curated subset for typo detection; distance-1
// hits against these are almost always typosquats.
var popularPackages = []string{
	"axios", "lodash", "request", "node-fetch", "chalk", "debug", "ms",
	"typescript", "express", "uuid", "jsonwebtoken", "dotenv", "commander",
	"yargs", "esbuild", "postcss", "moment", "ws", "zod", "cross-env",
	"npm-check-updates", "semver", "minimist", "glob", "mkdirp", "rimraf",
}

// secTyposquat (VX-SEC-011): dependency names one or two edits away from a
// popular package — typosquatting or a typo you'll regret either way.
// Distance 1 = high confidence (high); distance 2 covers transpositions
// like lodahs/lodash (medium).
var secTyposquat = packageRule{
	ID:    "VX-SEC-011",
	Title: "dependency name resembles a popular package",
	Sev:   SevHigh,
	Check: func(rel string, data []byte) []Finding {
		pkg, ok := parsePackageJSON(data)
		if !ok {
			return nil
		}
		var out []Finding
		for dep := range pkg.Dependencies {
			name := strings.TrimPrefix(dep, "@types/")
			for _, pop := range popularPackages {
				if name == pop {
					break // exact popular name: fine
				}
				d := editDistance(name, pop)
				if d == 1 || d == 2 {
					sev := SevHigh
					if d == 2 {
						sev = SevMedium
					}
					out = append(out, Finding{
						RuleID: "VX-SEC-011", Title: "dependency name resembles a popular package",
						Severity: sev, File: rel,
						Snippet: dep + " (≈ " + pop + ", distance " + string(rune('0'+d)) + ")",
						Why:     "close to a popular name: typosquat or typo — verify the source",
					})
					break
				}
			}
		}
		return out
	},
}

// editDistance is the classic Levenshtein DP over runes.
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = minInt(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func minInt(nums ...int) int {
	m := nums[0]
	for _, n := range nums[1:] {
		if n < m {
			m = n
		}
	}
	return m
}
