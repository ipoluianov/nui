// File dialog types re-exported from the platform layer so that applications
// only need to import ui.

package ui

import "github.com/ipoluianov/nui/internal/platforms"

type (
	FileDialogFilter             = platforms.FileDialogFilter
	OpenFileDialogOptions        = platforms.OpenFileDialogOptions
	SaveFileDialogOptions        = platforms.SaveFileDialogOptions
	SelectDirectoryDialogOptions = platforms.SelectDirectoryDialogOptions
)

// ErrNoFileDialog is returned when the system has no file dialog to show,
// e.g. Linux without zenity or kdialog
var ErrNoFileDialog = platforms.ErrNoFileDialog

// The system file dialogs block while they are open, so the form shows them on
// another goroutine and stays responsive; the result comes back on the UI
// thread. The dialog is owned by the form. Where the system has no file
// dialog (Linux without zenity and kdialog), nui shows its own, see
// UseBuiltinFileDialogs.

// ShowOpenFileDialog shows the system "Open File" dialog without blocking the
// form and calls onResult on the UI thread: paths is empty when the user
// cancelled. opts.AllowMultiple lets the user pick several files.
func (c *Form) ShowOpenFileDialog(opts OpenFileDialogOptions, onResult func(paths []string, err error)) {
	builtin := func() {
		c.showBuiltinFileDialog(fileDialogOpen, opts.Title, opts.DefaultDirectory, "", opts.Filters, opts.AllowMultiple, func(paths []string) {
			if onResult != nil {
				onResult(paths, nil)
			}
		})
	}
	if builtinFileDialogs {
		builtin()
		return
	}
	wnd := c.wnd
	go func() {
		paths, err := platforms.OpenFileDialog(wnd, opts)
		Invoke(func() {
			if err == ErrNoFileDialog {
				builtin()
				return
			}
			if onResult != nil {
				onResult(paths, err)
			}
		})
	}()
}

// ShowSaveFileDialog shows the system "Save File" dialog without blocking the
// form and calls onResult on the UI thread: path is "" when the user cancelled.
func (c *Form) ShowSaveFileDialog(opts SaveFileDialogOptions, onResult func(path string, err error)) {
	builtin := func() {
		c.showBuiltinFileDialog(fileDialogSave, opts.Title, opts.DefaultDirectory, opts.DefaultFileName, opts.Filters, false, func(paths []string) {
			if onResult != nil {
				path := ""
				if len(paths) > 0 {
					path = paths[0]
				}
				onResult(path, nil)
			}
		})
	}
	if builtinFileDialogs {
		builtin()
		return
	}
	wnd := c.wnd
	go func() {
		path, err := platforms.SaveFileDialog(wnd, opts)
		Invoke(func() {
			if err == ErrNoFileDialog {
				builtin()
				return
			}
			if onResult != nil {
				onResult(path, err)
			}
		})
	}()
}

// ShowSelectDirectoryDialog shows the system "Select Folder" dialog without
// blocking the form and calls onResult on the UI thread: path is "" when the
// user cancelled.
func (c *Form) ShowSelectDirectoryDialog(opts SelectDirectoryDialogOptions, onResult func(path string, err error)) {
	builtin := func() {
		c.showBuiltinFileDialog(fileDialogDirectory, opts.Title, opts.DefaultDirectory, "", nil, false, func(paths []string) {
			if onResult != nil {
				path := ""
				if len(paths) > 0 {
					path = paths[0]
				}
				onResult(path, nil)
			}
		})
	}
	if builtinFileDialogs {
		builtin()
		return
	}
	wnd := c.wnd
	go func() {
		path, err := platforms.SelectDirectoryDialog(wnd, opts)
		Invoke(func() {
			if err == ErrNoFileDialog {
				builtin()
				return
			}
			if onResult != nil {
				onResult(path, err)
			}
		})
	}()
}
