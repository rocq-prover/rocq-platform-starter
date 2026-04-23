package vscode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	sharedvscode "github.com/justme0606/rocq-platform-starter/shared/vscode"
)

// IsCoq returns true if the version refers to a Coq release (major version < 9).
func IsCoq(version string) bool {
	return sharedvscode.IsCoq(version)
}

// ExtensionIDForVersion returns the appropriate VSCode extension ID for the given version.
func ExtensionIDForVersion(rocqVersion string) string {
	return sharedvscode.ExtensionIDForVersion(rocqVersion)
}

// InstallExtension installs the given VSCode extension if not already present.
func InstallExtension(codeBin, extensionID string) error {
	return sharedvscode.InstallExtension(codeBin, extensionID)
}

// OpenWorkspace opens VSCode with the given workspace directory.
func OpenWorkspace(codeBin, workspaceDir string) error {
	return sharedvscode.OpenWorkspace(codeBin, workspaceDir)
}

// FindCode searches for the VSCode CLI executable on macOS.
func FindCode() (string, error) {
	// 1. Try PATH first
	path, err := exec.LookPath("code")
	if err == nil {
		return path, nil
	}

	// 2. Standard macOS app bundle location
	appBundlePath := "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
	if info, err := os.Stat(appBundlePath); err == nil && !info.IsDir() {
		return appBundlePath, nil
	}

	// 3. User Applications folder
	home, _ := os.UserHomeDir()
	if home != "" {
		userAppPath := filepath.Join(home, "Applications/Visual Studio Code.app/Contents/Resources/app/bin/code")
		if info, err := os.Stat(userAppPath); err == nil && !info.IsDir() {
			return userAppPath, nil
		}
	}

	// 4. Homebrew cask location
	brewPaths := []string{
		"/opt/homebrew/bin/code",
		"/usr/local/bin/code",
	}
	for _, p := range brewPaths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("VSCode (code) not found in PATH or common locations")
}
