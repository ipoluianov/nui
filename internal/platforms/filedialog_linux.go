//go:build linux
// +build linux

package platforms

import (
	"bytes"
	"os/exec"
	"strings"
)

// openFileDialog shells out to the desktop's own file-picker helper: zenity
// (GTK/GNOME) if present, else kdialog (Qt/KDE). Neither X11 nor GTK is
// otherwise linked into this package, so this avoids pulling in a GTK build
// dependency just to draw a dialog — it reuses whichever one the running
// desktop already ships.
func openFileDialog(parent Window, opts OpenFileDialogOptions) ([]string, error) {
	if bin, err := exec.LookPath("zenity"); err == nil {
		return openFileDialogZenity(bin, opts)
	}
	if bin, err := exec.LookPath("kdialog"); err == nil {
		return openFileDialogKdialog(bin, opts)
	}
	return nil, ErrNoFileDialog
}

func saveFileDialog(parent Window, opts SaveFileDialogOptions) (string, error) {
	if bin, err := exec.LookPath("zenity"); err == nil {
		return saveFileDialogZenity(bin, opts)
	}
	if bin, err := exec.LookPath("kdialog"); err == nil {
		return saveFileDialogKdialog(bin, opts)
	}
	return "", ErrNoFileDialog
}

func selectDirectoryDialog(parent Window, opts SelectDirectoryDialogOptions) (string, error) {
	if bin, err := exec.LookPath("zenity"); err == nil {
		return selectDirectoryDialogZenity(bin, opts)
	}
	if bin, err := exec.LookPath("kdialog"); err == nil {
		return selectDirectoryDialogKdialog(bin, opts)
	}
	return "", ErrNoFileDialog
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

func openFileDialogKdialog(bin string, opts OpenFileDialogOptions) ([]string, error) {
	args := make([]string, 0, 4)
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

func saveFileDialogKdialog(bin string, opts SaveFileDialogOptions) (string, error) {
	dir := opts.DefaultDirectory
	if dir == "" {
		dir = "."
	}
	path := joinDirAndName(dir, opts.DefaultFileName)

	args := []string{"--getsavefilename", path}
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

func selectDirectoryDialogKdialog(bin string, opts SelectDirectoryDialogOptions) (string, error) {
	dir := opts.DefaultDirectory
	if dir == "" {
		dir = "."
	}

	args := []string{"--getexistingdirectory", dir}
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
