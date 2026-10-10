package files

import (
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

// entry is a file or a folder of the current folder.
type entry struct {
	name string
	path string
	dir  bool
	size int64
	mode fs.FileMode
	mod  time.Time
}

// kind is the text of the Type column.
func (en entry) kind() string {
	switch {
	case en.dir:
		return "Folder"
	case en.mode&fs.ModeSymlink != 0:
		return "Link"
	case en.mode&fs.ModeType != 0:
		return "Special"
	}
	ext := strings.TrimPrefix(filepath.Ext(en.name), ".")
	if ext == "" {
		return "File"
	}
	return strings.ToUpper(ext) + " file"
}

func isHidden(name string) bool { return strings.HasPrefix(name, ".") }

// readDir reads the entries of a folder; links to folders count as folders.
func readDir(dir string) ([]entry, error) {
	list, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]entry, 0, len(list))
	for _, d := range list {
		en := entry{name: d.Name(), path: filepath.Join(dir, d.Name())}
		info, err := os.Stat(en.path)
		if err != nil {
			// A broken link or no access: still listed, as far as known
			if info, err = d.Info(); err != nil {
				continue
			}
		}
		en.dir = info.IsDir()
		en.size = info.Size()
		en.mode = info.Mode()
		en.mod = info.ModTime()
		if d.Type()&fs.ModeSymlink != 0 {
			en.mode |= fs.ModeSymlink
		}
		entries = append(entries, en)
	}
	return entries, nil
}

const (
	colName = iota
	colSize
	colType
	colModified
)

var columnNames = []string{"Name", "Size", "Type", "Modified"}

func (e *explorer) newTable() *ui.Table {
	t := ui.NewTable()
	t.SetColumnCount(len(columnNames))
	for col, name := range columnNames {
		t.SetColumnName(col, name)
	}
	t.SetColumnWidth(colName, 230)
	t.SetColumnWidth(colSize, 80)
	t.SetColumnWidth(colType, 90)
	t.SetColumnWidth(colModified, 130)
	t.SetColumnHAlign(colSize, ui.HAlignRight)
	t.SetStretchLastColumn(true)
	t.SetEditTriggerDoubleClick(false)
	t.SetEditTriggerEnter(false)
	t.SetEditTriggerF2(false)
	t.SetEditTriggerKeyDown(false)
	t.SetMultiselect(true)

	t.SetOnColumnClick(func(col int) {
		if col == e.sortCol {
			e.sortDesc = !e.sortDesc
		} else {
			e.sortCol, e.sortDesc = col, false
		}
		selected := ""
		if en, ok := e.current(); ok {
			selected = en.name
		}
		e.fillTable()
		e.selectName(selected)
	})
	t.SetOnSelectionChanged(func(row, col int) { e.selectionChanged() })
	t.SetOnCellMouseDblClick(func() {
		ev := ui.CurrentEvent().Parameter.(*ui.EventTableCellMouseDblClick)
		ev.Processed = true
		e.open()
	})
	t.SetOnKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
		switch {
		case key == ui.KeyEnter && !mods.Ctrl && !mods.Shift && !mods.Alt:
			e.open()
			return true
		case key == ui.KeyBackspace:
			e.goUp()
			return true
		}
		return false
	})

	menu := ui.NewContextMenu(t)
	menu.AddItem("Open", e.open).SetImage(icons.Folder(icons.Size))
	menu.AddItem("Open in System", func() {
		if en, ok := e.current(); ok {
			if err := ui.OpenURL("file://" + en.path); err != nil {
				e.showError(err)
			}
		}
	})
	menu.AddSeparator()
	menu.AddItem("Copy Path", func() {
		var paths []string
		for _, row := range t.SelectedRows() {
			paths = append(paths, e.rows[row].path)
		}
		if len(paths) > 0 {
			ui.ClipboardSetText(strings.Join(paths, "\n"))
			e.form.ShowToast("Copied: "+strings.Join(paths, ", "), ui.ToastInfo)
		}
	})
	menu.AddSeparator()
	menu.AddItem("Properties", e.properties).SetImage(icons.Info(icons.Size))
	t.SetContextMenu(menu)
	return t
}

// fillTable shows the entries matching the search, folders first, sorted
// by the chosen column.
func (e *explorer) fillTable() {
	filter := strings.ToLower(strings.TrimSpace(e.search.Text()))
	e.rows = e.rows[:0]
	for _, en := range e.entries {
		if !e.showHidden && isHidden(en.name) {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(en.name), filter) {
			continue
		}
		e.rows = append(e.rows, en)
	}
	sort.SliceStable(e.rows, func(i, j int) bool {
		a, b := e.rows[i], e.rows[j]
		if a.dir != b.dir {
			return a.dir
		}
		var less bool
		switch e.sortCol {
		case colSize:
			less = a.size < b.size
		case colType:
			less = a.kind() < b.kind()
		case colModified:
			less = a.mod.Before(b.mod)
		default:
			less = strings.ToLower(a.name) < strings.ToLower(b.name)
		}
		if e.sortDesc {
			return !less
		}
		return less
	})

	for col := range columnNames {
		var img image.Image
		if col == e.sortCol {
			img = sortArrow(e.sortDesc)
		}
		e.table.SetColumnImage(col, img, icons.Size)
	}

	folderIcon, fileIcon := icons.Folder(icons.Size), icons.File(icons.Size)
	t := e.table
	t.ClearRows()
	var folders, files int
	var total int64
	for row, en := range e.rows {
		t.SetCellText2(row, colName, en.name)
		t.SetCellContraction(row, colName, true)
		t.SetCellText2(row, colType, en.kind())
		t.SetCellText2(row, colModified, en.mod.Format("2006-01-02 15:04"))
		if en.dir {
			folders++
			t.SetCellImage(row, colName, folderIcon, icons.Size)
		} else {
			files++
			total += en.size
			t.SetCellImage(row, colName, fileIcon, icons.Size)
			t.SetCellText2(row, colSize, formatSize(en.size))
		}
		if isHidden(en.name) {
			t.SetCellColor(row, colName, ui.ColorFromHex("#8b949e"))
		}
	}
	t.SetRowCount(len(e.rows))

	text := fmt.Sprintf("%s, %s, %s", count(folders, "folder"), count(files, "file"), formatSize(total))
	if len(e.rows) < len(e.entries) {
		text += fmt.Sprintf(" (%d hidden or filtered out)", len(e.entries)-len(e.rows))
	}
	e.status.SetText(text)
	e.selectionChanged()
}

// count is "1 file", "2 files".
func count(n int, noun string) string {
	if n != 1 {
		noun += "s"
	}
	return fmt.Sprintf("%d %s", n, noun)
}

// current is the entry of the current row.
func (e *explorer) current() (entry, bool) {
	row := e.table.CurrentRow()
	if row < 0 || row >= len(e.rows) {
		return entry{}, false
	}
	return e.rows[row], true
}

// selectName makes the entry with the name current, if it is shown.
func (e *explorer) selectName(name string) {
	for row, en := range e.rows {
		if en.name == name {
			e.table.SetCurrentCell2(row, colName)
			return
		}
	}
}

func (e *explorer) selectionChanged() {
	rows := e.table.SelectedRows()
	switch {
	case len(rows) > 1:
		var total int64
		for _, row := range rows {
			if row < len(e.rows) {
				total += e.rows[row].size
			}
		}
		e.selection.SetText(fmt.Sprintf("%d selected, %s", len(rows), formatSize(total)))
	case len(rows) == 1 && rows[0] < len(e.rows):
		en := e.rows[rows[0]]
		if en.dir {
			e.selection.SetText(en.name)
		} else {
			e.selection.SetText(fmt.Sprintf("%s, %s", en.name, formatSize(en.size)))
		}
	default:
		e.selection.SetText("")
	}
	en, ok := e.current()
	if !ok {
		e.preview.clear()
		return
	}
	e.preview.show(en)
}

// open enters a folder or shows a file in the preview.
func (e *explorer) open() {
	en, ok := e.current()
	if !ok {
		return
	}
	if en.dir {
		if e.navigate(en.path, true) && len(e.rows) > 0 {
			e.table.SetCurrentCell2(0, colName)
		}
		return
	}
	e.preview.show(en)
	e.preview.focus()
}

func (e *explorer) properties() {
	en, ok := e.current()
	if !ok {
		return
	}
	size := formatSize(en.size)
	if en.dir {
		if list, err := os.ReadDir(en.path); err == nil {
			size = fmt.Sprintf("%d items", len(list))
		} else {
			size = "unknown (" + err.Error() + ")"
		}
	} else if en.size >= 1024 {
		size += fmt.Sprintf(" (%d bytes)", en.size)
	}
	text := strings.Join([]string{
		"Name: " + en.name,
		"Location: " + filepath.Dir(en.path),
		"Type: " + en.kind(),
		"Size: " + size,
		"Modified: " + en.mod.Format("2006-01-02 15:04:05"),
		"Permissions: " + en.mode.String(),
	}, "\n")
	if target, err := os.Readlink(en.path); err == nil {
		text += "\nLink to: " + target
	}
	ui.ShowMessageBox(e.table, "Properties", text)
}

// sortArrow is the header icon of the sort column: a small triangle.
func sortArrow(desc bool) image.Image {
	return icons.Draw(icons.Size, func(dc *gg.Context) {
		if desc {
			dc.RotateAbout(gg.Radians(180), 8, 8)
		}
		dc.SetHexColor("#8B949E")
		dc.MoveTo(8, 5)
		dc.LineTo(12, 10)
		dc.LineTo(4, 10)
		dc.ClosePath()
		dc.Fill()
	})
}
