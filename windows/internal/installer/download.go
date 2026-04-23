package installer

import (
	sharedinstaller "github.com/justme0606/rocq-platform-starter/shared/installer"
)

// ProgressFunc is called with bytes downloaded and total size (-1 if unknown).
type ProgressFunc = sharedinstaller.ProgressFunc

// Download fetches url to a temporary file and reports progress.
// Returns the path to the downloaded file.
func Download(url, destDir string, progress ProgressFunc) (string, error) {
	return sharedinstaller.Download(url, destDir, "rocq-platform-installer.exe", progress)
}
