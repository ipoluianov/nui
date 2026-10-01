package ex12menubar

import (
	"image"
	"strings"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows the main menu of a form. Its menus are context menus
// (items with icons, separators, submenus) that drop down from the titles; while
// one is open, moving the mouse along the bar switches between them. A click
// on the bar keeps the focus, so the Edit items act on the text box.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Main Menu")
	form.SetSize(520, 320)

	panel := form.Panel()
	status := panel.AddLabel(0, 0, "Choose a menu item")
	setStatus := func(text string) { status.SetText(text) }

	text := ui.NewTextBox()
	text.SetMultiline(true)
	text.SetText("Click here, then use the Edit menu")
	text.SetYExpandable(true)
	panel.AddWidget(1, 0, text)

	bar := ui.NewMenuBar()

	file := bar.AddMenu("File")
	file.AddItem("New", func() {
		text.SetText("")
		setStatus("New")
	}).SetImage(iconNew())
	file.AddItem("Open...", func() { setStatus("Open") }).SetImage(iconFolder())
	recent := ui.NewContextMenu(nil)
	for _, name := range []string{"report.txt", "notes.md", "todo.txt"} {
		recent.AddItem(name, func() { setStatus("Open recent: " + name) })
	}
	// An item without an icon: its text lines up with the others
	file.AddItemWithSubmenu("Open Recent", recent)
	file.AddItem("Save", func() { setStatus("Save") }).SetImage(iconSave())
	file.AddSeparator()
	file.AddItem("Exit", func() { form.Close() }).SetImage(iconCross())

	edit := bar.AddMenu("Edit")
	edit.AddItem("Upper Case", func() {
		text.SetText(strings.ToUpper(text.Text()))
		setStatus("Upper Case")
	}).SetImage(iconArrow(true))
	edit.AddItem("Lower Case", func() {
		text.SetText(strings.ToLower(text.Text()))
		setStatus("Lower Case")
	}).SetImage(iconArrow(false))
	edit.AddSeparator()
	edit.AddItem("Clear", func() {
		text.SetText("")
		setStatus("Clear")
	}).SetImage(iconCross())

	view := bar.AddMenu("View")
	view.AddItem("Light Theme", func() { ui.ApplyLightTheme() }).SetImage(iconSun())
	view.AddItem("Dark Theme", func() { ui.ApplyDarkTheme() }).SetImage(iconMoon())

	help := bar.AddMenu("Help")
	help.AddItem("About", func() { setStatus("nui main menu example") }).SetImage(iconInfo())

	form.SetMenuBar(bar)
	return form
}

// The icons are drawn here so the example needs no image files; an
// application would load its own (e.g. PNGs with image.Decode).

const iconSize = 16

func drawIcon(draw func(dc *gg.Context)) image.Image {
	dc := gg.NewContext(iconSize, iconSize)
	draw(dc)
	return dc.Image()
}

// iconNew: a sheet of paper with a folded corner
func iconNew() image.Image {
	return drawIcon(func(dc *gg.Context) {
		dc.MoveTo(3, 1.5)
		dc.LineTo(10, 1.5)
		dc.LineTo(13, 4.5)
		dc.LineTo(13, 14.5)
		dc.LineTo(3, 14.5)
		dc.ClosePath()
		dc.SetHexColor("#F5F5F5")
		dc.FillPreserve()
		dc.SetHexColor("#757575")
		dc.SetLineWidth(1)
		dc.Stroke()
		dc.MoveTo(10, 1.5)
		dc.LineTo(10, 4.5)
		dc.LineTo(13, 4.5)
		dc.Stroke()
	})
}

// iconFolder: a yellow folder
func iconFolder() image.Image {
	return drawIcon(func(dc *gg.Context) {
		dc.SetHexColor("#E0A526")
		dc.DrawRoundedRectangle(1, 3, 6, 3, 1)
		dc.Fill()
		dc.SetHexColor("#FFC940")
		dc.DrawRoundedRectangle(1, 5, 14, 9, 1.5)
		dc.Fill()
	})
}

// iconSave: a floppy disk
func iconSave() image.Image {
	return drawIcon(func(dc *gg.Context) {
		dc.SetHexColor("#1D6FB5")
		dc.DrawRoundedRectangle(1.5, 1.5, 13, 13, 1.5)
		dc.Fill()
		dc.SetHexColor("#FFFFFF")
		dc.DrawRectangle(4, 2, 8, 5)
		dc.Fill()
		dc.SetHexColor("#CFD8DC")
		dc.DrawRectangle(4, 9.5, 8, 4)
		dc.Fill()
	})
}

// iconCross: a red cross
func iconCross() image.Image {
	return drawIcon(func(dc *gg.Context) {
		dc.SetHexColor("#E53935")
		dc.SetLineWidth(2.5)
		dc.SetLineCapRound()
		dc.DrawLine(4, 4, 12, 12)
		dc.DrawLine(12, 4, 4, 12)
		dc.Stroke()
	})
}

// iconArrow: a green arrow up or down
func iconArrow(up bool) image.Image {
	return drawIcon(func(dc *gg.Context) {
		if !up {
			dc.RotateAbout(gg.Radians(180), iconSize/2, iconSize/2)
		}
		dc.SetHexColor("#43A047")
		dc.MoveTo(8, 1.5)
		dc.LineTo(14, 8)
		dc.LineTo(10.5, 8)
		dc.LineTo(10.5, 14.5)
		dc.LineTo(5.5, 14.5)
		dc.LineTo(5.5, 8)
		dc.LineTo(2, 8)
		dc.ClosePath()
		dc.Fill()
	})
}

// iconSun: a sun with rays
func iconSun() image.Image {
	return drawIcon(func(dc *gg.Context) {
		dc.SetHexColor("#FFB300")
		dc.DrawCircle(8, 8, 3.5)
		dc.Fill()
		dc.SetLineWidth(1.5)
		dc.SetLineCapRound()
		for i := 0; i < 8; i++ {
			a := gg.Radians(float64(i) * 45)
			dc.Push()
			dc.RotateAbout(a, 8, 8)
			dc.DrawLine(8, 1.5, 8, 2.8)
			dc.Stroke()
			dc.Pop()
		}
	})
}

// iconMoon: a crescent moon
func iconMoon() image.Image {
	return drawIcon(func(dc *gg.Context) {
		// A circle without the part a shifted circle covers
		dc.DrawCircle(11, 5, 5.5)
		dc.Clip()
		dc.InvertMask()
		dc.SetHexColor("#7E57C2")
		dc.DrawCircle(8, 8, 6.5)
		dc.Fill()
	})
}

// iconInfo: a blue circle with an "i"
func iconInfo() image.Image {
	return drawIcon(func(dc *gg.Context) {
		dc.SetHexColor("#1D6FB5")
		dc.DrawCircle(8, 8, 7)
		dc.Fill()
		dc.SetHexColor("#FFFFFF")
		dc.DrawCircle(8, 4.5, 1.3)
		dc.Fill()
		dc.DrawRoundedRectangle(7, 6.8, 2, 6, 0.8)
		dc.Fill()
	})
}
