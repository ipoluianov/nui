// File dialog types re-exported from the platform layer so that applications
// only need to import ui.

package ui

import "github.com/ipoluianov/nui/internal/platforms"

type (
	FileDialogFilter      = platforms.FileDialogFilter
	OpenFileDialogOptions = platforms.OpenFileDialogOptions
	SaveFileDialogOptions = platforms.SaveFileDialogOptions
)

// ErrNoFileDialog is returned when the system has no file dialog to show,
// e.g. Linux without zenity or kdialog
var ErrNoFileDialog = platforms.ErrNoFileDialog
