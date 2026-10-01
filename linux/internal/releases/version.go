package releases

import "regexp"

// majorMinorRe captures the leading "<major>.<minor>" of a Rocq version,
// ignoring any patch level, pre-release or build suffix.
var majorMinorRe = regexp.MustCompile(`^(\d+)\.(\d+)`)

// RocqMajorMinor reduces a Rocq version to "<major>.<minor>". Both "9.0.1" and
// "9.1+rc1" reduce to "9.0" and "9.1"; anything unparseable yields "".
//
// The suffix matters: package-pick-9.1~2026.01.sh pins COQ_PLATFORM_COQ_TAG
// "9.1+rc1", so naive splitting on "." keeps "+rc1" in the minor component.
//
// Exported because internal/gui also needs it, to decide whether a release
// qualifies for Docker mode.
func RocqMajorMinor(version string) string {
	m := majorMinorRe.FindStringSubmatch(version)
	if m == nil {
		return ""
	}
	return m[1] + "." + m[2]
}
