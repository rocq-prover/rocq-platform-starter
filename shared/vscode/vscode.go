package vscode

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	RocqExtensionID = "rocq-prover.vsrocq"
	CoqExtensionID  = "coq-community.vscoq"
)

// IsCoq returns true if the version refers to a Coq release (major version < 9).
func IsCoq(version string) bool {
	parts := strings.SplitN(version, ".", 2)
	if len(parts) > 0 {
		if major, err := strconv.Atoi(parts[0]); err == nil && major < 9 {
			return true
		}
	}
	return false
}

// ExtensionIDForVersion returns the appropriate VSCode extension ID for the given version.
func ExtensionIDForVersion(rocqVersion string) string {
	if IsCoq(rocqVersion) {
		return CoqExtensionID
	}
	return RocqExtensionID
}

// InstallExtension installs the given VSCode extension if not already present.
func InstallExtension(codeBin, extensionID string) error {
	// Check if already installed
	out, err := exec.Command(codeBin, "--list-extensions").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.EqualFold(strings.TrimSpace(line), extensionID) {
				return nil // already installed
			}
		}
	}

	cmd := exec.Command(codeBin, "--install-extension", extensionID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install extension: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// OpenWorkspace opens VSCode with the given workspace directory.
func OpenWorkspace(codeBin, workspaceDir string) error {
	cmd := exec.Command(codeBin, workspaceDir)
	return cmd.Start()
}

// OpenWorkspaceWithDisabledExtensions opens VSCode with specified extensions
// disabled. Used by the Docker flow to prevent vsrocq from starting on the
// host (it will be properly installed inside the container by devcontainer.json).
func OpenWorkspaceWithDisabledExtensions(codeBin, workspaceDir string, extensionIDs []string) error {
	args := []string{}
	for _, id := range extensionIDs {
		args = append(args, "--disable-extension", id)
	}
	args = append(args, workspaceDir)
	cmd := exec.Command(codeBin, args...)
	return cmd.Start()
}
