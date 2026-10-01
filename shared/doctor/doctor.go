// Package doctor holds the diagnostic checks that are identical on every
// platform. How a Rocq installation is *found* differs completely (opam
// switches on Linux, .app bundles on macOS, the registry on Windows), so
// discovery stays in each platform's own doctor; what is inspected afterwards
// -- VSCode, the binaries on PATH, the workspace -- does not.
//
// Every check reports through an onLog callback and must never exit the
// process: the GUI runs the doctor in-process.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// versionTimeout caps a --print-version probe. Some binaries in a broken
// installation never return.
const versionTimeout = 5 * time.Second

// CheckVSCode reports the VSCode CLI and the Rocq/Coq extensions installed in
// it. findCode is the platform's VSCode lookup.
func CheckVSCode(onLog func(string), findCode func() (string, error)) (vsrocqFound, vscoqFound bool) {
	codeBin, err := findCode()
	if err != nil {
		onLog("  VSCode not found")
		return false, false
	}
	onLog(fmt.Sprintf("  CLI: %s", codeBin))

	out, err := exec.Command(codeBin, "--list-extensions", "--show-versions").Output()
	if err != nil {
		onLog("  (could not list extensions)")
		return false, false
	}

	onLog("  Extensions:")
	anyExt := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "rocq") && !strings.Contains(lower, "coq") {
			continue
		}
		onLog(fmt.Sprintf("    %s", line))
		anyExt = true
		if strings.Contains(lower, "vsrocq") {
			vsrocqFound = true
		}
		if strings.Contains(lower, "vscoq") {
			vscoqFound = true
		}
	}
	if !anyExt {
		onLog("    (no Rocq/Coq extensions)")
	}
	if !vsrocqFound {
		onLog("  ⚠ vsrocq extension not found")
	}
	if vscoqFound {
		onLog("  ⚠ vscoq extension detected (deprecated, use vsrocq instead)")
	}

	return vsrocqFound, vscoqFound
}

// BinariesOptions configures CheckBinaries.
type BinariesOptions struct {
	// Names are the binaries to look for, in report order.
	Names []string
	// TryExeSuffix also tries name+".exe" (Windows).
	TryExeSuffix bool
	// ProbeVersion runs each binary with --print-version.
	ProbeVersion bool
	// SkipVersionFor lists binaries to never probe, such as language servers
	// that block instead of printing a version.
	SkipVersionFor []string
}

// CheckBinaries reports which of the named binaries are on PATH.
func CheckBinaries(onLog func(string), opts BinariesOptions) {
	suffixes := []string{""}
	if opts.TryExeSuffix {
		suffixes = append(suffixes, ".exe")
	}

	anyFound := false
	for _, name := range opts.Names {
		for _, suffix := range suffixes {
			p, err := exec.LookPath(name + suffix)
			if err != nil {
				continue
			}
			onLog(fmt.Sprintf("  %s → %s", name, p))
			anyFound = true

			if opts.ProbeVersion && !slices.Contains(opts.SkipVersionFor, name) {
				if ver := probeVersion(p); ver != "" {
					onLog(fmt.Sprintf("    version: %s", ver))
				}
			}
			break
		}
	}

	if !anyFound {
		onLog("  (none found in PATH)")
	}
}

// probeVersion runs binPath --print-version under a timeout, returning "" on
// any failure.
func probeVersion(binPath string) string {
	ctx, cancel := context.WithTimeout(context.Background(), versionTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, binPath, "--print-version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CheckWorkspace reports the workspace directory, the language server path
// recorded in its .vscode/settings.json, and each of extraFiles (the opam
// activation scripts on Linux; nothing elsewhere).
func CheckWorkspace(onLog func(string), workspaceName string, extraFiles []string) {
	home, err := os.UserHomeDir()
	if err != nil {
		onLog("  (could not determine home directory)")
		return
	}

	wsDir := filepath.Join(home, workspaceName)
	info, err := os.Stat(wsDir)
	if err != nil || !info.IsDir() {
		onLog(fmt.Sprintf("  %s not found", wsDir))
		return
	}
	onLog(fmt.Sprintf("  ✓ %s", wsDir))

	reportSettings(onLog, filepath.Join(wsDir, ".vscode", "settings.json"))

	for _, name := range extraFiles {
		if _, err := os.Stat(filepath.Join(wsDir, name)); err == nil {
			onLog(fmt.Sprintf("  ✓ %s present", name))
		} else {
			onLog(fmt.Sprintf("  ⚠ %s not found", name))
		}
	}
}

// reportSettings reports the vsrocq.path entry of a workspace settings file.
func reportSettings(onLog func(string), settingsPath string) {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		onLog("  .vscode/settings.json not found")
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}
	if v, ok := settings["vsrocq.path"]; ok {
		onLog(fmt.Sprintf("  settings.json: vsrocq.path = %v", v))
	} else {
		onLog("  settings.json: vsrocq.path not set")
	}
}

// shellOnlyNames are the launcher files a Rocq Platform installer leaves
// behind even when the rest of the install failed. A directory containing
// nothing else is incomplete.
var shellOnlyNames = []string{"coq-shell", "coq-shell.lnk", "coq-shell.bat", "coq-shell.sh"}

// InspectInstallDir returns a human-readable warning when dir does not look
// like a usable Rocq installation, or "" when it does. Callers pass an
// already-resolved directory: on macOS that means Contents/Resources inside
// the .app, not the bundle itself.
func InspectInstallDir(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("cannot read directory: %v", err)
	}
	if len(entries) == 0 {
		return "installation directory is empty"
	}

	for _, e := range entries {
		if !slices.Contains(shellOnlyNames, strings.ToLower(e.Name())) {
			return ""
		}
	}
	return "installation contains only coq-shell — installation appears incomplete"
}
