package releases

import "testing"

func TestGHAssetSHA256(t *testing.T) {
	const hex = "ca1d2a1a7e137dc82327a128bc1d709426f3038a4771a85f29a8d5524b56ab47"

	cases := []struct {
		name   string
		digest string
		want   string
	}{
		{"sha256 digest", "sha256:" + hex, hex},
		{"surrounding whitespace", "  sha256:" + hex + "\n", hex},
		{"absent", "", ""},
		{"other algorithm", "sha512:" + hex, ""},
		{"no prefix", hex, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := (GHAsset{Digest: c.digest}).SHA256(); got != c.want {
				t.Errorf("SHA256() = %q, want %q", got, c.want)
			}
		})
	}
}

// assets mirrors the shape of a real Rocq Platform release: an Intel DMG, an
// Apple Silicon DMG, a Windows exe, plus unsigned noise that must be ignored.
var assets = []GHAsset{
	{Name: "Rocq-Platform-release-unsigned.dmg", BrowserDownloadURL: "u1"},
	{Name: "signed_Rocq-Platform-release-2025.08.2-intel.dmg", BrowserDownloadURL: "intel"},
	{Name: "signed_Rocq-Platform-release-2025.08.2.dmg", BrowserDownloadURL: "arm"},
	{Name: "signed_Rocq-Platform-release-2025.08.1-Windows-x86_64.exe", BrowserDownloadURL: "exe"},
	{Name: "SHA256SUMS.txt", BrowserDownloadURL: "sums"},
}

func TestFindSignedAsset_PrefersAppleSilicon(t *testing.T) {
	got, ok := FindSignedAsset(assets, ".dmg", NotIntel)
	if !ok {
		t.Fatal("no asset found, want the Apple Silicon DMG")
	}
	if got.BrowserDownloadURL != "arm" {
		t.Errorf("picked %q, want the non-Intel DMG", got.BrowserDownloadURL)
	}
}

// TestFindSignedAsset_FallsBackToRejected guards the reason prefer is a
// preference and not a filter: a release shipping only an Intel DMG must still
// resolve rather than failing outright.
func TestFindSignedAsset_FallsBackToRejected(t *testing.T) {
	onlyIntel := []GHAsset{
		{Name: "signed_Rocq-Platform-intel.dmg", BrowserDownloadURL: "intel"},
	}

	got, ok := FindSignedAsset(onlyIntel, ".dmg", NotIntel)
	if !ok {
		t.Fatal("no asset found, want the Intel DMG as fallback")
	}
	if got.BrowserDownloadURL != "intel" {
		t.Errorf("picked %q, want the Intel fallback", got.BrowserDownloadURL)
	}
}

func TestFindSignedAsset_NilPreferTakesFirstMatch(t *testing.T) {
	got, ok := FindSignedAsset(assets, ".exe", nil)
	if !ok {
		t.Fatal("no asset found, want the Windows exe")
	}
	if got.BrowserDownloadURL != "exe" {
		t.Errorf("picked %q, want the exe", got.BrowserDownloadURL)
	}
}

func TestFindSignedAsset_IgnoresUnsignedAndOtherSuffixes(t *testing.T) {
	if got, ok := FindSignedAsset(assets, ".tar.gz", nil); ok {
		t.Errorf("found %q for an absent suffix, want no match", got.Name)
	}

	unsignedOnly := []GHAsset{{Name: "Rocq-Platform.dmg", BrowserDownloadURL: "u"}}
	if got, ok := FindSignedAsset(unsignedOnly, ".dmg", nil); ok {
		t.Errorf("found unsigned asset %q, want no match", got.Name)
	}
}

func TestFindSignedAsset_CarriesTheDigest(t *testing.T) {
	const hex = "b858d2272f943e27499c54f70362088a945d1cd1393563c5c23dbcadd702ea0b"
	withDigest := []GHAsset{
		{Name: "signed_x.exe", BrowserDownloadURL: "exe", Digest: "sha256:" + hex},
	}

	got, ok := FindSignedAsset(withDigest, ".exe", nil)
	if !ok {
		t.Fatal("no asset found")
	}
	if got.SHA256() != hex {
		t.Errorf("SHA256() = %q, want the digest to survive selection", got.SHA256())
	}
}

func TestNotIntel(t *testing.T) {
	intel := []string{
		"signed_Rocq-intel.dmg", "signed_Rocq-INTEL.dmg",
		"signed_Rocq-x86_64.dmg", "signed_Rocq-amd64.dmg",
	}
	for _, n := range intel {
		if NotIntel(n) {
			t.Errorf("NotIntel(%q) = true, want false", n)
		}
	}

	if !NotIntel("signed_Rocq-Platform-release-2025.08.2.dmg") {
		t.Error("NotIntel on a plain DMG = false, want true")
	}
}
