package ui

import (
	"image"
	"image/color"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
)

// ToastKind is the look of a toast: the color of its stripe
type ToastKind int

const (
	ToastInfo ToastKind = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// ToastDefaultDuration is how long ShowToast keeps a toast on the screen
const ToastDefaultDuration = 3 * time.Second

const (
	toastMaxVisible = 5
	toastMaxWidth   = 360
	toastMargin     = 16
	toastGap        = 8
	toastPaddingX   = 14
	toastPaddingY   = 10
	toastStripe     = 4
)

// toast is a short message in the bottom-right corner of the form that
// disappears by itself, or when clicked. It is shown in a native popup
// window, so it may stick out of a small form; where the platform has no
// popups, it's drawn inside the form.
type toast struct {
	lines   []string
	kind    ToastKind
	expires time.Time
	w, h    int
	// x, y is the position in the form's client coordinates
	x, y  int
	popup platforms.PopupWindow
}

// ShowToast shows a short message in the bottom-right corner of the form for
// ToastDefaultDuration. A click on it closes it at once. The newest toast is
// at the bottom; the oldest ones go away when there are too many.
func (c *Form) ShowToast(text string, kind ToastKind) {
	c.ShowToastFor(text, kind, ToastDefaultDuration)
}

// ShowToastFor is ShowToast with the time the toast stays on the screen
func (c *Form) ShowToastFor(text string, kind ToastKind, duration time.Duration) {
	if c == nil {
		return
	}
	t := &toast{kind: kind, expires: time.Now().Add(duration)}
	t.lines = wrapTextLines(text, ThemeFontFamily(), ThemeFontSize(), toastMaxWidth-toastPaddingX*2-toastStripe)
	textWidth := 0
	for _, line := range t.lines {
		w, _, err := MeasureText(ThemeFontFamily(), ThemeFontSize(), line)
		if err == nil && w > textWidth {
			textWidth = w
		}
	}
	t.w = min(toastMaxWidth, textWidth+toastPaddingX*2+toastStripe)
	t.h = len(t.lines)*ThemeLineHeight() + toastPaddingY*2

	c.toasts = append(c.toasts, t)
	for len(c.toasts) > toastMaxVisible {
		c.closeToast(c.toasts[0])
	}
	c.layoutToasts()
}

// ShowToast shows a toast in the form of the widget (see Form.ShowToast)
func ShowToast(w Widgeter, text string, kind ToastKind) {
	if w == nil || w.Form() == nil {
		return
	}
	w.Form().ShowToast(text, kind)
}

// layoutToasts stacks the toasts up from the bottom-right corner, the newest
// at the bottom, and moves their windows there.
func (c *Form) layoutToasts() {
	y := c.height - toastMargin
	for i := len(c.toasts) - 1; i >= 0; i-- {
		t := c.toasts[i]
		y -= t.h
		t.x = c.width - toastMargin - t.w
		t.y = y
		y -= toastGap
		c.showToastPopup(t)
	}
	c.Update()
}

// showToastPopup shows the toast's window at its place. Without popups the
// toast is drawn inside the form (see toastsPaint).
func (c *Form) showToastPopup(t *toast) {
	if c.wnd == nil || c.toastPopupsUnavailable {
		return
	}
	if t.popup == nil {
		t.popup = platforms.CreatePopupWindow(c.wnd, true)
		if t.popup == nil {
			c.toastPopupsUnavailable = true
			return
		}
		t.popup.OnPaint(func(rgba *image.RGBA) {
			cnv := NewCanvasScaled(rgba, t.popup.Scale())
			cnv.SetDirectTranslateAndClip(0, 0, t.w, t.h)
			// The window is rectangular: no rounded corners
			drawToast(cnv, t, 0, 0, 0)
		})
		t.popup.OnMouseButtonDown(func(btn MouseButton, x, y int) {
			c.closeToast(t)
			c.layoutToasts()
		})
	}
	screenX, screenY := c.wnd.ClientToScreen(t.x, t.y)
	t.popup.ShowAt(screenX, screenY, t.w, t.h)
}

func (c *Form) closeToast(t *toast) {
	for i, other := range c.toasts {
		if other == t {
			c.toasts = append(c.toasts[:i], c.toasts[i+1:]...)
			break
		}
	}
	if t.popup != nil {
		t.popup.Close()
		t.popup = nil
	}
	c.Update()
}

// closeToasts closes all the toasts, e.g. before the form's window goes away
func (c *Form) closeToasts() {
	for len(c.toasts) > 0 {
		c.closeToast(c.toasts[0])
	}
}

// toastsProcessTimer closes the expired toasts
func (c *Form) toastsProcessTimer() {
	if len(c.toasts) == 0 {
		return
	}
	now := time.Now()
	closed := false
	for _, t := range append([]*toast(nil), c.toasts...) {
		if now.After(t.expires) {
			c.closeToast(t)
			closed = true
		}
	}
	if closed {
		c.layoutToasts()
	}
}

// toastsProcessMouseDown closes the toast drawn inside the form under the
// mouse; returns true if there was one. Toasts in popups get their own clicks.
func (c *Form) toastsProcessMouseDown(x, y int) bool {
	for _, t := range c.toasts {
		if t.popup == nil && x >= t.x && x < t.x+t.w && y >= t.y && y < t.y+t.h {
			c.closeToast(t)
			c.layoutToasts()
			return true
		}
	}
	return false
}

// toastsPaint draws the toasts that have no popup window, over the form
func (c *Form) toastsPaint(cnv *Canvas) {
	for _, t := range c.toasts {
		if t.popup == nil {
			drawToast(cnv, t, t.x, t.y, themeControlRadius)
		}
	}
}

func toastAccent(kind ToastKind) color.RGBA {
	p := CurrentPalette()
	switch kind {
	case ToastSuccess:
		return p.Success
	case ToastWarning:
		return p.Warning
	case ToastError:
		return p.Error
	}
	return p.Highlight
}

func drawToast(cnv *Canvas, t *toast, x, y, radius int) {
	p := CurrentPalette()
	cnv.FillFrame(x, y, t.w, t.h, radius, p.PopupBase, p.Border)
	cnv.FillRect(x+1, y+1, toastStripe, t.h-2, toastAccent(t.kind))

	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(p.Text)
	cnv.SetFontFamily(ThemeFontFamily())
	cnv.SetFontSize(ThemeFontSize())
	lineHeight := ThemeLineHeight()
	textX := x + toastStripe + toastPaddingX
	for i, line := range t.lines {
		cnv.DrawText(textX, y+toastPaddingY+i*lineHeight, t.w-toastStripe-toastPaddingX*2, lineHeight, line)
	}
}
