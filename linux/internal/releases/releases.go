package releases

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/rocq-prover/rocq-platform-starter/shared/github"
	sharedreleases "github.com/rocq-prover/rocq-platform-starter/shared/releases"

	"github.com/rocq-prover/rocq-platform-starter/linux/internal/manifest"
)

type ghContent struct {
	Name string `json:"name"`
}

const (
	repoContentsURL = "https://api.github.com/repos/rocq-prover/platform/contents/package_picks"
	rawContentURL   = "https://raw.githubusercontent.com/rocq-prover/platform/main/package_picks/"
)

// FetchReleases returns available release tags from GitHub, filtered to exclude
// old "v" prefixed tags.
func FetchReleases() ([]string, error) {
	return sharedreleases.FetchReleases()
}

// FetchRocqVersion fetches the Rocq version for a given release tag from the GitHub release body.
func FetchRocqVersion(tag string) (string, error) {
	return sharedreleases.FetchRocqVersion(tag)
}

// packagePickInfo holds parsed data from a package-pick shell script.
type packagePickInfo struct {
	coqTag         string            // COQ_PLATFORM_COQ_TAG (e.g. "9.0.1" or "8.20.1")
	ocamlVersion   string            // COQ_PLATFORM_OCAML_VERSION (e.g. "4.14.2")
	pinnedPackages map[string]string // PIN.name.version -> name: version
}

var (
	varRe = regexp.MustCompile(`^(\w+)=["']?([^"'\s]+)["']?`)
	pinRe = regexp.MustCompile(`PIN\.([^."]+)\.([\w.~+-]+)`)
	pkgRe = regexp.MustCompile(`(?:^|[\s"])([a-z][\w-]*)\.(v?\d[\w.~+-]*)`)
)

// parsePackagePick parses a package-pick shell script and extracts relevant info.
func parsePackagePick(content string) *packagePickInfo {
	info := &packagePickInfo{
		pinnedPackages: make(map[string]string),
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)

		// Parse variable assignments: VAR=value or VAR="value" or VAR='value'
		if m := varRe.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "COQ_PLATFORM_COQ_TAG":
				info.coqTag = m[2]
			case "COQ_PLATFORM_OCAML_VERSION":
				info.ocamlVersion = m[2]
			}
		}

		// Parse package references in PACKAGES lines
		if strings.Contains(line, "PACKAGES") {
			for _, m := range pinRe.FindAllStringSubmatch(line, -1) {
				info.pinnedPackages[m[1]] = m[2]
			}
			for _, m := range pkgRe.FindAllStringSubmatch(line, -1) {
				if _, exists := info.pinnedPackages[m[1]]; !exists {
					info.pinnedPackages[m[1]] = m[2]
				}
			}
		}
	}

	return info
}

// pickFileRe matches a package-pick filename: the Rocq major.minor it pins,
// then the platform cycle it belongs to, e.g. "package-pick-9.1~2026.01.sh".
var pickFileRe = regexp.MustCompile(`^package-pick-(\d+\.\d+)~(\d{4}\.\d{2})\.sh$`)

// tagYearMonth reduces a platform release tag to its YYYY.MM, e.g.
// "2025.08.1" -> "2025.08".
func tagYearMonth(tag string) (string, error) {
	parts := strings.SplitN(tag, ".", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid tag format: %s", tag)
	}
	return parts[0] + "." + parts[1], nil
}

// selectPackagePick picks the package-pick file for a release out of a
// package_picks directory listing.
//
// The filename encodes the Rocq major.minor it pins and the platform cycle it
// was introduced in; neither is the release's own date. Matching on the release
// date alone is wrong twice over: release 2026.07.0 ships Rocq 9.1, whose pick
// is package-pick-9.1~2026.01.sh and so was never found at all, while releases
// such as 2022.09.1 (Coq 8.16) matched package-pick-8.15~2022.09.sh because two
// Rocq versions share that cycle.
//
// rocqVersion is the version advertised by the release, and may be empty when it
// could not be inferred; the date-only match is then kept as a fallback so old
// releases whose notes cannot be parsed still resolve.
func selectPackagePick(names []string, tag, rocqVersion string) (string, error) {
	yearMonth, err := tagYearMonth(tag)
	if err != nil {
		return "", err
	}

	if majorMinor := rocqMajorMinor(rocqVersion); majorMinor != "" {
		type candidate struct{ cycle, name string }
		var cands []candidate
		for _, n := range names {
			if m := pickFileRe.FindStringSubmatch(n); m != nil && m[1] == majorMinor {
				cands = append(cands, candidate{cycle: m[2], name: n})
			}
		}
		// YYYY.MM is zero-padded, so lexicographic order is chronological.
		sort.Slice(cands, func(i, j int) bool { return cands[i].cycle < cands[j].cycle })

		if len(cands) > 0 {
			// The cycle matching the release wins; otherwise the newest cycle
			// that predates it; otherwise the newest known for that version.
			best := ""
			for _, c := range cands {
				if c.cycle == yearMonth {
					return c.name, nil
				}
				if c.cycle <= yearMonth {
					best = c.name
				}
			}
			if best != "" {
				return best, nil
			}
			return cands[len(cands)-1].name, nil
		}
	}

	// Fallback: the legacy date-only match.
	suffix := "~" + yearMonth + ".sh"
	for _, n := range names {
		if strings.HasSuffix(n, suffix) {
			return n, nil
		}
	}

	return "", fmt.Errorf("no package-pick file found for release %s (Rocq %q, looked for package-pick-<major.minor>~*.sh and *%s)",
		tag, rocqVersion, suffix)
}

// findPackagePickFile lists package_picks on GitHub and selects the file for a
// release. rocqVersion may be empty.
func findPackagePickFile(tag, rocqVersion string) (string, error) {
	// List package_picks directory
	resp, err := github.Get(repoContentsURL)
	if err != nil {
		return "", fmt.Errorf("list package_picks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("list package_picks: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read package_picks listing: %w", err)
	}

	var contents []ghContent
	if err := json.Unmarshal(body, &contents); err != nil {
		return "", fmt.Errorf("parse package_picks listing: %w", err)
	}

	names := make([]string, 0, len(contents))
	for _, c := range contents {
		names = append(names, c.Name)
	}

	return selectPackagePick(names, tag, rocqVersion)
}

// fetchPackagePick downloads and parses a package-pick file.
func fetchPackagePick(filename string) (*packagePickInfo, error) {
	url := rawContentURL + filename
	resp, err := github.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", filename, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filename, err)
	}

	return parsePackagePick(string(body)), nil
}

// FetchManifestForTag fetches a specific release from GitHub, reads its package-pick
// file, and builds a Linux manifest with the actual pinned versions.
func FetchManifestForTag(tag string) (*manifest.Manifest, error) {
	// The release notes tell us which Rocq version this release ships, which is
	// what identifies the package-pick file. Best effort: selectPackagePick
	// falls back to a date-only match when this cannot be determined.
	rocqFromRelease, err := sharedreleases.FetchRocqVersion(tag)
	if err != nil {
		rocqFromRelease = ""
	}

	// Find and fetch the package-pick file for this release
	pickFile, err := findPackagePickFile(tag, rocqFromRelease)
	if err != nil {
		return nil, fmt.Errorf("find package-pick: %w", err)
	}

	pick, err := fetchPackagePick(pickFile)
	if err != nil {
		return nil, fmt.Errorf("fetch package-pick: %w", err)
	}

	rocqVersion := pick.coqTag
	if rocqVersion == "" {
		return nil, fmt.Errorf("COQ_PLATFORM_COQ_TAG not found in %s", pickFile)
	}

	ocamlCompiler := "ocaml-base-compiler." + pick.ocamlVersion
	if pick.ocamlVersion == "" {
		ocamlCompiler = "ocaml-base-compiler.4.14.2" // fallback
	}

	// Build package list from pinned packages only — never guess versions.
	// Packages relevant to the installer, in priority order.
	relevantPackages := []struct {
		name     string
		optional string // "" = required, "skip_vscode" or "with_rocqide" = optional
	}{
		// Core Rocq/Coq packages
		{"coq", ""},
		{"rocq-runtime", ""},
		{"rocq-core", ""},
		{"rocq-stdlib", ""},
		{"rocq-prover", ""},
		// Language servers
		{"vsrocq-language-server", "skip_vscode"},
		{"vscoq-language-server", "skip_vscode"},
		// IDEs
		{"rocqide", "with_rocqide"},
		{"coqide", "with_rocqide"},
	}

	var packages []manifest.OpamPackage
	for _, pkg := range relevantPackages {
		if ver, ok := pick.pinnedPackages[pkg.name]; ok {
			packages = append(packages, manifest.OpamPackage{
				Name:     pkg.name,
				Version:  ver,
				Optional: pkg.optional,
			})
		}
	}

	m := &manifest.Manifest{Base: manifest.NewBase(rocqVersion, tag)}
	m.Assets.Linux.X86_64 = manifest.Asset{
		Type: "opam",
		Opam: manifest.OpamConfig{
			OCamlCompiler: ocamlCompiler,
			SwitchPrefix:  "CP",
			RepoName:      "rocq-released",
			RepoURL:       "https://rocq-prover.org/opam/released",
			Packages:      packages,
		},
	}

	return m, nil
}
