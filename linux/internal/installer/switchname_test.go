package installer

import "testing"

func TestSwitchName(t *testing.T) {
	cases := []struct {
		rocqVersion, platformRelease, want string
	}{
		// Existing switches keep the exact names they already have.
		{"9.0.1", "2025.08.1", "CP.2025.08.1~9.0"},
		{"9.0.1", "2025.08.3", "CP.2025.08.3~9.0"},
		{"9.0.0", "2025.08.1", "CP.2025.08.1~9.0"},
		{"8.20.1", "2025.01.0", "CP.2025.01.0~8.20"},

		// package-pick-9.1~2026.01.sh pins COQ_PLATFORM_COQ_TAG "9.1+rc1".
		// Splitting on "." keeps "+rc1" in the minor component and produces
		// CP.2026.07.0~9.1+rc1, which is not the documented format.
		{"9.1+rc1", "2026.07.0", "CP.2026.07.0~9.1"},
		{"9.1.0", "2026.07.0", "CP.2026.07.0~9.1"},
		{"9.1~beta2", "2026.07.0", "CP.2026.07.0~9.1"},

		// Unparseable versions are passed through rather than mangled.
		{"dev", "2026.07.0", "CP.2026.07.0~dev"},
	}

	for _, c := range cases {
		if got := SwitchName(c.rocqVersion, c.platformRelease); got != c.want {
			t.Errorf("SwitchName(%q, %q) = %q, want %q",
				c.rocqVersion, c.platformRelease, got, c.want)
		}
	}
}
