package regression

import (
	"strings"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// The behavior of the text box, as the user sees it: kept by the rewrite of
// its storage

func textBoxForm(t *testing.T, multiline bool) (*uitest.Window, *ui.TextBox) {
	t.Helper()
	form := newForm(400, 300)
	tb := ui.NewTextBox()
	tb.SetMultiline(multiline)
	form.Panel().AddWidget(0, 0, tb)
	if !multiline {
		form.Panel().AddVSpacer(1, 0)
	}
	w := uitest.Show(t, form)
	w.Click(tb)
	return w, tb
}

var noMods = ui.KeyModifiers{}
var shift = ui.KeyModifiers{Shift: true}

func TestTextBoxTypingAndLines(t *testing.T) {
	w, tb := textBoxForm(t, true)
	w.Type("one\ntwo\nthree")
	if tb.Text() != "one\ntwo\nthree" {
		t.Fatalf("text %q", tb.Text())
	}
	if lines := tb.Lines(); len(lines) != 3 || lines[1] != "two" {
		t.Fatalf("lines %q", lines)
	}
	// Backspace at the start of a line joins it to the one above
	w.Key(ui.KeyHome, noMods)
	w.Key(ui.KeyBackspace, noMods)
	if tb.Text() != "one\ntwothree" {
		t.Errorf("after Backspace: %q", tb.Text())
	}
	// Delete at the end of a line joins the next one
	w.Key(ui.KeyArrowUp, noMods)
	w.Key(ui.KeyEnd, noMods)
	w.Key(ui.KeyDelete, noMods)
	if tb.Text() != "onetwothree" {
		t.Errorf("after Delete: %q", tb.Text())
	}
	// Enter splits the line at the cursor
	w.Key(ui.KeyHome, noMods)
	for range 3 {
		w.Key(ui.KeyArrowRight, noMods)
	}
	w.Key(ui.KeyEnter, noMods)
	if tb.Text() != "one\ntwothree" {
		t.Errorf("after Enter: %q", tb.Text())
	}
	// Up and down keep the column within the shorter line
	w.Key(ui.KeyEnd, noMods) // end of "twothree"
	w.Key(ui.KeyArrowUp, noMods)
	w.Type("!")
	if tb.Text() != "one!\ntwothree" {
		t.Errorf("after Up and typing: %q", tb.Text())
	}
}

func TestTextBoxSelectionAndClipboard(t *testing.T) {
	w, tb := textBoxForm(t, true)
	w.Type("hello world\nsecond")
	w.Shortcut("Ctrl+A")
	if tb.SelectedText() != "hello world\nsecond" {
		t.Errorf("select all: %q", tb.SelectedText())
	}
	// Shift+arrows select; typing replaces the selection
	w.Key(ui.KeyArrowUp, noMods)
	w.Key(ui.KeyHome, noMods)
	for range 5 {
		w.Key(ui.KeyArrowRight, shift)
	}
	if tb.SelectedText() != "hello" {
		t.Fatalf("shift+right: %q", tb.SelectedText())
	}
	w.Shortcut("Ctrl+C")
	w.Type("HI")
	if tb.Text() != "HI world\nsecond" {
		t.Errorf("typing over the selection: %q", tb.Text())
	}
	w.Key(ui.KeyEnd, noMods)
	w.Shortcut("Ctrl+V")
	if tb.Text() != "HI worldhello\nsecond" {
		t.Errorf("paste: %q", tb.Text())
	}
	// A selection over two lines, cut
	w.Key(ui.KeyArrowLeft, shift)
	w.Key(ui.KeyArrowDown, shift)
	w.Shortcut("Ctrl+X")
	// From the end of the first line (the anchor) to the column 6 of the
	// second one
	clip, _ := ui.ClipboardGetText()
	if tb.Text() != "HI worldhello" || clip != "\nsecond" {
		t.Errorf("cut over two lines: text %q, clipboard %q", tb.Text(), clip)
	}
}

func TestTextBoxUndoRedoGrouping(t *testing.T) {
	w, tb := textBoxForm(t, true)
	tb.SetText("start")
	w.Shortcut("Ctrl+A")
	w.Type("abc")
	w.Key(ui.KeyEnter, noMods)
	w.Type("def")
	w.Shortcut("Ctrl+Z")
	if tb.Text() != "abc\n" {
		t.Errorf("undo of a run of typing: %q", tb.Text())
	}
	w.Shortcut("Ctrl+Z")
	w.Shortcut("Ctrl+Z")
	if tb.Text() != "start" {
		t.Errorf("undo to the start: %q", tb.Text())
	}
	w.Shortcut("Ctrl+Y")
	if tb.Text() != "abc" {
		t.Errorf("redo: %q", tb.Text())
	}
	// SetText starts a new history
	tb.SetText("new")
	if tb.CanUndo() {
		t.Error("SetText keeps the history")
	}
}

func TestTextBoxSingleLine(t *testing.T) {
	w, tb := textBoxForm(t, false)
	ui.ClipboardSetText("one\ntwo")
	w.Shortcut("Ctrl+V")
	if tb.Text() != "onetwo" {
		t.Errorf("paste into one line: %q", tb.Text())
	}
	tb.SetReadOnly(true)
	w.Type("x")
	w.Key(ui.KeyBackspace, noMods)
	if tb.Text() != "onetwo" {
		t.Errorf("read-only changed: %q", tb.Text())
	}
}

// A click puts the cursor at the nearest character boundary
func TestTextBoxClickPlacesCursor(t *testing.T) {
	w, tb := textBoxForm(t, true)
	tb.SetText("aaaa\nbbbb\ncccc")
	w.Click(uitest.At(tb, tb.Width()-20, 2+tb.OneLineHeight()+tb.OneLineHeight()/2)) // right of the 2nd line
	w.Type("!")
	if tb.Text() != "aaaa\nbbbb!\ncccc" {
		t.Errorf("click right of line 2 and type: %q", tb.Text())
	}
}

func TestTextBoxGolden(t *testing.T) {
	form := newForm(300, 260)
	tb := ui.NewTextBox()
	tb.SetMultiline(true)
	form.Panel().AddWidget(0, 0, tb)
	single := ui.NewTextBox()
	single.SetText("single line")
	form.Panel().AddWidget(1, 0, single)
	pass := ui.NewTextBox()
	pass.SetIsPassword(true)
	pass.SetText("secret")
	form.Panel().AddWidget(2, 0, pass)
	w := uitest.Show(t, form)
	var lines []string
	for i := range 30 {
		lines = append(lines, "line "+string(rune('A'+i%26))+" of the text")
	}
	tb.SetText(strings.Join(lines, "\n"))
	w.Click(tb)
	for range 20 {
		w.Key(ui.KeyArrowDown, noMods)
	}
	for range 6 {
		w.Key(ui.KeyArrowRight, shift)
	}
	w.Key(ui.KeyArrowDown, shift)
	w.MatchGolden("textbox-scrolled-selection")
}

// A property change doesn't make a multiline box expandable
// again; SetMultiline itself does, at once
func TestTextBoxKeepsSizePolicy(t *testing.T) {
	form := newForm(300, 200)
	tb := ui.NewTextBox()
	tb.SetMultiline(true)
	if !tb.XExpandable() || !tb.YExpandable() {
		t.Errorf("multiline: expandable %v %v", tb.XExpandable(), tb.YExpandable())
	}
	form.Panel().AddWidget(0, 0, tb)
	uitest.Show(t, form)
	tb.SetXExpandable(false)
	tb.SetYExpandable(false)
	tb.SetProp("placeholder", "x")
	tb.SetText("text")
	if tb.XExpandable() || tb.YExpandable() {
		t.Errorf("after SetProp and SetText: expandable %v %v", tb.XExpandable(), tb.YExpandable())
	}
}

// The cursor, the selection and the edits from the code
func TestTextBoxAPI(t *testing.T) {
	w, tb := textBoxForm(t, true)
	tb.SetText("one\ntwo three\nfour")
	if tb.LineCount() != 3 || tb.Line(1) != "two three" || tb.Line(5) != "" {
		t.Fatalf("lines: %d, %q", tb.LineCount(), tb.Line(1))
	}

	tb.SetCursorPosition(ui.TextPos{Line: 1, Col: 100}) // kept within the line
	if got := tb.CursorPosition(); got != (ui.TextPos{Line: 1, Col: 9}) {
		t.Errorf("cursor %+v", got)
	}

	tb.SetSelection(ui.TextPos{Line: 1, Col: 4}, ui.TextPos{Line: 1, Col: 9})
	if tb.SelectedText() != "three" || !tb.HasSelection() {
		t.Errorf("selected %q", tb.SelectedText())
	}
	from, to := tb.Selection()
	if from != (ui.TextPos{Line: 1, Col: 4}) || to != (ui.TextPos{Line: 1, Col: 9}) {
		t.Errorf("selection %+v - %+v", from, to)
	}

	// InsertText replaces the selection and is undone as one step;
	// OnTextChanged, not OnTextEdited
	changed, edited := count(&tb.OnTextChanged), count(&tb.OnTextEdited)
	tb.InsertText("3\n3½")
	if tb.Text() != "one\ntwo 3\n3½\nfour" || changed.n != 1 || edited.n != 0 {
		t.Errorf("after InsertText: %q, changed %d, edited %d", tb.Text(), changed.n, edited.n)
	}
	if got := tb.CursorPosition(); got != (ui.TextPos{Line: 2, Col: 2}) {
		t.Errorf("cursor after InsertText %+v", got)
	}
	w.Shortcut("Ctrl+Z")
	if tb.Text() != "one\ntwo three\nfour" {
		t.Errorf("undo of InsertText: %q", tb.Text())
	}

	// The offsets in the whole text, for searching it
	for _, offset := range []int{0, 3, 4, 13, 18} {
		if got := tb.OffsetOf(tb.PosAt(offset)); got != offset {
			t.Errorf("offset %d -> %+v -> %d", offset, tb.PosAt(offset), got)
		}
	}
	if pos := tb.PosAt(strings.Index(tb.Text(), "four")); pos != (ui.TextPos{Line: 2, Col: 0}) {
		t.Errorf("PosAt(four) = %+v", pos)
	}

	// Ctrl+Home and Ctrl+End go to the start and the end of the text
	w.Shortcut("Ctrl+End")
	if got := tb.CursorPosition(); got != (ui.TextPos{Line: 2, Col: 4}) {
		t.Errorf("Ctrl+End: %+v", got)
	}
	w.Shortcut("Ctrl+Home")
	if got := tb.CursorPosition(); got != (ui.TextPos{}) {
		t.Errorf("Ctrl+Home: %+v", got)
	}

	// The clipboard from the code, and with Cmd (macOS)
	tb.SetSelection(ui.TextPos{Line: 0, Col: 0}, ui.TextPos{Line: 0, Col: 3})
	tb.Copy()
	tb.SetCursorPosition(ui.TextPos{Line: 2, Col: 4})
	tb.Paste()
	w.Key(ui.KeyA, ui.KeyModifiers{Cmd: true})
	w.Key(ui.KeyX, ui.KeyModifiers{Cmd: true})
	if tb.Text() != "" {
		t.Errorf("Cmd+A, Cmd+X: %q", tb.Text())
	}
	if clip, _ := ui.ClipboardGetText(); clip != "one\ntwo three\nfourone" {
		t.Errorf("clipboard %q", clip)
	}
}

// The input method is told where the caret is: under the menu bar, in the
// scrolled text
func TestTextInputRect(t *testing.T) {
	form := newForm(300, 200)
	form.SetMenuBar(ui.NewMenuBar())
	form.MenuBar().AddMenu("File")
	box := ui.NewTextBox()
	box.SetMultiline(true)
	box.SetText("first\nsecond line")
	form.Panel().AddWidget(0, 0, box)
	w := uitest.Show(t, form)
	w.Click(box)
	box.SetCursorPosition(ui.TextPos{Line: 1, Col: 6})
	w.Settle()
	w.Screenshot()
	x, y, _, height := w.TextInputRect()
	bx, by := box.RectClientAreaOnWindow()
	lineHeight := box.OneLineHeight()
	if y < by+lineHeight || y > by+lineHeight+4 || height != lineHeight {
		t.Errorf("caret y %d height %d; the box at %d, lines of %d", y, height, by, lineHeight)
	}
	sixChars, _, _ := ui.MeasureText(box.FontFamily(), box.FontSize(), "second")
	if x < bx+sixChars || x > bx+sixChars+12 {
		t.Errorf("caret x %d; the box at %d, %d px of text before it", x, bx, sixChars)
	}
}

// A day chosen in the calendar of a date picker closes it on the press: the
// release goes to the closed popup. A click on a text box after it used to
// give its release to the calendar - the text box stayed "pressed" and
// selected the text as the mouse moved, with no button held.
func TestTextBoxAfterDatePickerPopup(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(400, 300)
	picker := ui.NewDatePicker()
	notes := ui.NewTextBox()
	notes.SetText("some notes here")
	form.Panel().AddWidget(0, 0, picker)
	form.Panel().AddWidget(1, 0, notes)
	form.Panel().AddVSpacer(2, 0)
	w := uitest.Show(t, form)

	w.Click(picker)
	popup := w.Popup()
	if popup == nil {
		t.Fatal("the calendar didn't open")
	}
	// A day of the month's grid, in the middle of the calendar
	w.Click(uitest.At(popup, popup.Width()/2, popup.Height()/2))
	if w.Popup() != nil {
		t.Fatal("choosing a day didn't close the calendar")
	}

	w.Click(uitest.At(notes, 10, notes.Height()/2))
	w.Move(uitest.At(notes, notes.Width()-10, notes.Height()/2))
	if notes.HasSelection() {
		t.Errorf("moving the mouse with no button held selected %q", notes.SelectedText())
	}
}

// The input method takes the keys only while the focus is in a text box:
// on a button or a list a held key repeats, no composition starts
func TestTextInputActiveInTextBoxOnly(t *testing.T) {
	form := ui.NewForm()
	form.SetSize(300, 200)
	button := ui.NewButton("OK")
	text := ui.NewTextBox()
	form.Panel().AddWidget(0, 0, button)
	form.Panel().AddWidget(1, 0, text)
	w := uitest.Show(t, form)

	w.Click(text)
	if !w.TextInputActive() {
		t.Error("the text box has the focus: the input method must be on")
	}
	w.Click(button)
	if w.TextInputActive() {
		t.Error("a button has the focus: the input method must be off")
	}
	text.Focus()
	if !w.TextInputActive() {
		t.Error("focused by the code: on at once, before a frame")
	}
	text.ClearFocus()
	if w.TextInputActive() {
		t.Error("no focus: off")
	}
}
