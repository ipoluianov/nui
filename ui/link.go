package ui

import (
	"os/exec"
	"runtime"
)

// Link is a clickable text, like a hyperlink: underlined under the mouse,
// activated by a click, Enter or Space. It calls its function, or opens its
// URL in the browser when it has no function.
//
//	help := ui.NewLink("Documentation", nil)
//	help.SetURL("https://example.com/docs")
type Link struct {
	Widget
	text    string
	url     string
	onClick func()
	visited bool
}

func NewLink(text string, onClick func()) *Link {
	var c Link
	c.InitWidget()
	c.SetTypeName("Link")
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetCanBeFocused(true)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool { return true })
	c.SetOnMouseUp(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		if button == MouseButtonLeft && x >= 0 && x < c.textWidth() && y >= 0 && y < c.Height() {
			c.Activate()
		}
		return true
	})
	c.onClick = onClick
	c.SetText(text)
	return &c
}

func (c *Link) Text() string {
	return c.text
}

func (c *Link) SetText(text string) {
	c.text = text
	c.SetMinWidth(c.textWidth() + 2)
	c.SetMaxWidth(c.textWidth() + 2)
	c.form.UpdateLayout()
}

func (c *Link) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

// SetURL sets the address opened in the browser when the link has no
// function; it's also shown as the tooltip unless one is set.
func (c *Link) SetURL(url string) {
	c.url = url
	c.form.Update()
}

func (c *Link) URL() string {
	return c.url
}

// SetOnClick sets the function called when the link is activated.
func (c *Link) SetOnClick(f func()) {
	c.onClick = f
}

// Visited reports whether the link was activated; it's drawn in a softer color.
func (c *Link) Visited() bool {
	return c.visited
}

// Activate does what a click does.
func (c *Link) Activate() {
	if !c.Enabled() {
		return
	}
	c.visited = true
	c.form.Update()
	switch {
	case c.onClick != nil:
		c.onClick()
	case c.url != "":
		_ = OpenURL(c.url)
	}
}

func (c *Link) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	if key == KeyEnter || key == KeySpace {
		c.Activate()
		return true
	}
	return false
}

func (c *Link) textWidth() int {
	w, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.text)
	if err != nil {
		return 0
	}
	return w
}

func (c *Link) draw(cnv *Canvas) {
	p := CurrentPalette()
	col := p.Link
	switch {
	case !c.Enabled():
		col = p.DisabledText
	case c.visited:
		col = MixColors(p.Link, p.Text, 0.3)
	}
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(col)
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.DrawText(0, 0, c.Width(), c.Height(), c.text)

	w := c.textWidth()
	_, textHeight, _ := MeasureText(c.FontFamily(), c.FontSize(), "Ag")
	underlineY := (c.Height()+textHeight)/2 - 2
	if c.Enabled() && (c.IsHovered() || c.IsFocused()) {
		cnv.FillRect(0, underlineY, w, 1, col)
	}
	if c.IsFocused() {
		cnv.SetColor(withAlpha(p.Highlight, 120))
		cnv.DrawRect(0, 1, min(w+2, c.Width()), c.Height()-2)
	}
}

func (c *Link) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.SetMinWidth(c.textWidth() + 2)
	c.SetMaxWidth(c.textWidth() + 2)
}

// OpenURL opens the address (a web page, a file, "mailto:...") with the
// system's default application.
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
