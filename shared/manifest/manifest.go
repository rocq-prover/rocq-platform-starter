// Package manifest holds the parts of manifest/latest.json that are identical
// on every platform. Each platform embeds Base and declares only its own
// assets block, because the JSON shape genuinely differs: macOS and Windows
// download a signed installer, Linux builds an opam switch.
package manifest

import (
	"fmt"
	"io/fs"
	"strings"
)

// Base contains the manifest fields shared by all platforms. Platform
// manifests embed it, so its fields are promoted both for JSON decoding and
// for callers (m.RocqVersion, m.Docker, ...).
type Base struct {
	Channel         string        `json:"channel"`
	RocqVersion     string        `json:"rocq_version"`
	PlatformRelease string        `json:"platform_release"`
	Docker          *DockerConfig `json:"docker,omitempty"`
}

// DownloadAsset describes an asset fetched over HTTP: the signed .dmg on
// macOS, the signed .exe on Windows.
type DownloadAsset struct {
	Type   string `json:"type"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// OpamPackage is one entry of the Linux asset's pinned package list.
// Optional marks a package the installer may skip: "skip_vscode" for the
// language server, "with_rocqide" for RocqIDE.
type OpamPackage struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Optional string `json:"optional,omitempty"`
}

// OpamConfig is the Linux asset: everything needed to build the opam switch.
type OpamConfig struct {
	OCamlCompiler string        `json:"ocaml_compiler"`
	SwitchPrefix  string        `json:"switch_prefix"`
	RepoName      string        `json:"repo_name"`
	RepoURL       string        `json:"repo_url"`
	Packages      []OpamPackage `json:"packages"`
}

// DockerVariant is one of the pre-built Rocq Platform images.
type DockerVariant struct {
	Image       string `json:"image"`
	Description string `json:"description"`
}

// DockerConfig is the manifest's optional docker section, used by the Linux
// Dev Container mode.
type DockerConfig struct {
	Registry       string                   `json:"registry"`
	Tag            string                   `json:"tag"`
	User           string                   `json:"user"`
	OpamSwitch     string                   `json:"opam_switch"`
	VsrocqtopPath  string                   `json:"vsrocqtop_path"`
	Variants       map[string]DockerVariant `json:"variants"`
	DefaultVariant string                   `json:"default_variant"`
}

// NewBase builds the common fields for a manifest assembled at runtime from a
// GitHub release. Platform code must set this through the embedded Base field
// rather than inline, because assigning promoted fields inside a struct
// literal requires Go 1.27 and these modules target 1.22.
func NewBase(rocqVersion, platformRelease string) Base {
	return Base{
		Channel:         "stable",
		RocqVersion:     rocqVersion,
		PlatformRelease: platformRelease,
	}
}

// Load reads a manifest file from an embedded filesystem and unmarshals it
// using the provided parse function.
func Load[T any](fsys fs.FS, path string, parse func([]byte) (*T, error)) (*T, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	return parse(data)
}

// RequireURL returns an error when a download asset carries no URL. target
// names the asset for the message, e.g. "macOS arm64".
func RequireURL(asset DownloadAsset, target string) error {
	if strings.TrimSpace(asset.URL) == "" {
		return fmt.Errorf("manifest: no %s asset URL", target)
	}
	return nil
}
