package ui

import (
	"image/color"
	"math"
)

// Slider chooses a number in a range by dragging a thumb along a track.
// A click on the track moves the thumb there; the arrows move it by a step,
// PageUp/PageDown by a page, Home/End to the ends; so does the mouse wheel.
//
//	volume := ui.NewSlider()
//	volume.SetRange(0, 100)
//	volume.SetStep(1)
//	volume.SetOnValueChanged(func() { player.SetVolume(volume.Value()) })
type Slider struct {
	Widget
	min, max     float64
	value        float64
	step         float64
	pageStep     float64
	tickInterval float64
	vertical     bool

	dragging       bool
	dragOffset     int // from the thumb's center to where it was grabbed
	onValueChanged func()
}

const sliderTrackThickness = 4

func NewSlider() *Slider {
	var c Slider
	c.InitWidget()
	c.SetTypeName("Slider")
	c.SetCanBeFocused(true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.mouseDown)
	c.SetOnMouseMove(c.mouseMove)
	c.SetOnMouseUp(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		c.dragging = false
		c.form.Update()
		return true
	})
	c.SetOnMouseWheel(func(deltaX, deltaY int) bool {
		if !c.Enabled() {
			return false
		}
		c.changeValue(c.value + float64(deltaY+deltaX)*c.keyStep())
		return true
	})
	c.max = 100
	c.setOrientation(false)
	return &c
}

// SetVertical puts the track upright, the minimum at the bottom.
func (c *Slider) SetVertical(vertical bool) {
	c.setOrientation(vertical)
	c.form.UpdateLayout()
}

func (c *Slider) IsVertical() bool {
	return c.vertical
}

func (c *Slider) setOrientation(vertical bool) {
	c.vertical = vertical
	if vertical {
		c.themeHeight = nil
		c.SetMinSize(ThemeControlHeight(), 80)
		c.SetMaxSize(ThemeControlHeight(), 10000)
		c.SetXExpandable(false)
		c.SetYExpandable(true)
	} else {
		c.SetMinWidth(100)
		c.SetMaxWidth(10000)
		c.SetXExpandable(true)
		c.SetYExpandable(false)
		c.setThemeHeight(ThemeControlHeight, true)
	}
}

// SetRange sets the minimum and the maximum; the value is kept in them.
func (c *Slider) SetRange(min, max float64) {
	if max < min {
		min, max = max, min
	}
	c.min, c.max = min, max
	c.value = c.snap(c.value)
	c.form.Update()
}

func (c *Slider) SetMin(min float64) { c.SetRange(min, math.Max(min, c.max)) }
func (c *Slider) SetMax(max float64) { c.SetRange(math.Min(c.min, max), max) }
func (c *Slider) Min() float64       { return c.min }
func (c *Slider) Max() float64       { return c.max }

// SetStep makes the value a multiple of step from the minimum; 0 (the
// default) allows any value, and the keys move by 1/100 of the range.
func (c *Slider) SetStep(step float64) {
	c.step = math.Max(0, step)
	c.value = c.snap(c.value)
	c.form.Update()
}

func (c *Slider) Step() float64 {
	return c.step
}

// SetPageStep sets how far PageUp/PageDown move; 0 (the default) is 1/10
// of the range.
func (c *Slider) SetPageStep(step float64) {
	c.pageStep = math.Max(0, step)
}

// SetTickInterval draws a tick every interval along the track; 0 draws none.
func (c *Slider) SetTickInterval(interval float64) {
	c.tickInterval = math.Max(0, interval)
	c.form.Update()
}

// SetValue sets the value, kept in the range and on the step, without
// calling SetOnValueChanged's function.
func (c *Slider) SetValue(value float64) {
	c.value = c.snap(value)
	c.form.Update()
}

func (c *Slider) Value() float64 {
	return c.value
}

// SetOnValueChanged sets the function called when the user changes the
// value, also while dragging.
func (c *Slider) SetOnValueChanged(f func()) {
	c.onValueChanged = f
}

func (c *Slider) snap(v float64) float64 {
	v = math.Max(c.min, math.Min(c.max, v))
	if c.step > 0 {
		v = c.min + math.Round((v-c.min)/c.step)*c.step
		v = math.Min(v, c.max)
	}
	return v
}

func (c *Slider) keyStep() float64 {
	if c.step > 0 {
		return c.step
	}
	return (c.max - c.min) / 100
}

func (c *Slider) page() float64 {
	if c.pageStep > 0 {
		return c.pageStep
	}
	return math.Max(c.keyStep(), (c.max-c.min)/10)
}

func (c *Slider) changeValue(v float64) {
	v = c.snap(v)
	if v == c.value {
		return
	}
	c.value = v
	c.form.Update()
	if c.onValueChanged != nil {
		c.onValueChanged()
	}
}

// Geometry: positions along the track, in pixels from the widget's start
// (left, or top for a vertical slider)

func (c *Slider) thumbRadius() int {
	return ThemeIndicatorSize() / 2
}

func (c *Slider) trackLength() int {
	if c.vertical {
		return c.Height()
	}
	return c.Width()
}

// posOf returns where the value is along the track.
func (c *Slider) posOf(v float64) int {
	r := c.thumbRadius() + 1
	length := c.trackLength() - r*2
	t := 0.0
	if c.max > c.min {
		t = (v - c.min) / (c.max - c.min)
	}
	if c.vertical {
		t = 1 - t
	}
	return r + int(math.Round(t*float64(length)))
}

// valueAt returns the value at a position along the track.
func (c *Slider) valueAt(pos int) float64 {
	r := c.thumbRadius() + 1
	length := c.trackLength() - r*2
	if length <= 0 {
		return c.min
	}
	t := math.Max(0, math.Min(1, float64(pos-r)/float64(length)))
	if c.vertical {
		t = 1 - t
	}
	return c.min + t*(c.max-c.min)
}

func (c *Slider) along(x, y int) int {
	if c.vertical {
		return y
	}
	return x
}

func (c *Slider) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft || !c.Enabled() {
		return true
	}
	c.Focus()
	pos := c.along(x, y)
	thumb := c.posOf(c.value)
	if abs(pos-thumb) <= c.thumbRadius()+2 {
		c.dragOffset = pos - thumb // grabbed the thumb: it doesn't jump
	} else {
		c.dragOffset = 0
		c.changeValue(c.valueAt(pos))
	}
	c.dragging = true
	return true
}

func (c *Slider) mouseMove(x, y int, mods KeyModifiers) bool {
	if c.dragging && c.form.mouseLeftButtonPressed {
		c.changeValue(c.valueAt(c.along(x, y) - c.dragOffset))
	}
	return true
}

func (c *Slider) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	if !c.Enabled() {
		return false
	}
	switch key {
	case KeyArrowRight, KeyArrowUp:
		c.changeValue(c.value + c.keyStep())
	case KeyArrowLeft, KeyArrowDown:
		c.changeValue(c.value - c.keyStep())
	case KeyPageUp:
		c.changeValue(c.value + c.page())
	case KeyPageDown:
		c.changeValue(c.value - c.page())
	case KeyHome:
		c.changeValue(c.min)
	case KeyEnd:
		c.changeValue(c.max)
	default:
		return false
	}
	return true
}

func (c *Slider) draw(cnv *Canvas) {
	p := CurrentPalette()
	enabled := c.Enabled()
	accent := p.Highlight
	rest := MixColors(p.Border, p.Window, 0.2)
	if !enabled {
		accent = MixColors(p.Highlight, p.Window, 0.55)
		rest = MixColors(p.Border, p.Window, 0.5)
	}

	r := c.thumbRadius()
	thumb := c.posOf(c.value)
	start, end := c.posOf(c.min), c.posOf(c.max)
	cross := c.Height() / 2
	if c.vertical {
		cross = c.Width() / 2
	}
	half := sliderTrackThickness / 2

	// rect draws along the track from a to b (in either order)
	rect := func(a, b, crossFrom, thickness int, col color.RGBA) {
		if a > b {
			a, b = b, a
		}
		if c.vertical {
			cnv.FillRoundedRectAA(crossFrom, a, thickness, b-a, thickness/2, col)
		} else {
			cnv.FillRoundedRectAA(a, crossFrom, b-a, thickness, thickness/2, col)
		}
	}
	rect(start, end, cross-half, sliderTrackThickness, rest)
	rect(start, thumb, cross-half, sliderTrackThickness, accent)

	if c.tickInterval > 0 && c.max > c.min {
		count := int((c.max - c.min) / c.tickInterval)
		if count <= 200 {
			for i := 0; i <= count; i++ {
				pos := c.posOf(c.min + float64(i)*c.tickInterval)
				if c.vertical {
					cnv.FillRect(cross+r+2, pos, 4, 1, p.Border)
				} else {
					cnv.FillRect(pos, cross+r+2, 1, 4, p.Border)
				}
			}
		}
	}

	// Thumb: a disc in the accent color with a light center, bigger when
	// hovered or dragged
	cx, cy := thumb, cross
	if c.vertical {
		cx, cy = cross, thumb
	}
	outer := r
	if enabled && (c.IsHovered() || c.dragging) {
		outer = r + 1
	}
	if c.IsFocused() && enabled {
		focus := withAlpha(p.Highlight, 70)
		cnv.FillRoundedRectAA(cx-outer-3, cy-outer-3, (outer+3)*2, (outer+3)*2, outer+3, focus)
	}
	cnv.FillRoundedRectAA(cx-outer, cy-outer, outer*2, outer*2, outer, accent)
	inner := outer - 4
	if inner > 0 {
		cnv.FillRoundedRectAA(cx-inner, cy-inner, inner*2, inner*2, inner, p.Base)
	}
}

func (c *Slider) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	if c.vertical {
		c.SetMinSize(ThemeControlHeight(), c.minHeight)
		c.SetMaxSize(ThemeControlHeight(), 10000)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
