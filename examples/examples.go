// Package examples is a set of small applications built with the ui
// package: each shows the widgets the way a real program uses them. Run
// them all from the launcher (go run .) or one by name (go run . notepad).
package examples

import (
	"image"
	"math"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/calculator"
	"github.com/ipoluianov/nui/examples/contacts"
	"github.com/ipoluianov/nui/examples/converter"
	"github.com/ipoluianov/nui/examples/files"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/examples/monitor"
	"github.com/ipoluianov/nui/examples/notepad"
	"github.com/ipoluianov/nui/examples/paint"
	"github.com/ipoluianov/nui/examples/plotter"
	"github.com/ipoluianov/nui/examples/tasks"
	"github.com/ipoluianov/nui/ui"
)

// App is one example application.
type App struct {
	Name        string // used on the command line: go run . <name>
	Title       string
	Description string
	Icon        image.Image
	NewForm     func() *ui.Form
}

const appIconSize = 40

// Apps are the example applications, in the launcher's order.
var Apps = []App{
	{"notepad", "Notepad",
		"Text editor: main menu, toolbar, files, find, printing",
		appIcon("#1E88E5", markLines), notepad.NewForm},
	{"calculator", "Calculator",
		"Button grid, keyboard input, history, clipboard",
		appIcon("#546E7A", markCalculator), calculator.NewForm},
	{"contacts", "Contacts",
		"Table with search and sorting, edit dialog, details pane",
		appIcon("#43A047", markPerson), contacts.NewForm},
	{"files", "Files",
		"File explorer: folder tree, file list, image and text preview",
		appIcon("#FFA000", markFolder), files.NewForm},
	{"paint", "Paint",
		"Drawing on a custom widget, tools, colors, undo",
		appIcon("#E53935", markBrush), paint.NewForm},
	{"tasks", "Tasks",
		"To-do planner: calendar, property grid, reminders, tray icon",
		appIcon("#8E24AA", markCheck), tasks.NewForm},
	{"monitor", "Network Monitor",
		"Live time charts of simulated servers, alerts",
		appIcon("#00897B", markPulse), monitor.NewForm},
	{"converter", "Unit Converter",
		"Translated into English, Russian and Chinese, switched live",
		appIcon("#3949AB", markArrows), converter.NewForm},
	{"plotter", "Function Plotter",
		"Formula input, sliders, chart, animation",
		appIcon("#F4511E", markWave), plotter.NewForm},
}

// FindApp returns the application with the name, or nil.
func FindApp(name string) *App {
	for i := range Apps {
		if Apps[i].Name == name {
			return &Apps[i]
		}
	}
	return nil
}

// RunApp runs one application as the main window.
func RunApp(app *App) {
	form := app.NewForm()
	form.Show()
	form.Exec()
}

// Run runs the launcher: a window with the list of the applications. Each
// opens in its own window; several can be open at once.
func Run() {
	form := ui.NewForm()
	form.SetTitle("nui examples")
	form.SetSize(640, 720)
	panel := form.Panel()

	list := panel.AddPanel(0, 0)
	list.SetCellPadding(6)
	for i, app := range Apps {
		open := func() { app.NewForm().Show() }
		btn := ui.NewToolButton(app.Icon, "Open "+app.Title, open)
		btn.SetFlat(true)
		list.AddWidget(i, 0, btn)

		text := list.AddPanel(i, 1)
		text.SetPanelPadding(0)
		title := ui.NewLink(app.Title, open)
		text.AddWidget(0, 0, title)
		text.AddLabel(1, 0, app.Description)
		text.AddHSpacer(0, 1)
	}
	panel.AddVSpacer(1, 0)

	settings := panel.AddPanel(2, 0)
	dark := ui.NewToggleSwitch("Dark theme")
	dark.SetChecked(ui.IsDarkTheme)
	dark.SetOnStateChanged(func() {
		if dark.Checked() {
			ui.ApplyDarkTheme()
		} else {
			ui.ApplyLightTheme()
		}
	})
	settings.AddWidget(0, 0, dark)

	font := ui.NewComboBox()
	font.SetMinWidth(280)
	font.AddItem("Built-in font (Noto Sans)", nil)
	systemFont := ui.SystemUIFontName()
	font.AddItem("System font ("+systemFont+")", nil)
	font.SetSelectedIndex(0)
	font.SetOnSelectedIndexChanged(func() {
		if font.SelectedIndex() == 0 {
			ui.UseBuiltinFont()
			return
		}
		if err := ui.UseSystemFont(systemFont); err != nil {
			form.ShowToast(err.Error(), ui.ToastError)
		}
	})
	settings.AddWidget(0, 1, font)
	settings.AddHSpacer(0, 2)

	form.Show()
	form.Exec()
}

// The application icons: a colored rounded square with a white mark.

func appIcon(color string, mark func(dc *gg.Context)) image.Image {
	return icons.App(appIconSize, color, mark)
}

func markLines(dc *gg.Context) {
	for _, y := range []float64{5, 8, 11} {
		dc.DrawLine(4, y, 12, y)
	}
	dc.Stroke()
}

func markCalculator(dc *gg.Context) {
	dc.DrawRectangle(4.5, 3.5, 7, 2.5)
	dc.Stroke()
	for _, y := range []float64{8.5, 11.5} {
		for _, x := range []float64{5, 8, 11} {
			dc.DrawCircle(x, y, 0.8)
			dc.Fill()
		}
	}
}

func markPerson(dc *gg.Context) {
	dc.DrawCircle(8, 6, 2.3)
	dc.Fill()
	dc.DrawArc(8, 13.5, 4, gg.Radians(180), gg.Radians(360))
	dc.Fill()
}

func markFolder(dc *gg.Context) {
	dc.DrawRoundedRectangle(3.5, 5, 9, 7, 1)
	dc.Stroke()
	dc.DrawLine(3.5, 7, 12.5, 7)
	dc.Stroke()
}

func markBrush(dc *gg.Context) {
	dc.SetLineWidth(2)
	dc.DrawLine(11.5, 4, 7, 9.5)
	dc.Stroke()
	dc.DrawCircle(5.5, 11, 2)
	dc.Fill()
}

func markCheck(dc *gg.Context) {
	dc.SetLineWidth(2)
	dc.MoveTo(4, 8.5)
	dc.LineTo(7, 11.5)
	dc.LineTo(12, 5)
	dc.Stroke()
}

func markPulse(dc *gg.Context) {
	dc.MoveTo(3, 9)
	dc.LineTo(6, 9)
	dc.LineTo(7.5, 4.5)
	dc.LineTo(9, 12)
	dc.LineTo(10.5, 8)
	dc.LineTo(13, 8)
	dc.Stroke()
}

func markArrows(dc *gg.Context) {
	dc.DrawLine(4, 6, 12, 6)
	dc.DrawLine(10, 4, 12, 6)
	dc.DrawLine(4, 10, 12, 10)
	dc.DrawLine(4, 10, 6, 12)
	dc.Stroke()
}

func markWave(dc *gg.Context) {
	for x := 3.0; x <= 13; x += 0.25 {
		dc.LineTo(x, 8-3*math.Sin((x-3)/10*2*math.Pi))
	}
	dc.Stroke()
}
