//go:build linux
// +build linux

package platforms

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
)

// openFileDialog shells out to the desktop's own file-picker helper: zenity
// (GTK/GNOME) or kdialog (Qt/KDE), the one of the running desktop first.
// Neither X11 nor GTK is otherwise linked into this package, so this avoids
// pulling in a GTK build dependency just to draw a dialog — it reuses
// whichever one the running desktop already ships.
//
// The dialog is modal over parent: the parent gets no keyboard or mouse
// input while it is open (as for a dialog of ShowModal), and kdialog is
// made transient for it (zenity 4 ignores --attach).
func openFileDialog(parent Window, opts OpenFileDialogOptions) ([]string, error) {
	var res []string
	err := withHelper(parent, func(kind, bin string, attach []string) (err error) {
		if kind == "kdialog" {
			res, err = openFileDialogKdialog(bin, opts, attach)
		} else {
			res, err = openFileDialogZenity(bin, opts)
		}
		return
	})
	return res, err
}

func saveFileDialog(parent Window, opts SaveFileDialogOptions) (string, error) {
	var res string
	err := withHelper(parent, func(kind, bin string, attach []string) (err error) {
		if kind == "kdialog" {
			res, err = saveFileDialogKdialog(bin, opts, attach)
		} else {
			res, err = saveFileDialogZenity(bin, opts)
		}
		return
	})
	return res, err
}

func selectDirectoryDialog(parent Window, opts SelectDirectoryDialogOptions) (string, error) {
	var res string
	err := withHelper(parent, func(kind, bin string, attach []string) (err error) {
		if kind == "kdialog" {
			res, err = selectDirectoryDialogKdialog(bin, opts, attach)
		} else {
			res, err = selectDirectoryDialogZenity(bin, opts)
		}
		return
	})
	return res, err
}

// withHelper runs a dialog by the helper of the desktop (kdialog first on
// KDE, else zenity first), modal over parent: its input is dropped for as
// long as the helper runs (the modalChildCount of ShowModal)
func withHelper(parent Window, run func(kind, bin string, attach []string) error) error {
	kinds := []string{"zenity", "kdialog"}
	if strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "KDE") {
		kinds = []string{"kdialog", "zenity"}
	}
	var attach []string
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		attach = kdialogAttach(p)
		atomic.AddInt32(&p.platform.modalChildCount, 1)
		defer atomic.AddInt32(&p.platform.modalChildCount, -1)
	}
	for _, kind := range kinds {
		if bin, err := exec.LookPath(kind); err == nil {
			return run(kind, bin, attach)
		}
	}
	return ErrNoFileDialog
}

// kdialogAttach are the arguments that make a kdialog transient for a
// window and place it in its middle: the window manager places a dialog of
// a foreign window where it likes, on another screen too
func kdialogAttach(p *nativeWindow) []string {
	args := []string{"--attach", strconv.FormatUint(uint64(p.platform.window), 10)}
	if geo := centeredOver(p); geo != "" {
		args = append(args, "--geometry", geo)
	}
	return args
}

// centeredOver is the geometry WxH+X+Y of a dialog in the middle of a window
func centeredOver(p *nativeWindow) string {
	px, py := p.Pos()
	pw, ph := p.Size()
	if pw <= 0 || ph <= 0 {
		return ""
	}
	w, h := min(900, pw-40), min(620, ph-40)
	if w < 400 || h < 300 {
		return ""
	}
	return fmt.Sprintf("%dx%d+%d+%d", w, h, px+(pw-w)/2, py+(ph-h)/2)
}

func openFileDialogZenity(bin string, opts OpenFileDialogOptions) ([]string, error) {
	args := []string{"--file-selection"}

	if opts.Title != "" {
		args = append(args, "--title="+opts.Title)
	}
	if opts.DefaultDirectory != "" {
		args = append(args, "--filename="+withTrailingSlash(opts.DefaultDirectory))
	}
	if opts.AllowMultiple {
		args = append(args, "--multiple", "--separator=\n")
	}
	args = append(args, zenityFilterArgs(opts.Filters)...)

	out, err := runDialog(bin, args...)
	if err != nil {
		return nil, err
	}
	return splitDialogOutput(out), nil
}

func openFileDialogKdialog(bin string, opts OpenFileDialogOptions, attach []string) ([]string, error) {
	args := append([]string(nil), attach...)
	if opts.AllowMultiple {
		args = append(args, "--multiple", "--separate-output")
	}
	args = append(args, "--getopenfilename")

	dir := opts.DefaultDirectory
	if dir == "" {
		dir = "."
	}
	args = append(args, dir)
	args = append(args, kdialogFilterArgs(opts.Filters)...)
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	out, err := runDialog(bin, args...)
	if err != nil {
		return nil, err
	}
	return splitDialogOutput(out), nil
}

func saveFileDialogZenity(bin string, opts SaveFileDialogOptions) (string, error) {
	args := []string{"--file-selection", "--save", "--confirm-overwrite"}

	if opts.Title != "" {
		args = append(args, "--title="+opts.Title)
	}
	if path := joinDirAndName(opts.DefaultDirectory, opts.DefaultFileName); path != "" {
		args = append(args, "--filename="+path)
	}
	args = append(args, zenityFilterArgs(opts.Filters)...)

	return runDialog(bin, args...)
}

func saveFileDialogKdialog(bin string, opts SaveFileDialogOptions, attach []string) (string, error) {
	dir := opts.DefaultDirectory
	if dir == "" {
		dir = "."
	}
	path := joinDirAndName(dir, opts.DefaultFileName)

	args := append(append([]string(nil), attach...), "--getsavefilename", path)
	args = append(args, kdialogFilterArgs(opts.Filters)...)
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	return runDialog(bin, args...)
}

func selectDirectoryDialogZenity(bin string, opts SelectDirectoryDialogOptions) (string, error) {
	args := []string{"--file-selection", "--directory"}

	if opts.Title != "" {
		args = append(args, "--title="+opts.Title)
	}
	if opts.DefaultDirectory != "" {
		args = append(args, "--filename="+withTrailingSlash(opts.DefaultDirectory))
	}

	return runDialog(bin, args...)
}

func selectDirectoryDialogKdialog(bin string, opts SelectDirectoryDialogOptions, attach []string) (string, error) {
	dir := opts.DefaultDirectory
	if dir == "" {
		dir = "."
	}

	args := append(append([]string(nil), attach...), "--getexistingdirectory", dir)
	if opts.Title != "" {
		args = append(args, "--title", opts.Title)
	}

	return runDialog(bin, args...)
}

// runDialog runs a dialog-helper command and returns its trimmed stdout. A
// user-cancelled dialog (both zenity and kdialog exit 1 in that case) is
// reported as ("", nil) rather than an error.
func runDialog(bin string, args ...string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", err
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

func splitDialogOutput(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func withTrailingSlash(dir string) string {
	if !strings.HasSuffix(dir, "/") {
		return dir + "/"
	}
	return dir
}

func joinDirAndName(dir, name string) string {
	if dir == "" {
		return name
	}
	return withTrailingSlash(dir) + name
}

func zenityFilterArgs(filters []FileDialogFilter) []string {
	args := make([]string, 0, len(filters))
	for _, f := range filters {
		filter := strings.Join(f.Patterns, " ")
		if f.DisplayName != "" {
			filter = f.DisplayName + " | " + filter
		}
		args = append(args, "--file-filter="+filter)
	}
	return args
}

func kdialogFilterArgs(filters []FileDialogFilter) []string {
	if len(filters) == 0 {
		return nil
	}
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		parts = append(parts, strings.Join(f.Patterns, " ")+"|"+f.DisplayName)
	}
	return []string{strings.Join(parts, "\n")}
}
