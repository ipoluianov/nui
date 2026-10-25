package ui

import "github.com/ipoluianov/nui/internal/platforms"

// Activate brings the form's window to the front and gives it the focus:
// a minimized window is restored (a maximized one stays maximized), one
// under the others comes on top. For a second start of the application
// that asks the running one to show itself (see AcquireSingleInstance), or
// a notification clicked in the tray. A hidden form is shown.
func (c *Form) Activate() {
	platforms.RunOnUIThread(func() {
		if c.wnd == nil || c.closed || platforms.IsHeadless() {
			c.Show()
			return
		}
		activateWindow(c)
	})
}
