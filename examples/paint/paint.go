// Package paint is a tiny raster paint program.
package paint

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

const (
	docWidth  = 800
	docHeight = 600
)

type app struct {
	form   *ui.Form
	canvas *paintCanvas
	path   string // the file of the document, "" - not saved yet

	toolButtons        []*ui.ToolButton
	primary, secondary *ui.ColorPicker
	status             *ui.Label
	position           *ui.Label
}

// NewForm shows a paint program: a custom widget that draws on an image
// with the mouse, a toolbar of checkable flat tool buttons with tooltips and
// shortcut letters, color pickers, a palette in a custom popup, a slider,
// a check box, the main menu with file dialogs, undo and redo, and a status
// bar.
func NewForm() *ui.Form {
	a := &app{form: ui.NewForm()}
	a.form.SetSize(1100, 760)
	a.canvas = newPaintCanvas(docWidth, docHeight)
	a.canvas.onChanged = a.updateTitle
	a.canvas.onMouseMoved = func(p image.Point, in bool) {
		if in {
			a.position.SetText(fmt.Sprintf("%d, %d px", p.X, p.Y))
		} else {
			a.position.SetText("")
		}
	}
	a.canvas.onPicked = a.setColor

	panel := a.form.Panel()
	panel.SetPanelPadding(0)
	main := panel.AddPanel(0, 0)
	main.SetPanelPadding(0)
	main.AddWidget(0, 0, a.makeToolbar())
	scroll := ui.NewScrollArea()
	scroll.AddWidget(0, 0, a.canvas)
	main.AddWidget(0, 1, scroll)
	main.AddWidget(0, 2, a.makeSidePanel())

	statusBar := panel.AddPanel(1, 0)
	statusBar.SetPanelPadding(4)
	a.position = statusBar.AddLabel(0, 0, "")
	a.position.SetMinWidth(110)
	statusBar.AddHSpacer(0, 1)
	a.status = statusBar.AddLabel(0, 2, "")

	a.form.SetMenuBar(a.makeMenu())
	a.form.OnClose = func() bool {
		if !a.canvas.modified {
			return true
		}
		a.confirmDiscard(func() { a.form.Close() })
		return false
	}

	a.selectTool(toolPencil)
	a.updateTitle()
	return a.form
}

func (a *app) makeToolbar() ui.Widgeter {
	bar := ui.NewPanel()
	bar.SetPanelPadding(4)
	for i, info := range tools {
		t := tool(i)
		btn := ui.NewToolButton(info.icon(toolIconSize), info.name+" ("+info.shortcut+")", func() { a.selectTool(t) })
		btn.SetFlat(true)
		btn.SetButtonSize(40, 40)
		bar.AddWidget(i, 0, btn)
		a.toolButtons = append(a.toolButtons, btn)
		a.form.AddShortcut(info.shortcut, func() { a.selectTool(t) })
	}
	bar.AddVSpacer(len(tools), 0)
	return bar
}

func (a *app) makeSidePanel() ui.Widgeter {
	side := ui.NewPanel()
	side.SetPanelPadding(8)
	side.SetCellPadding(6)
	side.SetMinWidth(210)
	side.SetMaxWidth(210)

	side.AddLabel(0, 0, "Primary")
	a.primary = ui.NewColorPicker()
	a.primary.SetColor(a.canvas.primary)
	a.primary.SetOnColorChanged(func(col color.RGBA) { a.canvas.primary = col })
	side.AddWidget(1, 0, a.primary)

	side.AddLabel(2, 0, "Secondary")
	a.secondary = ui.NewColorPicker()
	a.secondary.SetTooltip("Right click with Fill or Color Picker; the eraser paints with it")
	a.secondary.SetColor(a.canvas.secondary)
	a.secondary.SetOnColorChanged(func(col color.RGBA) { a.canvas.secondary = col })
	side.AddWidget(3, 0, a.secondary)

	var paletteButton *ui.Button
	paletteButton = side.AddButton(4, 0, "Quick colors...", func() {
		newPalette(a.setColor).openBelow(paletteButton)
	})

	sizeRow := side.AddPanel(5, 0)
	sizeRow.SetPanelPadding(0)
	sizeRow.AddLabel(0, 0, "Size")
	sizeRow.AddHSpacer(0, 1)
	sizeLabel := sizeRow.AddLabel(0, 2, "")
	size := ui.NewSlider()
	size.SetRange(1, 50)
	size.SetStep(1)
	size.SetValue(float64(a.canvas.size))
	showSize := func() {
		a.canvas.size = int(size.Value())
		sizeLabel.SetText(fmt.Sprintf("%d px", a.canvas.size))
	}
	size.SetOnValueChanged(showSize)
	showSize()
	side.AddWidget(6, 0, size)

	fill := ui.NewCheckbox("Fill shapes")
	fill.SetOnStateChanged(func() { a.canvas.fillShapes = fill.Checked() })
	side.AddWidget(7, 0, fill)

	side.AddVSpacer(8, 0)
	return side
}

func (a *app) makeMenu() *ui.MenuBar {
	bar := ui.NewMenuBar()

	file := bar.AddMenu("&File")
	file.AddItem("&New", func() { a.confirmDiscard(a.newDocument) }).SetImage(icons.New(icons.Size)).SetShortcut("Mod+N")
	file.AddItem("&Open...", func() { a.confirmDiscard(a.open) }).SetImage(icons.Folder(icons.Size)).SetShortcut("Mod+O")
	file.AddItem("&Save", func() { a.save() }).SetImage(icons.Save(icons.Size)).SetShortcut("Mod+S")
	file.AddItem("Save &As...", func() { a.saveAs() })
	file.AddSeparator()
	file.AddItem("E&xit", func() { a.form.RequestClose() }).SetImage(icons.Cross(icons.Size))

	edit := bar.AddMenu("&Edit")
	edit.AddItem("&Undo", a.canvas.Undo).SetImage(icons.Undo(icons.Size)).SetShortcut("Mod+Z")
	edit.AddItem("&Redo", a.canvas.Redo).SetImage(icons.Redo(icons.Size)).SetShortcut("Mod+Y")
	edit.AddSeparator()
	edit.AddItem("&Clear", func() {
		a.canvas.edit(func(img *image.RGBA) {
			b := img.Bounds()
			copy(img.Pix, newDocument(b.Dx(), b.Dy()).Pix)
		})
	})

	img := bar.AddMenu("&Image")
	img.AddItem("&Invert Colors", func() { a.canvas.edit(invertColors) })
	img.AddItem("&Flip Horizontal", func() { a.canvas.edit(flipHorizontal) })

	return bar
}

func (a *app) selectTool(t tool) {
	a.canvas.tool = t
	for i, btn := range a.toolButtons {
		btn.SetChecked(tool(i) == t)
	}
	a.updateStatus()
}

// setColor sets the primary or the secondary color: from the palette or
// the eyedropper.
func (a *app) setColor(col color.RGBA, primary bool) {
	if primary {
		a.canvas.primary = col
		a.primary.SetColor(col)
	} else {
		a.canvas.secondary = col
		a.secondary.SetColor(col)
	}
}

func (a *app) updateTitle() {
	name := "Untitled"
	if a.path != "" {
		name = filepath.Base(a.path)
	}
	if a.canvas.modified {
		name += " *"
	}
	a.form.SetTitle(name + " - Paint")
	a.updateStatus()
}

func (a *app) updateStatus() {
	if a.status == nil {
		return
	}
	b := a.canvas.doc.Bounds()
	a.status.SetText(fmt.Sprintf("%d x %d px    %s", b.Dx(), b.Dy(), a.canvas.tool))
}

// confirmDiscard runs then at once if the document has no unsaved changes,
// otherwise after the user agreed to lose them.
func (a *app) confirmDiscard(then func()) {
	if !a.canvas.modified {
		then()
		return
	}
	ui.ShowQuestionMessageBoxYesNo(a.canvas, "Paint", "The picture has unsaved changes. Discard them?",
		func() {
			a.canvas.modified = false
			then()
		}, nil)
}

func (a *app) newDocument() {
	a.path = ""
	a.canvas.setDocument(newDocument(docWidth, docHeight))
}

var imageFilters = []ui.FileDialogFilter{
	{DisplayName: "Images", Patterns: []string{"*.png", "*.jpg", "*.jpeg"}},
}

func (a *app) open() {
	a.form.ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: "Open Picture", Filters: imageFilters},
		func(paths []string, err error) {
			if err != nil || len(paths) == 0 {
				return
			}
			img, err := loadImage(paths[0])
			if err != nil {
				ui.ShowMessageBox(a.canvas, "Open", err.Error())
				return
			}
			a.path = paths[0]
			a.canvas.setDocument(img)
		})
}

// save saves to the document's file, asking for one the first time.
func (a *app) save() {
	if a.path == "" {
		a.saveAs()
		return
	}
	if err := savePNG(a.path, a.canvas.doc); err != nil {
		ui.ShowMessageBox(a.canvas, "Save", err.Error())
		return
	}
	a.canvas.modified = false
	a.updateTitle()
	a.form.ShowToast("Saved "+filepath.Base(a.path), ui.ToastInfo)
}

func (a *app) saveAs() {
	opts := ui.SaveFileDialogOptions{
		Title:           "Save Picture",
		DefaultFileName: "picture.png",
		Filters:         []ui.FileDialogFilter{{DisplayName: "PNG", Patterns: []string{"*.png"}}},
	}
	a.form.ShowSaveFileDialog(opts, func(path string, err error) {
		if err != nil || path == "" {
			return
		}
		if filepath.Ext(path) == "" {
			path += ".png"
		}
		a.path = path
		a.save()
	})
}

func loadImage(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return toRGBA(img), nil
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
