package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The file dialog of nui itself: shown where the system has none (Linux
// without zenity and kdialog), or always after UseBuiltinFileDialogs(true).
// A folder's files in a table, the path above it, the name and the filter
// below; Enter or a double click opens a folder or picks a file.

var builtinFileDialogs bool

// UseBuiltinFileDialogs makes the file dialogs of the forms nui's own
// instead of the system's: the same on every system, and testable with
// uitest. By default nui's dialog is shown only where the system has none.
func UseBuiltinFileDialogs(use bool) {
	builtinFileDialogs = use
}

type fileDialogMode int

const (
	fileDialogOpen fileDialogMode = iota
	fileDialogSave
	fileDialogDirectory
)

// fileDialog is the content of the dialog
type fileDialog struct {
	DialogContent
	mode     fileDialogMode
	title    string
	multiple bool
	filters  []FileDialogFilter
	dir      string
	entries  []os.DirEntry
	model    *fileDialogModel
	onResult func(paths []string)
	done     bool // the result is given

	path       *TextBox
	table      *Table
	name       *TextBox
	filter     *ComboBox
	showHidden *Checkbox
	ok         *Button
}

// fileDialogModel shows the entries of the folder: name, size, modified
type fileDialogModel struct {
	ModelSignal
	d *fileDialog
}

func (m *fileDialogModel) RowCount() int    { return len(m.d.entries) }
func (m *fileDialogModel) ColumnCount() int { return 3 }

func (m *fileDialogModel) ColumnName(col int) string {
	return [...]string{UIText().FileName, UIText().FileSize, UIText().FileModified}[col]
}

func (m *fileDialogModel) Text(row, col int) string {
	e := m.d.entries[row]
	switch col {
	case 0:
		return e.Name()
	case 1:
		if e.IsDir() {
			return ""
		}
		if info, err := e.Info(); err == nil {
			return formatFileSize(info.Size())
		}
	case 2:
		if info, err := e.Info(); err == nil {
			return info.ModTime().Format(time.DateTime[:16])
		}
	}
	return ""
}

func (m *fileDialogModel) Style(row, col int) ItemStyle {
	if col == 0 && m.d.entries[row].IsDir() {
		return ItemStyle{Icon: folderIcon()}
	}
	if col == 1 {
		return ItemStyle{HAlign: HAlignRight}
	}
	return ItemStyle{}
}

// formatFileSize is the size in B, KB, MB, GB
func formatFileSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	size, i := float64(n)/unit, 0
	for size >= unit && i < 3 {
		size /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", size, [...]string{"KB", "MB", "GB", "TB"}[i])
}

var folderIconImage image.Image

// folderIcon is a small yellow folder
func folderIcon() image.Image {
	if folderIconImage == nil {
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		body := color.RGBA{0xF0, 0xB4, 0x29, 0xFF}
		tab := color.RGBA{0xD9, 0x9A, 0x16, 0xFF}
		draw.Draw(img, image.Rect(1, 3, 7, 5), &image.Uniform{tab}, image.Point{}, draw.Src)
		draw.Draw(img, image.Rect(1, 5, 15, 14), &image.Uniform{body}, image.Point{}, draw.Src)
		folderIconImage = img
	}
	return folderIconImage
}

// showBuiltinFileDialog shows nui's dialog over the form; onResult gets
// the paths picked, none if cancelled
func (c *Form) showBuiltinFileDialog(mode fileDialogMode, title, dir, name string, filters []FileDialogFilter, multiple bool, onResult func(paths []string)) {
	d := &fileDialog{mode: mode, title: title, multiple: multiple, filters: filters, onResult: onResult}
	d.InitWidget()
	d.model = &fileDialogModel{d: d}

	top := NewHBox()
	up := NewButton(UIText().Up)
	up.SetOnClick(func() { d.openDir(filepath.Dir(d.dir)) })
	top.Add(up)
	home := NewButton(UIText().Home)
	home.SetOnClick(func() {
		if h, err := os.UserHomeDir(); err == nil {
			d.openDir(h)
		}
	})
	top.Add(home)
	d.path = NewTextBox()
	d.path.SetXExpandable(true)
	d.path.OnKeyDown.Connect(func(e *KeyEvent) {
		if e.Key == KeyEnter {
			e.Handled = true
			d.openDir(d.path.Text())
		}
	})
	top.Add(d.path)
	d.AddWidget(0, 0, top)

	d.table = NewTable()
	d.table.SetModel(d.model)
	d.table.SetSelectingRows(true)
	d.table.SetMultiselect(multiple)
	d.table.SetColumnWidth(0, 300)
	d.table.SetColumnWidth(1, 90)
	d.table.SetColumnWidth(2, 140)
	d.table.SetMinSize(560, 300)
	d.table.SetXExpandable(true)
	d.table.SetYExpandable(true)
	d.table.OnSelectionChanged.Connect(func(TableCell) { d.selectionChanged() })
	d.table.OnCellDoubleClick.Connect(func(e *TableCellEvent) {
		e.Handled = true
		d.activate(e.Row)
	})
	d.table.OnKeyDown.Connect(func(e *KeyEvent) {
		if e.Key == KeyEnter {
			e.Handled = true
			d.activate(d.table.CurrentRow())
		}
	})
	d.AddWidget(1, 0, d.table)

	bottom := NewFormLayout()
	if mode != fileDialogDirectory {
		d.name = NewTextBox()
		d.name.SetText(name)
		d.name.OnKeyDown.Connect(func(e *KeyEvent) {
			if e.Key == KeyEnter {
				e.Handled = true
				d.accept()
			}
		})
		bottom.AddRow(UIText().FileName+":", d.name)
	}
	if len(filters) > 0 {
		d.filter = NewComboBox()
		for _, f := range filters {
			d.filter.AddItem(fmt.Sprintf("%s (%s)", f.DisplayName, strings.Join(f.Patterns, " ")), nil)
		}
		d.filter.SetSelectedIndex(0)
		d.filter.OnSelectionChanged.Connect(func(int) { d.openDir(d.dir) })
		bottom.AddRow(UIText().FileType+":", d.filter)
	}
	d.showHidden = NewCheckbox(UIText().ShowHiddenFiles)
	d.showHidden.OnCheckedChanged.Connect(func(bool) { d.openDir(d.dir) })
	bottom.AddRowWidget(d.showHidden)
	buttons := NewDialogButtonBox()
	okText := UIText().Open
	if mode == fileDialogSave {
		okText = UIText().Save
	} else if mode == fileDialogDirectory {
		okText = UIText().SelectFolder
	}
	d.ok = buttons.AddButton(okText, ButtonAccept, d.accept)
	buttons.AddCancel(func() { d.finish(nil) })
	// Esc and the close button: the result is "cancelled" too
	d.OnDialogReject = func() bool {
		d.finish(nil)
		return true
	}
	d.AddWidget(2, 0, bottom)
	d.AddWidget(3, 0, buttons)
	d.SetRowStretch(1, 1)

	d.OnDialogShow = func() {
		d.Form().SetTitle(title)
		start := dir
		if start == "" {
			start, _ = os.Getwd()
		}
		d.openDir(start)
		if d.name != nil && name != "" {
			d.name.Focus()
			d.name.SelectAllText()
		} else {
			d.table.Focus()
		}
	}
	c.Panel().ShowDialog(d)
}

// openDir shows the folder; a path that is not a folder leaves the dialog
// where it is
func (d *fileDialog) openDir(dir string) {
	dir = expandHome(strings.TrimSpace(dir))
	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		d.path.SetText(d.dir)
		return
	}
	d.dir = abs
	d.path.SetText(abs)
	d.entries = d.entries[:0]
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") && !d.showHidden.Checked() {
			continue
		}
		if e.IsDir() || isDirLink(abs, e) {
			d.entries = append(d.entries, e)
			continue
		}
		if d.mode != fileDialogDirectory && d.matches(e.Name()) {
			d.entries = append(d.entries, e)
		}
	}
	// Folders first, then by name, as file managers sort them
	slices.SortStableFunc(d.entries, func(a, b os.DirEntry) int {
		if ad, bd := a.IsDir(), b.IsDir(); ad != bd {
			if ad {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})
	d.model.Changed()
	d.table.SetSelectedRows(nil)
	d.table.SetScrollY(0)
}

// isDirLink reports whether the entry is a link to a folder
func isDirLink(dir string, e os.DirEntry) bool {
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, e.Name()))
	return err == nil && info.IsDir()
}

// expandHome turns ~ at the start into the home folder
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// matches reports whether the name passes the filter chosen
func (d *fileDialog) matches(name string) bool {
	if d.filter == nil {
		return true
	}
	i := d.filter.SelectedIndex()
	if i < 0 || i >= len(d.filters) {
		return true
	}
	for _, pattern := range d.filters[i].Patterns {
		if ok, _ := filepath.Match(strings.ToLower(pattern), strings.ToLower(name)); ok {
			return true
		}
	}
	return false
}

// selectionChanged puts the files selected into the name field
func (d *fileDialog) selectionChanged() {
	if d.name == nil {
		return
	}
	var names []string
	for _, row := range d.table.SelectedRows() {
		if row >= 0 && row < len(d.entries) && !d.entries[row].IsDir() && !isDirLink(d.dir, d.entries[row]) {
			names = append(names, d.entries[row].Name())
		}
	}
	switch {
	case len(names) == 1:
		d.name.SetText(names[0])
	case len(names) > 1:
		d.name.SetText(`"` + strings.Join(names, `" "`) + `"`)
	}
}

// activate opens the folder of the row, or picks its file
func (d *fileDialog) activate(row int) {
	if row < 0 || row >= len(d.entries) {
		return
	}
	e := d.entries[row]
	if e.IsDir() || isDirLink(d.dir, e) {
		d.openDir(filepath.Join(d.dir, e.Name()))
		return
	}
	d.name.SetText(e.Name())
	d.accept()
}

// accept picks the files of the name field: a folder typed there is opened
// instead
func (d *fileDialog) accept() {
	// A folder: the one open (Enter and a double click open the others)
	if d.mode == fileDialogDirectory {
		d.finish([]string{d.dir})
		return
	}
	names := parseFileNames(d.name.Text())
	if len(names) == 0 {
		return
	}
	var paths []string
	for _, name := range names {
		path := expandHome(name)
		if !filepath.IsAbs(path) {
			path = filepath.Join(d.dir, path)
		}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			d.name.SetText("")
			d.openDir(path)
			return
		}
		paths = append(paths, path)
	}
	switch d.mode {
	case fileDialogOpen:
		for _, p := range paths {
			if _, err := os.Stat(p); err != nil {
				ShowMessageBox(d, d.title, fmt.Sprintf(UIText().FileNotFound, filepath.Base(p)))
				return
			}
		}
		if !d.multiple {
			paths = paths[:1]
		}
		d.finish(paths)
	case fileDialogSave:
		path := d.withExtension(paths[0])
		if _, err := os.Stat(path); err == nil {
			ShowQuestionMessageBoxYesNo(d, UIText().Save, fmt.Sprintf(UIText().FileExistsReplace, filepath.Base(path)), func() {
				d.finish([]string{path})
			}, nil)
			return
		}
		d.finish([]string{path})
	}
}

// withExtension adds the extension of the filter chosen to a name without
// one, when the filter has a single "*.ext"
func (d *fileDialog) withExtension(path string) string {
	if d.filter == nil || filepath.Ext(path) != "" {
		return path
	}
	i := d.filter.SelectedIndex()
	if i < 0 || i >= len(d.filters) || len(d.filters[i].Patterns) != 1 {
		return path
	}
	ext := strings.TrimPrefix(d.filters[i].Patterns[0], "*")
	if !strings.HasPrefix(ext, ".") || strings.ContainsAny(ext, "*?[") {
		return path
	}
	return path + ext
}

// parseFileNames splits the name field: one name, or several in quotes
func parseFileNames(text string) []string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, `"`) {
		if text == "" {
			return nil
		}
		return []string{text}
	}
	var names []string
	for _, part := range strings.Split(text, `"`) {
		if part = strings.TrimSpace(part); part != "" {
			names = append(names, part)
		}
	}
	return names
}

// finish gives the paths, none if cancelled, and closes the dialog
func (d *fileDialog) finish(paths []string) {
	if d.done {
		return
	}
	d.done = true
	d.RunInParent(func() {
		if d.onResult != nil {
			d.onResult(paths)
		}
	})
	d.Form().Close()
}
