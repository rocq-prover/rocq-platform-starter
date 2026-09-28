package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fixtureContent = "rocq"
	fixtureSHA256  = "ef5352bbceb5e7942d7629250e8081af0bfc2566c422e5331a3dff7dbc8f8c21"

	// A well-formed hash that matches nothing in these tests.
	wrongSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
)

// writeFixture writes content to a file in a per-test temp dir.
func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// TestVerifySHA256_EmptyExpectedSkips pins down the behaviour that turned the
// missing manifest checksums into a silent no-op: an empty expected value is
// accepted without the file even being read. Callers must therefore treat an
// absent checksum as "skipped", not as "verified".
func TestVerifySHA256_EmptyExpectedSkips(t *testing.T) {
	for _, expected := range []string{"", "   ", "\n", "\t "} {
		if err := VerifySHA256("/nonexistent/path/that/does/not/exist", expected); err != nil {
			t.Errorf("VerifySHA256(_, %q) = %v, want nil (skip)", expected, err)
		}
	}
}

func TestVerifySHA256_Match(t *testing.T) {
	path := writeFixture(t, fixtureContent)

	if err := VerifySHA256(path, fixtureSHA256); err != nil {
		t.Errorf("VerifySHA256 with the correct hash = %v, want nil", err)
	}
}

func TestVerifySHA256_MatchIsCaseInsensitive(t *testing.T) {
	path := writeFixture(t, fixtureContent)

	if err := VerifySHA256(path, strings.ToUpper(fixtureSHA256)); err != nil {
		t.Errorf("VerifySHA256 with an uppercase hash = %v, want nil", err)
	}
}

func TestVerifySHA256_MatchIgnoresSurroundingWhitespace(t *testing.T) {
	path := writeFixture(t, fixtureContent)

	if err := VerifySHA256(path, "  "+fixtureSHA256+"\n"); err != nil {
		t.Errorf("VerifySHA256 with a padded hash = %v, want nil", err)
	}
}

func TestVerifySHA256_Mismatch(t *testing.T) {
	path := writeFixture(t, fixtureContent)

	err := VerifySHA256(path, wrongSHA256)
	if err == nil {
		t.Fatal("VerifySHA256 with a wrong hash = nil, want an error")
	}
	if !strings.Contains(err.Error(), "SHA256 mismatch") {
		t.Errorf("error = %q, want it to mention a SHA256 mismatch", err)
	}
	// The error must carry both hashes so the log is actionable.
	if !strings.Contains(err.Error(), fixtureSHA256) {
		t.Errorf("error = %q, want it to report the computed hash", err)
	}
}

// TestVerifySHA256_DetectsSingleByteChange is the property that actually
// matters for a downloaded installer.
func TestVerifySHA256_DetectsSingleByteChange(t *testing.T) {
	tampered := writeFixture(t, fixtureContent+"!")

	if err := VerifySHA256(tampered, fixtureSHA256); err == nil {
		t.Error("VerifySHA256 accepted a tampered file, want a mismatch error")
	}
}

func TestVerifySHA256_MissingFileWithNonEmptyExpected(t *testing.T) {
	err := VerifySHA256(filepath.Join(t.TempDir(), "absent"), wrongSHA256)
	if err == nil {
		t.Fatal("VerifySHA256 on a missing file = nil, want an error")
	}
	if !strings.Contains(err.Error(), "open for checksum") {
		t.Errorf("error = %q, want it to mention opening the file", err)
	}
}

func TestVerifySHA256_EmptyFile(t *testing.T) {
	path := writeFixture(t, "")

	// sha256 of the empty byte string.
	const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if err := VerifySHA256(path, emptySHA256); err != nil {
		t.Errorf("VerifySHA256 on an empty file = %v, want nil", err)
	}
	if err := VerifySHA256(path, fixtureSHA256); err == nil {
		t.Error("VerifySHA256 matched an empty file against a non-empty hash")
	}
}
