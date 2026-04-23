package installer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justme0606/rocq-platform-starter/windows/internal/vscode"
)

// FindLanguageServerTop searches for vsrocqtop or vscoqtop in the installation directory.
// It first checks <installDir>/bin/, then searches recursively.
func FindLanguageServerTop(installDir, rocqVersion string) (string, error) {
	binBase := "vsrocqtop"
	if vscode.IsCoq(rocqVersion) {
		binBase = "vscoqtop"
	}
	names := []string{binBase, binBase + ".exe"}

	debugLog("[%s] searching in %s", binBase, installDir)

	// Check the expected locations first
	for _, name := range names {
		direct := filepath.Join(installDir, "bin", name)
		debugLog("[%s] checking %s", binBase, direct)
		if info, err := os.Stat(direct); err == nil && !info.IsDir() {
			debugLog("[%s] FOUND at %s", binBase, direct)
			return direct, nil
		}
	}

	// Recursive search
	debugLog("[%s] not in bin/, starting recursive search...", binBase)
	var found string
	err := filepath.Walk(installDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if !info.IsDir() {
			name := info.Name()
			if name == binBase || name == binBase+".exe" {
				found = path
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		debugLog("[%s] walk error: %v", binBase, err)
		return "", fmt.Errorf("search %s: %w", binBase, err)
	}

	if found == "" {
		debugLog("[%s] NOT FOUND anywhere in %s", binBase, installDir)
		return "", fmt.Errorf("%s not found in %s", binBase, installDir)
	}

	debugLog("[%s] FOUND at %s", binBase, found)
	return found, nil
}
