package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rocq-prover/rocq-platform-starter/linux/internal/vscode"
)

// FindLanguageServerTop searches for the vsrocqtop or vscoqtop binary in the opam switch.
// Search order:
// 1. opam switch bin directory (<switch>/bin/)
// 2. exec.LookPath (PATH)
// 3. Known paths: ~/.local/bin, /usr/local/bin
func FindLanguageServerTop(switchName, rocqVersion string) (string, error) {
	binName := "vsrocqtop"
	if vscode.IsCoq(rocqVersion) {
		binName = "vscoqtop"
	}

	debugLog("[%s] searching for %s", binName, binName)

	// 1. Search in the opam switch bin directory
	if switchName != "" {
		out, err := exec.Command("opam", "var", "--switch="+switchName, "bin").Output()
		if err == nil {
			binDir := strings.TrimSpace(string(out))
			topPath := filepath.Join(binDir, binName)
			if info, err := os.Stat(topPath); err == nil && !info.IsDir() {
				debugLog("[%s] FOUND in opam switch: %s", binName, topPath)
				return topPath, nil
			}
		}
	}

	// 2. PATH lookup
	if path, err := exec.LookPath(binName); err == nil {
		debugLog("[%s] FOUND in PATH: %s", binName, path)
		return path, nil
	}

	// 3. Known paths
	home, _ := os.UserHomeDir()
	var knownPaths []string
	if home != "" {
		knownPaths = append(knownPaths, filepath.Join(home, ".local", "bin", binName))
	}
	knownPaths = append(knownPaths, "/usr/local/bin/"+binName)

	for _, p := range knownPaths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			debugLog("[%s] FOUND at known path: %s", binName, p)
			return p, nil
		}
	}

	debugLog("[%s] NOT FOUND", binName)
	return "", fmt.Errorf("%s not found", binName)
}
