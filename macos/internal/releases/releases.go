package releases

import (
	"fmt"
	"strings"

	sharedreleases "github.com/justme0606/rocq-platform-starter/shared/releases"

	"github.com/justme0606/rocq-platform-starter/macos/internal/manifest"
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

func findSignedDMG(assets []sharedreleases.GHAsset) (string, string) {
	for _, a := range assets {
		if strings.HasPrefix(a.Name, "signed_") && strings.HasSuffix(a.Name, ".dmg") {
			// Prefer non-intel DMG (arm64)
			if !strings.Contains(strings.ToLower(a.Name), "intel") &&
				!strings.Contains(strings.ToLower(a.Name), "x86_64") &&
				!strings.Contains(strings.ToLower(a.Name), "amd64") {
				return a.BrowserDownloadURL, a.Name
			}
		}
	}
	// Fallback: any signed DMG
	for _, a := range assets {
		if strings.HasPrefix(a.Name, "signed_") && strings.HasSuffix(a.Name, ".dmg") {
			return a.BrowserDownloadURL, a.Name
		}
	}
	return "", ""
}

// FetchManifestForTag fetches a specific release from GitHub and builds a macOS manifest.
func FetchManifestForTag(tag string) (*manifest.Manifest, error) {
	rel, err := sharedreleases.FetchReleaseDetail(tag)
	if err != nil {
		return nil, fmt.Errorf("fetch release %s: %w", tag, err)
	}

	rocqVersion := sharedreleases.InferRocqVersion(rel.Body)
	if rocqVersion == "" {
		return nil, fmt.Errorf("could not infer Rocq version from release %s body", tag)
	}

	dmgURL, _ := findSignedDMG(rel.Assets)
	if dmgURL == "" {
		return nil, fmt.Errorf("no signed .dmg asset found for release %s", tag)
	}

	m := &manifest.Manifest{
		Channel:         "stable",
		RocqVersion:     rocqVersion,
		PlatformRelease: tag,
		Assets: manifest.Assets{
			MacOS: struct {
				ARM64 manifest.Asset `json:"arm64"`
			}{
				ARM64: manifest.Asset{
					Type: "dmg",
					URL:  dmgURL,
				},
			},
		},
	}

	return m, nil
}
