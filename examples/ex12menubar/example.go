package ex12menubar

import (
	"strings"

	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows the main menu of a form. Its menus are context menus
// (items, separators, submenus, icons) that drop down from the titles; while
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
	})
	file.AddItem("Open...", func() { setStatus("Open") })
	recent := ui.NewContextMenu(nil)
	for _, name := range []string{"report.txt", "notes.md", "todo.txt"} {
		recent.AddItem(name, func() { setStatus("Open recent: " + name) })
	}
	file.AddItemWithSubmenu("Open Recent", recent)
	file.AddItem("Save", func() { setStatus("Save") })
	file.AddSeparator()
	file.AddItem("Exit", func() { form.Close() })

	edit := bar.AddMenu("Edit")
	edit.AddItem("Upper Case", func() {
		text.SetText(strings.ToUpper(text.Text()))
		setStatus("Upper Case")
	})
	edit.AddItem("Lower Case", func() {
		text.SetText(strings.ToLower(text.Text()))
		setStatus("Lower Case")
	})
	edit.AddSeparator()
	edit.AddItem("Clear", func() {
		text.SetText("")
		setStatus("Clear")
	})

	view := bar.AddMenu("View")
	view.AddItem("Light Theme", func() { ui.ApplyLightTheme() })
	view.AddItem("Dark Theme", func() { ui.ApplyDarkTheme() })

	help := bar.AddMenu("Help")
	help.AddItem("About", func() { setStatus("nui main menu example") })

	form.SetMenuBar(bar)
	return form
}
