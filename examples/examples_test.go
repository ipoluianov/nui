package examples

import "testing"

// Every application builds its window and lays it out without a panic.
func TestAppsBuild(t *testing.T) {
	names := map[string]bool{}
	for _, app := range Apps {
		if names[app.Name] {
			t.Errorf("duplicate name %q", app.Name)
		}
		names[app.Name] = true
		t.Run(app.Name, func(t *testing.T) {
			if app.Name == "tasks" {
				// Its tray icon is created on the UI thread, which a test
				// doesn't run
				t.Skip("needs the UI thread")
			}
			form := app.NewForm()
			form.SetSize(1000, 700)
			form.UpdateLayout()
		})
	}
}
