package gui

import (
	"fmt"
	"io/fs"
	"sort"
	"time"

	sharedgui "github.com/justme0606/rocq-platform-starter/shared/gui"

	"github.com/justme0606/rocq-platform-starter/linux/internal/doctor"
	"github.com/justme0606/rocq-platform-starter/linux/internal/installer"
	"github.com/justme0606/rocq-platform-starter/linux/internal/manifest"
	"github.com/justme0606/rocq-platform-starter/linux/internal/releases"
)

const totalSteps = 7

// Run creates and runs the GUI application.
func Run(m *manifest.Manifest, templates fs.FS, icon []byte, version string, showLog bool) {
	currentManifest := m

	cfg := &sharedgui.AppConfig{
		Version:    version,
		TotalSteps: totalSteps,
		StepNames: []string{
			"Check/install opam",
			"Initialize opam",
			"Create opam switch",
			"Configure repository",
			"Install Rocq packages",
			"Create workspace",
			"Configure VSCode",
		},
		RocqVersion:     m.RocqVersion,
		PlatformRelease: m.PlatformRelease,
		ShowLog:         showLog,
		Icon:            icon,

		FetchReleases:    releases.FetchReleases,
		FetchRocqVersion: releases.FetchRocqVersion,
		FetchManifestForTag: func(tag string) error {
			newManifest, err := releases.FetchManifestForTag(tag)
			if err != nil {
				return err
			}
			currentManifest = newManifest
			return nil
		},
		GetRocqVersion:     func() string { return currentManifest.RocqVersion },
		GetPlatformRelease: func() string { return currentManifest.PlatformRelease },

		FindExisting: installer.FindExistingInstallations,
		ExistingLogMsg: func(item string) string {
			return fmt.Sprintf("Existing opam switch detected: %s", item)
		},
		ExistingDialogMsg: "Existing opam switches were found.\nSelect one to reuse, or install a new switch:",
		NewInstallLabel: func() string {
			return fmt.Sprintf("Install new (%s)", installer.SwitchName(currentManifest.RocqVersion, currentManifest.PlatformRelease))
		},

		RunDoctor: func(onLog func(string)) {
			doctor.Run(onLog)
		},

		RunInstall: func(ctx *sharedgui.InstallContext, existingSelection string, skipInstall bool) {
			runInstall(ctx, currentManifest, templates, skipInstall)
		},
	}

	// Configure Docker button if the manifest has a docker section
	configureDocker := func() {
		if currentManifest.Docker == nil {
			cfg.DockerInstall = nil
			return
		}
		dc := currentManifest.Docker
		if _, ok := dc.Variants[dc.DefaultVariant]; !ok {
			cfg.DockerInstall = nil
			return
		}

		// Build sorted variants slice: default first, then the rest alphabetically
		var others []string
		for name := range dc.Variants {
			if name != dc.DefaultVariant {
				others = append(others, name)
			}
		}
		sort.Strings(others)

		ordered := append([]string{dc.DefaultVariant}, others...)
		var variants []sharedgui.DockerVariant
		for _, name := range ordered {
			v := dc.Variants[name]
			variants = append(variants, sharedgui.DockerVariant{
				Name:        name,
				Description: v.Description,
			})
		}

		cfg.DockerInstall = &sharedgui.DockerInstallConfig{
			StepCount: 5,
			StepNames: []string{
				"Check Docker",
				"Pull image",
				"Create workspace",
				"Configure VSCode",
				"Open VSCode",
			},
			Variants:       variants,
			DefaultVariant: dc.DefaultVariant,
			RunInstall: func(ctx *sharedgui.InstallContext, variant string) {
				// Read current docker config at invocation time
				curDC := currentManifest.Docker
				curVariant, ok := curDC.Variants[variant]
				if !ok {
					curVariant = curDC.Variants[curDC.DefaultVariant]
				}
				curImage := curDC.Registry + "/" + curVariant.Image + ":" + curDC.Tag
				runDockerInstall(ctx, curImage, curDC.VsrocqtopPath, curDC.User, templates)
			},
		}
	}
	configureDocker()

	// Wrap FetchManifestForTag to update Docker config on release change
	origFetch := cfg.FetchManifestForTag
	cfg.FetchManifestForTag = func(tag string) error {
		if err := origFetch(tag); err != nil {
			return err
		}
		configureDocker()
		return nil
	}

	sharedgui.Run(cfg)
}

func runInstall(ctx *sharedgui.InstallContext, m *manifest.Manifest, templates fs.FS, skipInstall bool) {
	startTime := time.Now()

	logger, err := installer.NewLogger()
	if err != nil {
		ctx.LogPanel.Append(fmt.Sprintf("WARNING: could not create log file: %v", err))
	}
	if logger != nil {
		defer logger.Close()
	}

	switchName := installer.SwitchName(m.RocqVersion, m.PlatformRelease)

	cfg := &installer.Config{
		Manifest:    m,
		Templates:   templates,
		SkipInstall: skipInstall,
		Logger:      logger,
		OnStep: func(step int, label string, fraction float64) {
			// Show infinite progress bar during long opam operations (steps 2-5)
			if step >= 2 && step <= 5 && fraction < 1.0 {
				ctx.ProgressBar.Hide()
				ctx.InfiniteBar.Show()
			} else {
				ctx.InfiniteBar.Hide()
				ctx.ProgressBar.Show()
			}
			ctx.OnStep(step, label, fraction)
		},
	}

	result, err := installer.Run(cfg)
	if err != nil {
		if logger != nil {
			logger.Log("ERROR: %v", err)
		}
		ctx.LogPanel.Append(fmt.Sprintf("ERROR: %v", err))
		sharedgui.ShowError(ctx.Window, ctx.InstallBtn, err.Error())
		return
	}

	ctx.ProgressBar.SetValue(1.0)

	elapsed := sharedgui.FormatDuration(time.Since(startTime))

	if !result.VSCodeFound {
		ctx.StatusLabel.SetText(fmt.Sprintf("Rocq Platform installed in %s — VSCode not found", elapsed))
		ctx.LogPanel.Append(fmt.Sprintf("Rocq Platform installed successfully in %s.", elapsed))
		ctx.LogPanel.Append("VSCode was not found. Install VSCode then re-run this installer to configure the workspace.")
		ctx.LogPanel.Append(fmt.Sprintf("Opam switch: %s", switchName))
		ctx.LogPanel.Append("Activate with: source ~/rocq-workspace/activate.sh")

		if ctx.Checklist != nil {
			ctx.Checklist.AppendSummary("")
			ctx.Checklist.AppendSummary(fmt.Sprintf("Rocq Platform installed successfully in %s.", elapsed))
			ctx.Checklist.AppendSummary(fmt.Sprintf("Opam switch: %s", switchName))
			ctx.Checklist.AppendSummary("Activate with: source ~/rocq-workspace/activate.sh")
		}

		sharedgui.ShowVSCodeDialog(ctx.Window)
		return
	}

	ctx.StatusLabel.SetText(fmt.Sprintf("Installation complete! (%s)", elapsed))
	ctx.LogPanel.Append(fmt.Sprintf("Installation complete! (%s)", elapsed))
	ctx.LogPanel.Append(fmt.Sprintf("Opam switch: %s", switchName))
	ctx.LogPanel.Append(fmt.Sprintf("Workspace: ~/%s", installer.WorkspaceName))
	ctx.LogPanel.Append("Activate with: source ~/rocq-workspace/activate.sh")

	if ctx.Checklist != nil {
		ctx.Checklist.AppendSummary("")
		ctx.Checklist.AppendSummary(fmt.Sprintf("Installation complete! (%s)", elapsed))
		ctx.Checklist.AppendSummary(fmt.Sprintf("Opam switch: %s", switchName))
		ctx.Checklist.AppendSummary(fmt.Sprintf("Workspace: ~/%s", installer.WorkspaceName))
		ctx.Checklist.AppendSummary("Activate with: source ~/rocq-workspace/activate.sh")
	}

	sharedgui.ShowSuccess(ctx.Window,
		fmt.Sprintf("Rocq Platform has been installed successfully in %s.\n\n", elapsed)+
			fmt.Sprintf("Opam switch: %s\n", switchName)+
			fmt.Sprintf("Workspace: ~/%s\n\n", installer.WorkspaceName)+
			"Activate with:\n  source ~/rocq-workspace/activate.sh")
}

func runDockerInstall(ctx *sharedgui.InstallContext, image, vsrocqtopPath, user string, templates fs.FS) {
	startTime := time.Now()

	logger, err := installer.NewLogger()
	if err != nil {
		ctx.LogPanel.Append(fmt.Sprintf("WARNING: could not create log file: %v", err))
	}
	if logger != nil {
		defer logger.Close()
	}

	cfg := &installer.DockerConfig{
		Image:         image,
		VsrocqtopPath: vsrocqtopPath,
		User:          user,
		Templates:     templates,
		Logger:        logger,
		OnStep: func(step int, label string, fraction float64) {
			// Show infinite progress bar during docker pull (step 2)
			if step == 2 && fraction < 1.0 {
				ctx.ProgressBar.Hide()
				ctx.InfiniteBar.Show()
			} else {
				ctx.InfiniteBar.Hide()
				ctx.ProgressBar.Show()
			}
			ctx.OnStep(step, label, fraction)
		},
	}

	result, err := installer.RunDocker(cfg)
	if err != nil {
		if logger != nil {
			logger.Log("ERROR: %v", err)
		}
		ctx.LogPanel.Append(fmt.Sprintf("ERROR: %v", err))
		sharedgui.ShowError(ctx.Window, ctx.InstallBtn, err.Error())
		return
	}

	ctx.ProgressBar.SetValue(1.0)

	elapsed := sharedgui.FormatDuration(time.Since(startTime))

	if !result.VSCodeFound {
		ctx.StatusLabel.SetText(fmt.Sprintf("Docker setup done in %s — VSCode not found", elapsed))
		ctx.LogPanel.Append(fmt.Sprintf("Docker setup completed in %s.", elapsed))
		ctx.LogPanel.Append("VSCode was not found. Install VSCode and the Dev Containers extension to use the workspace.")
		ctx.LogPanel.Append(fmt.Sprintf("Workspace: ~/%s", installer.WorkspaceName))

		if ctx.Checklist != nil {
			ctx.Checklist.AppendSummary("")
			ctx.Checklist.AppendSummary(fmt.Sprintf("Docker setup completed in %s.", elapsed))
			ctx.Checklist.AppendSummary(fmt.Sprintf("Workspace: ~/%s", installer.WorkspaceName))
			ctx.Checklist.AppendSummary("Install VSCode to use the Dev Container.")
		}

		sharedgui.ShowVSCodeDialog(ctx.Window)
		return
	}

	ctx.StatusLabel.SetText(fmt.Sprintf("Docker setup complete! (%s)", elapsed))
	ctx.LogPanel.Append(fmt.Sprintf("Docker setup complete! (%s)", elapsed))
	ctx.LogPanel.Append(fmt.Sprintf("Image: %s", image))
	ctx.LogPanel.Append(fmt.Sprintf("Workspace: ~/%s", installer.WorkspaceName))
	ctx.LogPanel.Append("Open the workspace in VSCode and use 'Reopen in Container'.")

	if ctx.Checklist != nil {
		ctx.Checklist.AppendSummary("")
		ctx.Checklist.AppendSummary(fmt.Sprintf("Docker setup complete! (%s)", elapsed))
		ctx.Checklist.AppendSummary(fmt.Sprintf("Image: %s", image))
		ctx.Checklist.AppendSummary(fmt.Sprintf("Workspace: ~/%s", installer.WorkspaceName))
		ctx.Checklist.AppendSummary("Use 'Reopen in Container' in VSCode.")
	}

	sharedgui.ShowSuccess(ctx.Window,
		fmt.Sprintf("Docker setup completed successfully in %s.\n\n", elapsed)+
			fmt.Sprintf("Image: %s\n", image)+
			fmt.Sprintf("Workspace: ~/%s\n\n", installer.WorkspaceName)+
			"Open the workspace in VSCode and select\n'Reopen in Container' to start coding.")
}
