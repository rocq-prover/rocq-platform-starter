package installer

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sharedinstaller "github.com/justme0606/rocq-bootstrap/shared/installer"

	"github.com/justme0606/rocq-bootstrap/macos/internal/manifest"
	"github.com/justme0606/rocq-bootstrap/macos/internal/vscode"
	"github.com/justme0606/rocq-bootstrap/macos/internal/workspace"
)

// Logger wraps the shared Logger type.
type Logger = sharedinstaller.Logger

// NewLogger creates a log file under ~/.rocq-setup/logs/.
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

// DefaultInstallDir returns the default installation directory for macOS.
func DefaultInstallDir() string {
	return "/Applications"
}

// StepFunc is called to report progress: step number (1-7), label, and fraction (0.0–1.0).
type StepFunc func(step int, label string, fraction float64)

// Config holds all parameters for the installation pipeline.
type Config struct {
	Manifest    *manifest.Manifest
	Templates   fs.FS
	SkipInstall bool   // If true, skip download/checksum/install steps (reuse existing installation)
	ExistingApp string // Path to existing .app if reusing
	OnStep      StepFunc
	Logger      *Logger
}

// FindExistingInstallations searches for all existing Rocq Platform installations.
// Returns a deduplicated list of .app paths or binary directories.
func FindExistingInstallations() []string {
	debugLog("[detect] === Starting existing installation search ===")
	var found []string
	seen := make(map[string]bool)

	addIfNew := func(path string) {
		if !seen[path] {
			seen[path] = true
			found = append(found, path)
		}
	}

	home, _ := os.UserHomeDir()
	searchDirs := []string{"/Applications"}
	if home != "" {
		searchDirs = append(searchDirs, filepath.Join(home, "Applications"))
	}

	// 1. Glob for *[Rr]ocq*.app and *[Cc]oq*.app in /Applications and ~/Applications
	for _, dir := range searchDirs {
		debugLog("[detect] searching in %s", dir)
		for _, pattern := range []string{"*[Rr]ocq*.app", "*[Cc]oq*.app"} {
			matches, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				continue
			}
			for _, m := range matches {
				info, err := os.Stat(m)
				if err == nil && info.IsDir() {
					debugLog("[detect] => Found: %s", m)
					addIfNew(m)
				}
			}
		}
	}

	// 2. PATH lookup
	debugLog("[detect] checking PATH for rocq")
	if rocqPath, err := exec.LookPath("rocq"); err == nil {
		debugLog("[detect] => Found rocq in PATH: %s", rocqPath)
		dir := filepath.Dir(rocqPath)
		for i := 0; i < 6; i++ {
			if strings.HasSuffix(dir, ".app") {
				addIfNew(dir)
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// 3. Homebrew paths
	debugLog("[detect] checking Homebrew paths")
	for _, p := range []string{"/opt/homebrew/bin/rocq", "/usr/local/bin/rocq"} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			debugLog("[detect] => Found: %s", p)
			addIfNew(filepath.Dir(p))
		}
	}

	if len(found) == 0 {
		debugLog("[detect] === No existing installation found ===")
	}
	return found
}

// Result holds information about the installation outcome.
type Result struct {
	VSCodeFound       bool   // Whether VSCode was detected on the system
	InstalledApp      string // Path to the installed .app
	VsrocqtopPath     string // Path to vsrocqtop binary
	VsrocqtopWarning  string // Non-empty if vsrocqtop was not found
}

// Run executes the installation pipeline.
func Run(cfg *Config) (*Result, error) {
	asset := cfg.Manifest.Assets.MacOS.ARM64

	result := &Result{}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	workspaceDir := filepath.Join(home, WorkspaceName)

	var installedAppPath string

	if cfg.SkipInstall && cfg.ExistingApp != "" {
		// Reuse existing installation
		installedAppPath = cfg.ExistingApp
		cfg.Logger.Log("Reusing existing installation: %s", installedAppPath)
		cfg.OnStep(1, "Rocq Platform already installed, skipping download.", 1.0)
		cfg.OnStep(2, "Skipped (already installed).", 1.0)
		cfg.OnStep(3, "Skipped (already installed).", 1.0)
	} else {
		tempDir := filepath.Join(os.TempDir(), "rocq-bootstrap")

		// Step 1: Download DMG
		cfg.OnStep(1, "Downloading Rocq Platform DMG...", 0.0)
		cfg.Logger.Log("Downloading %s", asset.URL)
		dmgPath, err := Download(asset.URL, tempDir, func(downloaded, total int64) {
			if total > 0 {
				cfg.OnStep(1, "Downloading Rocq Platform DMG...", float64(downloaded)/float64(total))
			}
		})
		if err != nil {
			return nil, fmt.Errorf("download: %w", err)
		}
		cfg.Logger.Log("Downloaded to %s", dmgPath)
		defer os.RemoveAll(tempDir)

		// Step 2: Verify SHA256
		cfg.OnStep(2, "Verifying checksum...", 0.0)
		cfg.Logger.Log("Verifying SHA256 (expected: %q)", asset.SHA256)
		if err := VerifySHA256(dmgPath, asset.SHA256); err != nil {
			return nil, fmt.Errorf("checksum: %w", err)
		}
		cfg.Logger.Log("Checksum OK (or skipped)")
		cfg.OnStep(2, "Checksum verified.", 1.0)

		// Step 3: Mount DMG → find .app → copy to /Applications → unmount
		cfg.OnStep(3, "Installing Rocq Platform...", 0.0)

		cfg.Logger.Log("Mounting DMG: %s", dmgPath)
		mountPoint, err := MountDMG(dmgPath)
		if err != nil {
			return nil, fmt.Errorf("mount DMG: %w", err)
		}

		appSrc, err := FindAppInDMG(mountPoint)
		if err != nil {
			UnmountDMG(mountPoint)
			return nil, fmt.Errorf("find app in DMG: %w", err)
		}
		cfg.Logger.Log("Found app in DMG: %s", appSrc)

		cfg.OnStep(3, fmt.Sprintf("Copying %s to Applications...", filepath.Base(appSrc)), 0.5)

		installedAppPath, err = InstallApp(appSrc, false)
		if err != nil {
			UnmountDMG(mountPoint)
			return nil, fmt.Errorf("install app: %w", err)
		}
		cfg.Logger.Log("App installed to: %s", installedAppPath)

		// Unmount DMG
		cfg.Logger.Log("Detaching DMG")
		if err := UnmountDMG(mountPoint); err != nil {
			cfg.Logger.Log("WARNING: failed to unmount DMG: %v", err)
		}

		cfg.OnStep(3, "Rocq Platform installed.", 1.0)
	}

	result.InstalledApp = installedAppPath

	// Step 4: Find language server binary
	topBinLabel := "vsrocqtop"
	if vscode.IsCoq(cfg.Manifest.RocqVersion) {
		topBinLabel = "vscoqtop"
	}
	cfg.OnStep(4, fmt.Sprintf("Locating %s...", topBinLabel), 0.0)
	vsrocqtopPath, err := FindLanguageServerTop(installedAppPath, cfg.Manifest.RocqVersion)
	if err != nil {
		warnMsg := fmt.Sprintf("%s not found: %v. VSCode will not be able to check proofs until this is resolved.", topBinLabel, err)
		cfg.Logger.Log("WARNING: %s", warnMsg)
		result.VsrocqtopWarning = warnMsg
		cfg.OnStep(4, fmt.Sprintf("%s not found (will skip VSCode settings).", topBinLabel), 1.0)
	} else {
		cfg.Logger.Log("Found %s: %s", topBinLabel, vsrocqtopPath)
		result.VsrocqtopPath = vsrocqtopPath
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
		settingsKey := "vsrocq.path"
		if vscode.IsCoq(cfg.Manifest.RocqVersion) {
			settingsKey = "vscoq.path"
		}
		if err := workspace.WriteVSCodeSettings(workspaceDir, settingsKey, vsrocqtopPath); err != nil {
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
