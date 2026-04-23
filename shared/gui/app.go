package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	windowWidth  = 620
	windowHeight = 520
)

// InstallContext holds references to all UI widgets needed during installation.
type InstallContext struct {
	Window      fyne.Window
	StatusLabel *widget.Label
	ProgressBar *widget.ProgressBar
	InfiniteBar *widget.ProgressBarInfinite
	StepLabel   *widget.Label
	InstallBtn  *widget.Button
	LogPanel    *LogPanel
	Checklist   *StepChecklist
	TotalSteps  int
	StepNames   []string

	lastLoggedStep int
}

// OnStep updates the UI for a step transition.
// It updates the status label, step label, progress bar, and log panel with
// structured "[step/total]" prefixes. When a step completes (fraction >= 1.0),
// the last log line is updated in-place with a "✓" marker.
func (ctx *InstallContext) OnStep(step int, label string, fraction float64) {
	overall := (float64(step-1) + fraction) / float64(ctx.TotalSteps)
	ctx.StatusLabel.SetText(label)
	ctx.StepLabel.SetText(fmt.Sprintf("Step %d/%d", step, ctx.TotalSteps))
	ctx.ProgressBar.SetValue(overall)

	prefix := fmt.Sprintf("[%d/%d]", step, ctx.TotalSteps)

	if step != ctx.lastLoggedStep {
		ctx.LogPanel.Append(fmt.Sprintf("%s %s", prefix, label))
		ctx.lastLoggedStep = step
		if ctx.Checklist != nil {
			ctx.Checklist.Show()
			ctx.Checklist.SetInProgress(step, label)
		}
	} else if ctx.Checklist != nil && fraction < 1.0 {
		ctx.Checklist.UpdateDetail(step, label)
	}
	if fraction >= 1.0 {
		// Determine the completion label: use step name if available, otherwise the current label
		doneLabel := label
		if step >= 1 && step <= len(ctx.StepNames) {
			doneLabel = ctx.StepNames[step-1]
		}
		ctx.LogPanel.UpdateLast(fmt.Sprintf("%s ✓ %s", prefix, doneLabel))
		if ctx.Checklist != nil {
			ctx.Checklist.SetDone(step, label)
		}
	}
}

// AppConfig holds all the platform-specific callbacks and configuration
// needed to run the shared GUI.
type AppConfig struct {
	Version    string
	TotalSteps int
	StepNames  []string

	// Manifest info (from the initially loaded manifest)
	RocqVersion     string
	PlatformRelease string

	// Release operations
	FetchReleases       func() ([]string, error)
	FetchRocqVersion    func(tag string) (string, error)
	FetchManifestForTag func(tag string) error // updates internal manifest state
	GetRocqVersion      func() string
	GetPlatformRelease  func() string

	// Existing installations
	FindExisting      func() []string
	ExistingLogMsg    func(item string) string // e.g. "Existing opam switch detected: %s"
	ExistingDialogMsg string                   // dialog message shown when existing installations are found
	NewInstallLabel   func() string            // label for the "install new" radio option

	// Doctor
	RunDoctor func(onLog func(string))

	// Install execution
	RunInstall func(ctx *InstallContext, existingSelection string, skipInstall bool)

	// ShowLog controls whether the log panel is visible (use --log flag)
	ShowLog bool

	// Icon (PNG bytes)
	Icon []byte
}

// Run creates and runs the shared GUI application.
func Run(cfg *AppConfig) {
	a := app.New()
	a.Settings().SetTheme(NewRocqTheme())

	var iconRes fyne.Resource
	if len(cfg.Icon) > 0 {
		iconRes = fyne.NewStaticResource("rocq-icon.png", cfg.Icon)
		a.SetIcon(iconRes)
	}

	windowTitle := "Rocq Platform Starter"
	if cfg.Version != "" && cfg.Version != "dev" {
		windowTitle += " - " + cfg.Version
	}
	w := a.NewWindow(windowTitle)
	w.Resize(fyne.NewSize(windowWidth, windowHeight))
	w.SetFixedSize(false)

	// --- Header: icon + title + version info ---
	var headerIcon *canvas.Image
	if iconRes != nil {
		headerIcon = canvas.NewImageFromResource(iconRes)
		headerIcon.FillMode = canvas.ImageFillContain
		headerIcon.SetMinSize(fyne.NewSize(64, 64))
	}

	titleRocq := canvas.NewText("Rocq", RocqOrange)
	titleRocq.TextSize = 20
	titleRocq.TextStyle = fyne.TextStyle{Bold: true}

	titleRest := canvas.NewText(" Platform Starter", RocqBlue)
	titleRest.TextSize = 20
	titleRest.TextStyle = fyne.TextStyle{Bold: true}

	titleRow := container.NewHBox(titleRocq, titleRest)

	titleBlock := container.NewVBox(titleRow)

	var header *fyne.Container
	if headerIcon != nil {
		header = container.NewHBox(headerIcon, container.NewCenter(titleBlock))
	} else {
		header = container.NewHBox(container.NewCenter(titleBlock))
	}

	headerSep := widget.NewSeparator()

	// --- Log panel (defined early so release callbacks can update it) ---
	logP := NewLogPanel()

	resetLog := func() {
		logP.Clear()
		logP.Append(fmt.Sprintf("Rocq version: %s", cfg.GetRocqVersion()))
		logP.Append(fmt.Sprintf("Platform release: %s", cfg.GetPlatformRelease()))
		logP.Append("Click 'Install' to begin.")
	}
	resetLog()

	// --- Release selector ---
	labelToTag := map[string]string{}

	initialLabel := cfg.PlatformRelease + " \u2014 " + VersionDisplayName(cfg.RocqVersion)
	labelToTag[initialLabel] = cfg.PlatformRelease
	labelToTag[cfg.PlatformRelease] = cfg.PlatformRelease

	releaseSelect := widget.NewSelect([]string{initialLabel}, func(selected string) {})
	releaseSelect.Selected = initialLabel

	releaseLabel := widget.NewLabelWithStyle("Release:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	releaseRow := container.NewBorder(nil, nil, releaseLabel, nil, releaseSelect)

	resolveTag := func(label string) string {
		if tag, ok := labelToTag[label]; ok {
			return tag
		}
		return label
	}

	// Fetch available releases in background with versions
	go func() {
		tags, err := cfg.FetchReleases()
		if err != nil {
			return
		}

		type tagVersion struct {
			tag     string
			version string
		}
		results := make(chan tagVersion, len(tags))
		for _, tag := range tags {
			go func(t string) {
				ver, err := cfg.FetchRocqVersion(t)
				if err != nil {
					results <- tagVersion{tag: t}
					return
				}
				results <- tagVersion{tag: t, version: ver}
			}(tag)
		}

		versionMap := map[string]string{}
		for range tags {
			r := <-results
			if r.version != "" {
				versionMap[r.tag] = r.version
			}
		}

		var options []string
		for _, tag := range tags {
			label := tag
			if ver, ok := versionMap[tag]; ok {
				label = tag + " \u2014 " + VersionDisplayName(ver)
			}
			labelToTag[label] = tag
			options = append(options, label)
		}

		releaseSelect.Options = options
		if len(options) > 0 {
			releaseSelect.Selected = options[0]
		}
		releaseSelect.Refresh()

		// Fetch full manifest for the most recent release
		latestTag := resolveTag(releaseSelect.Selected)
		if err := cfg.FetchManifestForTag(latestTag); err != nil {
			return
		}
		resetLog()
	}()

	releaseSelect.OnChanged = func(selected string) {
		tag := resolveTag(selected)
		if tag == cfg.GetPlatformRelease() {
			return
		}
		releaseSelect.Disable()
		go func() {
			if err := cfg.FetchManifestForTag(tag); err != nil {
				releaseSelect.Enable()
				return
			}
			resetLog()
			releaseSelect.Enable()
		}()
	}

	// --- Progress section ---
	totalSteps := cfg.TotalSteps
	statusLabel := widget.NewLabel("Ready to install")
	statusLabel.Wrapping = fyne.TextWrapWord
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	stepLabel := widget.NewLabel(fmt.Sprintf("Step 0/%d", totalSteps))
	stepLabel.Alignment = fyne.TextAlignTrailing

	progressBar := widget.NewProgressBar()
	progressBar.Min = 0
	progressBar.Max = 1.0

	infiniteBar := widget.NewProgressBarInfinite()
	infiniteBar.Hide()

	progressStack := container.NewStack(progressBar, infiniteBar)

	statusRow := container.NewBorder(nil, nil, nil, stepLabel, statusLabel)

	progressSection := container.NewVBox(
		statusRow,
		progressStack,
	)

	// --- Step checklist (hidden until install starts) ---
	checklist := NewStepChecklist(cfg.StepNames)

	// --- Log panel layout ---
	logScroll := container.NewScroll(logP.Display)
	logScroll.SetMinSize(fyne.NewSize(580, 220))

	logHeader := widget.NewLabelWithStyle("Log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	logSection := container.NewBorder(
		logHeader,
		nil, nil, nil,
		logScroll,
	)
	if !cfg.ShowLog {
		logSection.Hide()
	}

	// --- Install button ---
	installing := false
	var installBtn *widget.Button
	installBtn = widget.NewButtonWithIcon("Install", theme.DownloadIcon(), func() {
		installing = true
		installBtn.Disable()
		releaseSelect.Disable()

		ctx := &InstallContext{
			Window:      w,
			StatusLabel: statusLabel,
			ProgressBar: progressBar,
			InfiniteBar: infiniteBar,
			StepLabel:   stepLabel,
			InstallBtn:  installBtn,
			LogPanel:    logP,
			Checklist:   checklist,
			TotalSteps:  totalSteps,
			StepNames:   cfg.StepNames,
		}

		existingItems := cfg.FindExisting()
		if len(existingItems) > 0 {
			msg := widget.NewLabel(cfg.ExistingDialogMsg)
			msg.Wrapping = fyne.TextWrapWord

			newLabel := cfg.NewInstallLabel()
			options := append(existingItems, newLabel)
			radio := widget.NewRadioGroup(options, nil)
			radio.SetSelected(existingItems[0])

			radioScroll := container.NewScroll(radio)
			radioScroll.SetMinSize(fyne.NewSize(400, 200))

			closeBtn := widget.NewButton("Close", nil)
			closeBtn.Importance = widget.HighImportance
			confirmBtn := widget.NewButton("Continue", nil)
			confirmBtn.Importance = widget.HighImportance

			buttons := container.NewHBox(layout.NewSpacer(), closeBtn, confirmBtn)
			content := container.NewVBox(msg, radioScroll, buttons)
			d := dialog.NewCustomWithoutButtons("Existing Installation Detected", content, w)

			closeBtn.OnTapped = func() {
				d.Hide()
				installBtn.Enable()
				releaseSelect.Enable()
			}
			confirmBtn.OnTapped = func() {
				d.Hide()
				selected := radio.Selected
				if selected == newLabel {
					logP.Append("Starting fresh installation...")
					go cfg.RunInstall(ctx, "", false)
				} else {
					logP.Append(fmt.Sprintf("Reusing %s...", selected))
					go cfg.RunInstall(ctx, selected, true)
				}
			}

			d.Show()
		} else {
			logP.Append("Starting installation...")
			go cfg.RunInstall(ctx, "", false)
		}
	})
	installBtn.Importance = widget.HighImportance

	// --- Doctor button ---
	var doctorBtn *widget.Button
	doctorBtn = widget.NewButtonWithIcon("Doctor", theme.InfoIcon(), func() {
		installBtn.Disable()
		doctorBtn.Disable()
		statusLabel.SetText("Running diagnostics...")
		progressBar.Hide()
		infiniteBar.Show()

		go func() {
			var lines []string
			cfg.RunDoctor(func(msg string) {
				lines = append(lines, msg)
			})

			infiniteBar.Hide()
			progressBar.Show()
			statusLabel.SetText("Ready to install")

			richText := widget.NewRichText()
			richText.Wrapping = fyne.TextWrapWord
			richText.ParseMarkdown("```\n" + strings.Join(lines, "\n") + "\n```")

			scroll := container.NewScroll(richText)
			scroll.SetMinSize(fyne.NewSize(560, 350))

			closeBtn := widget.NewButton("Close", nil)
			closeBtn.Importance = widget.HighImportance

			content := container.NewBorder(nil, container.NewCenter(closeBtn), nil, nil, scroll)
			d := dialog.NewCustomWithoutButtons("Doctor \u2014 System Diagnostic", content, w)

			closeBtn.OnTapped = func() {
				d.Hide()
			}

			d.Show()

			if !installing {
				installBtn.Enable()
			}
			doctorBtn.Enable()
		}()
	})
	doctorBtn.Importance = widget.HighImportance

	versionLabel := widget.NewLabelWithStyle(cfg.Version, fyne.TextAlignTrailing, fyne.TextStyle{})
	versionLabel.Importance = widget.LowImportance

	bottomBar := container.NewPadded(
		container.NewBorder(nil, nil, nil, versionLabel,
			container.NewCenter(container.NewHBox(doctorBtn, installBtn)),
		),
	)

	// --- Main layout ---
	content := container.NewPadded(
		container.NewBorder(
			container.NewVBox(
				header,
				headerSep,
				releaseRow,
				progressSection,
			),
			bottomBar,
			nil, nil,
			container.NewVBox(
				checklist.Container,
				logSection,
				layout.NewSpacer(),
			),
		),
	)

	w.SetContent(content)
	w.ShowAndRun()
}
