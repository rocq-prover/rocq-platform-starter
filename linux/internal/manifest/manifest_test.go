package manifest

import (
	"testing"

	rootfs "github.com/rocq-prover/rocq-platform-starter/linux"
)

// TestLoadEmbedded parses the manifest that is actually compiled into the
// binary. It is the regression guard for embedding sharedmanifest.Base: the
// promoted fields must still decode from the same flat JSON.
func TestLoadEmbedded(t *testing.T) {
	m, err := Load(rootfs.EmbeddedManifest, "embedded/manifest/latest.json")
	if err != nil {
		t.Fatalf("Load = %v, want nil", err)
	}

	// Promoted from the embedded Base.
	if m.Channel == "" {
		t.Error("Channel is empty")
	}
	if m.RocqVersion == "" {
		t.Error("RocqVersion is empty")
	}
	if m.PlatformRelease == "" {
		t.Error("PlatformRelease is empty")
	}

	// Platform-specific block.
	opam := m.Assets.Linux.X86_64.Opam
	if opam.OCamlCompiler == "" {
		t.Error("opam.OCamlCompiler is empty")
	}
	if opam.RepoURL == "" {
		t.Error("opam.RepoURL is empty")
	}
	if len(opam.Packages) == 0 {
		t.Fatal("opam.Packages is empty")
	}
	for _, p := range opam.Packages {
		if p.Name == "" || p.Version == "" {
			t.Errorf("package %+v has an empty name or version", p)
		}
	}
}

// TestLoadEmbeddedDocker covers the section that silently went missing from
// the macOS and Windows embedded copies.
func TestLoadEmbeddedDocker(t *testing.T) {
	m, err := Load(rootfs.EmbeddedManifest, "embedded/manifest/latest.json")
	if err != nil {
		t.Fatalf("Load = %v, want nil", err)
	}

	if m.Docker == nil {
		t.Fatal("Docker section is nil; the Linux Docker mode would be hidden")
	}
	if m.Docker.Registry == "" || m.Docker.Tag == "" {
		t.Error("Docker registry or tag is empty")
	}
	if m.Docker.VsrocqtopPath == "" {
		t.Error("Docker vsrocqtop_path is empty")
	}
	if _, ok := m.Docker.Variants[m.Docker.DefaultVariant]; !ok {
		t.Errorf("default_variant %q is not in variants %v",
			m.Docker.DefaultVariant, m.Docker.Variants)
	}
}

func TestParseRejectsNonOpamLinuxAsset(t *testing.T) {
	_, err := Parse([]byte(`{
	  "channel": "stable",
	  "rocq_version": "9.0.0",
	  "platform_release": "2025.08.1",
	  "assets": {"linux": {"x86_64": {"type": "dmg"}}}
	}`))
	if err == nil {
		t.Fatal("Parse accepted a non-opam Linux asset, want an error")
	}
}

func TestParseRejectsMalformedJSON(t *testing.T) {
	if _, err := Parse([]byte(`{"channel":`)); err == nil {
		t.Fatal("Parse accepted malformed JSON, want an error")
	}
}
