package notepad

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ipoluianov/nui/ui"
)

const (
	maxRecent   = 6
	maxFileSize = 4 << 20 // bigger files are not for a notepad
)

var fileFilters = []ui.FileDialogFilter{
	{DisplayName: "Text files", Patterns: []string{"*.txt", "*.md"}},
	{DisplayName: "All files", Patterns: []string{"*"}},
}

// setDocument shows a document: a file just opened or a new one
func (n *notepad) setDocument(path, text string) {
	n.loading = true
	n.text.SetText(text)
	n.loading = false
	n.path = path
	n.findPos = 0
	n.modified = false
	n.updateTitle()
	n.updateStatus()
	n.text.Focus()
}

// confirmDiscard runs then once the changes of the document are saved or
// the user chose to drop them. If the save is canceled, then is not run.
func (n *notepad) confirmDiscard(then func()) {
	if !n.modified {
		then()
		return
	}
	ui.ShowQuestionMessageBoxYesNo(n.form.Panel(), appName,
		"Do you want to save the changes to "+n.fileName()+"?",
		func() { n.save(then) }, then)
}

func (n *notepad) newDocument() {
	n.confirmDiscard(func() { n.setDocument("", "") })
}

func (n *notepad) open() {
	n.confirmDiscard(func() {
		opts := ui.OpenFileDialogOptions{Title: "Open", Filters: fileFilters}
		if n.path != "" {
			opts.DefaultDirectory = filepath.Dir(n.path)
		}
		n.form.ShowOpenFileDialog(opts, func(paths []string, err error) {
			if err != nil {
				n.showError(err)
				return
			}
			if len(paths) > 0 {
				n.openFile(paths[0])
			}
		})
	})
}

// openFile reads the file into the editor; the caller has already dealt
// with the unsaved changes
func (n *notepad) openFile(path string) {
	info, err := os.Stat(path)
	if err != nil {
		n.showError(err)
		return
	}
	if info.IsDir() {
		n.showError(fmt.Errorf("%s is a folder", filepath.Base(path)))
		return
	}
	if info.Size() > maxFileSize {
		n.showError(fmt.Errorf("%s is too big for Notepad (%d KB)", filepath.Base(path), info.Size()>>10))
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		n.showError(err)
		return
	}
	if !utf8.Valid(data) {
		n.showError(fmt.Errorf("%s is not a text file", filepath.Base(path)))
		return
	}
	// The text box works with "\n" line ends; a file saved back gets them
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	n.setDocument(path, text)
	n.addRecent(path)
}

// save writes the document to its file, asking for a name if it has none,
// then runs then (may be nil)
func (n *notepad) save(then func()) {
	if n.path == "" {
		n.saveAs(then)
		return
	}
	n.writeFile(n.path, then)
}

func (n *notepad) saveAs(then func()) {
	opts := ui.SaveFileDialogOptions{Title: "Save As", DefaultFileName: n.fileName(), Filters: fileFilters}
	if n.path == "" {
		opts.DefaultFileName = "Untitled.txt"
	} else {
		opts.DefaultDirectory = filepath.Dir(n.path)
	}
	n.form.ShowSaveFileDialog(opts, func(path string, err error) {
		if err != nil {
			n.showError(err)
			return
		}
		if path != "" {
			n.writeFile(path, then)
		}
	})
}

func (n *notepad) writeFile(path string, then func()) {
	if err := os.WriteFile(path, []byte(n.text.Text()), 0o644); err != nil {
		n.showError(err)
		return
	}
	n.path = path
	n.modified = false
	n.updateTitle()
	n.updateStatus()
	n.addRecent(path)
	n.form.ShowToast("Saved "+filepath.Base(path), ui.ToastSuccess)
	if then != nil {
		then()
	}
}

func (n *notepad) print() {
	job := ui.NewTextPrintJob(n.fileName(), n.text.Text(), &ui.TextPrintOptions{FontFamily: ui.FontFamilyMono})
	n.form.Print(job, func(err error) {
		if err != nil && err != ui.ErrPrintCanceled {
			n.showError(err)
		}
	})
}

func (n *notepad) exportPDF() {
	name := strings.TrimSuffix(n.fileName(), filepath.Ext(n.fileName())) + ".pdf"
	opts := ui.SaveFileDialogOptions{
		Title:           "Export PDF",
		DefaultFileName: name,
		Filters:         []ui.FileDialogFilter{{DisplayName: "PDF", Patterns: []string{"*.pdf"}}},
	}
	n.form.ShowSaveFileDialog(opts, func(path string, err error) {
		if err != nil {
			n.showError(err)
			return
		}
		if path == "" {
			return
		}
		job := ui.NewTextPrintJob(n.fileName(), n.text.Text(), &ui.TextPrintOptions{FontFamily: ui.FontFamilyMono})
		if err := job.SavePDF(path); err != nil {
			n.showError(err)
			return
		}
		n.form.ShowToast("Exported "+filepath.Base(path), ui.ToastSuccess)
	})
}

// addRecent puts the file at the top of File > Open Recent
func (n *notepad) addRecent(path string) {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	n.recent = slices.DeleteFunc(n.recent, func(p string) bool { return p == path })
	n.recent = slices.Insert(n.recent, 0, path)
	if len(n.recent) > maxRecent {
		n.recent = n.recent[:maxRecent]
	}
	n.updateRecentMenu()
}

func (n *notepad) updateRecentMenu() {
	n.recentMenu.RemoveAllItems()
	if len(n.recent) == 0 {
		n.recentMenu.AddItem("(no recent files)", nil)
		return
	}
	for i, path := range n.recent {
		label := fmt.Sprintf("&%d  %s", i+1, filepath.Base(path))
		n.recentMenu.AddItem(label, func() {
			n.confirmDiscard(func() { n.openFile(path) })
		})
	}
	n.recentMenu.AddSeparator()
	n.recentMenu.AddItem("&Clear List", func() {
		n.recent = nil
		n.updateRecentMenu()
	})
}

func (n *notepad) showError(err error) {
	n.form.ShowToastFor(err.Error(), ui.ToastError, 5*time.Second)
}
