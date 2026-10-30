package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// newRepaintTestForm makes a form with a widget of each kind, in a few
// states, laid out
func newRepaintTestForm() *Form {
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.Local)
	form := NewForm()
	form.processResize(1100, 760)
	p := form.Panel()

	col1 := p.AddPanel(0, 0)
	col1.AddLabel(0, 0, "Label")
	col1.AddButton(1, 0, "Button", nil)
	checked := NewCheckbox("Checked")
	checked.SetChecked(true)
	col1.AddWidget(2, 0, checked)
	col1.AddWidget(3, 0, NewCheckbox("Unchecked"))
	radio := NewRadioButton("Radio on")
	radio.SetChecked(true)
	col1.AddWidget(4, 0, radio)
	toggle := NewToggleSwitch("Toggle")
	toggle.SetChecked(true)
	col1.AddWidget(5, 0, toggle)
	combo := NewComboBox()
	combo.SetItems([]string{"First item", "Second"})
	combo.SetSelectedIndex(0)
	col1.AddWidget(6, 0, combo)
	slider := NewSlider()
	slider.SetValue(30)
	col1.AddWidget(7, 0, slider)
	col1.AddWidget(8, 0, NewProgressBar(0, 100, 60))
	text := NewTextBox()
	text.SetText("Selected text")
	text.SelectAllText()
	col1.AddWidget(9, 0, text)
	col1.AddWidget(10, 0, NewLink("A link", nil))
	datePicker := NewDatePicker()
	datePicker.SetDate(date)
	col1.AddWidget(11, 0, datePicker)

	col2 := p.AddPanel(0, 1)
	group := NewGroupBox("Group box")
	group.AddLabel(0, 0, "Inside the group")
	col2.AddWidget(0, 0, group)
	tabs := NewTabWidget()
	tabs.AddPage("Tab one", NewLabel("Page one"))
	tabs.AddPage("Tab two", NewLabel("Page two"))
	col2.AddWidget(1, 0, tabs)
	markdown := NewMarkdownView()
	markdown.SetMarkdown("# Title\n\nSome **bold**, *italic*, `code` and a [link](http://example.com).\n\n- one\n- two\n")
	col2.AddWidget(2, 0, markdown)
	tree := NewTreeView()
	root := tree.AddNode(nil, "Root")
	tree.AddNode(root, "Child 1")
	tree.ExpandAll()
	col2.AddWidget(3, 0, tree)
	calendar := NewCalendar()
	calendar.SetDate(date)
	col2.AddWidget(4, 0, calendar)

	col3 := p.AddPanel(0, 2)
	table := NewTable()
	table.SetColumnCount(3)
	table.SetRowCount(30)
	for r := 0; r < 30; r++ {
		for c := 0; c < 3; c++ {
			table.SetCellText2(r, c, fmt.Sprint(r, ":", c))
		}
	}
	col3.AddWidget(0, 0, table)
	chart := NewChart()
	var points []ChartPoint
	for i := 0; i < 50; i++ {
		x := float64(i) / 5
		points = append(points, ChartPoint{X: x, Y: x * x / 10})
	}
	chart.SetData(points)
	col3.AddWidget(1, 0, chart)
	col3.AddWidget(2, 0, NewColorPicker())
	form.UpdateLayout()
	return form
}

// What is painted within a clip is what the whole paint has there: the
// frames painted in part rely on it
func TestPaintClipped(t *testing.T) {
	bg := color.RGBA{240, 240, 240, 255}
	for _, scale := range []float64{1, 1.5, 2} {
		form := newRepaintTestForm()
		r := image.Rect(0, 0, int(1100*scale+0.5), int(760*scale+0.5))
		whole := image.NewRGBA(r)
		fillPixels(whole, r, bg)
		form.paintClipped(whole, scale, r)
		rnd := rand.New(rand.NewSource(1))
		failures := 0
		for i := 0; i < 100 && failures < 5; i++ {
			x0, y0 := rnd.Intn(r.Dx()), rnd.Intn(r.Dy())
			clip := image.Rect(x0, y0, x0+1+rnd.Intn(300), y0+1+rnd.Intn(200)).Intersect(r)
			img := image.NewRGBA(r)
			copy(img.Pix, whole.Pix)
			// What was there must not show through
			fillPixels(img, clip, color.RGBA{255, 0, 255, 255})
			fillPixels(img, clip, bg)
			form.paintClipped(img, scale, clip)
			if diff, n := pixelDiff(img, whole); n > 0 {
				failures++
				logical := image.Rect(int(float64(diff.Min.X)/scale), int(float64(diff.Min.Y)/scale),
					int(float64(diff.Max.X)/scale)+1, int(float64(diff.Max.Y)/scale)+1)
				t.Errorf("scale %v, clip %v: %d pixels differ in %v; widgets there: %s",
					scale, clip, n, diff, strings.Join(form.widgetsIn(logical), ", "))
			}
		}
	}
}

// pixelDiff returns the bounds of the pixels that differ and their count
func pixelDiff(a, b *image.RGBA) (image.Rectangle, int) {
	var diff image.Rectangle
	n := 0
	if bytes.Equal(a.Pix, b.Pix) {
		return diff, 0
	}
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] || a.Pix[i+3] != b.Pix[i+3] {
			n++
			x, y := i/4%a.Rect.Dx(), i/4/a.Rect.Dx()
			diff = diff.Union(image.Rect(x, y, x+1, y+1))
		}
	}
	return diff, n
}

// The shapes a View2D draws in the transform of its drawing - dashed
// lines, markers - are the same within any clip
func TestPaintClippedView2D(t *testing.T) {
	for _, mode := range []string{"dash", "marker", "plain"} {
		view := NewView2D()
		view.SetBackground(color.RGBA{30, 30, 40, 255})
		view.SetPainter(func(p *View2DPaint) {
			hl := color.RGBA{60, 140, 255, 255}
			sel := NewPath()
			for i := 0; i < 20; i++ {
				sel.Rect(float64(i*37%300), float64(i*53%200), 40, 25)
			}
			switch mode {
			case "dash":
				p.Canvas.StrokePath(sel, StrokeStyle{Width: 2, Dash: []float64{6, 3}, NonScaling: true}, hl)
			case "plain":
				p.Canvas.StrokePath(sel, StrokeStyle{Width: 2, NonScaling: true}, hl)
			case "marker":
				p.InScreen(func(cnv *Canvas) {
					for i := 0; i < 40; i++ {
						cnv.DrawMarker(MarkerSquare, float64(i*37%800)+0.3, float64(i*53%600)+0.7, 8, hl, color.RGBA{255, 255, 255, 255})
					}
				})
			}
		})
		form := newTestForm(t, view)
		view.SetView(PointF{X: 150, Y: 100}, 1.7)
		r := image.Rect(0, 0, 500, 400)
		whole := image.NewRGBA(r)
		form.paintClipped(whole, 1, r)
		rnd := rand.New(rand.NewSource(1))
		for i := 0; i < 100; i++ {
			x0, y0 := rnd.Intn(500), rnd.Intn(400)
			clip := image.Rect(x0, y0, x0+1+rnd.Intn(200), y0+1+rnd.Intn(200)).Intersect(r)
			img := image.NewRGBA(r)
			copy(img.Pix, whole.Pix)
			fillPixels(img, clip, color.RGBA{})
			form.paintClipped(img, 1, clip)
			if d, n := pixelDiff(img, whole); n > 0 {
				t.Errorf("%s: clip %v: %d differ in %v", mode, clip, n, d)
				break
			}
		}
	}
}
