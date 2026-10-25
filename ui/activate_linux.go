package ui

// activateWindow: Show does nothing for a shown window, so it is hidden
// first: mapping the window again brings it out of the minimized state, on
// top of the others. The window manager forgets the maximized state of a
// hidden window, so it is set again.
func activateWindow(c *Form) {
	maximized := c.IsMaximized()
	c.Hide()
	c.Show()
	if maximized {
		c.Maximize()
	}
}
