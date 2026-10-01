package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	shareddoctor "github.com/rocq-prover/rocq-platform-starter/shared/doctor"

	"github.com/rocq-prover/rocq-platform-starter/windows/internal/installer"
	"github.com/rocq-prover/rocq-platform-starter/windows/internal/vscode"
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
	onLog("=== Rocq Platform Installations ===")
	installFound := checkInstallationsWindows(onLog)

	onLog("")
	onLog("=== Binaries in PATH ===")
	shareddoctor.CheckBinaries(onLog, shareddoctor.BinariesOptions{
		Names:        probedBinaries,
		TryExeSuffix: true,
	})

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
		if warning := shareddoctor.InspectInstallDir(inst.path); warning != "" {
			onLog(fmt.Sprintf("    \u26a0 %s", warning))
		}
	}
	return true
}

// checkDirContent verifies that an installation directory is not empty
// or contains only coq-shell (which indicates a broken/incomplete installation).
func alreadyFound(found []installation, path string) bool {
	for _, f := range found {
		if strings.EqualFold(f.path, path) {
			return true
		}
	}
	return false
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
