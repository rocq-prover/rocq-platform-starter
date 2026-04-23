package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justme0606/rocq-platform-starter/linux/internal/vscode"
)

// Run performs system diagnostics and reports findings via onLog callback.
func Run(onLog func(string)) {
	onLog("=== Opam ===")
	opamFound := checkOpam(onLog)

	onLog("")
	onLog("=== Rocq Platform Switches ===")
	installFound := checkSwitches(onLog)

	onLog("")
	onLog("=== Binaries in PATH ===")
	checkBinaries(onLog)

	onLog("")
	onLog("=== VSCode ===")
	vsrocqFound, vscoqFound := checkVSCode(onLog)

	onLog("")
	onLog("=== Workspace ===")
	checkWorkspace(onLog)

	onLog("")
	onLog("=== Potential Issues ===")
	checkIssues(onLog, opamFound, installFound, vsrocqFound, vscoqFound)
}

func checkOpam(onLog func(string)) bool {
	path, err := exec.LookPath("opam")
	if err != nil {
		onLog("  \u26a0 opam not found in PATH")
		return false
	}
	onLog(fmt.Sprintf("  \u2713 opam: %s", path))

	out, err := exec.Command("opam", "--version").Output()
	if err == nil {
		ver := strings.TrimSpace(string(out))
		onLog(fmt.Sprintf("  Version: %s", ver))
		if !strings.HasPrefix(ver, "2.") {
			onLog("  \u26a0 opam >= 2.x recommended")
		}
	}
	return true
}

func checkSwitches(onLog func(string)) bool {
	out, err := exec.Command("opam", "switch", "list", "--short").Output()
	if err != nil {
		onLog("  (could not list opam switches)")
		return false
	}

	found := false
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "CP.") || strings.HasPrefix(name, "coq-") {
			found = true
			onLog(fmt.Sprintf("  \u2713 %s", name))
			checkSwitchPackages(name, onLog)
			checkSwitchBinaries(name, onLog)
		}
	}

	if !found {
		onLog("  \u26a0 No Rocq/Coq Platform switches found (CP.* or coq-*)")
	}
	return found
}

func checkSwitchPackages(switchName string, onLog func(string)) {
	out, err := exec.Command("opam", "list", "--switch="+switchName, "--installed", "--short", "-V").Output()
	if err != nil {
		onLog("    (could not list packages)")
		return
	}

	rocqPkgs := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "rocq") || strings.Contains(lower, "coq") || strings.Contains(lower, "vsrocq") {
			rocqPkgs = append(rocqPkgs, line)
		}
	}

	if len(rocqPkgs) > 0 {
		onLog("    Packages:")
		for _, pkg := range rocqPkgs {
			onLog(fmt.Sprintf("      %s", pkg))
		}
	} else {
		onLog("    \u26a0 No Rocq/Coq packages found in switch")
	}
}

func checkSwitchBinaries(switchName string, onLog func(string)) {
	out, err := exec.Command("opam", "var", "--switch="+switchName, "bin").Output()
	if err != nil {
		return
	}
	binDir := strings.TrimSpace(string(out))

	binaries := []string{"rocq", "vsrocqtop", "coqc", "coqtop"}
	for _, bin := range binaries {
		binPath := filepath.Join(binDir, bin)
		if info, err := os.Stat(binPath); err == nil && !info.IsDir() {
			onLog(fmt.Sprintf("    \u2713 %s", binPath))
		}
	}
}

func checkBinaries(onLog func(string)) {
	binaries := []string{"rocq", "coqtop", "coqc", "vsrocqtop"}
	anyFound := false

	for _, name := range binaries {
		if p, err := exec.LookPath(name); err == nil {
			onLog(fmt.Sprintf("  %s \u2192 %s", name, p))
			anyFound = true

			// Skip version check for vsrocqtop (LSP server, hangs on --print-version)
			if name == "vsrocqtop" {
				continue
			}

			// Try to get version with a timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			out, err := exec.CommandContext(ctx, p, "--print-version").Output()
			cancel()
			if err == nil {
				ver := strings.TrimSpace(string(out))
				if ver != "" {
					onLog(fmt.Sprintf("    version: %s", ver))
				}
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

func checkWorkspace(onLog func(string)) {
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

		// Check activation scripts
		for _, script := range []string{"activate.sh", "activate-shell.sh"} {
			scriptPath := filepath.Join(wsDir, script)
			if _, err := os.Stat(scriptPath); err == nil {
				onLog(fmt.Sprintf("  \u2713 %s present", script))
			} else {
				onLog(fmt.Sprintf("  \u26a0 %s not found", script))
			}
		}
	} else {
		onLog(fmt.Sprintf("  %s not found", wsDir))
	}
}

func checkIssues(onLog func(string), opamFound, installFound, vsrocqFound, vscoqFound bool) {
	anyIssue := false

	if !opamFound {
		onLog("  \u26a0 opam is not installed \u2014 required for Rocq Platform on Linux")
		anyIssue = true
	}

	if !installFound {
		onLog("  \u26a0 No Rocq Platform switch found \u2014 run the installer to set it up")
		anyIssue = true
	}

	// Check for multiple CP.* switches
	if opamFound {
		out, _ := exec.Command("opam", "switch", "list", "--short").Output()
		cpCount := 0
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "CP.") {
				cpCount++
			}
		}
		if cpCount > 1 {
			onLog("  \u26a0 Multiple Rocq Platform switches detected \u2014 potential confusion")
			anyIssue = true
		}
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
