package ui

// ToggleSwitch is an on/off switch with a text, for settings that apply at
// once: a click, Space or Enter flips it. The knob slides to its new side.
//
//	wifi := ui.NewToggleSwitch("Wi-Fi")
//	wifi.SetOnStateChanged(func() { network.SetWifi(wifi.Checked()) })
type ToggleSwitch struct {
	Widget
	text           string
	checked        bool
	onStateChanged func()
	// knob is where the knob is drawn: 0 off, 1 on; it moves towards the
	// state on the timer
	knob float64
}

const (
	toggleAnimationStep = 0.25
	toggleTextGap       = 10
)

func NewToggleSwitch(text string) *ToggleSwitch {
	var c ToggleSwitch
	c.InitWidget()
	c.SetTypeName("ToggleSwitch")
	c.SetMaxWidth(10000)
	c.setThemeHeight(ThemeControlHeight, true)
	c.SetMouseCursor(MouseCursorPointer)
	c.SetCanBeFocused(true)
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool { return true })
	c.SetOnMouseUp(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		if button == MouseButtonLeft && x >= 0 && x < c.Width() && y >= 0 && y < c.Height() && c.Enabled() {
			c.toggle()
		}
		return true
	})
	c.AddTimer(15, c.animate)
	c.SetText(text)
	return &c
}

func (c *ToggleSwitch) Text() string {
	return c.text
}

func (c *ToggleSwitch) SetText(text string) {
	c.text = text
	c.updateMinWidth()
	c.form.UpdateLayout()
}

func (c *ToggleSwitch) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

func (c *ToggleSwitch) Checked() bool {
	return c.checked
}

// SetChecked sets the state without calling SetOnStateChanged's function
// and without the animation.
func (c *ToggleSwitch) SetChecked(checked bool) {
	c.checked = checked
	c.knob = 0
	if checked {
		c.knob = 1
	}
	c.form.Update()
}

// SetOnStateChanged sets the function called when the user flips the switch.
func (c *ToggleSwitch) SetOnStateChanged(f func()) {
	c.onStateChanged = f
}

func (c *ToggleSwitch) toggle() {
	c.checked = !c.checked
	c.form.Update()
	if c.onStateChanged != nil {
		c.onStateChanged()
	}
}

func (c *ToggleSwitch) animate() {
	target := 0.0
	if c.checked {
		target = 1
	}
	if c.knob == target {
		return
	}
	if c.knob < target {
		c.knob = min(target, c.knob+toggleAnimationStep)
	} else {
		c.knob = max(target, c.knob-toggleAnimationStep)
	}
	c.form.Update()
}

func (c *ToggleSwitch) ProcessKeyDown(key Key, mods KeyModifiers) bool {
	if c.onKeyDown != nil && c.onKeyDown(key, mods) {
		return true
	}
	if (key == KeySpace || key == KeyEnter) && c.Enabled() {
		c.toggle()
		return true
	}
	return false
}

// trackSize is the size of the pill the knob slides in.
func (c *ToggleSwitch) trackSize() (int, int) {
	h := ThemeIndicatorSize() + 4
	return h * 9 / 5, h
}

func (c *ToggleSwitch) updateMinWidth() {
	w, _ := c.trackSize()
	textWidth, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.text)
	if err != nil || c.text == "" {
		textWidth = -toggleTextGap
	}
	c.SetMinWidth(w + toggleTextGap + textWidth + 2)
}

func (c *ToggleSwitch) draw(cnv *Canvas) {
	p := CurrentPalette()
	w, h := c.trackSize()
	y := (c.Height() - h) / 2
	enabled := c.Enabled()

	off := MixColors(p.Border, p.Window, 0.1)
	track := MixColors(off, p.Highlight, c.knob)
	knobColor := MixColors(p.Base, p.HighlightedText, c.knob)
	if !enabled {
		track = MixColors(track, p.Window, 0.5)
		knobColor = MixColors(knobColor, p.Window, 0.3)
	} else if c.IsHovered() {
		track = hoverColor(track, p.Text)
	}
	if c.IsFocused() && enabled {
		cnv.FillRoundedRectAA(0, y-2, w+4, h+4, (h+4)/2, withAlpha(p.Highlight, 70))
		cnv.FillRoundedRectAA(2, y, w, h, h/2, track)
	} else {
		cnv.FillRoundedRectAA(2, y, w, h, h/2, track)
	}

	const inset = 3
	d := h - inset*2
	knobX := 2 + inset + int(float64(w-inset*2-d)*c.knob+0.5)
	cnv.FillRoundedRectAA(knobX, y+inset, d, d, d/2, knobColor)

	textX := 2 + w + toggleTextGap
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(indicatorTextColor(&c.Widget))
	cnv.SetFontFamily(c.FontFamily())
	cnv.SetFontSize(c.FontSize())
	cnv.DrawText(textX, 0, c.Width()-textX, c.Height(), c.text)
}

func (c *ToggleSwitch) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.updateMinWidth()
}
