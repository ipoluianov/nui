package platforms

import "strings"

func parentWindowID(parent Window) windowId {
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		return p.hwnd
	}
	return -1
}

// extensionsList flattens every filter's patterns into a single list of bare
// extensions (NSOpenPanel/NSSavePanel have no Windows/GTK-style grouped,
// switchable filter, just one allowed-extensions list). "*" or "*.*" means
// "any file" and is dropped, since that's the default when no filter is set
// at all.
func extensionsList(filters []FileDialogFilter) []string {
	seen := make(map[string]bool)
	var exts []string
	for _, f := range filters {
		for _, p := range f.Patterns {
			ext := strings.TrimPrefix(strings.TrimPrefix(p, "*"), ".")
			if ext == "" || ext == "*" {
				continue
			}
			if !seen[ext] {
				seen[ext] = true
				exts = append(exts, ext)
			}
		}
	}
	return exts
}

func openFileDialog(parent Window, opts OpenFileDialogOptions) ([]string, error) {
	exts := extensionsList(opts.Filters)
	return showOpenFileDialog(parentWindowID(parent), opts.Title, opts.DefaultDirectory, exts, opts.AllowMultiple), nil
}

func saveFileDialog(parent Window, opts SaveFileDialogOptions) (string, error) {
	exts := extensionsList(opts.Filters)
	return showSaveFileDialog(parentWindowID(parent), opts.Title, opts.DefaultDirectory, opts.DefaultFileName, exts), nil
}

func selectDirectoryDialog(parent Window, opts SelectDirectoryDialogOptions) (string, error) {
	return showSelectDirectoryDialog(parentWindowID(parent), opts.Title, opts.DefaultDirectory), nil
}
