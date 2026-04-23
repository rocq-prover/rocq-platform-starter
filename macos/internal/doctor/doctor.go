package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justme0606/rocq-platform-starter/macos/internal/vscode"
)

// Run performs system diagnostics and reports findings via onLog callback.
func Run(onLog func(string)) {
	onLog("=== Rocq/Coq Installations ===")
	installFound := checkInstallationsMacOS(onLog)

	onLog("")
	onLog("=== Binaries in PATH ===")
	checkBinariesMacOS(onLog)

	onLog("")
	onLog("=== opam ===")
	checkOpam(onLog)

	onLog("")
	onLog("=== VSCode ===")
	vsrocqFound, vscoqFound := checkVSCode(onLog)

	onLog("")
	onLog("=== Workspace ===")
	checkWorkspaceMacOS(onLog)

	onLog("")
	onLog("=== Potential Issues ===")
	checkIssues(onLog, installFound, vsrocqFound, vscoqFound)
}

// installation holds info about a found Rocq installation.
type installation struct {
	path    string
	version string
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
func checkAppContent(dir string) string {
	// For .app bundles, check Contents/Resources
	resourcesDir := dir
	if strings.HasSuffix(dir, ".app") {
		resourcesDir = filepath.Join(dir, "Contents", "Resources")
	}
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return fmt.Sprintf("cannot read directory: %v", err)
	}
	if len(entries) == 0 {
		return "installation directory is empty"
	}
	// Check if directory contains only coq-shell (broken installation)
	nonShellCount := 0
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if name != "coq-shell" && name != "coq-shell.lnk" && name != "coq-shell.bat" && name != "coq-shell.sh" {
			nonShellCount++
		}
	}
	if nonShellCount == 0 {
		return "installation contains only coq-shell \u2014 installation appears incomplete"
	}
	return ""
}

func alreadyFound(found []installation, path string) bool {
	for _, f := range found {
		if f.path == path {
			return true
		}
	}
	return false
}

func checkBinariesMacOS(onLog func(string)) {
	binaries := []string{"rocq", "coqtop", "coqc", "vsrocqtop"}
	anyFound := false

	for _, name := range binaries {
		if p, err := exec.LookPath(name); err == nil {
			onLog(fmt.Sprintf("  %s \u2192 %s", name, p))
			anyFound = true
		}
	}

	if !anyFound {
		onLog("  (none found in PATH)")
	}
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

func checkVSCode(onLog func(string)) (vsrocqFound, vscoqFound bool) {
	codeBin, err := vscode.FindCode()
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
	lines := strings.Split(string(out), "\n")
	anyExt := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.Contains(lower, "rocq") || strings.Contains(lower, "coq") {
			onLog(fmt.Sprintf("    %s", line))
			anyExt = true
			if strings.Contains(lower, "vsrocq") {
				vsrocqFound = true
			}
			if strings.Contains(lower, "vscoq") {
				vscoqFound = true
			}
		}
	}
	if !anyExt {
		onLog("    (no Rocq/Coq extensions)")
	}
	if !vsrocqFound {
		onLog("  \u26a0 vsrocq extension not found")
	}
	if vscoqFound {
		onLog("  \u26a0 vscoq extension detected (deprecated, use vsrocq instead)")
	}

	return vsrocqFound, vscoqFound
}

func checkWorkspaceMacOS(onLog func(string)) {
	home, err := os.UserHomeDir()
	if err != nil {
		onLog("  (could not determine home directory)")
		return
	}

	wsDir := filepath.Join(home, "rocq-workspace")
	if info, err := os.Stat(wsDir); err == nil && info.IsDir() {
		onLog(fmt.Sprintf("  \u2713 %s", wsDir))

		settingsPath := filepath.Join(wsDir, ".vscode", "settings.json")
		if data, err := os.ReadFile(settingsPath); err == nil {
			var settings map[string]interface{}
			if err := json.Unmarshal(data, &settings); err == nil {
				if v, ok := settings["vsrocq.path"]; ok {
					onLog(fmt.Sprintf("  settings.json: vsrocq.path = %v", v))
				} else {
					onLog("  settings.json: vsrocq.path not set")
				}
			}
		} else {
			onLog("  .vscode/settings.json not found")
		}
	} else {
		onLog(fmt.Sprintf("  %s not found", wsDir))
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
