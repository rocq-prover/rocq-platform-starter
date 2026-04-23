package vscode

import (
	"fmt"
	"os/exec"

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

// FindCode searches for the VSCode CLI executable.
func FindCode() (string, error) {
	// Try PATH first
	path, err := exec.LookPath("code")
	if err == nil {
		return path, nil
	}

	// Try common Linux install locations
	candidates := []string{
		"/usr/bin/code",
		"/snap/bin/code",
		"/usr/share/code/bin/code",
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("VSCode (code) not found in PATH or common locations")
}
