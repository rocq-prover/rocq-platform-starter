package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ProgressFunc is called with bytes downloaded and total size (-1 if unknown).
type ProgressFunc func(downloaded, total int64)

// Download fetches url into destDir/destFilename and reports progress.
// Returns the path to the downloaded file.
func Download(url, destDir, destFilename string, progress ProgressFunc) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "rocq-platform-starter/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	destPath := filepath.Join(destDir, destFilename)
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	total := resp.ContentLength
	var downloaded int64
	lastReport := time.Now().Add(-time.Second) // ensure first chunk triggers a report

	buf := make([]byte, 256*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				return "", fmt.Errorf("write file: %w", writeErr)
			}
			downloaded += int64(n)
			if progress != nil && time.Since(lastReport) >= 200*time.Millisecond {
				progress(downloaded, total)
				lastReport = time.Now()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("read body: %w", readErr)
		}
	}
	// Final progress report to ensure 100% is shown.
	if progress != nil {
		progress(downloaded, total)
	}

	return destPath, nil
}
