package manifest

import (
	"testing"

	rootfs "github.com/rocq-prover/rocq-platform-starter/windows"
)

// TestLoadEmbedded parses the manifest compiled into the binary. It is the
// regression guard for embedding sharedmanifest.Base: the promoted fields must
// still decode from the same flat JSON.
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

	asset := m.Assets.Windows.X86_64
	if asset.URL == "" {
		t.Error("asset URL is empty")
	}
	// An empty sha256 makes the integrity check a no-op, so the committed
	// manifest must carry one.
	if len(asset.SHA256) != 64 {
		t.Errorf("asset SHA256 = %q, want 64 hex characters", asset.SHA256)
	}
}

func TestParseRejectsMissingURL(t *testing.T) {
	_, err := Parse([]byte(`{
	  "channel": "stable",
	  "rocq_version": "9.0.0",
	  "platform_release": "2025.08.1",
	  "assets": {"windows": {"x86_64": {"type": "exe"}}}
	}`))
	if err == nil {
		t.Fatal("Parse accepted an asset without a URL, want an error")
	}
}

func TestParseRejectsMalformedJSON(t *testing.T) {
	if _, err := Parse([]byte(`{"channel":`)); err == nil {
		t.Fatal("Parse accepted malformed JSON, want an error")
	}
}
