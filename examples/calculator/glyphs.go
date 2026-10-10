package calculator

import (
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

// The UI font has no "√" and "⌫", so these keys draw their symbols.

const glyphSize = 20

// glyphSqrt is "√x".
func glyphSqrt(dc *gg.Context) {
	dc.MoveTo(1, 9)
	dc.LineTo(3, 8)
	dc.LineTo(5.5, 14)
	dc.LineTo(8.5, 2.5)
	dc.LineTo(15.5, 2.5)
	dc.Stroke()
	dc.DrawLine(10, 6.5, 14.5, 12.5)
	dc.DrawLine(14.5, 6.5, 10, 12.5)
	dc.Stroke()
}

// glyphBackspace is a key cap pointing left with a cross.
func glyphBackspace(dc *gg.Context) {
	dc.MoveTo(0.5, 8)
	dc.LineTo(4.5, 3.5)
	dc.LineTo(15, 3.5)
	dc.LineTo(15, 12.5)
	dc.LineTo(4.5, 12.5)
	dc.ClosePath()
	dc.Stroke()
	dc.DrawLine(7.5, 6, 11.5, 10)
	dc.DrawLine(11.5, 6, 7.5, 10)
	dc.Stroke()
}

// setGlyph makes the button show the glyph, in the color of its text.
func setGlyph(b *ui.Button, draw func(dc *gg.Context)) {
	var img image.Image
	var imgColor color.RGBA
	b.SetOnPostPaint(func(cnv *ui.Canvas) {
		col := ui.ThemeForegroundColor("")
		if !b.Enabled() {
			col = ui.ThemeForegroundColorDisabled()
		}
		if img == nil || col != imgColor {
			imgColor = col
			img = icons.Draw(glyphSize, func(dc *gg.Context) {
				dc.SetColor(col)
				dc.SetLineWidth(1.3)
				dc.SetLineCapRound()
				dc.SetLineJoinRound()
				draw(dc)
			})
		}
		cnv.DrawImage((b.Width()-glyphSize)/2, (b.Height()-glyphSize)/2, img)
	})
}
