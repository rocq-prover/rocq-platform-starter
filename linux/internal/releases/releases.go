package releases

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	sharedreleases "github.com/justme0606/rocq-platform-starter/shared/releases"

	"github.com/justme0606/rocq-platform-starter/linux/internal/manifest"
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

		// Parse PIN references in PACKAGES lines
		if strings.Contains(line, "PIN.") {
			// Extract all PIN.name.version patterns from the line
			if m := pinRe.FindStringSubmatch(line); m != nil {
				info.pinnedPackages[m[1]] = m[2]
			}
		}
	}

	return info
}

// findPackagePickFile finds the matching package-pick filename for a release tag.
// Tag format: "2025.08.1" → look for file containing "~2025.08" in its name.
func findPackagePickFile(tag string) (string, error) {
	// Extract YYYY.MM from tag (e.g. "2025.08.1" → "2025.08")
	parts := strings.SplitN(tag, ".", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid tag format: %s", tag)
	}
	yearMonth := parts[0] + "." + parts[1]

	// List package_picks directory
	resp, err := http.Get(repoContentsURL)
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

	// Find file matching ~YYYY.MM.sh
	suffix := "~" + yearMonth + ".sh"
	for _, c := range contents {
		if strings.HasSuffix(c.Name, suffix) {
			return c.Name, nil
		}
	}

	return "", fmt.Errorf("no package-pick file found for release %s (looked for *%s)", tag, suffix)
}

// fetchPackagePick downloads and parses a package-pick file.
func fetchPackagePick(filename string) (*packagePickInfo, error) {
	url := rawContentURL + filename
	resp, err := http.Get(url)
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
	// Find and fetch the package-pick file for this release
	pickFile, err := findPackagePickFile(tag)
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

	m := &manifest.Manifest{
		Channel:         "stable",
		RocqVersion:     rocqVersion,
		PlatformRelease: tag,
		Assets: manifest.Assets{
			Linux: struct {
				X86_64 manifest.Asset `json:"x86_64"`
			}{
				X86_64: manifest.Asset{
					Type: "opam",
					Opam: manifest.OpamConfig{
						OCamlCompiler: ocamlCompiler,
						SwitchPrefix:  "CP",
						RepoName:      "rocq-released",
						RepoURL:       "https://rocq-prover.org/opam/released",
						Packages:      packages,
					},
				},
			},
		},
	}

	return m, nil
}
