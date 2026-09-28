package doctor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// collect returns an onLog callback and a pointer to the lines it received.
func collect() (func(string), *[]string) {
	var lines []string
	return func(s string) { lines = append(lines, s) }, &lines
}

func joined(lines *[]string) string { return strings.Join(*lines, "\n") }

// ---------- InspectInstallDir ----------

func TestInspectInstallDir_Healthy(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "bin"), "")
	mustWrite(t, filepath.Join(dir, "coq-shell"), "")

	if w := InspectInstallDir(dir); w != "" {
		t.Errorf("InspectInstallDir = %q, want no warning", w)
	}
}

func TestInspectInstallDir_Empty(t *testing.T) {
	w := InspectInstallDir(t.TempDir())
	if !strings.Contains(w, "empty") {
		t.Errorf("InspectInstallDir = %q, want it to report an empty directory", w)
	}
}

// TestInspectInstallDir_ShellOnly covers the broken-install signature: the
// launcher survived but nothing else did.
func TestInspectInstallDir_ShellOnly(t *testing.T) {
	for _, names := range [][]string{
		{"coq-shell"},
		{"coq-shell.bat"},
		{"coq-shell.lnk"},
		{"coq-shell.sh"},
		{"coq-shell", "coq-shell.bat"},
		{"COQ-SHELL"}, // case must not matter
	} {
		dir := t.TempDir()
		for _, n := range names {
			mustWrite(t, filepath.Join(dir, n), "")
		}
		if w := InspectInstallDir(dir); !strings.Contains(w, "incomplete") {
			t.Errorf("for %v: InspectInstallDir = %q, want an incomplete-install warning", names, w)
		}
	}
}

func TestInspectInstallDir_Missing(t *testing.T) {
	w := InspectInstallDir(filepath.Join(t.TempDir(), "absent"))
	if !strings.Contains(w, "cannot read directory") {
		t.Errorf("InspectInstallDir = %q, want a read failure", w)
	}
}

// ---------- CheckWorkspace ----------

func TestCheckWorkspace_Missing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	onLog, lines := collect()
	CheckWorkspace(onLog, "rocq-workspace", nil)

	if !strings.Contains(joined(lines), "not found") {
		t.Errorf("log = %q, want a not-found report", joined(lines))
	}
}

func TestCheckWorkspace_ReportsSettingsPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	ws := filepath.Join(home, "rocq-workspace")
	mustMkdir(t, filepath.Join(ws, ".vscode"))
	mustWrite(t, filepath.Join(ws, ".vscode", "settings.json"),
		`{"vsrocq.path": "/opt/bin/vsrocqtop"}`)

	onLog, lines := collect()
	CheckWorkspace(onLog, "rocq-workspace", nil)

	out := joined(lines)
	if !strings.Contains(out, "/opt/bin/vsrocqtop") {
		t.Errorf("log = %q, want the vsrocq.path value", out)
	}
}

func TestCheckWorkspace_SettingsWithoutKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	ws := filepath.Join(home, "rocq-workspace")
	mustMkdir(t, filepath.Join(ws, ".vscode"))
	mustWrite(t, filepath.Join(ws, ".vscode", "settings.json"), `{"other": 1}`)

	onLog, lines := collect()
	CheckWorkspace(onLog, "rocq-workspace", nil)

	if !strings.Contains(joined(lines), "vsrocq.path not set") {
		t.Errorf("log = %q, want 'vsrocq.path not set'", joined(lines))
	}
}

// TestCheckWorkspace_ExtraFiles is the Linux case: the opam activation scripts
// are reported present or missing individually.
func TestCheckWorkspace_ExtraFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	ws := filepath.Join(home, "rocq-workspace")
	mustMkdir(t, ws)
	mustWrite(t, filepath.Join(ws, "activate.sh"), "")

	onLog, lines := collect()
	CheckWorkspace(onLog, "rocq-workspace", []string{"activate.sh", "activate-shell.sh"})

	out := joined(lines)
	if !strings.Contains(out, "activate.sh present") {
		t.Errorf("log = %q, want activate.sh reported present", out)
	}
	if !strings.Contains(out, "activate-shell.sh not found") {
		t.Errorf("log = %q, want activate-shell.sh reported missing", out)
	}
}

// ---------- CheckBinaries ----------

func TestCheckBinaries_NoneOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	onLog, lines := collect()
	CheckBinaries(onLog, BinariesOptions{Names: []string{"rocq", "coqc"}})

	if !strings.Contains(joined(lines), "(none found in PATH)") {
		t.Errorf("log = %q, want the none-found line", joined(lines))
	}
}

func TestCheckBinaries_ReportsFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fixtures are not executable on Windows")
	}
	dir := t.TempDir()
	mustWriteExec(t, filepath.Join(dir, "rocq"), "#!/bin/sh\necho 9.0.0\n")
	t.Setenv("PATH", dir)

	onLog, lines := collect()
	CheckBinaries(onLog, BinariesOptions{Names: []string{"rocq", "coqc"}})

	out := joined(lines)
	if !strings.Contains(out, "rocq") {
		t.Errorf("log = %q, want rocq reported", out)
	}
	if strings.Contains(out, "(none found in PATH)") {
		t.Errorf("log = %q, want no none-found line", out)
	}
	if strings.Contains(out, "version:") {
		t.Errorf("log = %q, want no version without ProbeVersion", out)
	}
}

func TestCheckBinaries_ProbeVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fixtures are not executable on Windows")
	}
	dir := t.TempDir()
	mustWriteExec(t, filepath.Join(dir, "rocq"), "#!/bin/sh\necho 9.0.0\n")
	t.Setenv("PATH", dir)

	onLog, lines := collect()
	CheckBinaries(onLog, BinariesOptions{Names: []string{"rocq"}, ProbeVersion: true})

	if !strings.Contains(joined(lines), "version: 9.0.0") {
		t.Errorf("log = %q, want the probed version", joined(lines))
	}
}

// TestCheckBinaries_SkipVersionFor guards the reason the option exists:
// vsrocqtop is a language server and never returns from --print-version.
func TestCheckBinaries_SkipVersionFor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fixtures are not executable on Windows")
	}
	dir := t.TempDir()
	mustWriteExec(t, filepath.Join(dir, "vsrocqtop"), "#!/bin/sh\necho SHOULD-NOT-RUN\n")
	t.Setenv("PATH", dir)

	onLog, lines := collect()
	CheckBinaries(onLog, BinariesOptions{
		Names:          []string{"vsrocqtop"},
		ProbeVersion:   true,
		SkipVersionFor: []string{"vsrocqtop"},
	})

	out := joined(lines)
	if strings.Contains(out, "SHOULD-NOT-RUN") {
		t.Errorf("log = %q, want vsrocqtop never probed", out)
	}
	if !strings.Contains(out, "vsrocqtop") {
		t.Errorf("log = %q, want vsrocqtop still reported on PATH", out)
	}
}

func TestCheckBinaries_ExeSuffix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("LookPath already handles .exe on Windows")
	}
	dir := t.TempDir()
	mustWriteExec(t, filepath.Join(dir, "rocq.exe"), "#!/bin/sh\n")
	t.Setenv("PATH", dir)

	onLog, lines := collect()
	CheckBinaries(onLog, BinariesOptions{Names: []string{"rocq"}, TryExeSuffix: true})
	if !strings.Contains(joined(lines), "rocq.exe") {
		t.Errorf("log = %q, want rocq.exe found via the suffix", joined(lines))
	}

	onLog2, lines2 := collect()
	CheckBinaries(onLog2, BinariesOptions{Names: []string{"rocq"}})
	if !strings.Contains(joined(lines2), "(none found in PATH)") {
		t.Errorf("log = %q, want no match without TryExeSuffix", joined(lines2))
	}
}

// ---------- helpers ----------

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustWriteExec(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
