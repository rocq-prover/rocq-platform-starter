package installer

import (
	sharedinstaller "github.com/rocq-prover/rocq-platform-starter/shared/installer"
)

// Logger wraps the shared Logger type.
type Logger = sharedinstaller.Logger

// NewLogger creates a log file under ~/.rocq-setup/logs/.
func NewLogger() (*Logger, error) {
	return sharedinstaller.NewLogger()
}
