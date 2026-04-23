package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/justme0606/rocq-platform-starter/shared/startup"

	rootfs "github.com/justme0606/rocq-platform-starter/linux"
	"github.com/justme0606/rocq-platform-starter/linux/internal/gui"
	"github.com/justme0606/rocq-platform-starter/linux/internal/manifest"
)

var Version = "dev"

const (
	binaryName  = "rocq-platform-starter"
	desktopFile = `[Desktop Entry]
Name=Rocq-Platform-Starter
Comment=Rocq Platform Starter
Exec=rocq-platform-starter
Icon=rocq-platform-starter
Terminal=false
Type=Application
Categories=Development;Education;Science;
Keywords=Rocq;Coq;proof;assistant;opam;
`
)

func main() {
	showLog := false
	for _, arg := range os.Args[1:] {
		if arg == "--log" {
			showLog = true
		}
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--install":
			if err := installDesktop(); err != nil {
				fmt.Fprintf(os.Stderr, "Install failed: %v\n", err)
				os.Exit(1)
			}
			return
		case "--uninstall":
			if err := uninstallDesktop(); err != nil {
				fmt.Fprintf(os.Stderr, "Uninstall failed: %v\n", err)
				os.Exit(1)
			}
			return
		case "--help", "-h":
			fmt.Println("Usage: rocq-platform-starter [--install | --uninstall | --log | --help]")
			fmt.Println()
			fmt.Println("  (no args)     Launch the GUI installer")
			fmt.Println("  --install     Install as desktop application (~/.local)")
			fmt.Println("  --uninstall   Remove desktop application")
			fmt.Println("  --log         Show the log panel in the GUI")
			fmt.Println("  --help        Show this help")
			return
		}
	}

	var m *manifest.Manifest

	startup.Bootstrap(&startup.BootstrapConfig{
		LoadManifest: func() error {
			var err error
			m, err = manifest.Load(rootfs.EmbeddedManifest, "embedded/manifest/latest.json")
			return err
		},
		RunGUI:          func() { gui.Run(m, rootfs.EmbeddedTemplates, rootfs.EmbeddedIcon, Version, showLog) },
		RocqVersion:     func() string { return m.RocqVersion },
		PlatformRelease: func() string { return m.PlatformRelease },
	})
}

func installDesktop() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	binDir := filepath.Join(home, ".local", "bin")
	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps")
	desktopDir := filepath.Join(home, ".local", "share", "applications")

	// Create directories
	for _, dir := range []string{binDir, iconDir, desktopDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	// Copy self to ~/.local/bin/
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	destBin := filepath.Join(binDir, binaryName)
	if err := copyFile(self, destBin, 0o755); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	fmt.Printf("Installed binary:  %s\n", destBin)

	// Write embedded icon
	destIcon := filepath.Join(iconDir, "rocq-platform-starter.png")
	if err := os.WriteFile(destIcon, rootfs.EmbeddedIcon, 0o644); err != nil {
		return fmt.Errorf("write icon: %w", err)
	}
	fmt.Printf("Installed icon:    %s\n", destIcon)

	// Write .desktop file
	destDesktop := filepath.Join(desktopDir, "rocq-platform-starter.desktop")
	if err := os.WriteFile(destDesktop, []byte(desktopFile), 0o644); err != nil {
		return fmt.Errorf("write desktop file: %w", err)
	}
	fmt.Printf("Installed desktop: %s\n", destDesktop)

	fmt.Println()
	fmt.Println("Rocq-Platform-Starter installed as desktop application.")
	fmt.Printf("Make sure %s is in your PATH.\n", binDir)
	return nil
}

func uninstallDesktop() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	files := []string{
		filepath.Join(home, ".local", "bin", binaryName),
		filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps", "rocq-platform-starter.png"),
		filepath.Join(home, ".local", "share", "applications", "rocq-platform-starter.desktop"),
	}

	for _, f := range files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: could not remove %s: %v\n", f, err)
		} else if err == nil {
			fmt.Printf("Removed: %s\n", f)
		}
	}

	fmt.Println("Rocq-Platform-Starter uninstalled.")
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
