package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justme0606/rocq-bootstrap/macos/internal/vscode"
)

// FindLanguageServerTop searches for the vsrocqtop or vscoqtop binary depending on the version.
// Search order:
// 1. Inside the installed .app bundle (Contents/ walk, max depth 10)
// 2. exec.LookPath
// 3. Known paths: /usr/local/bin, /opt/homebrew/bin
// 4. Scan /Applications and ~/Applications for *rocq*.app / *coq*.app
func FindLanguageServerTop(installedAppPath, rocqVersion string) (string, error) {
	binName := "vsrocqtop"
	if vscode.IsCoq(rocqVersion) {
		binName = "vscoqtop"
	}

	debugLog("[%s] searching for %s", binName, binName)

	// 1. Search inside the installed .app bundle
	if installedAppPath != "" {
		contentsDir := filepath.Join(installedAppPath, "Contents")
		if info, err := os.Stat(contentsDir); err == nil && info.IsDir() {
			debugLog("[%s] searching in %s (max depth 10)", binName, contentsDir)
			found := walkForBinary(contentsDir, binName, 10)
			if found != "" {
				debugLog("[%s] FOUND in app bundle: %s", binName, found)
				return found, nil
			}
		}
	}

	// 2. PATH lookup
	if path, err := exec.LookPath(binName); err == nil {
		debugLog("[%s] FOUND in PATH: %s", binName, path)
		return path, nil
	}

	// 3. Known paths
	knownPaths := []string{
		"/usr/local/bin/" + binName,
		"/opt/homebrew/bin/" + binName,
	}
	for _, p := range knownPaths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			debugLog("[%s] FOUND at known path: %s", binName, p)
			return p, nil
		}
	}

	// 4. Scan /Applications and ~/Applications for rocq/coq .app bundles
	home, _ := os.UserHomeDir()
	searchDirs := []string{"/Applications"}
	if home != "" {
		searchDirs = append(searchDirs, filepath.Join(home, "Applications"))
	}

	for _, baseDir := range searchDirs {
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || !strings.HasSuffix(e.Name(), ".app") {
				continue
			}
			nameLower := strings.ToLower(e.Name())
			if !strings.Contains(nameLower, "rocq") && !strings.Contains(nameLower, "coq") {
				continue
			}
			appContents := filepath.Join(baseDir, e.Name(), "Contents")
			if info, err := os.Stat(appContents); err == nil && info.IsDir() {
				found := walkForBinary(appContents, binName, 10)
				if found != "" {
					debugLog("[%s] FOUND in app scan: %s", binName, found)
					return found, nil
				}
			}
		}
	}

	debugLog("[%s] NOT FOUND", binName)
	return "", fmt.Errorf("%s not found", binName)
}

// walkForBinary walks a directory tree up to maxDepth levels looking for the named binary.
func walkForBinary(root, binName string, maxDepth int) string {
	var found string
	rootDepth := strings.Count(root, string(os.PathSeparator))

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Enforce max depth
		depth := strings.Count(path, string(os.PathSeparator)) - rootDepth
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && info.Name() == binName {
			// Verify it's executable
			if info.Mode()&0o111 != 0 {
				found = path
				return filepath.SkipAll
			}
		}
		return nil
	})

	return found
}
