package gui

import (
	"fmt"
	"io/fs"
	"time"

	sharedgui "github.com/justme0606/rocq-bootstrap/shared/gui"

	"github.com/justme0606/rocq-bootstrap/macos/internal/doctor"
	"github.com/justme0606/rocq-bootstrap/macos/internal/installer"
	"github.com/justme0606/rocq-bootstrap/macos/internal/manifest"
	"github.com/justme0606/rocq-bootstrap/macos/internal/releases"
)

const totalSteps = 7

// Run creates and runs the GUI application.
func Run(m *manifest.Manifest, templates fs.FS, icon []byte, version string, showLog bool) {
	currentManifest := m

	cfg := &sharedgui.AppConfig{
		Version:    version,
		TotalSteps: totalSteps,
		StepNames: []string{
			"Download Rocq Platform",
			"Verify checksum",
			"Install application",
			"Locate language server",
			"Check for VSCode",
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
			return fmt.Sprintf("Existing Rocq Platform detected: %s", item)
		},
		ExistingDialogMsg: "Existing Rocq Platform installations were found.\nSelect one to reuse, or install a new one:",
		NewInstallLabel: func() string {
			return fmt.Sprintf("Install new (%s)", installer.DefaultInstallDir())
		},

		RunDoctor: func(onLog func(string)) {
			doctor.Run(onLog)
		},

		RunInstall: func(ctx *sharedgui.InstallContext, existingSelection string, skipInstall bool) {
			runInstall(ctx, currentManifest, templates, existingSelection, skipInstall)
		},
	}

	sharedgui.Run(cfg)
}

func runInstall(ctx *sharedgui.InstallContext, m *manifest.Manifest, templates fs.FS,
	existingApp string, skipInstall bool) {

	startTime := time.Now()

	logger, err := installer.NewLogger()
	if err != nil {
		ctx.LogPanel.Append(fmt.Sprintf("WARNING: could not create log file: %v", err))
	}
	if logger != nil {
		defer logger.Close()
	}

	cfg := &installer.Config{
		Manifest:    m,
		Templates:   templates,
		SkipInstall: skipInstall,
		ExistingApp: existingApp,
		Logger:      logger,
		OnStep:      ctx.OnStep,
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
		ctx.LogPanel.Append(fmt.Sprintf("Installed app: %s", result.InstalledApp))

		if ctx.Checklist != nil {
			ctx.Checklist.AppendSummary("")
			ctx.Checklist.AppendSummary(fmt.Sprintf("Rocq Platform installed successfully in %s.", elapsed))
			ctx.Checklist.AppendSummary(fmt.Sprintf("Installed app: %s", result.InstalledApp))
		}

		sharedgui.ShowVSCodeDialog(ctx.Window)
		return
	}

	ctx.StatusLabel.SetText(fmt.Sprintf("Installation complete! (%s)", elapsed))
	ctx.LogPanel.Append(fmt.Sprintf("Installation complete! (%s)", elapsed))
	ctx.LogPanel.Append(fmt.Sprintf("Installed app: %s", result.InstalledApp))
	ctx.LogPanel.Append("Workspace: ~/rocq-workspace")

	if result.VsrocqtopWarning != "" {
		ctx.LogPanel.Append(fmt.Sprintf("WARNING: %s", result.VsrocqtopWarning))
	}

	if ctx.Checklist != nil {
		ctx.Checklist.AppendSummary("")
		ctx.Checklist.AppendSummary(fmt.Sprintf("Installation complete! (%s)", elapsed))
		ctx.Checklist.AppendSummary(fmt.Sprintf("Installed app: %s", result.InstalledApp))
		ctx.Checklist.AppendSummary("Workspace: ~/rocq-workspace")
		if result.VsrocqtopWarning != "" {
			ctx.Checklist.AppendSummary(fmt.Sprintf("WARNING: %s", result.VsrocqtopWarning))
		}
	}

	successMsg := fmt.Sprintf("Rocq Platform has been installed successfully in %s.\n\n", elapsed) +
		fmt.Sprintf("Installed app: %s\n", result.InstalledApp) +
		"Workspace: ~/rocq-workspace"
	if result.VsrocqtopWarning != "" {
		successMsg += fmt.Sprintf("\n\nWarning: %s", result.VsrocqtopWarning)
	}
	sharedgui.ShowSuccess(ctx.Window, successMsg)
}
