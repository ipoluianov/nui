package main

import (
	"fmt"
	"os"

	"github.com/ipoluianov/nui/examples"
)

// Without arguments: the launcher of the example applications. With the
// name of one (go run . notepad): only that application.
func main() {
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
