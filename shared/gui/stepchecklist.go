package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// StepChecklist displays all installation steps as a checklist with visual
// indicators: ○ (pending), ▶ (in progress), ✓ (done).
// Each step has an optional detail line shown below the step name.
type StepChecklist struct {
	icons   []*canvas.Text
	names   []*canvas.Text
	details []*canvas.Text
	steps   []string
	Container *fyne.Container
}

// NewStepChecklist creates a checklist with all steps in pending state.
func NewStepChecklist(steps []string) *StepChecklist {
	sc := &StepChecklist{
		icons:   make([]*canvas.Text, len(steps)),
		names:   make([]*canvas.Text, len(steps)),
		details: make([]*canvas.Text, len(steps)),
		steps:   steps,
	}

	rows := make([]fyne.CanvasObject, 0, len(steps)*2)
	for i, name := range steps {
		icon := canvas.NewText(" ○ ", rocqLightDisabled)
		icon.TextSize = 13

		label := canvas.NewText(name, rocqLightDisabled)
		label.TextSize = 13

		detail := canvas.NewText("", rocqLightDisabled)
		detail.TextSize = 11
		detail.Hide()

		sc.icons[i] = icon
		sc.names[i] = label
		sc.details[i] = detail

		titleRow := container.NewHBox(icon, label)
		detailRow := container.NewHBox(spacer(28), detail)

		rows = append(rows, titleRow, detailRow)
	}

	sc.Container = container.New(&compactVBoxLayout{spacing: 1}, rows...)
	sc.Container.Hide()
	return sc
}

// spacer returns an invisible fixed-width rectangle for indentation.
func spacer(width float32) fyne.CanvasObject {
	r := canvas.NewRectangle(nil)
	r.SetMinSize(fyne.NewSize(width, 0))
	return r
}

// SetInProgress marks the given step (1-indexed) as in progress and all
// previous steps as done.
func (sc *StepChecklist) SetInProgress(step int, detail string) {
	if step < 1 || step > len(sc.steps) {
		return
	}
	for i := 0; i < step-1; i++ {
		sc.markDone(i)
	}
	idx := step - 1
	sc.icons[idx].Text = " ▶ "
	sc.icons[idx].Color = RocqOrange
	sc.icons[idx].Refresh()
	sc.names[idx].Color = rocqLightFg
	sc.names[idx].TextStyle = fyne.TextStyle{Bold: true}
	sc.names[idx].Refresh()
	sc.setDetail(idx, detail)
}

// UpdateDetail updates the detail text for the given step (1-indexed)
// without changing its status.
func (sc *StepChecklist) UpdateDetail(step int, detail string) {
	if step < 1 || step > len(sc.steps) {
		return
	}
	sc.setDetail(step-1, detail)
}

// SetDone marks the given step (1-indexed) as done.
func (sc *StepChecklist) SetDone(step int, detail string) {
	if step < 1 || step > len(sc.steps) {
		return
	}
	idx := step - 1
	sc.markDone(idx)
	if detail != "" {
		sc.setDetail(idx, detail)
	}
}

func (sc *StepChecklist) markDone(idx int) {
	sc.icons[idx].Text = " ✓ "
	sc.icons[idx].Color = rocqSuccess
	sc.icons[idx].Refresh()
	sc.names[idx].Color = rocqLightFg
	sc.names[idx].TextStyle = fyne.TextStyle{}
	sc.names[idx].Refresh()
}

func (sc *StepChecklist) setDetail(idx int, text string) {
	if text == "" {
		sc.details[idx].Hide()
		return
	}
	sc.details[idx].Text = text
	sc.details[idx].Refresh()
	sc.details[idx].Show()
}

// AppendSummary adds a summary line below the checklist steps.
func (sc *StepChecklist) AppendSummary(text string) {
	line := canvas.NewText(text, rocqLightFg)
	line.TextSize = 12
	sc.Container.Add(line)
}

// Show makes the checklist visible.
func (sc *StepChecklist) Show() {
	sc.Container.Show()
}

// Hide makes the checklist invisible.
func (sc *StepChecklist) Hide() {
	sc.Container.Hide()
}

// compactVBoxLayout is a vertical box layout with configurable spacing
// (instead of the default theme padding used by layout.VBoxLayout).
type compactVBoxLayout struct {
	spacing float32
}

func (l *compactVBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var w, h float32
	visible := 0
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		min := o.MinSize()
		if min.Width > w {
			w = min.Width
		}
		h += min.Height
		visible++
	}
	if visible > 1 {
		h += l.spacing * float32(visible-1)
	}
	return fyne.NewSize(w, h)
}

func (l *compactVBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	var y float32
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		min := o.MinSize()
		o.Move(fyne.NewPos(0, y))
		o.Resize(fyne.NewSize(size.Width, min.Height))
		y += min.Height + l.spacing
	}
}

var _ fyne.Layout = (*compactVBoxLayout)(nil)
