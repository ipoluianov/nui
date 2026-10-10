package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
	"weak"
)

// Tab with nothing focused (a fresh form, or after a click on a label)
// focuses the first focusable widget instead of panicking
func TestTabWithoutFocus(t *testing.T) {
	tb := NewTextBox()
	form := newTestForm(t, tb)
	form.Panel().AddWidget(2, 0, NewLabel("label"))
	if form.FocusedWidget() != nil {
		t.Fatal("a fresh form has a focused widget")
	}
	form.processKeyDown(KeyTab, KeyModifiers{})
	if form.FocusedWidget() != Widgeter(tb) {
		t.Errorf("focused after Tab: %v", form.FocusedWidget())
	}
}

// A removed widget loses the focus: keys don't go to it any more, and the
// next click elsewhere doesn't touch it
func TestRemoveFocusedWidget(t *testing.T) {
	label := NewLabel("label")
	form := newTestForm(t, label)
	box := NewPanel()
	tb := NewTextBox()
	box.AddWidget(0, 0, tb)
	form.Panel().AddWidget(2, 0, box)
	form.UpdateLayout()

	focusLost := 0
	tb.SetOnFocusLost(func() { focusLost++ })
	tb.Focus()
	form.hoverWidget = tb
	form.mouseLeftButtonPressedWidget = tb

	// The text box goes with its parent
	form.Panel().RemoveWidget(box)
	if form.FocusedWidget() != nil || form.hoverWidget != nil || form.mouseLeftButtonPressedWidget != nil {
		t.Fatalf("the form still refers to the removed widget: focused %t, hover %t, pressed %t",
			form.FocusedWidget() != nil, form.hoverWidget != nil, form.mouseLeftButtonPressedWidget != nil)
	}
	form.processChar('x')
	if tb.Text() != "" {
		t.Errorf("a key went to the removed text box: %q", tb.Text())
	}
	form.processMouseDown(MouseButtonLeft, label.X()+2, label.Y()+2)
	form.processMouseUp(MouseButtonLeft, label.X()+2, label.Y()+2)
	if focusLost != 0 {
		t.Errorf("onFocusLost of the removed widget called %d times", focusLost)
	}
	if tb.IsFocused() || tb.IsHovered() {
		t.Error("the removed widget reports focus or hover")
	}
	tb.ClearFocus() // detached: does nothing
}

// Esc in the table's cell editor keeps the cell's text, Enter applies it:
// removing the focused editor must not run its focus-lost commit
func TestTableEditorEscAndEnter(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(3)
	table.SetCellText2(1, 0, "old")
	table.SetCurrentCell2(1, 0)

	table.EditCurrentCell("new")
	form.processKeyDown(KeyEsc, KeyModifiers{})
	if got := table.GetCellText2(1, 0); got != "old" {
		t.Errorf("after Esc: %q, want old", got)
	}
	if form.FocusedWidget() != Widgeter(table) {
		t.Errorf("after Esc the focus is on %v, want the table", form.FocusedWidget())
	}

	table.EditCurrentCell("new")
	form.processKeyDown(KeyEnter, KeyModifiers{})
	if got := table.GetCellText2(1, 0); got != "new" {
		t.Errorf("after Enter: %q, want new", got)
	}
	// The closed editors don't stay among the table's inner widgets
	if n := len(table.innerWidgets); n != 0 {
		t.Errorf("inner widgets after the edits: %d, want 0", n)
	}
}

// A Y range narrower than the float resolution at its magnitude used to
// hang the paint: v += step did not move v
func TestTimeChartTinyYRange(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		setup func(c *TimeChart)
	}{
		{"manual 1e17", func(c *TimeChart) {
			c.AddArea().SetYRange(1e17, 1e17+16)
		}},
		{"manual empty", func(c *TimeChart) {
			c.AddArea().SetYRange(5, 5)
		}},
		{"manual inverted", func(c *TimeChart) {
			c.AddArea().SetYRange(5, 1)
		}},
		{"data a few ULPs apart", func(c *TimeChart) {
			src := NewTimeChartMemorySource()
			src.AddPoint(NewTimeChartValue(base, 1e6))
			src.AddPoint(NewTimeChartValue(base.Add(time.Second), math.Nextafter(1e6, 2e6)))
			c.AddArea().AddSeries("s", src)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chart := NewTimeChart()
			chart.SetSize(400, 300)
			chart.SetDefaultTimeRange(base.Add(-time.Minute), base.Add(time.Minute))
			tc.setup(chart)
			done := make(chan struct{})
			go func() {
				defer close(done)
				img := image.NewRGBA(image.Rect(0, 0, 400, 300))
				cnv := NewCanvas(img)
				cnv.SetDirectTranslateAndClip(0, 0, 400, 300)
				chart.draw(cnv)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("draw hangs")
			}
		})
	}
}

// Hovering an item of the top menu closes all the submenus open from it,
// not only the first; a submenu opened again is not listed twice
func TestContextMenuClosesNestedSubmenus(t *testing.T) {
	form := newTestForm(t, NewLabel("label"))
	top := NewContextMenu(form.Panel())
	sub := NewContextMenu(form.Panel())
	subsub := NewContextMenu(form.Panel())
	top.AddItem("Plain", nil)
	top.AddItemWithSubmenu("Sub", sub)
	sub.AddItemWithSubmenu("SubSub", subsub)
	popups := func() []Widgeter { return form.Panel().PopupWidgets }

	form.ShowContextMenu(top, 10, 10)
	sub.showMenu(100, 10, top)
	subsub.showMenu(200, 10, sub)
	if n := len(popups()); n != 3 {
		t.Fatalf("open popups: %d, want 3", n)
	}

	// The submenu opened again (hover timer, then a click on its item)
	sub.showMenu(100, 10, top)
	sub.showMenu(100, 10, top)
	if n := len(popups()); n != 2 {
		t.Fatalf("submenu opened twice: %d popups, want 2", n)
	}

	subsub.showMenu(200, 10, sub)
	form.Panel().CloseAfterPopupWidget(top)
	if n := len(popups()); n != 1 || popups()[0] != Widgeter(top) {
		t.Fatalf("after closing above the top menu: %d popups, want only the top menu", n)
	}
}

// Text that changes on every frame doesn't grow the cache of rendered texts
// beyond its budget
func TestRenderedTextsBudget(t *testing.T) {
	NativeFontRendering = false
	defer func() { NativeFontRendering = true }()
	clearAllRenderedTexts()
	defer clearAllRenderedTexts()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	line := strings.Repeat("W", 60)
	added := 0
	for i := 0; added < renderedTextsMaxBytes*5/4; i++ {
		before := renderedTextsBytesNow()
		DrawText(img, line+strconv.Itoa(i), color.Black, FontFamilySans, 100, 0, 0, 0, 0, 100, 100)
		after := renderedTextsBytesNow()
		if after > before {
			added += after - before
		} else {
			added = renderedTextsMaxBytes * 2 // evicted: the budget works
		}
		if after > renderedTextsMaxBytes {
			t.Fatalf("cache at %d bytes after %d texts, budget %d", after, i+1, renderedTextsMaxBytes)
		}
	}
	if renderedTextsBytesNow() > renderedTextsMaxBytes {
		t.Fatal("over the budget")
	}
}

func renderedTextsBytesNow() int {
	renderedTextsMu.Lock()
	defer renderedTextsMu.Unlock()
	total := 0
	for _, rt := range renderedTexts {
		total += len(rt.textImage.Pix)
	}
	if total != renderedTextsBytes {
		panic("renderedTextsBytes is out of sync")
	}
	return total
}

// A closed form is freed with its widgets once the application drops it;
// shown again, its widgets work as before
func TestClosedFormIsFreed(t *testing.T) {
	newForm := func() (*Form, *TextBox) {
		tb := NewTextBox()
		form := newTestForm(t, tb)
		menu := NewContextMenu(nil)
		menu.AddItemWithSubmenu("Sub", NewContextMenu(nil))
		tb.SetContextMenu(menu)
		return form, tb
	}

	form, tb := newForm()
	formRef, tbRef := weak.Make(form), weak.Make(tb)
	id := tb.Id()
	form.processWindowClose() // the close button
	if _, ok := lookupWidget(id); ok {
		t.Fatal("a widget of the closed form is still registered")
	}
	form, tb = nil, nil
	for i := 0; i < 5 && (formRef.Value() != nil || tbRef.Value() != nil); i++ {
		runtime.GC()
	}
	if formRef.Value() != nil || tbRef.Value() != nil {
		t.Error("the closed form or its widget is still in memory")
	}

	// Closed, then shown again (e.g. from a tray icon)
	form, tb = newForm()
	form.processWindowClose() // the close button
	form.reopen()
	if w, ok := lookupWidget(tb.Id()); !ok || w != Widgeter(tb) {
		t.Fatal("the widgets of the form shown again are not registered")
	}
	tb.Focus()
	if form.FocusedWidget() != Widgeter(tb) {
		t.Error("a widget of the form shown again can't take the focus")
	}
}

// Pasting inserts the text at once, with the result typing it gave: a line
// break only in a multiline box, the other control characters dropped, the
// selection replaced, one undo step and one onTextChanged
func TestTextBoxPaste(t *testing.T) {
	cases := []struct {
		name      string
		multiline bool
		text      string
		selectAll bool
		cursorX   int
		paste     string
		want      string
		wantX     int
		wantY     int
	}{
		{"middle of a line", false, "hello world", false, 5, ",", "hello, world", 6, 0},
		{"cyrillic", false, "абв", false, 1, "жё", "ажёбв", 3, 0},
		{"control characters", false, "ab", false, 1, "x\ty\rz", "axyzb", 4, 0},
		{"lines in a single-line box", false, "ab", false, 1, "1\n2\r\n3", "a123b", 4, 0},
		{"lines", true, "ab", false, 1, "1\n22\r\n333", "a1\n22\n333b", 3, 2},
		{"trailing line break", true, "ab", false, 2, "x\n", "abx\n", 0, 1},
		{"over a selection", true, "one\ntwo", true, 0, "new", "new", 3, 0},
		{"lines over a selection", true, "one\ntwo", true, 0, "a\nb", "a\nb", 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tb := NewTextBox()
			newTestForm(t, tb)
			tb.SetMultiline(tc.multiline)
			tb.SetText(tc.text)
			changes := 0
			tb.SetOnTextChanged(func() { changes++ })
			if tc.selectAll {
				tb.SelectAllText()
			} else {
				tb.moveCursor(tc.cursorX, 0, KeyModifiers{})
			}
			tb.modifyText(textboxModifyCommandInsertString, KeyModifiers{}, tc.paste)
			if got := tb.Text(); got != tc.want {
				t.Errorf("text %q, want %q", got, tc.want)
			}
			if tb.cursorPosX != tc.wantX || tb.cursorPosY != tc.wantY {
				t.Errorf("cursor %d,%d, want %d,%d", tb.cursorPosX, tb.cursorPosY, tc.wantX, tc.wantY)
			}
			if changes != 1 {
				t.Errorf("onTextChanged called %d times, want 1", changes)
			}
			tb.Undo()
			if got := tb.Text(); got != tc.text {
				t.Errorf("after undo %q, want %q", got, tc.text)
			}
		})
	}

	// 64 KB pastes in well under a second (it took minutes)
	tb := NewTextBox()
	newTestForm(t, tb)
	tb.SetMultiline(true)
	big := strings.Repeat(strings.Repeat("x", 63)+"\n", 1024)
	start := time.Now()
	tb.modifyText(textboxModifyCommandInsertString, KeyModifiers{}, big)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("a 64 KB paste took %v", d)
	}
	if tb.Text() != big {
		t.Error("the big paste is wrong")
	}
}

// buildLayoutTestForm builds a form with nested grids, expandable and hidden
// widgets, two widgets in one cell, gaps in the grid and sparse indexes
func buildLayoutTestForm() *Form {
	form := NewForm()
	form.processResize(900, 700)
	p := form.Panel()
	top := NewPanel()
	p.AddWidget(0, 0, top)
	top.AddWidget(0, 0, NewLabel("Name"))
	top.AddWidget(0, 1, NewTextBox())
	top.AddWidget(1, 0, NewLabel("A much longer label text"))
	tb := NewTextBox()
	tb.SetMultiline(true)
	top.AddWidget(1, 1, tb)
	top.AddWidget(2, 3, NewButton("Far"))
	hidden := NewButton("Hidden")
	top.AddWidget(3, 0, hidden)
	hidden.SetVisible(false)
	top.AddWidget(3, 0, NewButton("Second in the cell"))
	top.AddWidget(3, 0, NewButton("Third in the cell"))
	mid := NewPanel()
	p.AddWidget(1, 0, mid)
	for r := 0; r < 6; r++ {
		for col := 0; col < 4; col++ {
			if (r+col)%3 == 0 {
				mid.AddWidget(r, col, NewButton(fmt.Sprintf("B%d.%d", r, col)))
			} else if (r+col)%3 == 1 {
				mid.AddWidget(r, col, NewLabel(strings.Repeat("w", r+col)))
			}
		}
	}
	mid.AddHSpacer(0, 10)
	p.AddWidget(2, 0, NewCheckbox("check"))
	p.AddVSpacer(3, 0)
	sparse := NewPanel()
	p.AddWidget(10, 5, sparse)
	sparse.AddWidget(-2, -1, NewLabel("negative"))
	sparse.AddWidget(7, 4, NewLabel("seven"))
	form.UpdateLayout()
	return form
}

// The index of the grid cells finds in every cell the widget the lookup
// among all the children found; adding many widgets doesn't freeze
func TestGridCellsIndex(t *testing.T) {
	form := buildLayoutTestForm()
	var check func(w Widgeter)
	check = func(w Widgeter) {
		panel, ok := w.(*Panel)
		if !ok {
			return
		}
		base := &panel.Widget
		cells := base.gridCells()
		for x := -3; x <= 12; x++ {
			for y := -3; y <= 12; y++ {
				if got, want := cells[gridCell{x, y}], base.getWidgetInGridCell(x, y); got != want {
					t.Errorf("%s cell %d,%d: index %v, lookup %v", w.Name(), x, y, got, want)
				}
			}
		}
		for _, ch := range w.Widgets() {
			check(ch)
		}
	}
	check(form.Panel())

	form = NewForm()
	form.processResize(800, 600)
	panel := NewPanel()
	form.Panel().AddWidget(0, 0, panel)
	start := time.Now()
	for i := 0; i < 300; i++ {
		panel.AddWidget(i, 0, NewLabel("label"))
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("adding 300 labels took %v", d)
	}
}

// ClearRows removes the cells, the selection and the editor; SetRowCount
// alone keeps the cells of the rows beyond the count
func TestTableClearRows(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(3)
	table.SetCellText2(2, 1, "old")
	table.SetCellData2(2, 1, 42)

	// SetRowCount keeps the cells, as it always did
	table.SetRowCount(0)
	table.SetRowCount(3)
	if got := table.GetCellText2(2, 1); got != "old" {
		t.Fatalf("SetRowCount lost the cell: %q", got)
	}

	table.SetCurrentCell2(2, 1)
	table.EditCurrentCell("typed")
	changes := 0
	table.SetOnSelectionChanged(func(row, col int) { changes++ })

	table.ClearRows()
	if table.RowCount() != 0 || table.CurrentRow() != -1 || len(table.SelectedRows()) != 0 {
		t.Errorf("after ClearRows: %d rows, current %d, selected %v", table.RowCount(), table.CurrentRow(), table.SelectedRows())
	}
	if table.editorTextBox != nil || len(table.innerWidgets) != 0 {
		t.Error("the editor is still open")
	}
	if changes != 1 {
		t.Errorf("onSelectionChanged called %d times, want 1", changes)
	}

	// Filled again: no old data, the first row current as in a new table
	table.SetCellText2(0, 0, "new")
	table.SetRowCount(3)
	if got := table.GetCellText2(2, 1); got != "" {
		t.Errorf("old text came back: %q", got)
	}
	if got := table.GetCellData2(2, 1); got != nil {
		t.Errorf("old data came back: %v", got)
	}
	if got := table.GetCellText2(0, 0); got != "new" {
		t.Errorf("new text %q", got)
	}
	if table.CurrentRow() != 0 {
		t.Errorf("current row %d, want 0", table.CurrentRow())
	}
}

// Inputs that used to panic: a zero row height, the text property set over a
// selection, an inverted range to downsample
func TestSafeInputs(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(50)
	table.SetRowHeight(0)
	if table.RowHeight() < 1 {
		t.Errorf("table row height %d", table.RowHeight())
	}
	table.ProcessKeyDown(KeyPageDown, KeyModifiers{})
	img := image.NewRGBA(image.Rect(0, 0, 500, 400))
	cnv := NewCanvas(img)
	cnv.SetDirectTranslateAndClip(0, 0, 500, 400)
	table.ProcessPaint(cnv)

	tree := NewTreeView()
	newTestForm(t, tree)
	tree.SetRowHeight(-5)
	if tree.RowHeight() < 1 {
		t.Errorf("tree row height %d", tree.RowHeight())
	}

	tb := NewTextBox()
	newTestForm(t, tb)
	tb.SetMultiline(true)
	tb.SetText("hello world\nsecond line")
	tb.SelectAllText()
	tb.SetProp("text", "hi")
	if tb.Text() != "hi" {
		t.Errorf("text %q", tb.Text())
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []TimeChartPoint{NewTimeChartValue(base, 1), NewTimeChartValue(base.Add(time.Minute), 2)}
	if got := TimeChartDownsample(points, base.Add(time.Hour), base, time.Second); len(got) != 0 {
		t.Errorf("inverted range: %d points", len(got))
	}
}

// The cell editor applies the edit to the cell it was opened on, even when
// the current cell changes meanwhile
func TestTableEditorKeepsItsCell(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(3)
	for r := 0; r < 3; r++ {
		table.SetCellText2(r, 0, "row"+strconv.Itoa(r))
	}

	// Enter after code moved the current cell (a refresh restoring it)
	table.SetCurrentCell2(1, 0)
	table.EditCurrentCell("edited")
	table.SetCurrentCell2(2, 0)
	form.processKeyDown(KeyEnter, KeyModifiers{})
	if a, b := table.GetCellText2(1, 0), table.GetCellText2(2, 0); a != "edited" || b != "row2" {
		t.Errorf("Enter: row 1 %q, row 2 %q; want edited, row2", a, b)
	}

	// The focus lost after the selection was cleared
	table.SetCurrentCell2(0, 0)
	table.EditCurrentCell("again")
	table.ClearSelection()
	table.Focus()
	if got := table.GetCellText2(0, 0); got != "again" {
		t.Errorf("focus lost: row 0 %q, want again", got)
	}
	if _, ok := table.rows[-1]; ok {
		t.Error("an entry for row -1 was made")
	}

	// The cell is gone when the edit is applied: nothing is written
	table.SetCurrentCell2(2, 1)
	table.EditCurrentCell("lost")
	table.SetRowCount(2)
	form.processKeyDown(KeyEnter, KeyModifiers{})
	table.SetRowCount(3)
	if got := table.GetCellText2(2, 1); got != "" {
		t.Errorf("an edit of a removed row was written: %q", got)
	}
}

// The number box keeps its value when the focus passes through it or Enter
// is pressed without an edit: the text shown is rounded, the value isn't
func TestNumBoxKeepsPrecision(t *testing.T) {
	nb := NewNumBox()
	label := NewLabel("label")
	form := newTestForm(t, nb)
	form.Panel().AddWidget(2, 0, label)
	nb.SetDecimals(2)
	nb.SetValue(3.14159)
	changes := 0
	nb.SetOnValueChanged(func() { changes++ })

	nb.Focus()
	nb.onKeyDown(KeyEnter, KeyModifiers{})
	nb.ProcessFocusLost()
	if nb.Value() != 3.14159 || changes != 0 {
		t.Errorf("focus and Enter without an edit: value %v, changes %d", nb.Value(), changes)
	}

	// A step is a change by the user: from the exact value, rounded
	nb.onKeyDown(KeyArrowUp, KeyModifiers{})
	if nb.Value() != 3.15 || changes != 1 {
		t.Errorf("step: value %v, changes %d", nb.Value(), changes)
	}

	// Typed text is applied as before
	nb.SetText("2.5")
	nb.ProcessFocusLost()
	if nb.Value() != 2.5 || changes != 2 {
		t.Errorf("typed: value %v, changes %d", nb.Value(), changes)
	}

	// PropertyGrid: a click into the field and out doesn't write the rounded
	// value into the object
	s := &struct{ Ratio float64 }{Ratio: 0.123456}
	g := NewPropertyGrid()
	if err := g.SetObject(s); err != nil {
		t.Fatal(err)
	}
	newTestForm(t, g)
	editor := g.Property("Ratio").editor
	editor.Focus()
	editor.ProcessFocusLost()
	if s.Ratio != 0.123456 {
		t.Errorf("PropertyGrid wrote %v into the object", s.Ratio)
	}
}

// PropertyGrid edits whole numbers over the range of their type, writes the
// limits without an overflow, and keeps the time of day and the location of
// a time field when the date changes
func TestPropertyGridWideFields(t *testing.T) {
	zone := time.FixedZone("UTC+3", 3*3600)
	s := &struct {
		Big   int64
		Small int8
		Huge  uint64
		Due   time.Time
	}{Big: 5_000_000_000, Small: -100, Huge: 10_000_000_000_000_000_000,
		Due: time.Date(2024, 5, 6, 15, 30, 45, 0, zone)}
	g := NewPropertyGrid()
	if err := g.SetObject(s); err != nil {
		t.Fatal(err)
	}
	newTestForm(t, g)
	box := func(name string) *NumBox { return g.Property(name).editor.(*NumBox) }

	if box("Big").Value() != 5e9 || g.Property("Big").Value() != 5_000_000_000 {
		t.Errorf("Big shown as %v", box("Big").Value())
	}
	if box("Huge").Value() != 1e19 {
		t.Errorf("Huge shown as %v", box("Huge").Value())
	}
	if box("Small").Min() != -128 || box("Small").Max() != 127 {
		t.Errorf("Small range %v..%v", box("Small").Min(), box("Small").Max())
	}

	edit := func(name, text string) {
		b := box(name)
		b.SetText(text)
		b.ProcessFocusLost()
	}
	edit("Big", "6000000000")
	if s.Big != 6_000_000_000 {
		t.Errorf("Big written as %d", s.Big)
	}
	edit("Big", "99999999999999999999") // clamped to the top of int64
	if s.Big != math.MaxInt64 {
		t.Errorf("Big at the limit written as %d", s.Big)
	}
	edit("Huge", "12000000000000000000")
	if s.Huge != 12_000_000_000_000_000_000 {
		t.Errorf("Huge written as %d", s.Huge)
	}
	if s.Small != -100 {
		t.Errorf("an untouched field changed: Small %d", s.Small)
	}

	g.Property("Due").editor.(*DatePicker).changeDate(time.Date(2025, 1, 2, 0, 0, 0, 0, time.Local))
	want := time.Date(2025, 1, 2, 15, 30, 45, 0, zone)
	if !s.Due.Equal(want) || s.Due.Location() != zone {
		t.Errorf("Due written as %v, want %v", s.Due, want)
	}
}
