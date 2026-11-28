package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestAuditToolButtonEnabledUnderDisabledParent(t *testing.T) {
	panel := NewPanel()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	b := NewToolButton(img, "tip", func() {})
	panel.AddWidget(0, 0, b)
	newTestForm(t, panel)
	panel.SetEnabled(false)
	b.SetEnabled(false) // e.g. the action of the button became disabled
	panel.SetEnabled(true)
	if b.Enabled() {
		t.Errorf("ToolButton.SetEnabled(false) under a disabled parent was ignored: enabled again with the parent")
	}
}

func TestAuditSliderStepOffGrid(t *testing.T) {
	s := NewSlider()
	newTestForm(t, s)
	s.SetRange(0, 10)
	s.SetStep(4)
	s.SetValue(10)
	t.Logf("SetValue(10) with step 4 in 0..10 -> %v", s.Value())
	s.SetValue(8)
	s.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
	t.Logf("ArrowRight from 8 -> %v", s.Value())
	s.ProcessKeyDown(KeyEnd, KeyModifiers{})
	t.Logf("End -> %v", s.Value())
	if v := s.Value(); math.Mod(v, 4) != 0 {
		t.Errorf("value %v is not on the step 4 from 0", v)
	}
}

func TestAuditAccessibleSliderNaN(t *testing.T) {
	s := NewSlider()
	s.SetRange(0, 10)
	f := newTestForm(t, s)
	n := f.Accessible().Find(RoleSlider, "")
	if n == nil {
		t.Skip("no slider node")
	}
	n.SetCurrent(math.NaN())
	t.Logf("after accessible SetCurrent(NaN): slider value %v", s.Value())
}

func TestAuditRoleString(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AccessibleRole(-1).String() panics: %v", r)
		}
	}()
	_ = AccessibleRole(-1).String()
}

func FuzzAuditRegisterFont(f *testing.F) {
	f.Add(fontJetBrainsMono, uint32(0), byte(0))
	f.Add(fontJetBrainsMono[:4096], uint32(100), byte(0xff))
	f.Fuzz(func(t *testing.T, data []byte, off uint32, b byte) {
		d := append([]byte(nil), data...)
		if len(d) > 0 {
			d[int(off)%len(d)] ^= b
		}
		if err := RegisterFont("zzfuzz", d); err != nil {
			return
		}
		MeasureText("zzfuzz", 14, "Hello, world 1234 {}")
		renderText("Hello W", color.RGBA{A: 255}, "zzfuzz", 14)
		MeasureTextOutline("Hello", "zzfuzz", 20)
		p := NewPath()
		p.Text(0, 0, "Hello", "zzfuzz", 20)
		cnv, _ := NewImageCanvas(64, 32, 1)
		cnv.SetFontFamily("zzfuzz")
		cnv.DrawText(0, 0, 64, 32, "Hi")
		cnv.FillPath(p, color.RGBA{A: 255})
	})
}
