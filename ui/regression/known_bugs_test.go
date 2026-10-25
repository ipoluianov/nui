// Package regression holds the tests of the known bugs of nui
// (doc/production-readiness.md, section 5). Each test reproduces its bug:
// while the bug is there, the test is skipped with its number; once the bug
// is fixed, the test fails, asking to turn it into an ordinary test (drop
// the knownBug call and keep the check). So a fix is noticed, and a bug that
// comes back is caught.
package regression

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// knownBug checks the known bug number n: reproduced, it skips the test;
// not reproduced, it fails it
func knownBug(t *testing.T, n int, reproduced bool, what string) {
	t.Helper()
	if reproduced {
		t.Skipf("known bug #%d: %s", n, what)
	}
	t.Errorf("known bug #%d seems fixed (%s): make this an ordinary test", n, what)
}

func newForm(width, height int) *ui.Form {
	form := ui.NewForm()
	form.SetSize(width, height)
	return form
}

// #3: the release of the right (and the middle) button doesn't reach the
// widget
func TestBug03RightButtonUp(t *testing.T) {
	form := newForm(300, 200)
	target := ui.NewPanel()
	target.SetMinSize(100, 100)
	ups := 0
	target.SetOnMouseUp(func(button ui.MouseButton, x, y int, mods ui.KeyModifiers) bool {
		if button == ui.MouseButtonRight {
			ups++
		}
		return true
	})
	form.Panel().AddWidget(0, 0, target)
	w := uitest.Show(t, form)
	w.RightClick(target)
	knownBug(t, 3, ups == 0, "no right button release")
}

// #4: a disabled container doesn't disable its children
func TestBug04DisabledParent(t *testing.T) {
	form := newForm(300, 200)
	panel := form.Panel().AddPanel(0, 0)
	clicks := 0
	btn := panel.AddButton(0, 0, "Click", func() { clicks++ })
	panel.SetEnabled(false)
	w := uitest.Show(t, form)
	w.Click(btn)
	knownBug(t, 4, clicks > 0, "the button in a disabled panel is clicked")
}

// #5: a multiline text box turns expandable on any property change
func TestBug05TextBoxExpandable(t *testing.T) {
	form := newForm(300, 200)
	tb := ui.NewTextBox()
	tb.SetMultiline(true)
	form.Panel().AddWidget(0, 0, tb)
	uitest.Show(t, form)
	tb.SetXExpandable(false)
	tb.SetYExpandable(false)
	tb.SetProp("placeholder", "x")
	knownBug(t, 5, tb.XExpandable() || tb.YExpandable(), "SetProp turned it expandable")
}

// paintBox is a widget the test paints itself
func paintBox(paint func(cnv *ui.Canvas)) *ui.Widget {
	w := &ui.Widget{}
	w.InitWidget()
	w.SetMinSize(100, 60)
	w.SetOnPaint(paint)
	return w
}

// #6: Canvas.DrawImage ignores the Min of the image's bounds: a sub-image is
// drawn from the wrong place
func TestBug06DrawImageSubImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))
	red, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				src.SetRGBA(x, y, red)
			} else {
				src.SetRGBA(x, y, blue)
			}
		}
	}
	sub := src.SubImage(image.Rect(10, 0, 20, 10)) // the blue half
	form := newForm(200, 100)
	box := paintBox(func(cnv *ui.Canvas) { cnv.DrawImage(0, 0, sub) })
	form.Panel().AddWidget(0, 0, box)
	w := uitest.Show(t, form)
	x, y := box.RectClientAreaOnWindow()
	got := w.Screenshot().RGBAAt(x+2, y+2)
	knownBug(t, 6, got != blue, "the blue half is drawn as "+colorString(got))
}

// #7: Canvas.DrawText breaks lines only at "\r\n", not at "\n"
func TestBug07DrawTextNewline(t *testing.T) {
	inkBelow := func(text string) int {
		form := newForm(200, 120)
		box := paintBox(func(cnv *ui.Canvas) {
			cnv.SetColor(color.Black)
			cnv.SetFontSize(14)
			cnv.SetVAlign(ui.VAlignTop)
			cnv.DrawText(0, 0, 100, 60, text)
		})
		form.Panel().AddWidget(0, 0, box)
		w := uitest.Show(t, form)
		x, y := box.RectClientAreaOnWindow()
		img := w.Screenshot()
		form.Close()
		// The ink of the second line: below the first one
		dark := 0
		for py := y + 22; py < y+50; py++ {
			for px := x; px < x+100; px++ {
				if c := img.RGBAAt(px, py); c.R < 128 {
					dark++
				}
			}
		}
		return dark
	}
	crlf, lf := inkBelow("A\r\nB"), inkBelow("A\nB")
	if crlf == 0 {
		t.Fatal("no second line even with \\r\\n: the check is broken")
	}
	knownBug(t, 7, lf == 0, `"\n" doesn't break the line`)
}

// #10: Table.CopySelectionToClipboard copies only the current cell
func TestBug10TableCopySelection(t *testing.T) {
	form := newForm(400, 300)
	table := ui.NewTable()
	table.SetMultiselect(true)
	table.SetColumnCount(2)
	table.SetRowCount(2)
	table.SetCellText2(0, 0, "a")
	table.SetCellText2(0, 1, "b")
	table.SetCellText2(1, 0, "c")
	table.SetCellText2(1, 1, "d")
	form.Panel().AddWidget(0, 0, table)
	uitest.Show(t, form)
	table.SetCurrentCell2(0, 0)
	table.SelectAll()
	table.CopySelectionToClipboard()
	text, _ := ui.ClipboardGetText()
	knownBug(t, 10, !strings.Contains(text, "d"), "copied "+strings.ReplaceAll(text, "\n", `\n`))
}

// #11: the global key handler gets the keys while a popup is open
func TestBug11GlobalKeysUnderPopup(t *testing.T) {
	form := newForm(300, 200)
	combo := ui.NewComboBox()
	combo.AddItem("One", nil)
	combo.AddItem("Two", nil)
	form.Panel().AddWidget(0, 0, combo)
	global := 0
	form.SetOnGlobalKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
		if key == ui.KeyF5 {
			global++
		}
		return false
	})
	w := uitest.Show(t, form)
	w.Click(combo)
	if w.Popup() == nil {
		t.Fatal("the list didn't open: the check is broken")
	}
	w.Key(ui.KeyF5, ui.KeyModifiers{})
	knownBug(t, 11, global > 0, "F5 reached the global handler over the open list")
}

// #12: the documentation describes functions that don't exist
func TestBug12DocsNonexistentAPI(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "doc", "ui", "content", "*.md"))
	if len(files) == 0 {
		t.Skip("no doc/ui/content")
	}
	var found []string
	for _, f := range files {
		data, _ := os.ReadFile(f)
		for _, name := range []string{"ShowEnterStringDialog", "ShowAboutDialog", "UpdateMainForm"} {
			if strings.Contains(string(data), name) {
				found = append(found, filepath.Base(f)+": "+name)
			}
		}
	}
	knownBug(t, 12, len(found) > 0, strings.Join(found, ", "))
}

// #14: a Label is as high as a control whatever its font size: big text is
// cut off at the bottom
func TestBug14LabelHeight(t *testing.T) {
	form := newForm(400, 200)
	label := ui.NewLabel("Big text")
	label.SetFontSize(40)
	form.Panel().AddWidget(0, 0, label)
	form.Panel().AddVSpacer(1, 0)
	uitest.Show(t, form)
	_, textHeight, _ := ui.MeasureText(label.FontFamily(), label.FontSize(), "Big text")
	knownBug(t, 14, label.Height() < textHeight, "label "+itoa(label.Height())+" px, text "+itoa(textHeight)+" px")
}

func itoa(n int) string { return strconv.Itoa(n) }

func colorString(c color.RGBA) string { return fmt.Sprint(c) }
