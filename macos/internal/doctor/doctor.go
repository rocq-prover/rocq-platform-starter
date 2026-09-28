package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	shareddoctor "github.com/rocq-prover/rocq-platform-starter/shared/doctor"

	"github.com/rocq-prover/rocq-platform-starter/macos/internal/installer"
	"github.com/rocq-prover/rocq-platform-starter/macos/internal/vscode"
)

// probedBinaries are the Rocq/Coq binaries the doctor looks for on PATH.
var probedBinaries = []string{"rocq", "coqtop", "coqc", "vsrocqtop"}

// installation holds info about a found Rocq installation.
type installation struct {
	path    string
	version string
}

// Run performs system diagnostics and reports findings via onLog callback.
func Run(onLog func(string)) {
	onLog("=== Rocq/Coq Installations ===")
	installFound := checkInstallationsMacOS(onLog)

	onLog("")
	onLog("=== Binaries in PATH ===")
	shareddoctor.CheckBinaries(onLog, shareddoctor.BinariesOptions{Names: probedBinaries})

	onLog("")
	onLog("=== opam ===")
	checkOpam(onLog)

	onLog("")
	onLog("=== VSCode ===")
	vsrocqFound, vscoqFound := shareddoctor.CheckVSCode(onLog, vscode.FindCode)

	onLog("")
	onLog("=== Workspace ===")
	shareddoctor.CheckWorkspace(onLog, installer.WorkspaceName, nil)

	onLog("")
	onLog("=== Potential Issues ===")
	checkIssues(onLog, installFound, vsrocqFound, vscoqFound)
}

// checkAppContent resolves an .app bundle to its Resources directory before
// inspecting it, then defers to the shared check.
func checkAppContent(dir string) string {
	resourcesDir := dir
	if strings.HasSuffix(dir, ".app") {
		resourcesDir = filepath.Join(dir, "Contents", "Resources")
	}
	return shareddoctor.InspectInstallDir(resourcesDir)
}

func getRocqVersion(binPath string) string {
	out, err := exec.Command(binPath, "--print-version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
func checkInstallationsMacOS(onLog func(string)) bool {
	var found []installation

	home, _ := os.UserHomeDir()
	searchDirs := []string{"/Applications"}
	if home != "" {
		searchDirs = append(searchDirs, filepath.Join(home, "Applications"))
	}

	// 1. Glob for Rocq/Coq .app bundles
	for _, dir := range searchDirs {
		for _, pattern := range []string{"*[Rr]ocq*.app", "*[Cc]oq*.app"} {
			matches, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				continue
			}
			for _, m := range matches {
				info, err := os.Stat(m)
				if err != nil || !info.IsDir() {
					continue
				}
				// Try to find rocq binary inside the .app
				ver := ""
				binPath := filepath.Join(m, "Contents", "Resources", "bin", "rocq")
				if _, err := os.Stat(binPath); err == nil {
					ver = getRocqVersion(binPath)
				}
				found = append(found, installation{path: m, version: ver})
			}
		}
	}

	// 2. PATH lookup
	if rocqPath, err := exec.LookPath("rocq"); err == nil {
		dir := rocqPath
		// Walk up to find .app
		appPath := ""
		d := filepath.Dir(rocqPath)
		for i := 0; i < 6; i++ {
			if strings.HasSuffix(d, ".app") {
				appPath = d
				break
			}
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
		if appPath != "" {
			if !alreadyFound(found, appPath) {
				ver := getRocqVersion(rocqPath)
				found = append(found, installation{path: appPath, version: ver})
			}
		} else if !alreadyFound(found, dir) {
			ver := getRocqVersion(rocqPath)
			found = append(found, installation{path: rocqPath, version: ver})
		}
	}

	// 3. Homebrew paths
	for _, p := range []string{"/opt/homebrew/bin/rocq", "/usr/local/bin/rocq"} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			dir := filepath.Dir(p)
			if !alreadyFound(found, dir) {
				ver := getRocqVersion(p)
				found = append(found, installation{path: dir, version: ver})
			}
		}
	}

	if len(found) == 0 {
		onLog("  \u26a0 No Rocq Platform installation found")
		return false
	}
	for _, inst := range found {
		if inst.version != "" {
			onLog(fmt.Sprintf("  \u2713 %s  (%s)", inst.path, inst.version))
		} else {
			onLog(fmt.Sprintf("  \u2713 %s  (version unknown)", inst.path))
		}
		if warning := checkAppContent(inst.path); warning != "" {
			onLog(fmt.Sprintf("    \u26a0 %s", warning))
		}
	}
	return true
}

// checkAppContent verifies that an installation directory/app bundle is not empty
// or contains only coq-shell (which indicates a broken/incomplete installation).
func alreadyFound(found []installation, path string) bool {
	for _, f := range found {
		if f.path == path {
			return true
		}
	}
	return false
}
func checkOpam(onLog func(string)) {
	opamPath, err := exec.LookPath("opam")
	if err != nil {
		onLog("  opam not found")
		return
	}
	onLog(fmt.Sprintf("  opam: %s", opamPath))

	out, err := exec.Command("opam", "--version").Output()
	if err == nil {
		onLog(fmt.Sprintf("  version: %s", strings.TrimSpace(string(out))))
	}

	switchOut, err := exec.Command("opam", "switch", "list", "--short").Output()
	if err != nil {
		onLog("  (could not list switches)")
		return
	}

	lines := strings.Split(string(switchOut), "\n")
	anySwitch := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.Contains(lower, "rocq") || strings.Contains(lower, "coq") || strings.Contains(lower, "cp.") {
			onLog(fmt.Sprintf("  switch: %s", line))
			anySwitch = true
		}
	}
	if !anySwitch {
		onLog("  (no Rocq/Coq-related switches)")
	}
}
func checkIssues(onLog func(string), installFound, vsrocqFound, vscoqFound bool) {
	anyIssue := false

	if !installFound {
		onLog("  \u26a0 Rocq Platform is not installed \u2014 run the installer to set it up")
		anyIssue = true
	}

	// Count installations
	home, _ := os.UserHomeDir()
	searchDirs := []string{"/Applications"}
	if home != "" {
		searchDirs = append(searchDirs, filepath.Join(home, "Applications"))
	}

	installCount := 0
	for _, dir := range searchDirs {
		for _, pattern := range []string{"*[Rr]ocq*.app", "*[Cc]oq*.app"} {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern))
			installCount += len(matches)
		}
	}

	if installCount > 1 {
		onLog("  \u26a0 Multiple Rocq/Coq installations detected — potential conflicts")
		anyIssue = true
	}

	if !vsrocqFound {
		onLog("  \u26a0 vsrocq extension not installed \u2014 required for Rocq support in VSCode")
		anyIssue = true
	}

	if vscoqFound {
		onLog("  \u26a0 vscoq extension is installed \u2014 deprecated, may conflict with vsrocq")
		anyIssue = true
	}

	if !anyIssue {
		onLog("  (no issues detected)")
	}
}
