package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SetupEarlyLog creates an early log file to capture errors before the GUI starts.
// Returns nil if the log file could not be created.
func SetupEarlyLog() *os.File {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	logDir := filepath.Join(home, ".rocq-setup", "logs")
	os.MkdirAll(logDir, 0o755)

	name := fmt.Sprintf("rocq-platform-starter-%s.log", time.Now().Format("20060102-150405"))
	f, err := os.Create(filepath.Join(logDir, name))
	if err != nil {
		return nil
	}
	return f
}

// BootstrapConfig holds the callbacks needed to bootstrap the application.
type BootstrapConfig struct {
	LoadManifest    func() error  // loads the manifest, stores internally
	RunGUI          func()        // launches the GUI
	RocqVersion     func() string // for logging
	PlatformRelease func() string // for logging
}

// Bootstrap handles the common startup sequence: early log, manifest loading,
// logging, and GUI launch.
func Bootstrap(cfg *BootstrapConfig) {
	earlyLog := SetupEarlyLog()
	if earlyLog != nil {
		defer earlyLog.Close()
		fmt.Fprintf(earlyLog, "[%s] rocq-platform-starter starting\n", time.Now().Format("15:04:05"))
	}

	if err := cfg.LoadManifest(); err != nil {
		msg := fmt.Sprintf("Fatal: %v", err)
		if earlyLog != nil {
			fmt.Fprintln(earlyLog, msg)
		}
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}

	if earlyLog != nil {
		fmt.Fprintf(earlyLog, "[%s] manifest loaded: Rocq %s (platform %s)\n",
			time.Now().Format("15:04:05"), cfg.RocqVersion(), cfg.PlatformRelease())
		fmt.Fprintf(earlyLog, "[%s] launching GUI\n", time.Now().Format("15:04:05"))
	}

	cfg.RunGUI()
}
