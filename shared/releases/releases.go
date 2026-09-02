package releases

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	ReleasesURL = "https://api.github.com/repos/rocq-prover/platform/releases"
	ReleaseURL  = "https://api.github.com/repos/rocq-prover/platform/releases/tags/"
)

// GHRelease represents a GitHub release.
type GHRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
}

// GHAsset represents a GitHub release asset.
type GHAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GHReleaseDetail represents detailed GitHub release info.
type GHReleaseDetail struct {
	TagName string    `json:"tag_name"`
	Body    string    `json:"body"`
	Assets  []GHAsset `json:"assets"`
}

// FetchReleases returns available release tags from GitHub, filtered to exclude
// old "v" prefixed tags.
func FetchReleases() ([]string, error) {
	resp, err := http.Get(ReleasesURL + "?per_page=30")
	if err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch releases: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read releases body: %w", err)
	}

	var releases []GHRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("parse releases: %w", err)
	}

	var tags []string
	for _, r := range releases {
		if !strings.HasPrefix(r.TagName, "v") && !r.Prerelease {
			tags = append(tags, r.TagName)
		}
	}

	sort.Slice(tags, func(i, j int) bool {
		return CompareVersionDesc(tags[i], tags[j])
	})

	return tags, nil
}

// CompareVersionDesc returns true if a should come before b (newest first).
// Tags use the format YYYY.MM.patch (e.g. "2025.08.1").
func CompareVersionDesc(a, b string) bool {
	ap := ParseVersion(a)
	bp := ParseVersion(b)
	for k := 0; k < len(ap) && k < len(bp); k++ {
		if ap[k] != bp[k] {
			return ap[k] > bp[k]
		}
	}
	return len(ap) > len(bp)
}

// ParseVersion splits a version tag into integer components.
func ParseVersion(tag string) []int {
	parts := strings.Split(tag, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		nums[i], _ = strconv.Atoi(p)
	}
	return nums
}

// versionBoldRe matches bold Rocq/Coq version strings: **Rocq 9.0.1**
var versionBoldRe = regexp.MustCompile(`\*\*(?:Rocq|Coq)\s+(\d+\.\d+\.\d+)\*\*`)

// versionPlainRe matches plain Rocq/Coq version strings: Rocq 9.1.0
var versionPlainRe = regexp.MustCompile(`(?:Rocq|Coq)\s+(\d+\.\d+\.\d+)`)

// InferRocqVersion extracts the Rocq/Coq version from a release body text.
// It first tries bold markdown patterns (**Rocq X.Y.Z**), then falls back
// to plain text matches (Rocq X.Y.Z).
func InferRocqVersion(body string) string {
	if m := versionBoldRe.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	if m := versionPlainRe.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

// FetchRocqVersion fetches the Rocq version for a given release tag from the GitHub release body.
func FetchRocqVersion(tag string) (string, error) {
	rel, err := FetchReleaseDetail(tag)
	if err != nil {
		return "", err
	}
	ver := InferRocqVersion(rel.Body)
	if ver == "" {
		return "", fmt.Errorf("version not found in release body")
	}
	return ver, nil
}

// FetchReleaseDetail fetches the full release details for a given tag from GitHub.
func FetchReleaseDetail(tag string) (*GHReleaseDetail, error) {
	resp, err := http.Get(ReleaseURL + tag)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var rel GHReleaseDetail
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}
