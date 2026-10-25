// Package benchmarks measures the costs that decide whether an interface
// stays responsive: a key press in a big text, adding many widgets to a
// shown form, filling and painting a big table. The forms are shown in the
// headless mode (see uitest), so the numbers are the library's own, without
// the system's drawing.
//
//	go test ./ui/benchmarks -bench . -benchmem
//	go test ./ui/benchmarks -bench . -short    # without the 20 000 lines
//
// The numbers measured last are in doc/benchmarks.md.
package benchmarks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func lines(n int) string {
	var sb strings.Builder
	for i := range n {
		if i > 0 {
			sb.WriteByte('\n')
		}
		fmt.Fprintf(&sb, "%05d the quick brown fox jumps over the lazy dog", i)
	}
	return sb.String()
}

var textSizes = []int{500, 2000, 20000}

// textBoxForm shows a multiline text box with n lines, focused, the cursor
// at the start
func textBoxForm(b *testing.B, n int) (*uitest.Window, *ui.TextBox) {
	if n > 2000 && testing.Short() {
		b.Skip("-short")
	}
	form := ui.NewForm()
	form.SetSize(1000, 700)
	tb := ui.NewTextBox()
	tb.SetMultiline(true)
	form.Panel().AddWidget(0, 0, tb)
	w := uitest.Show(b, form)
	tb.SetText(lines(n))
	w.Click(uitest.At(tb, 10, 10))
	return w, tb
}

// BenchmarkTextBoxSetText: loading a text into a multiline text box
func BenchmarkTextBoxSetText(b *testing.B) {
	for _, n := range textSizes {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			w, tb := textBoxForm(b, n)
			text := lines(n)
			for b.Loop() {
				tb.SetText(text)
				w.Settle()
			}
		})
	}
}

// BenchmarkTextBoxKey: typing a character into a multiline text box, as the
// user does it (the key press, the character, the release), without the
// repaint
func BenchmarkTextBoxKey(b *testing.B) {
	for _, n := range textSizes {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			w, _ := textBoxForm(b, n)
			for b.Loop() {
				w.Type("x")
			}
		})
	}
}

// BenchmarkTextBoxPaint: one frame of the form with a multiline text box,
// e.g. a blink of the cursor
func BenchmarkTextBoxPaint(b *testing.B) {
	for _, n := range textSizes {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			w, _ := textBoxForm(b, n)
			for b.Loop() {
				w.Screenshot()
			}
		})
	}
}

// BenchmarkAddWidgets: adding 1000 labels to a shown form one by one, as a
// list built in code is
func BenchmarkAddWidgets(b *testing.B) {
	for b.Loop() {
		form := ui.NewForm()
		form.SetSize(800, 600)
		w := uitest.Show(b, form)
		panel := form.Panel()
		for i := range 1000 {
			panel.AddWidget(i, 0, ui.NewLabel(fmt.Sprint("Label ", i)))
		}
		w.Settle()
		form.Close()
	}
}

// BenchmarkLabelSetText: changing the text of one label on a form with 300
// labels, e.g. a status that follows the mouse, and laying the form out for
// the next frame
func BenchmarkLabelSetText(b *testing.B) {
	form := ui.NewForm()
	form.SetSize(800, 600)
	panel := form.Panel()
	labels := make([]*ui.Label, 300)
	for i := range labels {
		labels[i] = ui.NewLabel(fmt.Sprint("Label ", i))
		panel.AddWidget(i/3, i%3, labels[i])
	}
	w := uitest.Show(b, form)
	i := 0
	for b.Loop() {
		labels[0].SetText(fmt.Sprint("Status ", i))
		w.Settle()
		labels[0].Width() // lays the form out if it was asked for
		i++
	}
}

func tableForm(b *testing.B) (*uitest.Window, *ui.Table) {
	form := ui.NewForm()
	form.SetSize(1000, 700)
	t := ui.NewTable()
	form.Panel().AddWidget(0, 0, t)
	w := uitest.Show(b, form)
	return w, t
}

func fillTable(t *ui.Table, rows, cols int) {
	t.SetColumnCount(cols)
	t.SetRowCount(rows)
	for r := range rows {
		for c := range cols {
			t.SetCellText2(r, c, fmt.Sprint(r, ":", c))
		}
	}
}

// BenchmarkTableFill: filling a table of 100 000 rows by 5 columns
func BenchmarkTableFill(b *testing.B) {
	for b.Loop() {
		w, t := tableForm(b)
		fillTable(t, 100_000, 5)
		w.Settle()
	}
}

// BenchmarkTablePaint: a frame of the form with the 100 000-row table
func BenchmarkTablePaint(b *testing.B) {
	w, t := tableForm(b)
	fillTable(t, 100_000, 5)
	for b.Loop() {
		w.Screenshot()
	}
}

// BenchmarkEmptyFormPaint: a frame of an empty form of 2560x1440, the floor
// of every repaint
func BenchmarkEmptyFormPaint(b *testing.B) {
	uitest.SetScreen(3000, 2000, 1)
	defer uitest.SetScreen(1920, 1080, 1)
	form := ui.NewForm()
	form.SetSize(2560, 1440)
	w := uitest.Show(b, form)
	for b.Loop() {
		w.Screenshot()
	}
}
