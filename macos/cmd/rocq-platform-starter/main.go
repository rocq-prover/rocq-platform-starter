package main

import (
	"os"

	"github.com/justme0606/rocq-platform-starter/shared/startup"

	rootfs "github.com/justme0606/rocq-platform-starter/macos"
	"github.com/justme0606/rocq-platform-starter/macos/internal/gui"
	"github.com/justme0606/rocq-platform-starter/macos/internal/manifest"
)

var Version = "dev"

func main() {
	showLog := false
	for _, arg := range os.Args[1:] {
		if arg == "--log" {
			showLog = true
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
