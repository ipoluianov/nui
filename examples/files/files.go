package files

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

// explorer is the state of the file explorer window.
type explorer struct {
	form *ui.Form

	dir        string   // the folder shown in the table
	history    []string // folders to go Back to
	showHidden bool

	entries  []entry // all the entries of dir
	rows     []entry // the entries shown: filtered and sorted
	sortCol  int
	sortDesc bool

	tree     *ui.TreeView
	syncTree bool // the tree selection is being set from code
	table    *ui.Table
	path     *ui.TextBox
	search   *ui.TextBox
	back     *ui.ToolButton

	preview previewPane

	status    *ui.Label
	selection *ui.Label
}

// NewForm is a read-only file explorer of the real file system: a folder
// tree loaded lazily, a sortable and searchable table of the folder's
// entries with icons and a context menu, and a preview pane for images and
// text files.
//
// It shows flat ToolButtons with tooltips, single-line TextBoxes reacting to
// Enter, Splitters, TreeView (SetHasChildren and SetOnExpand), Table (cell
// images, column clicks, double click), ImageBox, a read-only multiline
// TextBox, ContextMenu, clipboard, toasts, a message box and a Checkbox.
func NewForm() *ui.Form {
	e := &explorer{form: ui.NewForm()}
	e.form.SetTitle("Files")
	e.form.SetSize(1180, 720)

	panel := e.form.Panel()
	panel.AddWidget(0, 0, e.newToolbar())

	e.tree = e.newTree()
	e.table = e.newTable()
	right := ui.NewHSplitter()
	right.SetWidgets(e.table, e.preview.init())
	right.SetSecondSize(340)
	main := ui.NewHSplitter()
	main.SetWidgets(e.tree, right)
	main.SetFirstSize(240)
	panel.AddWidget(1, 0, main)

	panel.AddWidget(2, 0, e.newStatusBar())

	e.form.AddShortcut("Alt+Left", e.goBack)
	e.form.AddShortcut("Alt+Up", e.goUp)
	e.form.AddShortcut("F5", e.refresh)
	e.form.AddShortcut("Mod+L", func() {
		e.path.Focus()
		e.path.SelectAllText()
	})
	e.form.AddShortcut("Mod+F", func() { e.search.Focus() })

	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	e.navigate(home, false)
	return e.form
}

func (e *explorer) newToolbar() ui.Widgeter {
	bar := ui.NewPanel()
	tool := func(col int, img func(int) image.Image, tooltip string, onClick func()) *ui.ToolButton {
		b := ui.NewToolButton(img(20), tooltip, onClick)
		b.SetFlat(true)
		b.SetButtonSize(32, 32)
		bar.AddWidget(0, col, b)
		return b
	}
	e.back = tool(0, icons.Back, "Back (Alt+Left)", e.goBack)
	tool(1, func(size int) image.Image { return icons.Arrow(size, true) }, "Up (Alt+Up)", e.goUp)
	tool(2, iconHome, "Home folder", func() {
		if home, err := os.UserHomeDir(); err == nil {
			e.navigate(home, true)
		}
	})
	tool(3, iconRefresh, "Refresh (F5)", e.refresh)

	e.path = ui.NewTextBox()
	e.path.SetTooltip("Type a path and press Enter (Ctrl+L)")
	e.path.SetOnTextBoxKeyDown(func() {
		ev := ui.CurrentEvent().Parameter.(*ui.EventTextboxKeyDown)
		switch ev.Key {
		case ui.KeyEnter:
			e.navigateTyped(e.path.Text())
			ev.Processed = true
		case ui.KeyEsc:
			e.path.SetText(e.dir)
			e.table.Focus()
			ev.Processed = true
		}
	})
	bar.AddWidget(0, 4, e.path)

	e.search = ui.NewTextBox()
	e.search.SetHint("Search in folder")
	e.search.SetMinWidth(200)
	e.search.SetMaxWidth(200)
	e.search.SetOnTextChanged(e.fillTable)
	e.search.SetOnTextBoxKeyDown(func() {
		ev := ui.CurrentEvent().Parameter.(*ui.EventTextboxKeyDown)
		if ev.Key == ui.KeyEsc {
			e.search.SetText("")
			ev.Processed = true
		}
	})
	bar.AddWidget(0, 5, e.search)

	var theme *ui.ToolButton
	light := false
	theme = tool(6, icons.Sun, "Light theme", func() {
		light = !light
		if light {
			ui.ApplyLightTheme()
			theme.SetImage(icons.Moon(20))
			theme.SetTooltip("Dark theme")
		} else {
			ui.ApplyDarkTheme()
			theme.SetImage(icons.Sun(20))
			theme.SetTooltip("Light theme")
		}
	})
	return bar
}

func (e *explorer) newStatusBar() ui.Widgeter {
	bar := ui.NewPanel()
	e.status = bar.AddLabel(0, 0, "")
	bar.AddHSpacer(0, 1)
	e.selection = bar.AddLabel(0, 2, "")
	hidden := ui.NewCheckbox("Show hidden files")
	hidden.SetOnStateChanged(func() {
		e.showHidden = hidden.Checked()
		e.reloadTree()
		e.refresh()
	})
	bar.AddWidget(0, 3, hidden)
	return bar
}

// navigate shows the folder dir; addHistory remembers the current folder
// for Back. A folder that can't be read leaves everything as it was.
func (e *explorer) navigate(dir string, addHistory bool) bool {
	dir = filepath.Clean(dir)
	entries, err := readDir(dir)
	if err != nil {
		e.showError(err)
		return false
	}
	if addHistory && e.dir != "" && e.dir != dir {
		e.history = append(e.history, e.dir)
	}
	e.back.SetEnabled(len(e.history) > 0)
	e.dir = dir
	e.entries = entries
	e.path.SetText(dir)
	e.search.SetText("") // refills the table
	e.fillTable()
	e.selectInTree(dir)
	e.form.SetTitle("Files - " + dir)
	return true
}

// navigateTyped goes to a path typed by the user: "~" is the home folder,
// a file shows its folder with the file selected.
func (e *explorer) navigateTyped(path string) {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		e.showError(err)
		return
	}
	if !info.IsDir() {
		if e.navigate(filepath.Dir(path), true) {
			e.selectName(filepath.Base(path))
		}
		return
	}
	if e.navigate(path, true) {
		e.table.Focus()
	}
}

func (e *explorer) goBack() {
	for len(e.history) > 0 {
		dir := e.history[len(e.history)-1]
		e.history = e.history[:len(e.history)-1]
		if e.navigate(dir, false) {
			return
		}
	}
	e.back.SetEnabled(false)
}

// goUp shows the parent folder with the current folder selected in it.
func (e *explorer) goUp() {
	parent := filepath.Dir(e.dir)
	if parent == e.dir {
		return
	}
	child := filepath.Base(e.dir)
	if e.navigate(parent, true) {
		e.selectName(child)
	}
}

// refresh reads the folder again, keeping the selection and the search.
func (e *explorer) refresh() {
	entries, err := readDir(e.dir)
	if err != nil {
		e.showError(err)
		return
	}
	selected := ""
	if en, ok := e.current(); ok {
		selected = en.name
	}
	e.entries = entries
	e.fillTable()
	e.selectName(selected)
}

func (e *explorer) showError(err error) {
	e.form.ShowToast(err.Error(), ui.ToastError)
}

// iconHome is a house.
func iconHome(size int) image.Image {
	return icons.Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#E53935")
		dc.MoveTo(8, 1.5)
		dc.LineTo(15, 8)
		dc.LineTo(1, 8)
		dc.ClosePath()
		dc.Fill()
		dc.SetHexColor("#8D6E63")
		dc.DrawRectangle(3, 8, 10, 6.5)
		dc.Fill()
		dc.SetHexColor("#FFE082")
		dc.DrawRectangle(6.5, 10, 3, 4.5)
		dc.Fill()
	})
}

// iconRefresh is a circular arrow.
func iconRefresh(size int) image.Image {
	return icons.Draw(size, func(dc *gg.Context) {
		dc.SetHexColor("#43A047")
		dc.SetLineWidth(2)
		dc.DrawArc(8, 8, 5.5, gg.Radians(-60), gg.Radians(220))
		dc.Stroke()
		dc.MoveTo(15, 1.5)
		dc.LineTo(15, 7.5)
		dc.LineTo(9, 5.5)
		dc.ClosePath()
		dc.Fill()
	})
}

// formatSize is a size in bytes for people: 1.5 MB.
func formatSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value, suffix := float64(n)/unit, "KB"
	for _, s := range []string{"MB", "GB", "TB"} {
		if value < unit {
			break
		}
		value, suffix = value/unit, s
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}
