package releases

import "testing"

// realPickListing is the actual package_picks directory of
// rocq-prover/platform, so these tests exercise the data the app really sees.
var realPickListing = []string{
	"coq_platform_release.sh",
	"coq_platform_switch_name.sh",
	"package-pick-8.12.sh",
	"package-pick-8.13~2021.02.sh",
	"package-pick-8.13~2021.09.sh",
	"package-pick-8.13~2022.01.sh",
	"package-pick-8.14~2022.01.sh",
	"package-pick-8.14~2022.04.sh",
	"package-pick-8.15~2022.04.sh",
	"package-pick-8.15~2022.09.sh",
	"package-pick-8.16~2022.09.sh",
	"package-pick-8.16~2023.08.sh",
	"package-pick-8.17~2023.08.sh",
	"package-pick-8.18~2023.11.sh",
	"package-pick-8.18~mc2.sh",
	"package-pick-8.19-2024.01+ltac2-debugger.sh",
	"package-pick-8.19~2024.10.sh",
	"package-pick-8.20~2025.01.sh",
	"package-pick-9.0~2025.08.sh",
	"package-pick-9.1~2026.01.sh",
	"package-pick-dev.sh",
}

// TestSelectPackagePick_EveryPublishedRelease pins the resolution for every
// non-prerelease tag the dropdown offers, with the Rocq version each release
// advertises. Every one must resolve: an unresolved release is what made the
// installer fall back to the embedded 9.0 manifest while the UI showed 9.1.
func TestSelectPackagePick_EveryPublishedRelease(t *testing.T) {
	cases := []struct {
		tag, rocqVersion, want string
	}{
		// The reported bug: the release date (2026.07) is not the pick cycle
		// (2026.01), so a date-only match found nothing at all.
		{"2026.07.0", "9.1.0", "package-pick-9.1~2026.01.sh"},

		{"2025.08.3", "9.0.1", "package-pick-9.0~2025.08.sh"},
		{"2025.08.2", "9.0.1", "package-pick-9.0~2025.08.sh"},
		{"2025.08.1", "9.0.1", "package-pick-9.0~2025.08.sh"},
		{"2025.08.0", "9.0.1", "package-pick-9.0~2025.08.sh"},
		{"2025.01.0", "8.20.1", "package-pick-8.20~2025.01.sh"},
		{"2024.10.1", "8.19.2", "package-pick-8.19~2024.10.sh"},
		{"2024.10.0", "8.19.2", "package-pick-8.19~2024.10.sh"},
		{"2023.11.0", "8.18.0", "package-pick-8.18~2023.11.sh"},

		// No 8.17 pick predates this release, so the only 8.17 pick is used.
		{"2023.03.0", "8.17.1", "package-pick-8.17~2023.08.sh"},

		// These four used to resolve to the pick of the *wrong* Rocq version,
		// because two versions share the cycle and the date matched first.
		{"2022.09.1", "8.16.1", "package-pick-8.16~2022.09.sh"},
		{"2022.04.1", "8.15.2", "package-pick-8.15~2022.04.sh"},
		{"2022.04.0", "8.15.1", "package-pick-8.15~2022.04.sh"},
		{"2022.01.0", "8.14.1", "package-pick-8.14~2022.01.sh"},

		{"2021.09.0", "8.13.2", "package-pick-8.13~2021.09.sh"},
		{"2021.02.1", "8.13.2", "package-pick-8.13~2021.02.sh"},
		{"2021.02.0", "8.13.1", "package-pick-8.13~2021.02.sh"},
	}

	for _, c := range cases {
		got, err := selectPackagePick(realPickListing, c.tag, c.rocqVersion)
		if err != nil {
			t.Errorf("release %s (Rocq %s): %v", c.tag, c.rocqVersion, err)
			continue
		}
		if got != c.want {
			t.Errorf("release %s (Rocq %s) = %s, want %s", c.tag, c.rocqVersion, got, c.want)
		}
	}
}

// TestSelectPackagePick_FallsBackWithoutVersion covers release 2021.02.2, whose
// notes carry no parseable version. The legacy date-only match must still
// resolve it rather than failing.
func TestSelectPackagePick_FallsBackWithoutVersion(t *testing.T) {
	got, err := selectPackagePick(realPickListing, "2021.02.2", "")
	if err != nil {
		t.Fatalf("selectPackagePick = %v, want the date-only fallback to succeed", err)
	}
	if got != "package-pick-8.13~2021.02.sh" {
		t.Errorf("got %s, want package-pick-8.13~2021.02.sh", got)
	}
}

// TestSelectPackagePick_PrereleaseVersion covers the pick file pinning
// COQ_PLATFORM_COQ_TAG "9.1+rc1": the suffix must not prevent the match.
func TestSelectPackagePick_PrereleaseVersion(t *testing.T) {
	got, err := selectPackagePick(realPickListing, "2026.07.0", "9.1+rc1")
	if err != nil {
		t.Fatalf("selectPackagePick = %v", err)
	}
	if got != "package-pick-9.1~2026.01.sh" {
		t.Errorf("got %s, want package-pick-9.1~2026.01.sh", got)
	}
}

// TestSelectPackagePick_ExactCycleWins guards the tie-break when several picks
// exist for one Rocq version.
func TestSelectPackagePick_ExactCycleWins(t *testing.T) {
	// 8.13 has picks for 2021.02, 2021.09 and 2022.01.
	for _, c := range []struct{ tag, want string }{
		{"2021.02.0", "package-pick-8.13~2021.02.sh"},
		{"2021.09.0", "package-pick-8.13~2021.09.sh"},
		{"2022.01.5", "package-pick-8.13~2022.01.sh"},
		// Between cycles: the newest one that predates the release.
		{"2021.11.0", "package-pick-8.13~2021.09.sh"},
		// Older than every 8.13 pick: the oldest is still better than nothing.
		{"2020.01.0", "package-pick-8.13~2022.01.sh"},
	} {
		got, err := selectPackagePick(realPickListing, c.tag, "8.13.2")
		if err != nil {
			t.Errorf("%s: %v", c.tag, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %s, want %s", c.tag, got, c.want)
		}
	}
}

// TestSelectPackagePick_IgnoresNonCycleFiles makes sure the loose filenames in
// the directory (dev, mc2, the ltac2-debugger branch, the 8.12 pick with no
// cycle at all) are never selected by the version-aware path.
func TestSelectPackagePick_IgnoresNonCycleFiles(t *testing.T) {
	if got, err := selectPackagePick(realPickListing, "2024.01.0", "8.19.2"); err == nil {
		if got == "package-pick-8.19-2024.01+ltac2-debugger.sh" {
			t.Errorf("selected the ltac2-debugger branch pick (%s)", got)
		}
	}
	if _, err := selectPackagePick(realPickListing, "2030.01.0", "8.12.0"); err == nil {
		t.Error("selectPackagePick resolved 8.12, which has no cycle-tagged pick")
	}
}

func TestSelectPackagePick_Errors(t *testing.T) {
	if _, err := selectPackagePick(realPickListing, "2026", "9.1.0"); err == nil {
		t.Error("accepted a tag without a month, want an error")
	}
	if _, err := selectPackagePick(nil, "2026.07.0", "9.1.0"); err == nil {
		t.Error("accepted an empty listing, want an error")
	}
	if _, err := selectPackagePick(realPickListing, "2099.12.0", "7.4.0"); err == nil {
		t.Error("accepted an unknown Rocq version with no date match, want an error")
	}
}

func TestRocqMajorMinor(t *testing.T) {
	cases := map[string]string{
		"9.0.1":    "9.0",
		"9.1.0":    "9.1",
		"9.1+rc1":  "9.1",
		"8.20.1":   "8.20",
		"8.13":     "8.13",
		"9.1~beta": "9.1",
		"":         "",
		"dev":      "",
		"9":        "",
	}
	for in, want := range cases {
		if got := rocqMajorMinor(in); got != want {
			t.Errorf("rocqMajorMinor(%q) = %q, want %q", in, got, want)
		}
	}
}
