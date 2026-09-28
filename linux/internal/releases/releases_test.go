package releases

import (
	"testing"
)

// Minimal package pick with only PIN entries (old format).
const pickOnlyPIN = `
COQ_PLATFORM_COQ_TAG="9.0.1"
COQ_PLATFORM_OCAML_VERSION="4.14.2"

PACKAGES=""
PACKAGES="${PACKAGES} PIN.ocamlfind.1.9.5~relocatable"
PACKAGES="${PACKAGES} PIN.dune.3.19.1"
PACKAGES="${PACKAGES} PIN.coq.9.0.1"
PACKAGES="${PACKAGES} PIN.rocq-stdlib.9.0.0"
`

// Package pick mixing PIN and non-PIN entries (real format).
const pickMixed = `
COQ_PLATFORM_COQ_TAG="9.0.1"
COQ_PLATFORM_OCAML_VERSION="4.14.2"

PACKAGES=""
PACKAGES="${PACKAGES} PIN.ocamlfind.1.9.5~relocatable"
PACKAGES="${PACKAGES} PIN.dune.3.19.1"
PACKAGES="${PACKAGES} PIN.dune-configurator.3.19.1"
PACKAGES="${PACKAGES} PIN.coq.9.0.1"
PACKAGES="${PACKAGES} PIN.rocq-stdlib.9.0.0"

PACKAGES="${PACKAGES} rocqide.9.0.1"
PACKAGES="${PACKAGES} vsrocq-language-server.2.3.4"
`

// Multiple packages on a single line.
const pickMultiPerLine = `
COQ_PLATFORM_COQ_TAG="9.0.1"
COQ_PLATFORM_OCAML_VERSION="4.14.2"

PACKAGES=""
PACKAGES="${PACKAGES} elpi.3.1.0 rocq-elpi.3.1.0"
PACKAGES="${PACKAGES} coq-menhirlib.20240715 menhir.20240715"
`

// Realistic excerpt from the 9.0~2025.08 pick file.
const pickRealistic = `#!/usr/bin/env bash

COQ_PLATFORM_VERSION_TITLE="Rocq 9.0.1 (released March 2025) with the preview package pick from July 2025"
COQ_PLATFORM_VERSION_SORTORDER="2"
COQ_PLATFORM_PACKAGE_PICK_POSTFIX="~9.0~2025.08"
COQ_PLATFORM_COQ_BRANCH="v9.0.1"
COQ_PLATFORM_COQ_TAG="9.0.1"
COQ_PLATFORM_USE_DEV_REPOSITORY="N"
COQ_PLATFORM_OCAML_VERSION="4.14.2"

PACKAGES=""
PACKAGES="${PACKAGES} PIN.ocamlfind.1.9.5~relocatable"
PACKAGES="${PACKAGES} PIN.dune.3.19.1"
PACKAGES="${PACKAGES} PIN.dune-configurator.3.19.1"
PACKAGES="${PACKAGES} PIN.coq.9.0.1"
PACKAGES="${PACKAGES} PIN.rocq-stdlib.9.0.0"

if  [[ "${COQ_PLATFORM_EXTENT}"  =~ ^[iIfFxX] ]]
then
PACKAGES="${PACKAGES} rocqide.9.0.1"
PACKAGES="${PACKAGES} vsrocq-language-server.2.3.4"
fi

if  [[ "${COQ_PLATFORM_EXTENT}"  =~ ^[fFxX] ]]
then
  PACKAGES="${PACKAGES} sexplib.v0.16.0"
  PACKAGES="${PACKAGES} rocq-bignums.9.0.0+rocq9.0"
  PACKAGES="${PACKAGES} coq-ext-lib.0.13.0"
  PACKAGES="${PACKAGES} coq-stdpp.1.12.0"
  PACKAGES="${PACKAGES} elpi.3.1.0 rocq-elpi.3.1.0"
  PACKAGES="${PACKAGES} rocq-hierarchy-builder.1.10.2"
  PACKAGES="${PACKAGES} coq-mathcomp-ssreflect.2.4.0"
  PACKAGES="${PACKAGES} coq-flocq.4.2.1"
  PACKAGES="${PACKAGES} coq-interval.4.11.3"
  PACKAGES="${PACKAGES} coq-hott.9.0"
  PACKAGES="${PACKAGES} rocq-equations.1.3.1+9.0"
  PACKAGES="${PACKAGES} rocq-aac-tactics.9.0.0"
  PACKAGES="${PACKAGES} coq-hammer.1.3.2+9.0"
  PACKAGES="${PACKAGES} eprover.3.1"
  PACKAGES="${PACKAGES} z3_tptp.4.13.0"
  PACKAGES="${PACKAGES} rocq-libhyps.4.0"
  PACKAGES="${PACKAGES} rocq-relation-algebra.1.8.0"
fi
`

func TestParsePackagePick_Variables(t *testing.T) {
	info := parsePackagePick(pickMixed)

	if info.coqTag != "9.0.1" {
		t.Errorf("coqTag = %q, want %q", info.coqTag, "9.0.1")
	}
	if info.ocamlVersion != "4.14.2" {
		t.Errorf("ocamlVersion = %q, want %q", info.ocamlVersion, "4.14.2")
	}
}

func TestParsePackagePick_PINPackages(t *testing.T) {
	info := parsePackagePick(pickOnlyPIN)

	want := map[string]string{
		"ocamlfind":   "1.9.5~relocatable",
		"dune":        "3.19.1",
		"coq":         "9.0.1",
		"rocq-stdlib": "9.0.0",
	}
	for name, version := range want {
		got, ok := info.pinnedPackages[name]
		if !ok {
			t.Errorf("missing PIN package %q", name)
		} else if got != version {
			t.Errorf("package %q = %q, want %q", name, got, version)
		}
	}
}

func TestParsePackagePick_NonPINPackages(t *testing.T) {
	info := parsePackagePick(pickMixed)

	// Non-PIN packages that must be captured
	want := map[string]string{
		"rocqide":                "9.0.1",
		"vsrocq-language-server": "2.3.4",
	}
	for name, version := range want {
		got, ok := info.pinnedPackages[name]
		if !ok {
			t.Errorf("missing non-PIN package %q", name)
		} else if got != version {
			t.Errorf("package %q = %q, want %q", name, got, version)
		}
	}

	// PIN packages should still be present too
	if v, ok := info.pinnedPackages["coq"]; !ok || v != "9.0.1" {
		t.Errorf("PIN package coq missing or wrong: got %q", v)
	}
}

func TestParsePackagePick_MultiplePackagesPerLine(t *testing.T) {
	info := parsePackagePick(pickMultiPerLine)

	want := map[string]string{
		"elpi":          "3.1.0",
		"rocq-elpi":     "3.1.0",
		"coq-menhirlib": "20240715",
		"menhir":        "20240715",
	}
	for name, version := range want {
		got, ok := info.pinnedPackages[name]
		if !ok {
			t.Errorf("missing package %q (multiple-per-line)", name)
		} else if got != version {
			t.Errorf("package %q = %q, want %q", name, got, version)
		}
	}
}

func TestParsePackagePick_PINTakesPriority(t *testing.T) {
	// A PIN.name.version entry should not be overwritten by the pkgRe
	// match of the same token (PIN.coq.9.0.1 also matches pkgRe on coq.9.0.1).
	info := parsePackagePick(pickOnlyPIN)

	if v := info.pinnedPackages["coq"]; v != "9.0.1" {
		t.Errorf("PIN priority: coq = %q, want %q", v, "9.0.1")
	}
}

func TestParsePackagePick_VersionFormats(t *testing.T) {
	// Test various version formats seen in real pick files.
	content := `
COQ_PLATFORM_COQ_TAG="9.0.1"
COQ_PLATFORM_OCAML_VERSION="4.14.2"
PACKAGES=""
PACKAGES="${PACKAGES} sexplib.v0.16.0"
PACKAGES="${PACKAGES} rocq-bignums.9.0.0+rocq9.0"
PACKAGES="${PACKAGES} PIN.ocamlfind.1.9.5~relocatable"
PACKAGES="${PACKAGES} rocq-equations.1.3.1+9.0"
PACKAGES="${PACKAGES} coq-paramcoq.1.1.3+rocq9.0"
PACKAGES="${PACKAGES} coq-hott.9.0"
`
	info := parsePackagePick(content)

	want := map[string]string{
		"sexplib":        "v0.16.0",
		"rocq-bignums":   "9.0.0+rocq9.0",
		"ocamlfind":      "1.9.5~relocatable",
		"rocq-equations": "1.3.1+9.0",
		"coq-paramcoq":   "1.1.3+rocq9.0",
		"coq-hott":       "9.0",
	}
	for name, version := range want {
		got, ok := info.pinnedPackages[name]
		if !ok {
			t.Errorf("missing package %q", name)
		} else if got != version {
			t.Errorf("package %q = %q, want %q", name, got, version)
		}
	}
}

func TestParsePackagePick_IgnoresNonPackageLines(t *testing.T) {
	// Lines that don't contain PACKAGES should not produce matches.
	content := `
COQ_PLATFORM_COQ_TAG="9.0.1"
COQ_PLATFORM_OCAML_VERSION="4.14.2"
COQ_PLATFORM_PACKAGE_PICK_POSTFIX="~9.0~2025.08"
# This is a comment mentioning coq.9.0.1 but not in PACKAGES
echo "installing rocqide.9.0.1"
PACKAGES=""
PACKAGES="${PACKAGES} PIN.coq.9.0.1"
`
	info := parsePackagePick(content)

	if len(info.pinnedPackages) != 1 {
		t.Errorf("expected 1 pinned package, got %d: %v", len(info.pinnedPackages), info.pinnedPackages)
	}
	if v := info.pinnedPackages["coq"]; v != "9.0.1" {
		t.Errorf("coq = %q, want %q", v, "9.0.1")
	}
}

// TestParsePackagePick_RealisticInstallerPackages verifies that the packages
// the installer cares about (core Rocq, language server, IDE) are all correctly
// extracted from a realistic pick file.
func TestParsePackagePick_RealisticInstallerPackages(t *testing.T) {
	info := parsePackagePick(pickRealistic)

	// These are the packages that FetchManifestForTag looks up.
	required := map[string]string{
		"coq":                    "9.0.1",
		"rocq-stdlib":            "9.0.0",
		"vsrocq-language-server": "2.3.4",
		"rocqide":                "9.0.1",
	}
	for name, version := range required {
		got, ok := info.pinnedPackages[name]
		if !ok {
			t.Errorf("installer-relevant package %q is missing from parsed output", name)
		} else if got != version {
			t.Errorf("package %q = %q, want %q", name, got, version)
		}
	}
}

// TestParsePackagePick_RealisticPackageCount ensures that a realistic pick
// file produces a reasonable number of pinned packages (not just the PIN ones).
func TestParsePackagePick_RealisticPackageCount(t *testing.T) {
	info := parsePackagePick(pickRealistic)

	// The realistic excerpt has 5 PIN + ~25 non-PIN packages.
	// With both regexes working we should get well over 20.
	if got := len(info.pinnedPackages); got < 20 {
		t.Errorf("expected at least 20 pinned packages from realistic pick, got %d", got)
	}
}

func TestParsePackagePick_EmptyInput(t *testing.T) {
	info := parsePackagePick("")
	if info.coqTag != "" {
		t.Errorf("coqTag = %q, want empty", info.coqTag)
	}
	if len(info.pinnedPackages) != 0 {
		t.Errorf("expected no pinned packages, got %d", len(info.pinnedPackages))
	}
}
