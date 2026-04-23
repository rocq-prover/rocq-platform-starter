package workspace

import (
	"io/fs"

	sharedworkspace "github.com/justme0606/rocq-platform-starter/shared/workspace"
)

// Create creates the workspace directory with template files.
// Existing files are not overwritten.
func Create(workspaceDir string, templates fs.FS) error {
	return sharedworkspace.Create(workspaceDir, templates)
}

// WriteVSCodeSettings writes .vscode/settings.json with the language server path.
func WriteVSCodeSettings(workspaceDir, settingsKey, topPath string) error {
	return sharedworkspace.WriteVSCodeSettings(workspaceDir, settingsKey, topPath)
}

// WriteActivationScripts generates shell activation scripts for the opam switch.
func WriteActivationScripts(workspaceDir, switchName string) error {
	return sharedworkspace.WriteActivationScripts(workspaceDir, switchName)
}
