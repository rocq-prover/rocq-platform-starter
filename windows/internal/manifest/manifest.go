package manifest

import (
	"encoding/json"
	"fmt"
	"io/fs"

	sharedmanifest "github.com/justme0606/rocq-platform-starter/shared/manifest"
)

type Asset struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type Assets struct {
	Windows struct {
		X86_64 Asset `json:"x86_64"`
	} `json:"windows"`
}

type Manifest struct {
	Channel         string `json:"channel"`
	RocqVersion     string `json:"rocq_version"`
	PlatformRelease string `json:"platform_release"`
	Assets          Assets `json:"assets"`
}

// Parse parses a manifest from raw JSON bytes.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if m.Assets.Windows.X86_64.URL == "" {
		return nil, fmt.Errorf("manifest: no Windows x86_64 asset URL")
	}

	return &m, nil
}

// Load reads and parses the manifest from an embedded filesystem.
func Load(fsys fs.FS, path string) (*Manifest, error) {
	return sharedmanifest.Load(fsys, path, Parse)
}
