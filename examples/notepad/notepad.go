// Package notepad is a primitive text editor: one text file at a time, with
// the usual File, Edit, View and Help menus.
package notepad

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

const (
	appName         = "Notepad"
	defaultFontSize = 14.0
	minFontSize     = 8.0
	maxFontSize     = 40.0
	toolIconSize    = 18
	toolButtonSize  = 30
)

const welcomeText = `Welcome to Notepad!

This is a small text editor built with nui. Try it out:

  - File > Open (Ctrl+O), or drop a text file onto the window
  - Edit > Find (Ctrl+F), then F3 to find the next match
  - Edit > Go To Line (Ctrl+G)
  - View > Bigger / Smaller (Ctrl+= / Ctrl+-) changes the font size
  - F5 inserts the current date and time

Changes are not lost by accident: New, Open and closing the window
ask to save them first.
`

// notepad is the state of the editor window
type notepad struct {
	form *ui.Form
	text *ui.TextBox

	lblFile     *ui.Label
	lblModified *ui.Label
	lblCounts   *ui.Label

	path     string // "" for a new, unsaved document
	modified bool
	loading  bool // SetText from the code: not an edit of the user

	recent     []string
	recentMenu *ui.ContextMenu

	fontSize float64

	// The last search, for Find Next (F3)
	findText  string
	matchCase bool
	findPos   int // the rune offset to search from
}

// NewForm creates the Notepad window: a primitive text editor that opens and
// saves real files. It shows the main menu (mnemonics, shortcuts, icons, a
// submenu of recent files that changes), a toolbar of flat tool buttons with
// tooltips, a multiline text box with undo, the clipboard, file dialogs,
// dropping files on the window, printing and PDF export, question message
// boxes guarding unsaved changes, dialogs built on ui.DialogContent (Find, Go
// To Line), toasts and a status bar.
func NewForm() *ui.Form {
	return newNotepad().form
}

func newNotepad() *notepad {
	n := &notepad{fontSize: defaultFontSize}
	n.form = ui.NewForm()
	n.form.SetSize(800, 600)

	panel := n.form.Panel()
	panel.SetPanelPadding(4)
	panel.SetCellPadding(4)

	panel.AddWidget(0, 0, n.buildToolbar())

	n.text = ui.NewTextBox()
	n.text.SetMultiline(true)
	n.text.SetFontFamily(ui.FontFamilyMono)
	n.text.SetFontSize(n.fontSize)
	n.text.SetOnTextChanged(func() {
		if !n.loading {
			n.setModified(true)
		}
		n.updateStatus()
	})
	panel.AddWidget(1, 0, n.text)

	panel.AddWidget(2, 0, n.buildStatusBar())

	n.form.SetMenuBar(n.buildMenu())
	n.form.AddShortcut("F3", n.findNext)

	n.form.SetOnFilesDropped(func(files []string, x, y int) {
		if len(files) > 0 {
			n.confirmDiscard(func() { n.openFile(files[0]) })
		}
	})
	n.form.OnClose = func() bool {
		if !n.modified {
			return true
		}
		n.confirmDiscard(n.form.Close)
		return false
	}

	n.setDocument("", welcomeText)
	n.text.Focus()
	return n
}

func (n *notepad) buildMenu() *ui.MenuBar {
	bar := ui.NewMenuBar()

	file := bar.AddMenu("&File")
	file.AddItem("&New", n.newDocument).SetImage(icons.New(icons.Size)).SetShortcut("Mod+N")
	file.AddItem("&Open...", n.open).SetImage(icons.Folder(icons.Size)).SetShortcut("Mod+O")
	n.recentMenu = ui.NewContextMenu(nil)
	file.AddItemWithSubmenu("Open &Recent", n.recentMenu)
	n.updateRecentMenu()
	file.AddItem("&Save", func() { n.save(nil) }).SetImage(icons.Save(icons.Size)).SetShortcut("Mod+S")
	file.AddItem("Save &As...", func() { n.saveAs(nil) }).SetShortcut("Mod+Shift+S")
	file.AddSeparator()
	file.AddItem("&Print...", n.print).SetImage(icons.Printer(icons.Size)).SetShortcut("Mod+P")
	file.AddItem("&Export PDF...", n.exportPDF)
	file.AddSeparator()
	file.AddItem("E&xit", func() { n.form.RequestClose() }).SetImage(icons.Cross(icons.Size)).SetShortcut("Alt+F4")

	// Undo, the clipboard and Select All are also keys of the text box
	// itself; the menu items do the same from anywhere
	edit := bar.AddMenu("&Edit")
	edit.AddItem("&Undo", n.editAction(n.text.Undo)).SetImage(icons.Undo(icons.Size)).SetShortcut("Mod+Z")
	edit.AddItem("&Redo", n.editAction(n.text.Redo)).SetImage(icons.Redo(icons.Size)).SetShortcut("Mod+Y")
	edit.AddSeparator()
	edit.AddItem("Cu&t", n.editAction(func() { n.textKey(ui.KeyX) })).SetShortcut("Mod+X")
	edit.AddItem("&Copy", n.editAction(func() { n.textKey(ui.KeyC) })).SetShortcut("Mod+C")
	edit.AddItem("&Paste", n.editAction(func() { n.textKey(ui.KeyV) })).SetShortcut("Mod+V")
	edit.AddItem("Select &All", n.editAction(n.text.SelectAllText)).SetShortcut("Mod+A")
	edit.AddSeparator()
	edit.AddItem("&Find...", n.showFind).SetImage(icons.Search(icons.Size)).SetShortcut("Mod+F")
	edit.AddItem("Find &Next", n.findNext).SetShortcut("F3")
	edit.AddItem("&Go To Line...", n.showGoToLine).SetShortcut("Mod+G")
	edit.AddSeparator()
	edit.AddItem("U&pper Case", n.editAction(func() { n.changeCase(strings.ToUpper) })).
		SetImage(icons.Arrow(icons.Size, true)).SetShortcut("Mod+Shift+U")
	edit.AddItem("Lo&wer Case", n.editAction(func() { n.changeCase(strings.ToLower) })).
		SetImage(icons.Arrow(icons.Size, false)).SetShortcut("Mod+Shift+L")
	edit.AddItem("Insert &Date/Time", n.editAction(n.insertDateTime)).SetShortcut("F5")

	view := bar.AddMenu("&View")
	view.AddItem("&Bigger Font", func() { n.setFontSize(n.fontSize + 2) }).
		SetImage(icons.Plus(icons.Size)).SetShortcut("Mod+Plus")
	view.AddItem("&Smaller Font", func() { n.setFontSize(n.fontSize - 2) }).SetShortcut("Mod+Minus")
	view.AddItem("&Default Font Size", func() { n.setFontSize(defaultFontSize) }).SetShortcut("Mod+0")
	view.AddSeparator()
	view.AddItem("&Light Theme", ui.ApplyLightTheme).SetImage(icons.Sun(icons.Size))
	view.AddItem("Dar&k Theme", ui.ApplyDarkTheme).SetImage(icons.Moon(icons.Size))

	help := bar.AddMenu("&Help")
	help.AddItem("&About Notepad", func() {
		ui.ShowMessageBox(n.form.Panel(), "About Notepad",
			"Notepad 1.0\n\nA primitive text editor, an example application of the nui library.")
	}).SetImage(icons.Info(icons.Size)).SetShortcut("F1")

	return bar
}

func (n *notepad) buildToolbar() *ui.Panel {
	toolbar := ui.NewPanel()
	toolbar.SetPanelPadding(0)
	toolbar.SetCellPadding(2)
	col := 0
	tool := func(tooltip string, onClick func(), icon func(size int) image.Image) {
		btn := ui.NewToolButton(icon(toolIconSize), tooltip, onClick)
		btn.SetFlat(true)
		btn.SetButtonSize(toolButtonSize, toolButtonSize)
		toolbar.AddWidget(0, col, btn)
		col++
	}
	separator := func() {
		gap := ui.NewSpace()
		gap.SetSize(10, 1)
		toolbar.AddWidget(0, col, gap)
		col++
	}

	tool("New (Ctrl+N)", n.newDocument, icons.New)
	tool("Open (Ctrl+O)", n.open, icons.Folder)
	tool("Save (Ctrl+S)", func() { n.save(nil) }, icons.Save)
	separator()
	tool("Undo (Ctrl+Z)", n.editAction(n.text.Undo), icons.Undo)
	tool("Redo (Ctrl+Y)", n.editAction(n.text.Redo), icons.Redo)
	separator()
	tool("Find (Ctrl+F)", n.showFind, icons.Search)
	toolbar.AddHSpacer(0, col)
	return toolbar
}

func (n *notepad) buildStatusBar() *ui.Panel {
	status := ui.NewPanel()
	status.SetPanelPadding(0)
	status.SetCellPadding(12)
	n.lblFile = status.AddLabel(0, 0, "")
	n.lblModified = status.AddLabel(0, 1, "")
	status.AddHSpacer(0, 2)
	n.lblCounts = status.AddLabel(0, 3, "")
	return status
}

// editAction runs an Edit command on the text box and gives the focus back
// to it (a click on a tool button takes it)
func (n *notepad) editAction(f func()) func() {
	return func() {
		f()
		n.text.Focus()
	}
}

// textKey presses Ctrl+key in the text box: Cut, Copy and Paste of the text
// box are its keys
func (n *notepad) textKey(key ui.Key) {
	n.text.KeyDown(key, ui.KeyModifiers{Ctrl: true})
}

func (n *notepad) setFontSize(size float64) {
	n.fontSize = max(minFontSize, min(maxFontSize, size))
	n.text.SetFontSize(n.fontSize)
	n.updateStatus()
}

func (n *notepad) fileName() string {
	if n.path == "" {
		return "Untitled"
	}
	return filepath.Base(n.path)
}

func (n *notepad) setModified(modified bool) {
	if n.modified == modified {
		return
	}
	n.modified = modified
	n.updateTitle()
	n.updateStatus()
}

func (n *notepad) updateTitle() {
	mark := ""
	if n.modified {
		mark = "*"
	}
	n.form.SetTitle(n.fileName() + mark + " — " + appName)
}

func (n *notepad) updateStatus() {
	if n.lblFile == nil {
		return
	}
	n.lblFile.SetText(n.fileName())
	if n.modified {
		n.lblModified.SetText("Modified")
	} else {
		n.lblModified.SetText("")
	}
	text := n.text.Text()
	lines := strings.Count(text, "\n") + 1
	n.lblCounts.SetText(fmt.Sprintf("Lines: %d    Words: %d    Characters: %d    Font: %.0f pt",
		lines, len(strings.Fields(text)), utf8.RuneCountInString(text), n.fontSize))
}
