package installer

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justme0606/rocq-platform-starter/linux/internal/manifest"
	"github.com/justme0606/rocq-platform-starter/linux/internal/vscode"
	"github.com/justme0606/rocq-platform-starter/linux/internal/workspace"
)

func debugLog(format string, args ...interface{}) {
	log.Printf(format, args...)
}

const (
	WorkspaceName = "rocq-workspace"
)

// SwitchName returns the opam switch name for a given manifest.
// Format: CP.<platform_release>~<rocq_major.minor>
func SwitchName(rocqVersion, platformRelease string) string {
	rocqShort := rocqVersion
	if parts := strings.SplitN(rocqVersion, ".", 3); len(parts) >= 2 {
		rocqShort = parts[0] + "." + parts[1]
	}
	return "CP." + platformRelease + "~" + rocqShort
}

// StepFunc is called to report progress: step number (1-7), label, and fraction (0.0-1.0).
type StepFunc func(step int, label string, fraction float64)

// Config holds all parameters for the installation pipeline.
type Config struct {
	Manifest    *manifest.Manifest
	Templates   fs.FS
	SkipInstall bool // If true, skip opam install steps (reuse existing switch)
	OnStep      StepFunc
	Logger      *Logger
}

// Result holds information about the installation outcome.
type Result struct {
	VSCodeFound bool
	SwitchName  string
}

// FindExistingInstallations returns all opam switches matching CP.* or coq-*.
func FindExistingInstallations() []string {
	debugLog("[detect] === Searching for existing opam switches ===")

	out, err := exec.Command("opam", "switch", "list", "--short").Output()
	if err != nil {
		debugLog("[detect] opam switch list failed: %v", err)
		return nil
	}

	var switches []string
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if strings.HasPrefix(name, "CP.") || strings.HasPrefix(name, "coq-") {
			switches = append(switches, name)
		}
	}

	if len(switches) == 0 {
		debugLog("[detect] === No existing CP.*/coq-* switch found ===")
	}
	return switches
}

// Run executes the installation pipeline via opam.
// Steps:
//  1. Check/install opam
//  2. Initialize opam (opam init)
//  3. Create opam switch
//  4. Configure rocq-released repo
//  5. Install Rocq packages
//  6. Create workspace + activation scripts
//  7. Configure VSCode + open workspace
func Run(cfg *Config) (*Result, error) {
	opamCfg := cfg.Manifest.Assets.Linux.X86_64.Opam
	switchName := SwitchName(cfg.Manifest.RocqVersion, cfg.Manifest.PlatformRelease)

	result := &Result{SwitchName: switchName}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	workspaceDir := filepath.Join(home, WorkspaceName)

	if cfg.SkipInstall {
		cfg.Logger.Log("Reusing existing opam switch %s, skipping install steps", switchName)
		cfg.OnStep(1, "Opam already available, skipping.", 1.0)
		cfg.OnStep(2, "Skipped (reusing switch).", 1.0)
		cfg.OnStep(3, "Skipped (reusing switch).", 1.0)
		cfg.OnStep(4, "Skipped (reusing switch).", 1.0)
		cfg.OnStep(5, "Skipped (reusing switch).", 1.0)
	} else {
		// Step 1: Check/install opam
		cfg.OnStep(1, "Checking for opam...", 0.0)
		opamBin, err := ensureOpam(cfg.Logger)
		if err != nil {
			return nil, fmt.Errorf("opam: %w", err)
		}
		cfg.Logger.Log("opam found: %s", opamBin)
		cfg.OnStep(1, "Opam found.", 1.0)

		// Step 2: Initialize opam
		cfg.OnStep(2, "Initializing opam...", 0.0)
		if err := initOpam(cfg.Logger); err != nil {
			return nil, fmt.Errorf("opam init: %w", err)
		}
		cfg.OnStep(2, "Opam initialized.", 1.0)

		// Step 3: Create switch
		cfg.OnStep(3, fmt.Sprintf("Creating opam switch %s...", switchName), 0.0)
		if err := createSwitch(switchName, opamCfg.OCamlCompiler, cfg.Logger); err != nil {
			return nil, fmt.Errorf("create switch: %w", err)
		}
		cfg.OnStep(3, fmt.Sprintf("Switch %s ready.", switchName), 1.0)

		// Step 4: Configure repo
		cfg.OnStep(4, "Configuring opam repository...", 0.0)
		if err := configureRepo(switchName, opamCfg.RepoName, opamCfg.RepoURL, cfg.Logger); err != nil {
			return nil, fmt.Errorf("configure repo: %w", err)
		}
		cfg.OnStep(4, "Repository configured.", 1.0)

		// Step 5: Install packages
		cfg.OnStep(5, "Installing Rocq packages (this may take a while)...", 0.0)
		if err := installPackages(switchName, opamCfg.Packages, cfg.Logger, func(fraction float64) {
			cfg.OnStep(5, "Installing Rocq packages...", fraction)
		}); err != nil {
			return nil, fmt.Errorf("install packages: %w", err)
		}
		cfg.OnStep(5, "Rocq packages installed.", 1.0)
	}

	// Step 6: Create workspace + activation scripts
	cfg.OnStep(6, "Creating workspace...", 0.0)
	cfg.Logger.Log("Creating workspace at %s", workspaceDir)
	if err := workspace.Create(workspaceDir, cfg.Templates); err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	if err := workspace.WriteActivationScripts(workspaceDir, switchName); err != nil {
		return nil, fmt.Errorf("activation scripts: %w", err)
	}
	cfg.Logger.Log("Workspace created")
	cfg.OnStep(6, "Workspace created.", 1.0)

	// Step 7: Check for VSCode and configure
	cfg.OnStep(7, "Checking for VSCode...", 0.0)
	codeBin, err := vscode.FindCode()
	if err != nil {
		cfg.Logger.Log("VSCode not found: %v", err)
		cfg.OnStep(7, "VSCode not found.", 1.0)
		result.VSCodeFound = false
		return result, nil
	}
	result.VSCodeFound = true

	cfg.Logger.Log("VSCode CLI: %s", codeBin)
	extensionID := vscode.ExtensionIDForVersion(cfg.Manifest.RocqVersion)
	if err := vscode.InstallExtension(codeBin, extensionID); err != nil {
		cfg.Logger.Log("WARNING: extension install failed: %v", err)
	}

	// Write VSCode settings with language server path from the switch
	topPath := findLanguageServerTop(switchName, cfg.Manifest.RocqVersion)
	if topPath != "" {
		settingsKey := "vsrocq.path"
		if vscode.IsCoq(cfg.Manifest.RocqVersion) {
			settingsKey = "vscoq.path"
		}
		if err := workspace.WriteVSCodeSettings(workspaceDir, settingsKey, topPath); err != nil {
			return nil, fmt.Errorf("vscode config: %w", err)
		}
		cfg.Logger.Log("VSCode settings written with %s=%s", settingsKey, topPath)
	}

	cfg.Logger.Log("Opening VSCode with workspace %s", workspaceDir)
	if err := vscode.OpenWorkspace(codeBin, workspaceDir); err != nil {
		cfg.Logger.Log("WARNING: failed to open VSCode: %v", err)
	}
	cfg.OnStep(7, "Done!", 1.0)

	return result, nil
}

// ensureOpam checks for opam in PATH or installs it.
func ensureOpam(logger *Logger) (string, error) {
	path, err := exec.LookPath("opam")
	if err == nil {
		// Verify version
		out, err := exec.Command(path, "--version").Output()
		if err == nil {
			ver := strings.TrimSpace(string(out))
			logger.Log("opam version: %s", ver)
			if !strings.HasPrefix(ver, "2.") {
				return "", fmt.Errorf("opam >= 2.x required (found %s)", ver)
			}
		}
		return path, nil
	}
	return "", fmt.Errorf("opam not found in PATH. Please install opam: https://opam.ocaml.org/doc/Install.html")
}

// initOpam runs opam init if ~/.opam doesn't exist.
func initOpam(logger *Logger) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	opamDir := filepath.Join(home, ".opam")
	if _, err := os.Stat(opamDir); err == nil {
		logger.Log("opam already initialized (%s exists)", opamDir)
		return nil
	}

	logger.Log("Running opam init...")
	cmd := exec.Command("opam", "init", "-y", "--bare", "--disable-sandboxing")
	cmd.Env = append(os.Environ(), "OPAMCONFIRMLEVEL=unsafe-yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("opam init failed: %w\nOutput: %s", err, string(output))
	}
	logger.Log("opam init complete")
	return nil
}

// createSwitch creates the opam switch if it doesn't already exist.
func createSwitch(switchName, compiler string, logger *Logger) error {
	// Check if switch already exists
	out, err := exec.Command("opam", "switch", "list", "--short").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == switchName {
				logger.Log("Switch %s already exists", switchName)
				return nil
			}
		}
	}

	logger.Log("Creating switch %s with compiler %s", switchName, compiler)
	cmd := exec.Command("opam", "switch", "create", switchName, compiler, "-y")
	cmd.Env = append(os.Environ(), "OPAMCONFIRMLEVEL=unsafe-yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("opam switch create failed: %w\nOutput: %s", err, string(output))
	}
	logger.Log("Switch %s created", switchName)
	return nil
}

// configureRepo adds and configures the rocq-released repository for the switch.
func configureRepo(switchName, repoName, repoURL string, logger *Logger) error {
	logger.Log("Configuring repo %s -> %s (switch=%s)", repoName, repoURL, switchName)

	// Add repo (ignore error if already exists)
	cmd := exec.Command("opam", "repo", "add", "--switch="+switchName, repoName, repoURL, "-y")
	cmd.Env = append(os.Environ(), "OPAMCONFIRMLEVEL=unsafe-yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Repo might already exist, try set-url
		logger.Log("repo add failed (may exist), trying set-url: %s", string(output))
		cmd2 := exec.Command("opam", "repo", "set-url", "--switch="+switchName, repoName, repoURL, "-y")
		cmd2.Env = append(os.Environ(), "OPAMCONFIRMLEVEL=unsafe-yes")
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("repo set-url failed: %w\nOutput: %s", err2, string(out2))
		}
	}

	// Set repo priority
	cmd = exec.Command("opam", "repo", "priority", "--switch="+switchName, repoName, "1")
	cmd.Env = append(os.Environ(), "OPAMCONFIRMLEVEL=unsafe-yes")
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Log("WARNING: repo priority failed: %s", string(output))
	}

	// Update
	logger.Log("Updating opam repos...")
	cmd = exec.Command("opam", "update", "--switch="+switchName)
	cmd.Env = append(os.Environ(), "OPAMCONFIRMLEVEL=unsafe-yes")
	if output, err = cmd.CombinedOutput(); err != nil {
		logger.Log("WARNING: opam update failed: %s", string(output))
	}

	return nil
}

// installPackages installs the Rocq packages into the switch.
func installPackages(switchName string, packages []manifest.OpamPackage, logger *Logger, onProgress func(float64)) error {
	// Build package list (skip optional packages with "with_rocqide" flag)
	var pkgs []string
	for _, pkg := range packages {
		if pkg.Optional == "with_rocqide" {
			logger.Log("Skipping optional package %s (rocqide)", pkg.Name)
			continue
		}
		pkgs = append(pkgs, fmt.Sprintf("%s=%s", pkg.Name, pkg.Version))
	}

	logger.Log("Installing packages in switch %s: %v", switchName, pkgs)

	args := []string{"install", "--switch=" + switchName, "-y"}
	args = append(args, pkgs...)

	cmd := exec.Command("opam", args...)
	cmd.Env = append(os.Environ(), "OPAMCONFIRMLEVEL=unsafe-yes")

	// Capture stdout for progress
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start opam install: %w", err)
	}

	// Parse output for progress
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		logger.Log("[opam] %s", line)
		lineCount++
		// Estimate progress based on output lines (rough heuristic)
		fraction := float64(lineCount) / 200.0
		if fraction > 0.95 {
			fraction = 0.95
		}
		onProgress(fraction)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("opam install failed: %w", err)
	}

	return nil
}

// findLanguageServerTop locates the vsrocqtop or vscoqtop binary in the opam switch.
func findLanguageServerTop(switchName, rocqVersion string) string {
	out, err := exec.Command("opam", "var", "--switch="+switchName, "bin").Output()
	if err != nil {
		return ""
	}
	binDir := strings.TrimSpace(string(out))

	binName := "vsrocqtop"
	if vscode.IsCoq(rocqVersion) {
		binName = "vscoqtop"
	}

	topPath := filepath.Join(binDir, binName)
	if _, err := os.Stat(topPath); err == nil {
		return topPath
	}
	return ""
}
