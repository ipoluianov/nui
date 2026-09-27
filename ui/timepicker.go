package ui

import (
	"fmt"
	"time"
)

// TimePicker edits a time of day, 24-hour, in segments: HH:MM or HH:MM:SS.
// Left/Right (or a click) choose the segment, Up/Down and the mouse wheel
// change it, digits type it (two digits, or one that can't start a
// two-digit value, move to the next segment); the arrows at the right
// change the segment too.
//
//	start := ui.NewTimePicker()
//	start.SetTime(9, 30, 0)
//	start.SetOnTimeChanged(func() { meeting.Start = start.Duration() })
type TimePicker struct {
	Widget
	values      [3]int // hours, minutes, seconds
	showSeconds bool
	segment     int
	// typed is the first digit of a two-digit value being typed, -1 if none
	typed         int
	hoverSpin     int // -1 none, 0 up, 1 down
	onTimeChanged func()
}

var timePickerMax = [3]int{23, 59, 59}

func NewTimePicker() *TimePicker {
	var c TimePicker
	c.InitWidget()
	c.SetTypeName("TimePicker")
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetCanBeFocused(true)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseMove(func(x, y int, mods KeyModifiers) bool {
		spin := c.spinAt(x, y)
		if spin != c.hoverSpin {
			c.hoverSpin = spin
			c.form.Update()
		}
		cursor := MouseCursorIBeam
		if spin >= 0 {
			cursor = MouseCursorPointer
		}
		c.SetMouseCursor(cursor)
		return true
	})
	c.SetOnMouseLeave(func() { c.hoverSpin = -1 })
	c.SetOnMouseWheel(func(deltaX, deltaY int) bool {
		if c.Enabled() {
			c.stepSegment(deltaY)
		}
		return true
	})
	c.SetOnChar(c.onChar)
	c.SetOnFocusLost(func() { c.typed = -1 })
	c.typed = -1
	c.hoverSpin = -1
	c.updateWidth()
	return &c
}

// SetTime sets the time without calling SetOnTimeChanged's function; the
// values are kept in their ranges.
func (c *TimePicker) SetTime(hour, minute, second int) {
	c.values = [3]int{clampInt(hour, 0, 23), clampInt(minute, 0, 59), clampInt(second, 0, 59)}
	if !c.showSeconds {
		c.values[2] = 0
	}
	c.form.Update()
}

func (c *TimePicker) Hour() int   { return c.values[0] }
func (c *TimePicker) Minute() int { return c.values[1] }
func (c *TimePicker) Second() int { return c.values[2] }

// Duration returns the time as the time since midnight.
func (c *TimePicker) Duration() time.Duration {
	return time.Duration(c.values[0])*time.Hour + time.Duration(c.values[1])*time.Minute + time.Duration(c.values[2])*time.Second
}

// SetDuration sets the time from the time since midnight.
func (c *TimePicker) SetDuration(d time.Duration) {
	d %= 24 * time.Hour
	if d < 0 {
		d += 24 * time.Hour
	}
	c.SetTime(int(d/time.Hour), int(d%time.Hour/time.Minute), int(d%time.Minute/time.Second))
}

// SetShowSeconds adds the seconds segment.
func (c *TimePicker) SetShowSeconds(show bool) {
	c.showSeconds = show
	if !show {
		c.values[2] = 0
		c.segment = min(c.segment, 1)
	}
	c.updateWidth()
	c.form.UpdateLayout()
}

// SetOnTimeChanged sets the function called when the user changes the time.
func (c *TimePicker) SetOnTimeChanged(f func()) {
	c.onTimeChanged = f
}

// Text returns the time as shown: "09:30" or "09:30:00".
func (c *TimePicker) Text() string {
	if c.showSeconds {
		return fmt.Sprintf("%02d:%02d:%02d", c.values[0], c.values[1], c.values[2])
	}
	return fmt.Sprintf("%02d:%02d", c.values[0], c.values[1])
}

func (c *TimePicker) segments() int {
	if c.showSeconds {
		return 3
	}
	return 2
}

func (c *TimePicker) setValue(seg, v int) {
	v = clampInt(v, 0, timePickerMax[seg])
	if c.values[seg] == v {
		return
	}
	c.values[seg] = v
	c.form.Update()
	if c.onTimeChanged != nil {
		c.onTimeChanged()
	}
}

// stepSegment changes the current segment by delta, wrapping around.
func (c *TimePicker) stepSegment(delta int) {
	c.typed = -1
	n := timePickerMax[c.segment] + 1
	c.setValue(c.segment, ((c.values[c.segment]+delta)%n+n)%n)
}

// ---------------------------------------------------------------- Geometry

func (c *TimePicker) digitsWidth() int {
	w, _, err := MeasureText(c.FontFamily(), c.FontSize(), "00")
	if err != nil {
		return 20
	}
	return w + 2
}

func (c *TimePicker) colonWidth() int {
	w, _, err := MeasureText(c.FontFamily(), c.FontSize(), ":")
	if err != nil {
		return 4
	}
	return w + 2
}

func (c *TimePicker) segmentX(seg int) int {
	return themeTextInset + seg*(c.digitsWidth()+c.colonWidth())
}

func (c *TimePicker) spinWidth() int {
	return c.Height() * 3 / 4
}

func (c *TimePicker) updateWidth() {
	w := c.segmentX(c.segments()-1) + c.digitsWidth() + themeTextInset + c.spinWidth()
	c.SetMinWidth(w)
	c.SetMaxWidth(w + 40)
}

// spinAt returns 0 over the up arrow, 1 over the down arrow, -1 elsewhere.
func (c *TimePicker) spinAt(x, y int) int {
	if x < c.Width()-c.spinWidth() || x >= c.Width() || y < 0 || y >= c.Height() {
		return -1
	}
	if y < c.Height()/2 {
		return 0
	}
	return 1
}

// ---------------------------------------------------------------- Input

func (c *TimePicker) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft || !c.Enabled() {
		return true
	}
	c.Focus()
	switch c.spinAt(x, y) {
	case 0:
		c.stepSegment(1)
		return true
	case 1:
		c.stepSegment(-1)
		return true
	}
	c.typed = -1
	for seg := c.segments() - 1; seg >= 0; seg-- {
		if x >= c.segmentX(seg)-c.colonWidth()/2 {
			c.segment = seg
			break
		}
	}
	c.segment = max(c.segment, 0)
	c.form.Update()
	return true
}

func (c *TimePicker) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	if !c.Enabled() {
		return false
	}
	switch key {
	case KeyArrowUp:
		c.stepSegment(1)
	case KeyArrowDown:
		c.stepSegment(-1)
	case KeyPageUp:
		c.stepSegment(10)
	case KeyPageDown:
		c.stepSegment(-10)
	case KeyArrowLeft:
		c.typed = -1
		c.segment = max(0, c.segment-1)
	case KeyArrowRight:
		c.typed = -1
		c.segment = min(c.segments()-1, c.segment+1)
	case KeyHome:
		c.typed = -1
		c.setValue(c.segment, 0)
	case KeyEnd:
		c.typed = -1
		c.setValue(c.segment, timePickerMax[c.segment])
	case KeyBackspace, KeyDelete:
		c.typed = -1
		c.setValue(c.segment, 0)
	default:
		return false
	}
	c.form.Update()
	return true
}

// onChar types digits: the first digit is shown at once; a second one
// completes the value; then the next segment is chosen. A separator
// (":", ".", space) moves to the next segment too.
func (c *TimePicker) onChar(char rune, mods KeyModifiers) bool {
	if !c.Enabled() || mods.Ctrl || mods.Alt {
		return false
	}
	switch {
	case char >= '0' && char <= '9':
		d := int(char - '0')
		maxV := timePickerMax[c.segment]
		if c.typed >= 0 {
			c.setValue(c.segment, min(c.typed*10+d, maxV))
			c.typed = -1
			c.nextSegment()
		} else {
			c.setValue(c.segment, d)
			if d*10 > maxV {
				c.nextSegment() // no two-digit value starts with it
			} else {
				c.typed = d
			}
		}
	case char == ':' || char == '.' || char == ' ':
		c.typed = -1
		c.nextSegment()
	default:
		return false
	}
	c.form.Update()
	return true
}

func (c *TimePicker) nextSegment() {
	if c.segment < c.segments()-1 {
		c.segment++
	}
}

// ---------------------------------------------------------------- Painting

func (c *TimePicker) draw(cnv *Canvas) {
	p := CurrentPalette()
	fill, border := inputFrameColors(&c.Widget)
	cnv.FillFrame(0, 0, c.Width(), c.Height(), themeControlRadius, fill, border)

	textColor := colorToRGBA(c.ForegroundColor())
	if !c.Enabled() {
		textColor = p.DisabledText
	}
	focused := c.IsFocused()
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.SetVAlign(VAlignCenter)
	dw, cw := c.digitsWidth(), c.colonWidth()
	for seg := 0; seg < c.segments(); seg++ {
		x := c.segmentX(seg)
		col := textColor
		if focused && seg == c.segment {
			cnv.FillRoundedRectAA(x, 4, dw, c.Height()-8, themeControlRadius, p.Highlight)
			col = p.HighlightedText
		}
		text := fmt.Sprintf("%02d", c.values[seg])
		if focused && seg == c.segment && c.typed >= 0 {
			text = fmt.Sprintf("%d", c.typed) // the first digit of a value being typed
		}
		cnv.SetHAlign(HAlignCenter)
		cnv.SetColor(col)
		cnv.DrawText(x, 0, dw, c.Height(), text)
		if seg < c.segments()-1 {
			cnv.SetColor(textColor)
			cnv.DrawText(x+dw, 0, cw, c.Height(), ":")
		}
	}

	// Spin arrows
	sw := c.spinWidth()
	sx := c.Width() - sw
	cnv.FillRect(sx, 3, 1, c.Height()-6, p.Divider)
	for i, up := range []bool{true, false} {
		top := 1 + i*(c.Height()/2)
		h := c.Height()/2 - 1
		if c.hoverSpin == i && c.Enabled() {
			cnv.FillRect(sx+1, top, sw-2, h, hoverColor(fill, p.Text))
		}
		cx, cy := sx+sw/2, top+h/2
		const hw, hh = 4, 3
		if up {
			cnv.FillTriangle(cx-hw, cy+hh/2+1, cx+hw, cy+hh/2+1, cx, cy-hh/2-1, textColor)
		} else {
			cnv.FillTriangle(cx-hw, cy-hh/2-1, cx+hw, cy-hh/2-1, cx, cy+hh/2+1, textColor)
		}
	}
}

func (c *TimePicker) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.updateWidth()
}

func clampInt(v, lo, hi int) int {
	return max(lo, min(hi, v))
}
