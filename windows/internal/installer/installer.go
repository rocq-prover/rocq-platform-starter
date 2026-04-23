package installer

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"

	sharedinstaller "github.com/justme0606/rocq-platform-starter/shared/installer"

	"github.com/justme0606/rocq-platform-starter/windows/internal/manifest"
	"github.com/justme0606/rocq-platform-starter/windows/internal/vscode"
	"github.com/justme0606/rocq-platform-starter/windows/internal/workspace"
)

// Logger wraps the shared Logger type.
type Logger = sharedinstaller.Logger

// NewLogger creates a log file under %USERPROFILE%\.rocq-setup\logs\.
func NewLogger() (*Logger, error) {
	return sharedinstaller.NewLogger()
}

// debugLog prints debug messages to stderr (visible in debug builds with console).
func debugLog(format string, args ...interface{}) {
	log.Printf(format, args...)
}

const (
	WorkspaceName = "rocq-workspace"
)

// DefaultInstallDir returns the default Rocq Platform installation directory
// based on the manifest version info.
// Example: for rocqVersion "9.0.0" and platformRelease "2025.08.1",
// returns "C:\Rocq-platform~9.0~2025.08".
func DefaultInstallDir(rocqVersion, platformRelease string) string {
	// Extract major.minor from rocqVersion (e.g. "9.0.0" -> "9.0")
	rocqShort := rocqVersion
	if parts := strings.SplitN(rocqVersion, ".", 3); len(parts) >= 2 {
		rocqShort = parts[0] + "." + parts[1]
	}

	// Extract year.month from platformRelease (e.g. "2025.08.1" -> "2025.08")
	releaseShort := platformRelease
	if parts := strings.SplitN(platformRelease, ".", 3); len(parts) >= 2 {
		releaseShort = parts[0] + "." + parts[1]
	}

	return `C:\Rocq-platform~` + rocqShort + `~` + releaseShort
}

// StepFunc is called to report progress: step number (1-7), label, and fraction (0.0–1.0).
type StepFunc func(step int, label string, fraction float64)

// Config holds all parameters for the installation pipeline.
type Config struct {
	Manifest    *manifest.Manifest
	Templates   fs.FS
	InstallDir  string
	SkipInstall bool // If true, skip download/checksum/install steps (reuse existing installation)
	OnStep      StepFunc
	Logger      *Logger
}

// rocqBinaryNames lists the binary names to look for (with and without .exe).
var rocqBinaryNames = []string{"rocq", "rocq.exe", "vsrocqtop", "vsrocqtop.exe"}

// hasRocqInstallation checks whether a directory contains a Rocq Platform installation
// by looking for known binaries in expected locations,
// plus a shallow recursive search (depth 1) in subdirectories.
func hasRocqInstallation(dir string) bool {
	debugLog("[detect] checking directory: %s", dir)

	// Check bin/ and root directory for known binaries
	for _, name := range rocqBinaryNames {
		for _, sub := range []string{"bin", ""} {
			c := filepath.Join(dir, sub, name)
			if info, err := os.Stat(c); err == nil && !info.IsDir() {
				debugLog("[detect]   FOUND: %s", c)
				return true
			}
		}
	}

	// Shallow recursive search (depth 1) in subdirectories
	entries, err := os.ReadDir(dir)
	if err != nil {
		debugLog("[detect]   cannot read directory %s: %v", dir, err)
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			for _, name := range rocqBinaryNames {
				subPath := filepath.Join(dir, e.Name(), name)
				if info, err := os.Stat(subPath); err == nil && !info.IsDir() {
					debugLog("[detect]   FOUND: %s", subPath)
					return true
				}
			}
		}
	}
	debugLog("[detect]   no Rocq installation in %s", dir)
	return false
}

// FindExistingInstallations searches for all existing Rocq Platform installations.
// It checks: glob patterns, Windows registry, common paths, and the system PATH.
// Returns a deduplicated list of installation directory paths.
func FindExistingInstallations() []string {
	debugLog("[detect] === Starting existing installation search ===")
	var found []string
	seen := make(map[string]bool)

	addIfNew := func(dir string) {
		key := strings.ToLower(dir)
		if !seen[key] {
			seen[key] = true
			found = append(found, dir)
		}
	}

	// 1. Default Rocq Platform installation paths (e.g. C:\Rocq-platform~9.0~2025.08)
	debugLog("[detect] Step 1: globbing C:\\Rocq-platform~*")
	matches, err := filepath.Glob(`C:\Rocq-platform~*`)
	if err != nil {
		debugLog("[detect]   glob error: %v", err)
	} else {
		for _, m := range matches {
			if hasRocqInstallation(m) {
				debugLog("[detect] => Found at Rocq Platform dir: %s", m)
				addIfNew(m)
			}
		}
	}

	// 2. Windows registry: look for uninstall entries mentioning "Rocq"
	debugLog("[detect] Step 2: searching Windows registry")
	for _, dir := range findAllFromRegistry() {
		if hasRocqInstallation(dir) {
			debugLog("[detect] => Found via registry: %s", dir)
			addIfNew(dir)
		}
	}

	// 3. Common installation paths
	commonPaths := []string{
		`C:\Rocq`,
		`C:\Program Files\Rocq`,
		`C:\Program Files (x86)\Rocq`,
	}
	debugLog("[detect] Step 3: checking common paths: %v", commonPaths)
	for _, p := range commonPaths {
		if hasRocqInstallation(p) {
			debugLog("[detect] => Found at common path: %s", p)
			addIfNew(p)
		}
	}

	// 4. PATH lookup (try both "rocq" and "rocq.exe")
	debugLog("[detect] Step 4: looking up rocq in PATH")
	for _, name := range []string{"rocq", "rocq.exe"} {
		if rocqPath, err := exec.LookPath(name); err == nil {
			debugLog("[detect]   found %s in PATH: %s", name, rocqPath)
			dir := filepath.Dir(filepath.Dir(rocqPath))
			if hasRocqInstallation(dir) {
				addIfNew(dir)
			} else {
				dir = filepath.Dir(rocqPath)
				addIfNew(dir)
			}
		}
	}

	if len(found) == 0 {
		debugLog("[detect] === No existing installation found ===")
	}
	return found
}

// findAllFromRegistry searches the Windows uninstall registry keys for entries
// whose DisplayName contains "Rocq" and returns all InstallLocation values.
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

// Result holds information about the installation outcome.
type Result struct {
	VSCodeFound bool   // Whether VSCode was detected on the system
	InstallDir  string // The directory where Rocq Platform is installed
}

// Run executes the installation pipeline.
// Returns a Result with details about the installation, or an error.
func Run(cfg *Config) (*Result, error) {
	asset := cfg.Manifest.Assets.Windows.X86_64
	installDir := cfg.InstallDir
	if installDir == "" {
		installDir = DefaultInstallDir(cfg.Manifest.RocqVersion, cfg.Manifest.PlatformRelease)
	}
	debugLog("[install] install directory: %s", installDir)

	result := &Result{InstallDir: installDir}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	workspaceDir := filepath.Join(home, WorkspaceName)

	// Check if we should skip installation (existing installation reused).
	alreadyInstalled := cfg.SkipInstall || hasRocqInstallation(installDir)
	if alreadyInstalled {
		cfg.Logger.Log("Rocq Platform already installed in %s, skipping download and installation", installDir)
		cfg.OnStep(1, "Rocq Platform already installed, skipping download.", 1.0)
		cfg.OnStep(2, "Skipped (already installed).", 1.0)
		cfg.OnStep(3, "Skipped (already installed).", 1.0)
	} else {
		tempDir := filepath.Join(os.TempDir(), "rocq-platform-starter")

		// Step 1: Download
		cfg.OnStep(1, "Downloading Rocq Platform installer...", 0.0)
		cfg.Logger.Log("Downloading %s", asset.URL)
		exePath, err := Download(asset.URL, tempDir, func(downloaded, total int64) {
			if total > 0 {
				cfg.OnStep(1, "Downloading Rocq Platform installer...", float64(downloaded)/float64(total))
			}
		})
		if err != nil {
			return nil, fmt.Errorf("download: %w", err)
		}
		cfg.Logger.Log("Downloaded to %s", exePath)
		defer os.RemoveAll(tempDir)

		// Step 2: Verify SHA256
		cfg.OnStep(2, "Verifying checksum...", 0.0)
		cfg.Logger.Log("Verifying SHA256 (expected: %q)", asset.SHA256)
		if err := VerifySHA256(exePath, asset.SHA256); err != nil {
			return nil, fmt.Errorf("checksum: %w", err)
		}
		cfg.Logger.Log("Checksum OK (or skipped)")
		cfg.OnStep(2, "Checksum verified.", 1.0)

		// Step 3: Install Rocq Platform
		cfg.OnStep(3, "Installing Rocq Platform (follow the installer window)...", 0.0)
		cfg.Logger.Log("Running installer: %s -> %s", exePath, installDir)
		if err := RunInnoSetup(exePath, installDir); err != nil {
			return nil, fmt.Errorf("install: %w", err)
		}
		cfg.Logger.Log("Installation complete")
		cfg.OnStep(3, "Rocq Platform installed.", 1.0)
	}

	// Step 4: Find language server binary
	topBinLabel := "vsrocqtop"
	if vscode.IsCoq(cfg.Manifest.RocqVersion) {
		topBinLabel = "vscoqtop"
	}
	cfg.OnStep(4, fmt.Sprintf("Locating %s...", topBinLabel), 0.0)
	vsrocqtopPath, err := FindLanguageServerTop(installDir, cfg.Manifest.RocqVersion)
	if err != nil {
		cfg.Logger.Log("WARNING: %s not found: %v", topBinLabel, err)
		cfg.OnStep(4, fmt.Sprintf("%s not found (will skip VSCode settings).", topBinLabel), 1.0)
	} else {
		cfg.Logger.Log("Found %s: %s", topBinLabel, vsrocqtopPath)
		cfg.OnStep(4, fmt.Sprintf("Found %s.", topBinLabel), 1.0)
	}

	// Step 5: Check for VSCode
	cfg.OnStep(5, "Checking for VSCode...", 0.0)
	codeBin, err := vscode.FindCode()
	if err != nil {
		cfg.Logger.Log("VSCode not found: %v", err)
		cfg.OnStep(5, "VSCode not found.", 1.0)
		cfg.OnStep(6, "Skipped (VSCode not found).", 1.0)
		cfg.OnStep(7, "Skipped (VSCode not found).", 1.0)
		result.VSCodeFound = false
		return result, nil
	}
	result.VSCodeFound = true

	// VSCode found — install extension
	cfg.Logger.Log("VSCode CLI: %s", codeBin)
	extensionID := vscode.ExtensionIDForVersion(cfg.Manifest.RocqVersion)
	if err := vscode.InstallExtension(codeBin, extensionID); err != nil {
		cfg.Logger.Log("WARNING: extension install failed: %v", err)
	}
	cfg.OnStep(5, "VSCode extension installed.", 1.0)

	// Step 6: Create workspace
	cfg.OnStep(6, "Creating workspace...", 0.0)
	cfg.Logger.Log("Creating workspace at %s", workspaceDir)
	if err := workspace.Create(workspaceDir, cfg.Templates); err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	cfg.Logger.Log("Workspace created")
	cfg.OnStep(6, "Workspace created.", 1.0)

	// Step 7: Configure VSCode settings and open workspace
	cfg.OnStep(7, "Configuring VSCode...", 0.0)
	if vsrocqtopPath != "" {
		// Strip .exe extension — settings expect the path without it
		topClean := strings.TrimSuffix(vsrocqtopPath, ".exe")
		topForward := filepath.ToSlash(topClean)
		settingsKey := "vsrocq.path"
		if vscode.IsCoq(cfg.Manifest.RocqVersion) {
			settingsKey = "vscoq.path"
		}
		if err := workspace.WriteVSCodeSettings(workspaceDir, settingsKey, topForward); err != nil {
			return nil, fmt.Errorf("vscode config: %w", err)
		}
		cfg.Logger.Log("VSCode settings written with %s=%s", settingsKey, vsrocqtopPath)
	} else {
		cfg.Logger.Log("Skipping VSCode settings (%s not found)", topBinLabel)
	}

	// Open VSCode with the workspace
	cfg.Logger.Log("Opening VSCode with workspace %s", workspaceDir)
	if err := vscode.OpenWorkspace(codeBin, workspaceDir); err != nil {
		cfg.Logger.Log("WARNING: failed to open VSCode: %v", err)
	}
	cfg.OnStep(7, "Done!", 1.0)

	return result, nil
}
