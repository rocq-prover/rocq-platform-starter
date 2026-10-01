package gui

import "testing"

// TestDockerSupportsRocqVersion pins the Docker button's visibility rule: it
// is a hand-maintained allowlist, not every release, because the Docker mode
// only has pre-built GHCR images for Rocq 9.0 and 9.1 so far. Before this,
// releases.FetchManifestForTag never populated Docker at all, so picking any
// release other than the embedded default hid the button regardless of its
// Rocq version.
func TestDockerSupportsRocqVersion(t *testing.T) {
	cases := []struct {
		rocqVersion string
		want        bool
	}{
		{"9.0.0", true},
		{"9.0.1", true},
		{"9.1.0", true},
		{"9.1+rc1", true}, // the pre-release tag package-pick-9.1~2026.01.sh pins

		{"8.20.1", false},
		{"9.2.0", false},
		{"8.13.2", false},
		{"", false},
		{"dev", false},
	}

	for _, c := range cases {
		if got := dockerSupportsRocqVersion(c.rocqVersion); got != c.want {
			t.Errorf("dockerSupportsRocqVersion(%q) = %v, want %v", c.rocqVersion, got, c.want)
		}
	}
}
