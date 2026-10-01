package manifest

import (
	"encoding/json"
	"fmt"
	"io/fs"

	sharedmanifest "github.com/rocq-prover/rocq-platform-starter/shared/manifest"
)

// Re-exported so callers in this module keep using manifest.OpamPackage etc.
type (
	OpamPackage   = sharedmanifest.OpamPackage
	OpamConfig    = sharedmanifest.OpamConfig
	DockerVariant = sharedmanifest.DockerVariant
	DockerConfig  = sharedmanifest.DockerConfig
)

// Asset is the Linux asset: an opam switch description rather than a download.
type Asset struct {
	Type string     `json:"type"`
	Opam OpamConfig `json:"opam"`
}

type Assets struct {
	Linux struct {
		X86_64 Asset `json:"x86_64"`
	} `json:"linux"`
}

// Manifest is the Linux view of manifest/latest.json. Base is embedded, so
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

	if m.Assets.Linux.X86_64.Type != "opam" {
		return nil, fmt.Errorf("manifest: Linux x86_64 asset is not opam type")
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
