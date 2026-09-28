package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	shareddoctor "github.com/rocq-prover/rocq-platform-starter/shared/doctor"

	"github.com/rocq-prover/rocq-platform-starter/linux/internal/installer"
	"github.com/rocq-prover/rocq-platform-starter/linux/internal/vscode"
)

// probedBinaries are the Rocq/Coq binaries the doctor looks for on PATH.
var probedBinaries = []string{"rocq", "coqtop", "coqc", "vsrocqtop"}

// Run performs system diagnostics and reports findings via onLog callback.
func Run(onLog func(string)) {
	onLog("=== Opam ===")
	opamFound := checkOpam(onLog)

	onLog("")
	onLog("=== Rocq Platform Switches ===")
	installFound := checkSwitches(onLog)

	onLog("")
	onLog("=== Binaries in PATH ===")
	shareddoctor.CheckBinaries(onLog, shareddoctor.BinariesOptions{
		Names:        probedBinaries,
		ProbeVersion: true,
		// vsrocqtop is an LSP server: --print-version never returns.
		SkipVersionFor: []string{"vsrocqtop"},
	})

	onLog("")
	onLog("=== VSCode ===")
	vsrocqFound, vscoqFound := shareddoctor.CheckVSCode(onLog, vscode.FindCode)

	onLog("")
	onLog("=== Workspace ===")
	shareddoctor.CheckWorkspace(onLog, installer.WorkspaceName,
		[]string{"activate.sh", "activate-shell.sh"})

	onLog("")
	onLog("=== Docker ===")
	dockerFound := checkDocker(onLog)

	onLog("")
	onLog("=== Potential Issues ===")
	checkIssues(onLog, opamFound, installFound, vsrocqFound, vscoqFound, dockerFound)
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
func checkDocker(onLog func(string)) bool {
	path, err := exec.LookPath("docker")
	if err != nil {
		onLog("  ⚠ docker not found in PATH")
		return false
	}
	onLog(fmt.Sprintf("  ✓ docker: %s", path))

	out, err := exec.Command("docker", "--version").Output()
	if err == nil {
		ver := strings.TrimSpace(string(out))
		onLog(fmt.Sprintf("  Version: %s", ver))
	}

	// Check if daemon is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = exec.CommandContext(ctx, "docker", "info").CombinedOutput()
	cancel()
	if err != nil {
		onLog("  ⚠ Docker daemon is not running")
	} else {
		onLog("  ✓ Docker daemon is running")
	}

	return true
}
func checkIssues(onLog func(string), opamFound, installFound, vsrocqFound, vscoqFound, dockerFound bool) {
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
