package releases

import (
	sharedreleases "github.com/rocq-prover/rocq-platform-starter/shared/releases"

	"github.com/rocq-prover/rocq-platform-starter/macos/internal/manifest"
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

// FetchManifestForTag fetches a specific release from GitHub and builds a macOS
// manifest. The asset's sha256 comes from the digest GitHub publishes, so a
// release picked at runtime is verified like the embedded one.
func FetchManifestForTag(tag string) (*manifest.Manifest, error) {
	rocqVersion, asset, err := sharedreleases.ResolveDownloadAsset(tag, ".dmg", sharedreleases.NotIntel)
	if err != nil {
		return nil, err
	}

	m := &manifest.Manifest{Base: manifest.NewBase(rocqVersion, tag)}
	m.Assets.MacOS.ARM64 = manifest.Asset{
		Type:   "dmg",
		URL:    asset.BrowserDownloadURL,
		SHA256: asset.SHA256(),
	}

	return m, nil
}
