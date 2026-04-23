package manifest

import (
	"encoding/json"
	"fmt"
	"io/fs"

	sharedmanifest "github.com/justme0606/rocq-platform-starter/shared/manifest"
)

type OpamPackage struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Optional string `json:"optional,omitempty"`
}

type OpamConfig struct {
	OCamlCompiler string        `json:"ocaml_compiler"`
	SwitchPrefix  string        `json:"switch_prefix"`
	RepoName      string        `json:"repo_name"`
	RepoURL       string        `json:"repo_url"`
	Packages      []OpamPackage `json:"packages"`
}

type Asset struct {
	Type string     `json:"type"`
	Opam OpamConfig `json:"opam"`
}

type Assets struct {
	Linux struct {
		X86_64 Asset `json:"x86_64"`
	} `json:"linux"`
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

	if m.Assets.Linux.X86_64.Type != "opam" {
		return nil, fmt.Errorf("manifest: Linux x86_64 asset is not opam type")
	}

	return &m, nil
}

// Load reads and parses the manifest from an embedded filesystem.
func Load(fsys fs.FS, path string) (*Manifest, error) {
	return sharedmanifest.Load(fsys, path, Parse)
}
