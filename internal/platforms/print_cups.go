//go:build !windows

package platforms

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Printing on Linux and macOS goes through CUPS: the printers come from
// lpstat, and the document - a PDF made by nui - is sent with lp. There is no
// system print dialog without GTK or Qt (Linux) or AppKit's print panel
// (macOS), so nui shows its own.

// HasPrintDialog reports whether the system has a print dialog of its own
func HasPrintDialog() bool {
	return false
}

// PrintWithDialog is Windows only: elsewhere nui shows its own dialog
func PrintWithDialog(owner Window, req PrintRequest) error {
	return ErrNoPrinting
}

// Printers returns the CUPS printers that accept jobs, and the default one
func Printers() (names []string, defaultName string, err error) {
	if _, err := exec.LookPath("lpstat"); err != nil {
		return nil, "", fmt.Errorf("%w: lpstat not found (CUPS)", ErrNoPrinting)
	}
	out, _ := exec.Command("lpstat", "-e").Output()
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, line)
		}
	}
	sort.Strings(names)
	// "system default destination: name"
	out, _ = exec.Command("lpstat", "-d").Output()
	if _, after, ok := strings.Cut(string(out), ":"); ok {
		defaultName = strings.TrimSpace(after)
	}
	return names, defaultName, nil
}

// PrintFile sends the file to the CUPS printer with lp; options are lp's -o
// options, e.g. "media": "A4"
func PrintFile(printer, path string, copies int, title string, options map[string]string) error {
	args := []string{"-d", printer, "-n", strconv.Itoa(max(copies, 1))}
	if title != "" {
		args = append(args, "-t", title)
	}
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "-o", k+"="+options[k])
	}
	args = append(args, "--", path)
	out, err := exec.Command("lp", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("nui: lp: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
