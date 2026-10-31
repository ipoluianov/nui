package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"math"
	"testing"
)

var (
	red   = color.RGBA{255, 0, 0, 255}
	black = color.RGBA{0, 0, 0, 255}
)

// newScaledTestCanvas is a black canvas of w x h logical pixels at the
// scale, clipped to itself
func newScaledTestCanvas(w, h int, scale float64) (*Canvas, *image.RGBA) {
	img := image.NewRGBA(image.Rect(0, 0, int(float64(w)*scale+0.5), int(float64(h)*scale+0.5)))
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	cnv := NewCanvasScaled(img, scale)
	cnv.SetDirectTranslateAndClip(0, 0, w, h)
	return cnv, img
}

// litOutside returns the pixels lit (not black) outside the rectangle
func litOutside(img *image.RGBA, r image.Rectangle) []image.Point {
	var lit []image.Point
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if !(image.Point{x, y}).In(r) && img.RGBAAt(x, y) != black {
				lit = append(lit, image.Point{x, y})
			}
		}
	}
	return lit
}

// An opaque FillRect sets exactly its pixels to the color, within the clip
func TestFillRectOpaque(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	cnv.Save()
	cnv.TranslateAndClip(5, 5, 10, 10)
	cnv.FillRect(-3, 2, 8, 4, red) // sticks out of the clip on the left
	cnv.Restore()
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			want := black
			if x >= 5 && x < 10 && y >= 7 && y < 11 {
				want = red
			}
			if got := img.RGBAAt(x, y); got != want {
				t.Fatalf("pixel %d,%d = %v, want %v", x, y, got, want)
			}
		}
	}
}

// Rectangles side by side touch without a gap or an overlap at a
// fractional scale
func TestFillRectScaledTouch(t *testing.T) {
	for _, scale := range []float64{1.25, 1.5, 1.75} {
		cnv, img := newScaledTestCanvas(30, 4, scale)
		half := color.RGBA{255, 0, 0, 128}
		for x := 0; x < 30; x += 3 {
			cnv.FillRect(x, 0, 3, 4, half)
		}
		for x := 0; x < img.Rect.Dx(); x++ {
			if got := img.RGBAAt(x, 1).R; got != img.RGBAAt(0, 1).R {
				t.Fatalf("scale %v: column %d = %d, column 0 = %d", scale, x, got, img.RGBAAt(0, 1).R)
			}
		}
	}
}

// A horizontal or vertical line of width 1 covers the pixels from the first
// point to the last, without the last
func TestDrawLineStraight(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	cnv.DrawLine(3, 4, 12, 4, 1, white)
	cnv.DrawLine(15, 2, 15, 9, 1, white)
	if lit := litOutside(img, image.Rect(0, 0, 0, 0)); len(lit) != 9+7 {
		t.Fatalf("%d pixels lit, want %d", len(lit), 9+7)
	}
	for x := 3; x < 12; x++ {
		if img.RGBAAt(x, 4).R < 254 {
			t.Fatalf("pixel %d,4 not lit", x)
		}
	}
	for y := 2; y < 9; y++ {
		if img.RGBAAt(15, y).R < 254 {
			t.Fatalf("pixel 15,%d not lit", y)
		}
	}
}

// An opaque DrawRect draws the border of its width x height pixels only
func TestDrawRectOpaque(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		cnv, img := newScaledTestCanvas(20, 20, scale)
		cnv.SetColor(white)
		cnv.DrawRect(3, 4, 10, 6)
		s := int(scale)
		inner := image.Rect((3+1)*s, (4+1)*s, (3+10-1)*s, (4+6-1)*s)
		outer := image.Rect(3*s, 4*s, (3+10)*s, (4+6)*s)
		for y := 0; y < img.Rect.Dy(); y++ {
			for x := 0; x < img.Rect.Dx(); x++ {
				p := image.Point{x, y}
				border := p.In(outer) && !p.In(inner)
				if lit := img.RGBAAt(x, y).R >= 254; lit != border {
					t.Fatalf("scale %v: pixel %d,%d lit %v, want %v", scale, x, y, lit, border)
				}
			}
		}
	}
}

// Restore brings back the translation, the clip and the drawing state
func TestSaveRestore(t *testing.T) {
	cnv, _ := newTestCanvas(50, 50)
	cnv.SetColor(red)
	cnv.SetFontSize(11)
	cnv.TranslateAndClip(2, 3, 40, 40)
	before := *cnv.state
	cnv.Save()
	cnv.TranslateAndClip(5, 5, 10, 10)
	cnv.SetColor(white)
	cnv.SetFontSize(20)
	cnv.SetUnderline(true)
	cnv.Save()
	cnv.TranslateAndClip(1, 1, 2, 2)
	cnv.Restore()
	cnv.Restore()
	if *cnv.state != before {
		t.Fatalf("state %+v, want %+v", *cnv.state, before)
	}
	cnv.Restore() // more than saved: ignored
	if *cnv.state != before {
		t.Fatal("an extra Restore changed the state")
	}
}

// DrawImage draws only the part of the image within the clip
func TestDrawImageClip(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	cnv.TranslateAndClip(4, 4, 6, 6)
	cnv.DrawImage(-2, -2, src)
	if lit := litOutside(img, image.Rect(4, 4, 10, 10)); len(lit) != 0 {
		t.Fatalf("lit outside the clip: %v", lit)
	}
	if img.RGBAAt(4, 4) != white || img.RGBAAt(9, 9) != white {
		t.Fatal("the image is not drawn inside the clip")
	}
}

// No line leaves the clip, whatever its direction and width: the edge
// pixels of the clip belong to the widget, the next ones to its neighbors
func TestDrawLineStaysInClip(t *testing.T) {
	for _, scale := range []float64{1, 1.5, 2} {
		cnv, img := newScaledTestCanvas(40, 40, scale)
		cnv.TranslateAndClip(10, 10, 10, 10)
		clip := cnv.ClipRect()
		for _, l := range [][5]int{
			{10, -5, 10, 15, 1}, // vertical, just right of the clip
			{-5, 10, 15, 10, 1}, // horizontal, just below
			{9, -5, 9, 15, 1},   // vertical, on the last column
			{-5, 9, 15, 9, 1},   // horizontal, on the last row
			{0, 0, 15, 9, 1},    // diagonal toward the corner
			{-5, 15, 15, -5, 1}, // diagonal across
			{0, 0, 15, 9, 2},    // thick
			{-5, 5, 15, 5, 3},   // thick horizontal
			{5, -5, 5, 15, 3},   // thick vertical
			{0, 9, 30, 9, 2},    // thick on the last row
			{12, 0, 12, 9, 1},   // vertical, outside on the right
		} {
			cnv.DrawLine(l[0], l[1], l[2], l[3], l[4], white)
		}
		if lit := litOutside(img, clip); len(lit) != 0 {
			t.Errorf("scale %v: %d pixels outside the clip %v, e.g. %v", scale, len(lit), clip, lit[0])
		}
	}
}

// With an empty clip (a widget scrolled out of view) nothing is drawn
func TestDrawEmptyClip(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	cnv.TranslateAndClip(5, 5, 0, 0)
	cnv.SetColor(white)
	cnv.DrawLine(0, 0, 0, 5, 1, white)
	cnv.DrawLine(0, 0, 5, 3, 1, white)
	cnv.DrawLine(0, 0, 5, 0, 2, white)
	cnv.DrawRect(0, 0, 4, 4)
	cnv.FillRect(0, 0, 4, 4, white)
	if lit := litOutside(img, image.Rect(0, 0, 0, 0)); len(lit) != 0 {
		t.Fatalf("%d pixels drawn, e.g. %v", len(lit), lit[0])
	}
}

// DrawRect stays within the clip too
func TestDrawRectStaysInClip(t *testing.T) {
	cnv, img := newTestCanvas(30, 30)
	cnv.TranslateAndClip(10, 10, 10, 10)
	cnv.SetColor(color.RGBA{255, 255, 255, 128})
	cnv.DrawRect(-3, -3, 20, 20)
	cnv.DrawRect(5, 5, 6, 6)
	if lit := litOutside(img, image.Rect(10, 10, 20, 20)); len(lit) != 0 {
		t.Fatalf("%d pixels outside the clip, e.g. %v", len(lit), lit[0])
	}
}

// drawTestText draws "Text" in white with the underline or not
func drawTestText(underline bool) *image.RGBA {
	cnv, img := newTestCanvas(80, 40)
	cnv.SetColor(white)
	cnv.SetFontFamily(FontFamilySans)
	cnv.SetFontSize(14)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetUnderline(underline)
	cnv.DrawText(5, 0, 70, 40, "Text")
	return img
}

// An underlined text has a line under it as wide as the text; the text
// itself is the same
func TestDrawTextUnderline(t *testing.T) {
	plain := drawTestText(false)
	underlined := drawTestText(true)
	textWidth, textHeight, _ := MeasureText(FontFamilySans, 14, "Text")

	var added []image.Point
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			if plain.RGBAAt(x, y) != underlined.RGBAAt(x, y) {
				added = append(added, image.Point{x, y})
			}
		}
	}
	if len(added) == 0 {
		t.Fatal("no underline")
	}
	minX, maxX, minY, maxY := 80, 0, 40, 0
	for _, p := range added {
		minX, maxX = min(minX, p.X), max(maxX, p.X)
		minY, maxY = min(minY, p.Y), max(maxY, p.Y)
	}
	if minX != 5 || maxX != 5+textWidth-1 {
		t.Errorf("underline from x %d to %d, want 5 to %d", minX, maxX, 5+textWidth-1)
	}
	top := 20 - textHeight/2
	if maxY-minY > 1 || minY < top+textHeight/2 || maxY >= top+textHeight {
		t.Errorf("underline at y %d..%d, want a line in the lower half of the text (%d..%d)", minY, maxY, top, top+textHeight)
	}
}

// DrawText of a canvas whose color was never set draws in black, not panics
func TestDrawTextWithoutColor(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 20))
	cnv := NewCanvas(img)
	cnv.SetDirectTranslateAndClip(0, 0, 50, 20)
	cnv.SetFontFamily(FontFamilySans)
	cnv.SetFontSize(12)
	cnv.DrawText(0, 0, 50, 20, "abc")
	cnv.DrawRect(1, 1, 5, 5)
}

// DrawText leaves the translation and the clip as they were
func TestDrawTextKeepsState(t *testing.T) {
	cnv, _ := newTestCanvas(50, 50)
	cnv.SetColor(white)
	cnv.TranslateAndClip(3, 4, 40, 40)
	before := *cnv.state
	cnv.SetFontFamily(FontFamilySans)
	cnv.SetFontSize(12)
	before.fontFamily, before.fontSize = cnv.state.fontFamily, cnv.state.fontSize
	cnv.DrawText(5, 6, 20, 20, "one\ntwo")
	if *cnv.state != before {
		t.Fatalf("state %+v, want %+v", *cnv.state, before)
	}
}

// A line of no length draws nothing, of any width (a thick one hung: its
// direction was NaN)
func TestDrawLineZeroLength(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		for _, width := range []int{1, 2, 3} {
			cnv, img := newScaledTestCanvas(20, 20, scale)
			cnv.DrawLine(5, 5, 5, 5, width, white)
			if lit := litOutside(img, image.Rect(0, 0, 0, 0)); len(lit) != 0 {
				t.Errorf("scale %v, width %d: %d pixels lit", scale, width, len(lit))
			}
		}
	}
}

// A straight line going past the clip covers it up to its last pixel: the
// corners of the frames drawn with such lines are closed
func TestDrawLineReachesClipEdge(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		cnv, img := newScaledTestCanvas(40, 40, scale)
		cnv.TranslateAndClip(10, 10, 10, 10)
		cnv.DrawLine(-5, 3, 30, 3, 1, white) // horizontal, through both sides
		cnv.DrawLine(4, -5, 4, 30, 1, white) // vertical, through the top and the bottom
		s := int(scale)
		for i := 10 * s; i < 20*s; i++ {
			if img.RGBAAt(i, (10+3)*s).R < 254 {
				t.Errorf("scale %v: horizontal line misses pixel %d", scale, i)
			}
			if img.RGBAAt((10+4)*s, i).R < 254 {
				t.Errorf("scale %v: vertical line misses pixel %d", scale, i)
			}
		}
	}
}

// drawOver gives what draw.Draw with draw.Over gives, to the bit, also for
// the pixels of no premultiplied color and over translucent ones
func TestDrawOverMatchesDraw(t *testing.T) {
	rnd := uint32(1)
	next := func() byte {
		rnd = rnd*1664525 + 1013904223
		return byte(rnd >> 24)
	}
	src := image.NewRGBA(image.Rect(-3, 2, 37, 25))
	for i := 0; i < len(src.Pix); i += 4 {
		switch next() % 4 {
		case 0: // transparent
		case 1: // opaque
			src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = next(), next(), next(), 255
		case 2: // premultiplied
			a := next()
			src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = next()%(a+1), next()%(a+1), next()%(a+1), a
		default: // anything
			src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = next(), next(), next(), next()
		}
	}
	for _, c := range []struct {
		r  image.Rectangle
		sp image.Point
	}{
		{image.Rect(0, 0, 40, 23), image.Pt(-3, 2)},
		{image.Rect(-10, -5, 20, 10), image.Pt(0, 5)}, // sticks out of dst
		{image.Rect(30, 15, 60, 40), image.Pt(-3, 2)}, // and of src
		{image.Rect(5, 5, 6, 6), image.Pt(36, 24)},    // one pixel
		{image.Rect(5, 5, 9, 9), image.Pt(100, 100)},  // outside src
	} {
		want := image.NewRGBA(image.Rect(0, 0, 50, 30))
		for i := range want.Pix {
			want.Pix[i] = next()
		}
		got := image.NewRGBA(want.Rect)
		copy(got.Pix, want.Pix)
		draw.Draw(want, c.r, src, c.sp, draw.Over)
		drawOver(got, c.r, src, c.sp)
		if !bytes.Equal(got.Pix, want.Pix) {
			t.Errorf("r %v sp %v: differs from draw.Draw", c.r, c.sp)
		}
	}
}

// An opaque color is drawn as it is: 255 stays 255
func TestBlendOpaqueExact(t *testing.T) {
	cnv, img := newTestCanvas(20, 20)
	cnv.DrawLine(2, 2, 15, 2, 1, white)
	cnv.MixPixel(5, 5, white)
	cnv.SetColor(white)
	cnv.DrawRect(2, 8, 6, 6)
	for _, p := range []image.Point{{2, 2}, {14, 2}, {5, 5}, {2, 8}, {7, 13}} {
		if got := img.RGBAAt(p.X, p.Y); got != white {
			t.Errorf("pixel %v = %v, want white", p, got)
		}
	}
}

// Half a color over black is half of it, whatever type the color is of; a
// transparent color changes nothing
func TestBlendColorTypes(t *testing.T) {
	for _, col := range []color.Color{
		color.RGBA{255, 0, 0, 128},
		color.NRGBA{255, 0, 0, 128},
		color.NRGBA64{0xffff, 0, 0, 0x8080},
	} {
		cnv, img := newTestCanvas(10, 10)
		cnv.FillRect(0, 0, 4, 4, col)
		cnv.DrawLine(0, 6, 8, 6, 1, col)
		cnv.MixPixel(8, 8, col)
		for _, p := range []image.Point{{1, 1}, {3, 6}, {8, 8}} {
			if got := img.RGBAAt(p.X, p.Y); got != (color.RGBA{128, 0, 0, 255}) {
				t.Errorf("%T: pixel %v = %v, want 128, 0, 0", col, p, got)
			}
		}
	}
	cnv, img := newTestCanvas(10, 10)
	cnv.FillRect(0, 0, 10, 10, color.RGBA{50, 100, 150, 255})
	cnv.FillRect(0, 0, 10, 10, color.RGBA{255, 255, 255, 0})
	if got := img.RGBAAt(5, 5); got != (color.RGBA{50, 100, 150, 255}) {
		t.Errorf("a transparent fill changed the pixel to %v", got)
	}
}

// The corners of a translucent rectangle are blended once, as its sides
func TestDrawRectTranslucentCorners(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		cnv, img := newScaledTestCanvas(20, 20, scale)
		cnv.SetColor(color.RGBA{255, 255, 255, 128})
		cnv.DrawRect(3, 3, 10, 6)
		s := int(scale)
		side, corner := img.RGBAAt(6*s, 3*s), img.RGBAAt(3*s, 3*s)
		if side != corner || side.R != 128 {
			t.Errorf("scale %v: side %v, corner %v, want both 128", scale, side, corner)
		}
	}
}

// A diagonal or thick line of a translucent color is as translucent as a
// straight one, not opaque
func TestDrawLineTranslucent(t *testing.T) {
	half := color.RGBA{255, 255, 255, 128}
	cnv, img := newTestCanvas(30, 30)
	cnv.DrawLine(0, 0, 20, 20, 1, half)
	cnv.DrawLine(2, 25, 25, 25, 3, half)
	for _, p := range []image.Point{{10, 10}, {12, 25}} {
		if got := img.RGBAAt(p.X, p.Y).R; got < 100 || got > 140 {
			t.Errorf("pixel %v = %d, want about 128", p, got)
		}
	}
}

// A horizontal or a vertical wide line is width whole pixels across, its
// ends included, as sharp as FillRect
func TestDrawWideLineStraight(t *testing.T) {
	for _, c := range []struct {
		width       int
		top, bottom int // rows of a horizontal line at y 10
	}{
		{2, 10, 11},
		{3, 9, 11},
		{4, 9, 12},
	} {
		cnv, img := newTestCanvas(30, 30)
		cnv.DrawLine(5, 10, 20, 10, c.width, white)
		cnv.DrawLine(25, 3, 25, 18, c.width, white)
		for y := 0; y < 30; y++ {
			for x := 0; x < 22; x++ {
				want := x >= 5 && x <= 20 && y >= c.top && y <= c.bottom
				if got := img.RGBAAt(x, y) == white; got != want {
					t.Fatalf("width %d: pixel %d,%d lit %v, want %v", c.width, x, y, got, want)
				}
			}
		}
		left := 25 - (c.width-1)/2
		for y := 3; y <= 18; y++ {
			for x := left; x < left+c.width; x++ {
				if img.RGBAAt(x, y) != white {
					t.Fatalf("width %d: vertical pixel %d,%d not lit", c.width, x, y)
				}
			}
		}
	}
}

// A wide line of any direction has no gaps, nothing far from it, and covers
// as much as it should: its width along all of it, half circles at the ends
func TestDrawWideLineAnyAngle(t *testing.T) {
	for w := 2; w <= 6; w++ {
		for length := 3; length <= 40; length += 6 {
			for deg := 0; deg < 360; deg += 3 {
				a := float64(deg) * math.Pi / 180
				x1, y1 := 50, 50
				x2 := x1 + int(math.Round(float64(length)*math.Cos(a)))
				y2 := y1 + int(math.Round(float64(length)*math.Sin(a)))
				dx, dy := float64(x2-x1), float64(y2-y1)
				l := math.Hypot(dx, dy)
				if l == 0 {
					continue
				}
				cnv, img := newTestCanvas(100, 100)
				cnv.DrawLine(x1, y1, x2, y2, w, white)

				covered := 0.0
				for y := 0; y < 100; y++ {
					for x := 0; x < 100; x++ {
						v := img.RGBAAt(x, y).R
						if v == 0 {
							continue
						}
						covered += float64(v) / 255
						tt := math.Max(0, math.Min(1, (float64(x-x1)*dx+float64(y-y1)*dy)/(l*l)))
						if d := math.Hypot(float64(x)-(float64(x1)+tt*dx), float64(y)-(float64(y1)+tt*dy)); d > float64(w)/2+1.5 {
							t.Fatalf("w %d, %d°, length %d: pixel %d,%d is %.1f from the line", w, deg, length, x, y, d)
						}
					}
				}
				for i := 1; i < int(l); i++ {
					px := float64(x1) + dx*float64(i)/l
					py := float64(y1) + dy*float64(i)/l
					if img.RGBAAt(int(math.Round(px)), int(math.Round(py))).R < 200 {
						t.Fatalf("w %d, %d°, length %d: gap at %.1f,%.1f", w, deg, length, px, py)
					}
				}
				straight := x1 == x2 || y1 == y2
				want := float64(w)*l + math.Pi*float64(w*w)/4
				if straight {
					want = float64(w) * (l + 1)
				}
				if math.Abs(covered-want) > want*0.1 {
					t.Fatalf("w %d, %d°, length %d: covers %.1f pixels, want about %.1f", w, deg, length, covered, want)
				}
			}
		}
	}
}

// In a frame that repaints a part of the window ClipRect is the clip cut
// to the part, and PixelOrigin stays where the whole frame has it
func TestClipRectOfPart(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 2} {
		cnv, _ := newScaledTestCanvas(100, 100, scale)
		cnv.TranslateAndClip(13, 7, 50, 40)
		origin, clip := cnv.PixelOrigin(), cnv.ClipRect()
		if origin != clip.Min {
			t.Errorf("scale %g: origin %v, clip %v", scale, origin, clip)
		}
		part := image.Rect(clip.Min.X+1, clip.Min.Y+20, clip.Max.X+30, clip.Max.Y+30)
		cnv.setPart(part)
		if got := cnv.PixelOrigin(); got != origin {
			t.Errorf("scale %g: origin %v in the part, %v in the whole frame", scale, got, origin)
		}
		if got, want := cnv.ClipRect(), clip.Intersect(part); got != want {
			t.Errorf("scale %g: clip %v in the part, want %v", scale, got, want)
		}
	}
}
