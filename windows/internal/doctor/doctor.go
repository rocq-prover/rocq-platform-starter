package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	"github.com/justme0606/rocq-platform-starter/windows/internal/vscode"
)

// Run performs system diagnostics and reports findings via onLog callback.
func Run(onLog func(string)) {
	onLog("=== Rocq Platform Installations ===")
	installFound := checkInstallationsWindows(onLog)

	onLog("")
	onLog("=== Binaries in PATH ===")
	checkBinariesWindows(onLog)

	onLog("")
	onLog("=== VSCode ===")
	vsrocqFound, vscoqFound := checkVSCode(onLog)

	onLog("")
	onLog("=== Workspace ===")
	checkWorkspaceWindows(onLog)

	onLog("")
	onLog("=== Potential Issues ===")
	checkIssues(onLog, installFound, vsrocqFound, vscoqFound)
}

// installation holds info about a found Rocq installation.
type installation struct {
	path    string
	version string
}

func getRocqVersion(dir string) string {
	for _, bin := range []string{"rocq.exe", "rocq", "coqc.exe", "coqc"} {
		for _, sub := range []string{"bin", ""} {
			binPath := filepath.Join(dir, sub, bin)
			if info, err := os.Stat(binPath); err == nil && !info.IsDir() {
				out, err := exec.Command(binPath, "--print-version").Output()
				if err == nil {
					return strings.TrimSpace(string(out))
				}
			}
		}
	}
	return ""
}

func checkInstallationsWindows(onLog func(string)) bool {
	var found []installation

	// 1. Glob C:\Rocq-platform~* and C:\Coq-platform~*
	for _, pattern := range []string{`C:\Rocq-platform~*`, `C:\Coq-platform~*`, `C:\Rocq-Platform~*`, `C:\Coq-Platform~*`} {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if !alreadyFound(found, m) {
				ver := getRocqVersion(m)
				found = append(found, installation{path: m, version: ver})
			}
		}
	}

	// 2. Registry search
	registryInstalls := findAllFromRegistry()
	for _, dir := range registryInstalls {
		if !alreadyFound(found, dir) {
			ver := getRocqVersion(dir)
			found = append(found, installation{path: dir, version: ver})
		}
	}

	// 3. Common paths
	commonPaths := []string{
		`C:\Rocq`,
		`C:\Coq`,
		`C:\Program Files\Rocq`,
		`C:\Program Files (x86)\Rocq`,
		`C:\Program Files\Coq`,
		`C:\Program Files (x86)\Coq`,
		`C:\Program Files\Rocq Platform`,
		`C:\Program Files\Coq Platform`,
	}
	for _, p := range commonPaths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			if !alreadyFound(found, p) {
				ver := getRocqVersion(p)
				found = append(found, installation{path: p, version: ver})
			}
		}
	}

	// 4. PATH lookup
	for _, name := range []string{"rocq", "rocq.exe", "coqc", "coqc.exe", "coqtop", "coqtop.exe"} {
		if binPath, err := exec.LookPath(name); err == nil {
			dir := filepath.Dir(filepath.Dir(binPath))
			if !alreadyFound(found, dir) {
				ver := getRocqVersion(dir)
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
		if warning := checkDirContent(inst.path); warning != "" {
			onLog(fmt.Sprintf("    \u26a0 %s", warning))
		}
	}
	return true
}

// checkDirContent verifies that an installation directory is not empty
// or contains only coq-shell (which indicates a broken/incomplete installation).
func checkDirContent(dir string) string {
	entries, err := os.ReadDir(dir)
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
		if name != "coq-shell" && name != "coq-shell.lnk" && name != "coq-shell.bat" {
			nonShellCount++
		}
	}
	if nonShellCount == 0 {
		return "installation directory contains only coq-shell \u2014 installation appears incomplete"
	}
	return ""
}

func alreadyFound(found []installation, path string) bool {
	for _, f := range found {
		if strings.EqualFold(f.path, path) {
			return true
		}
	}
	return false
}

func findAllFromRegistry() []string {
	var results []string
	uninstallKey := `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`

	for _, rootKey := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		k, err := registry.OpenKey(rootKey, uninstallKey, registry.ENUMERATE_SUB_KEYS|registry.READ)
		if err != nil {
			continue
		}

		subkeys, err := k.ReadSubKeyNames(-1)
		k.Close()
		if err != nil {
			continue
		}

		for _, subkey := range subkeys {
			sk, err := registry.OpenKey(rootKey, uninstallKey+`\`+subkey, registry.READ)
			if err != nil {
				continue
			}

			displayName, _, err := sk.GetStringValue("DisplayName")
			if err != nil {
				sk.Close()
				continue
			}

			lower := strings.ToLower(displayName)
			if strings.Contains(lower, "rocq") || strings.Contains(lower, "coq") {
				installLoc, _, err := sk.GetStringValue("InstallLocation")
				sk.Close()
				if err == nil && installLoc != "" {
					results = append(results, installLoc)
				}
				continue
			}
			sk.Close()
		}
	}

	return results
}

func checkBinariesWindows(onLog func(string)) {
	binaries := []string{"rocq", "coqtop", "coqc", "vsrocqtop"}
	anyFound := false

	for _, name := range binaries {
		for _, suffix := range []string{"", ".exe"} {
			full := name + suffix
			if p, err := exec.LookPath(full); err == nil {
				onLog(fmt.Sprintf("  %s \u2192 %s", name, p))
				anyFound = true
				break
			}
		}
	}

	if !anyFound {
		onLog("  (none found in PATH)")
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

func checkWorkspaceWindows(onLog func(string)) {
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

	// Check for multiple installations
	seen := make(map[string]bool)
	for _, pattern := range []string{`C:\Rocq-platform~*`, `C:\Coq-platform~*`, `C:\Rocq-Platform~*`, `C:\Coq-Platform~*`} {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			seen[strings.ToLower(m)] = true
		}
	}
	for _, r := range findAllFromRegistry() {
		seen[strings.ToLower(r)] = true
	}
	if len(seen) > 1 {
		onLog("  \u26a0 Multiple Rocq/Coq installations detected — potential PATH conflicts")
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
