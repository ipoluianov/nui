package ui

// FormState is where a window is: its place and size when it isn't
// maximized, and whether it is maximized. Saved when the application quits
// and restored when it starts, the window opens where the user left it (see
// RememberState).
type FormState struct {
	X, Y, Width, Height int
	Maximized           bool
}

// State returns where the window is. Of a maximized window it returns the
// place and the size it gets back when restored.
func (c *Form) State() FormState {
	state := FormState{X: c.posX, Y: c.posY, Width: c.width, Height: c.height}
	if c.wnd != nil {
		state.X, state.Y = c.wnd.PosX(), c.wnd.PosY()
		state.Maximized = c.wnd.IsMaximized()
	}
	if state.Maximized && c.normal.Width > 0 {
		state.X, state.Y, state.Width, state.Height = c.normal.X, c.normal.Y, c.normal.Width, c.normal.Height
	}
	return state
}

// SetState puts the window where the state says, before it is shown: its
// place (kept on the screen, should the screens have changed), its size,
// maximized. A state of no size does nothing.
func (c *Form) SetState(state FormState) {
	if state.Width <= 0 || state.Height <= 0 {
		return
	}
	c.posX, c.posY = state.X, state.Y
	c.SetSize(state.Width, state.Height)
	c.normal = FormState{X: state.X, Y: state.Y, Width: state.Width, Height: state.Height}
	c.maximizeOnShow = state.Maximized
	c.clampOnShow = true
}

// RememberState restores the window's state saved in the settings under the
// key, if any, and saves it there when the window closes (Settings.Save
// included):
//
//	settings, _ := ui.OpenSettings("myapp")
//	form.RememberState(settings, "main")
//	form.Show()
func (c *Form) RememberState(settings *Settings, key string) {
	var state FormState
	if settings.Get(key, &state) {
		c.SetState(state)
	}
	c.onClosing(func() {
		settings.Set(key, c.State())
		settings.Save()
	})
}

// rememberNormal keeps the place and the size of the window when it isn't
// maximized
func (c *Form) rememberNormal() {
	if c.wnd == nil || c.wnd.IsMaximized() {
		return
	}
	c.normal = FormState{X: c.wnd.PosX(), Y: c.wnd.PosY(), Width: c.width, Height: c.height}
}

// clampToScreen keeps the window (its title bar at least) on the screen the
// saved place is on, or the nearest one
func (c *Form) clampToScreen() {
	ax, ay, aw, ah := c.wnd.ScreenWorkArea(c.posX, c.posY)
	if aw <= 0 || ah <= 0 {
		return
	}
	c.posX = max(ax, min(c.posX, ax+aw-min(c.width, aw)))
	c.posY = max(ay, min(c.posY, ay+ah-min(c.height, ah)))
}
