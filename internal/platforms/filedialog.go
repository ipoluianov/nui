package platforms

import "errors"

// ErrNoFileDialog is returned when the system has no file dialog to show,
// e.g. Linux without zenity or kdialog
var ErrNoFileDialog = errors.New("nui: no file dialog helper found (install zenity or kdialog)")

// FileDialogFilter restricts an open-file dialog to a named group of file
// patterns, e.g. {DisplayName: "Text files", Patterns: []string{"*.txt", "*.md"}}.
type FileDialogFilter struct {
	DisplayName string
	Patterns    []string
}

// OpenFileDialogOptions configures OpenFileDialog.
type OpenFileDialogOptions struct {
	Title            string
	DefaultDirectory string
	Filters          []FileDialogFilter
	AllowMultiple    bool
}

// OpenFileDialog shows the operating system's standard "Open File" dialog,
// owned by parent if given (parent may be nil), and blocks until the user
// picks a file or cancels. On cancel it returns (nil, nil).
func OpenFileDialog(parent Window, opts OpenFileDialogOptions) ([]string, error) {
	return openFileDialog(parent, opts)
}

// SaveFileDialogOptions configures SaveFileDialog.
type SaveFileDialogOptions struct {
	Title            string
	DefaultDirectory string
	DefaultFileName  string
	Filters          []FileDialogFilter
}

// SaveFileDialog shows the operating system's standard "Save File" dialog,
// owned by parent if given (parent may be nil), and blocks until the user
// picks a destination path or cancels. On cancel it returns ("", nil).
func SaveFileDialog(parent Window, opts SaveFileDialogOptions) (string, error) {
	return saveFileDialog(parent, opts)
}

// SelectDirectoryDialogOptions configures SelectDirectoryDialog.
type SelectDirectoryDialogOptions struct {
	Title            string
	DefaultDirectory string
}

// SelectDirectoryDialog shows the operating system's standard "Select Folder"
// dialog, owned by parent if given (parent may be nil), and blocks until the
// user picks a directory or cancels. On cancel it returns ("", nil).
func SelectDirectoryDialog(parent Window, opts SelectDirectoryDialogOptions) (string, error) {
	return selectDirectoryDialog(parent, opts)
}
