package main

import (
	"fmt"
	"os"

	"github.com/ipoluianov/nui/examples"
	"github.com/ipoluianov/nui/ui"
)

// Without arguments: the launcher of the example applications. With the
// name of one (go run . notepad): only that application.
//
// The platform features that are off by default are switched on for the
// checks of doc/platform-checks.md by the environment:
//
//	NUI_NATIVE_MENU=1  the menu bars in the menu at the top of the screen (macOS)
//	NUI_MAC_IME=1      the input method of macOS (accents, Chinese, emoji)
func main() {
	if os.Getenv("NUI_NATIVE_MENU") == "1" {
		ui.UseNativeMenuBar(true)
	}
	if os.Getenv("NUI_MAC_IME") == "1" {
		ui.UseMacInputMethod(true)
	}
	if len(os.Args) < 2 {
		examples.Run()
		return
	}
	app := examples.FindApp(os.Args[1])
	if app == nil {
		fmt.Fprintf(os.Stderr, "unknown example %q; available:\n", os.Args[1])
		for _, a := range examples.Apps {
			fmt.Fprintf(os.Stderr, "  %-12s %s\n", a.Name, a.Description)
		}
		os.Exit(2)
	}
	examples.RunApp(app)
}
