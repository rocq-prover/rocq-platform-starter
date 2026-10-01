package manifest

import (
	"encoding/json"
	"fmt"
	"io/fs"

	sharedmanifest "github.com/rocq-prover/rocq-platform-starter/shared/manifest"
)

// Asset is the signed .exe downloaded on Windows.
type Asset = sharedmanifest.DownloadAsset

type Assets struct {
	Windows struct {
		X86_64 Asset `json:"x86_64"`
	} `json:"windows"`
}

// Manifest is the Windows view of manifest/latest.json. Base is embedded, so
// Channel, RocqVersion, PlatformRelease and Docker are promoted.
type Manifest struct {
	sharedmanifest.Base
	Assets Assets `json:"assets"`
}

// Parse parses a manifest from raw JSON bytes.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if err := sharedmanifest.RequireURL(m.Assets.Windows.X86_64, "Windows x86_64"); err != nil {
		return nil, err
	}

	return &m, nil
}

// NewBase builds the common manifest fields for a runtime-assembled manifest.
func NewBase(rocqVersion, platformRelease string) sharedmanifest.Base {
	return sharedmanifest.NewBase(rocqVersion, platformRelease)
}

// Load reads and parses the manifest from an embedded filesystem.
func Load(fsys fs.FS, path string) (*Manifest, error) {
	return sharedmanifest.Load(fsys, path, Parse)
}
