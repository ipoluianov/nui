// Package recipes holds the code of doc/recipes.md: common tasks of an
// application on nui, each a small piece that builds and is tested.
package recipes

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// Editor is the main window of a document application: a menu bar, a
// toolbar, a status bar, commands with shortcuts and states, a question
// before unsaved changes are lost, the window where the user left it.
type Editor struct {
	Form     *ui.Form
	text     *ui.TextBox
	status   *ui.StatusBar
	path     string
	modified bool
}

// NewEditor makes the window; settings keep its place between runs.
func NewEditor(settings *ui.Settings) *Editor {
	e := &Editor{Form: ui.NewForm()}
	e.Form.SetSize(800, 600)

	// Commands: one action in the menu, on the toolbar and on its shortcut
	newDoc := ui.NewAction("&New", func() { e.ask(e.clear) }).SetShortcut("Mod+N")
	open := ui.NewAction("&Open...", func() { e.ask(e.open) }).SetShortcut("Mod+O")
	save := ui.NewAction("&Save", e.save).SetShortcut("Mod+S").
		SetEnabledFunc(func() bool { return e.modified })
	quit := ui.NewAction("&Quit", func() { e.Form.RequestClose() })

	bar := ui.NewMenuBar()
	file := bar.AddMenu("&File")
	file.AddAction(newDoc)
	file.AddAction(open)
	file.AddAction(save)
	file.AddSeparator()
	file.AddAction(quit)
	e.Form.SetMenuBar(bar)

	tools := ui.NewToolBar()
	tools.AddAction(newDoc)
	tools.AddAction(open)
	tools.AddAction(save)

	e.text = ui.NewTextBox()
	e.text.SetMultiline(true)
	e.text.OnTextEdited.Connect(func(string) { e.setModified(true) })

	e.status = ui.NewStatusBar()

	p := e.Form.Panel()
	p.AddWidget(0, 0, tools)
	p.AddWidget(1, 0, e.text) // a multiline box takes the free room
	p.AddWidget(2, 0, e.status)

	// The close button, Quit and the system's quit ask first
	e.Form.OnClose = func() bool {
		if !e.modified {
			return true
		}
		e.ask(func() { e.modified = false; e.Form.Close() })
		return false
	}
	if settings != nil {
		e.Form.RememberState(settings, "editor")
	}
	e.clear()
	return e
}

// ask runs then after the user agreed to lose the changes, or saved them
func (e *Editor) ask(then func()) {
	if !e.modified {
		then()
		return
	}
	ui.ShowQuestionMessageBoxYesNoCancel(e.Form.Panel(), "Unsaved changes",
		"Save the changes to "+e.name()+"?",
		func() { // Yes
			if e.save(); !e.modified {
				then()
			}
		},
		then, // No
		nil)  // Cancel, Esc, the close button
}

func (e *Editor) clear() {
	e.path = ""
	e.text.SetText("")
	e.setModified(false)
}

func (e *Editor) open() {
	e.Form.ShowOpenFileDialog(ui.OpenFileDialogOptions{
		Filters: []ui.FileDialogFilter{{DisplayName: "Text files", Patterns: []string{"*.txt"}}},
	}, func(paths []string, err error) {
		if err != nil || len(paths) == 0 {
			return // cancelled
		}
		data, err := os.ReadFile(paths[0])
		if err != nil {
			ui.ShowMessageBox(e.Form.Panel(), "Open", err.Error())
			return
		}
		e.path = paths[0]
		e.text.SetText(string(data))
		e.setModified(false)
	})
}

func (e *Editor) save() {
	if e.path == "" {
		// The dialog answers later; this save is left unsaved meanwhile
		e.Form.ShowSaveFileDialog(ui.SaveFileDialogOptions{DefaultFileName: "untitled.txt"},
			func(path string, err error) {
				if err == nil && path != "" {
					e.path = path
					e.save()
				}
			})
		return
	}
	if err := os.WriteFile(e.path, []byte(e.text.Text()), 0o644); err != nil {
		ui.ShowMessageBox(e.Form.Panel(), "Save", err.Error())
		return
	}
	e.setModified(false)
	e.status.ShowMessage("Saved", 3*time.Second)
}

func (e *Editor) setModified(m bool) {
	e.modified = m
	title := e.name()
	if m {
		title += " *"
	}
	e.Form.SetTitle(title + " - Editor")
}

func (e *Editor) name() string {
	if e.path == "" {
		return "Untitled"
	}
	return filepath.Base(e.path)
}
