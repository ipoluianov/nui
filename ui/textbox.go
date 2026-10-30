package ui

import (
	"fmt"
	"image/color"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type TextBox struct {
	Widget

	// OnTextChanged: the text changed, by the user or by SetText
	OnTextChanged Signal[string]
	// OnTextEdited: the user changed the text: typed, deleted, pasted, cut,
	// undid or redid an edit
	OnTextEdited Signal[string]
	// OnSubmit: Enter pressed in a single-line box; while a handler is
	// connected, Enter doesn't reach the form (its accept button)
	OnSubmit Signal[string]
	// OnEditingFinished: the user is done editing: pressed Enter in a
	// single-line box, or left the box after changing its text
	OnEditingFinished Signal[string]

	textAtFocus string // the text when the box got the focus

	// The text, a line a string, without the line breaks: edits change the
	// lines they touch only. The whole text is joined when asked for.
	lines []string
	// lineWidths are the measured widths of the lines, -1 if not measured
	lineWidths []int
	// measure counts the times the lines were measured anew (remeasure): the
	// widths in the undo history of an older one are of another font size
	// or scale of the screen
	measure   int
	text      string // the joined text, if textValid
	textValid bool
	// editByCode is set while the code edits the text (InsertText): the
	// edit is not the user's (see OnTextEdited)
	editByCode bool

	hintFunc func() string // see SetHintFunc

	cursorPosX          int
	cursorPosY          int
	selectionLeftX      int
	selectionLeftY      int
	selectionRightX     int
	selectionRightY     int
	mouseButtonPressed  bool
	cursorWidth         int
	leftAndRightPadding int

	// lineNumbers shows the numbers of the lines in a column at the left
	// (multiline only); gutterDigits and gutterW cache its width
	lineNumbers  bool
	gutterDigits int
	gutterW      int
	// highlighter gives the styled parts of a line, see SetHighlighter
	highlighter func(line int, text string) []TextSpan

	dragingCursor bool

	// textAlign places the lines in the box (SetTextAlign); visualArrows
	// moves the cursor by Left and Right on the screen
	textAlign    TextAlign
	visualArrows bool

	padding int

	cursorVisible         bool
	skipOneCursorBlinking bool

	propIsProcessing bool

	// The undo history, see Undo
	undoStack, redoStack []textBoxState
	// The last edit, to make one undo step of a run of typing (or of
	// Backspace, Delete): its kind, where it left the cursor and when
	lastEdit       textboxModifyCommand
	lastEditX      int
	lastEditY      int
	lastEditTime   time.Time
	lastEditActive bool
}

// textBoxState is a step of the undo history: the lines (a copy of the
// slice: the strings are shared) with their widths, and the cursor
type textBoxState struct {
	lines  []string
	widths []int
	// measure is the measuring the widths are of (see TextBox.measure)
	measure int
	x, y    int
}

// textBoxUndoLimit is how many steps Undo goes back
const textBoxUndoLimit = 500

// textBoxUndoPause ends a run of typing that undoes in one step
const textBoxUndoPause = 2 * time.Second

/*
Properties:
- text: string - The text of the textbox.
- hint: string - The hint text displayed when the textbox is empty.
- multiline: string - Whether the textbox supports multiple lines.
- readonly: bool - Whether the textbox is read-only.
- ispassword: bool - Whether the textbox masks input (for password entry).
*/

type textboxModifyCommand int

const textboxModifyCommandInsertChar textboxModifyCommand = 0
const textboxModifyCommandInsertString textboxModifyCommand = 1
const textboxModifyCommandInsertReturn textboxModifyCommand = 2
const textboxModifyCommandBackspace textboxModifyCommand = 3
const textboxModifyCommandDelete textboxModifyCommand = 4
const textboxModifyCommandSetText textboxModifyCommand = 5

type TextBoxSelection struct {
	X1, Y1, X2, Y2 int
	Text           string
}

func NewTextBox() *TextBox {
	var c TextBox
	c.initTextBox()
	return &c
}

// initTextBox makes the text box ready, in place: for the widgets built on
// it (LogView), whose handlers must be of their own value
func (c *TextBox) initTextBox() {
	c.InitWidget()
	c.SetTypeName("TextBox")
	c.scrollBarInset = 1 // the scroll bars of a multiline box are inside its frame

	c.lines = []string{""}
	c.lineWidths = []int{-1}
	c.SetOnKeyDown(func(key Key, mods KeyModifiers) bool {
		return c.KeyDown(key, mods)
	})
	c.OnFocusChanged.Connect(func(focused bool) {
		if focused {
			c.textAtFocus = c.Text()
		} else if c.Text() != c.textAtFocus {
			c.textAtFocus = c.Text()
			c.OnEditingFinished.Emit(c.Text())
		}
	})

	c.SetOnChar(func(char rune, mods KeyModifiers) bool {
		c.KeyChar(char, mods)
		return true
	})

	c.SetOnPaint(func(cnv *Canvas) {
		c.Draw(cnv, c.innerWidth, c.innerHeight)
	})

	c.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		c.MouseDown(button, x, y, mods)
		return true
	})

	c.SetOnMouseMove(func(x, y int, mods KeyModifiers) bool {
		c.MouseMove(x, y, mods)
		return true
	})

	c.SetOnMouseUp(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		c.MouseUp(button, x, y, mods)
		return true
	})

	c.AddTimer(250, func() {
		c.timerCursorBlinking()
	})

	c.SetCanBeFocused(true)
	c.SetXExpandable(true)
	c.SetYExpandable(false)
	c.SetMinWidth(100)
	c.setThemeHeight(ThemeControlHeight, false)
	//c.SetMaxSize(2000, DefaultUiLineHeight)

	//c.lines = make([]string, 1)
	c.cursorWidth = 1
	c.leftAndRightPadding = themeTextInset
	c.SetMultiline(false)
	c.cursorVisible = true
	c.ScrollToBegin()
	c.updateInnerSize()

	c.padding = 4
}

// Lines returns the lines of the text, without the line breaks
func (c *TextBox) Lines() []string {
	return slices.Clone(c.lines)
}

// LineCount returns the number of the lines, at least 1
func (c *TextBox) LineCount() int {
	return len(c.lines)
}

// Line returns the line i (from 0), "" out of the text
func (c *TextBox) Line(i int) string {
	if i < 0 || i >= len(c.lines) {
		return ""
	}
	return c.lines[i]
}

// splitLines splits the text into lines; "\r" is dropped
func splitLines(text string) []string {
	return strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
}

func (c *TextBox) SetReadOnly(readonly bool) {
	c.SetProp("readonly", readonly)
}

func (c *TextBox) ReadOnly() bool {
	return c.GetPropBool("readonly", false)
}

func (c *TextBox) SetIsPassword(isPassword bool) {
	c.SetProp("ispassword", isPassword)
}

func (c *TextBox) SetOnTextChanged(onTextChanged func()) {
	c.SetPropFunction("ontextchanged", onTextChanged)
}

type EventTextboxKeyDown struct {
	Key       Key
	Mods      KeyModifiers
	Processed bool
}

// SetOnTextBoxKeyDown sets the function called on a key press, before the
// text box handles it; CurrentEvent().Parameter is *EventTextboxKeyDown.
//
// Deprecated: use OnKeyDown (or OnSubmit for Enter): its handler gets the
// key and sets Handled.
func (c *TextBox) SetOnTextBoxKeyDown(onKeyDown func()) {
	c.SetPropFunction("onkeydown", onKeyDown)
}

func (c *TextBox) timerCursorBlinking() {
	if c.form.focusedWidget != nil {
		if c.form.focusedWidget.Id() == c.id {
			if !c.skipOneCursorBlinking {
				c.cursorVisible = !c.cursorVisible
				c.form.Update()
			}
			c.skipOneCursorBlinking = false
		}
	}
}

func (c *TextBox) redraw() {
}

// SetProp: the text is kept in the lines, not in the properties
func (c *TextBox) SetProp(key string, value any) {
	if key == "text" {
		text, _ := value.(string)
		if value != nil && text == "" {
			text = fmt.Sprint(value)
		}
		c.SetText(text)
		return
	}
	c.Widget.SetProp(key, value)
	switch key {
	case "fontfamily", "fontsize", "fontbold", "fontitalic", "ispassword":
		c.remeasure()
	}
}

// GetProp: the text is in the lines
func (c *TextBox) GetProp(key string) any {
	if key == "text" {
		return c.Text()
	}
	return c.Widget.GetProp(key)
}

func (c *TextBox) GetPropString(key string, defaultValue string) string {
	if key == "text" {
		return c.Text()
	}
	return c.Widget.GetPropString(key, defaultValue)
}

// SetText replaces the text; the cursor goes to the start, the undo
// history starts anew
func (c *TextBox) SetText(text string) {
	// The selection is of the old text
	c.clearSelection()
	c.modifyText(textboxModifyCommandSetText, KeyModifiers{}, text)
	c.ScrollToBegin()
	c.form.Update()
}

func (c *TextBox) Text() string {
	if !c.textValid {
		c.text = strings.Join(c.lines, "\n")
		c.textValid = true
	}
	return c.text
}

// remeasure measures the lines again, e.g. in another font
func (c *TextBox) remeasure() {
	c.measure++
	for i := range c.lineWidths {
		c.lineWidths[i] = -1
	}
	c.updateInnerSize()
	c.form.Update()
}

func (c *TextBox) SetHint(text string) {
	c.SetProp("hint", text)
}

func (c *TextBox) Hint() string {
	return c.GetPropString("hint", "")
}

func (c *TextBox) SetMultiline(multiline bool) {
	c.Widget.SetProp("multiline", multiline)
	c.applyMultiline()
}

// applyMultiline sets what the mode decides: a multiline box scrolls and
// grows in both directions, a single-line one only in width
func (c *TextBox) applyMultiline() {
	if c.Multiline() {
		c.allowScrollX = true
		c.allowScrollY = true
		c.SetXExpandable(true)
		c.SetYExpandable(true)
	} else {
		c.allowScrollX = false
		c.allowScrollY = false
		c.SetXExpandable(true)
		c.SetYExpandable(false)
	}
	c.updateInnerSize()
	c.form.Update()
}

func (c *TextBox) Multiline() bool {
	return c.GetPropBool("multiline", false)
}

// ///////////////////////////////////////////////////////////////////////////////
// Props
// ProcessPropChange: only the mode changes the size policy. (Any property
// used to turn a multiline box expandable again, undoing SetXExpandable and
// SetYExpandable: bug #5.)
func (c *TextBox) ProcessPropChange(key string, value interface{}) {
	if c.propIsProcessing {
		return
	}
	c.propIsProcessing = true
	defer func() { c.propIsProcessing = false }()
	if key == "multiline" {
		c.applyMultiline()
	}
	c.form.Update()
}

// AssemblyText joins the lines with line breaks
func (c *TextBox) AssemblyText(lines []string) string {
	return strings.Join(lines, "\n")
}

// updateInnerSize sets the size of the content: the lines measured once,
// again only when they change
func (c *TextBox) updateInnerSize() {
	_, textHeight, err := MeasureText(c.FontFamily(), c.FontSize(), "0")
	if err != nil {
		return
	}
	c.innerHeight = textHeight * len(c.lines)

	var maxTextWidth int
	for i, line := range c.lines {
		if c.lineWidths[i] < 0 {
			w, _, err := MeasureText(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(line))
			if err != nil {
				return
			}
			c.lineWidths[i] = w
		}
		maxTextWidth = max(maxTextWidth, c.lineWidths[i])
	}
	c.innerWidth = maxTextWidth + c.textX() + c.leftAndRightPadding*2
	if c.Multiline() {
		c.allowScrollY = true
	}

	if !c.Multiline() {
		c.innerHeight = c.Height()
	}
}

func (c *TextBox) lineToPasswordChars(line string) string {
	if c.GetPropBool("ispassword", false) {
		return strings.Repeat("*", utf8.RuneCountInString(line))
	}
	return line
}

// ProcessPaint draws the frame on the whole widget, under the text and the
// scroll bars: the text is drawn in the view only and the frame doesn't
// scroll with it
func (c *TextBox) ProcessPaint(cnv *Canvas) {
	fill, border := inputFrameColors(&c.Widget)
	cnv.FillFrame(0, 0, c.Width(), c.Height(), themeControlRadius, fill, border)
	c.Widget.ProcessPaint(cnv)
}

func (c *TextBox) Draw(ctx *Canvas, width, height int) {
	p := CurrentPalette()

	textColor := colorToRGBA(c.ForegroundColor())
	if !c.Enabled() {
		textColor = p.DisabledText
	}

	lines := c.lines

	oneLineHeight := c.OneLineHeight()

	var yStaticOffset int
	if c.Multiline() {
		yStaticOffset = 1
	} else {
		yStaticOffset = (c.Height() - oneLineHeight) / 2
		//yStaticOffset = 0
	}

	// Only the lines in the view are drawn
	first, last := 0, len(lines)-1
	if c.Multiline() && oneLineHeight > 0 {
		_, vh := c.viewportSize()
		first = max(0, (c.scrollY-yStaticOffset)/oneLineHeight)
		last = min(last, (c.scrollY+vh)/oneLineHeight+1)
	}

	// Selection: the ranges of x each line's selected characters take, a
	// few where text written right to left is mixed in
	if len(c.selectedLines()) > 0 {
		selection := c.selectionRange()
		for selY := max(selection.Y1, first); selY <= min(selection.Y2, last); selY++ {
			line := c.lineToPasswordChars(lines[selY])
			from, to := 0, utf8.RuneCountInString(line)
			if selY == selection.Y1 {
				from = selection.X1
			}
			if selY == selection.Y2 {
				to = min(to, selection.X2)
			}
			rectY := selY * oneLineHeight
			if !c.Multiline() {
				rectY = yStaticOffset
			}
			for _, span := range TextSelectionSpans(c.FontFamily(), c.FontSize(), line, from, to) {
				lx := c.lineX(selY)
				x0, x1 := lx+span[0], lx+span[1]
				// A line the selection goes on to is marked from the edge
				// of the box (in text written a character after another)
				if selY != selection.Y1 && !needsShaping(line) && c.textAlign == TextAlignLeft {
					x0 = 0
				}
				ctx.FillRect(x0, rectY, x1-x0, oneLineHeight, p.Selection)
			}
		}
	}

	// Text
	ctx.SetColor(textColor)
	ctx.SetHAlign(HAlignLeft)
	ctx.SetVAlign(VAlignCenter)
	ctx.SetFontFamily(c.FontFamily())
	ctx.SetFontSize(c.FontSize())
	for i := first; i <= last; i++ {
		textX := c.lineX(i)
		y := yStaticOffset + i*oneLineHeight
		if c.highlighter != nil && !c.GetPropBool("ispassword", false) {
			c.drawHighlighted(ctx, i, lines[i], textX, y, oneLineHeight, textColor)
			continue
		}
		w := width - textX - c.leftAndRightPadding
		if c.textAlign != TextAlignLeft {
			// Moved to the right of the content: as wide as it is
			w = max(w, c.lineWidths[i]+1)
		}
		ctx.DrawText(textX, y, w, oneLineHeight, c.lineToPasswordChars(lines[i]))
	}
	c.drawGutter(ctx, first, last, yStaticOffset, oneLineHeight)

	// Compare by id via IsFocused() rather than "c.form.focusedWidget == c":
	// for a TextBox nested inside a custom composite widget, the interface
	// value stored as focusedWidget can carry the promoted *Widget type
	// instead of *TextBox, so a direct pointer/interface comparison against
	// c never matches even though the widget is genuinely focused.
	focus := c.IsFocused()

	// Cursor
	if focus && c.cursorVisible {
		cursorPosInPixels := c.lineX(c.cursorPosY) + c.caretX(c.cursorPosY, c.cursorPosX)
		curX := cursorPosInPixels - (c.cursorWidth / 2)
		curY := yStaticOffset + c.cursorPosY*oneLineHeight
		ctx.FillRect(curX, curY, c.cursorWidth, oneLineHeight, textColor)
	}

	if len(lines) == 1 && lines[0] == "" && c.Hint() != "" && !focus {
		ctx.SetHAlign(HAlignLeft)
		if c.Multiline() {
			ctx.SetVAlign(VAlignTop)
		} else {
			ctx.SetVAlign(VAlignCenter)
		}
		ctx.SetColor(p.PlaceholderText)
		ctx.DrawText(c.textX(), 0, c.w, c.h, c.Hint())
	}
}

// caretRect is the caret in the content of the text box (as Draw draws
// it): where an input method shows its window
func (c *TextBox) caretRect() (x, y, width, height int) {
	lineHeight := c.OneLineHeight()
	y = (c.Height() - lineHeight) / 2
	if c.Multiline() {
		y = 1 + c.cursorPosY*lineHeight
	}
	line := min(c.cursorPosY, len(c.lines)-1)
	x = c.lineX(line) + c.caretX(line, c.cursorPosX)
	return x, y, 1, lineHeight
}

func (c *TextBox) KeyChar(ch rune, mods KeyModifiers) {
	if c.GetPropBool("readonly", false) {
		return
	}

	c.redraw()
	if ch < 32 {
		return
	}

	c.modifyText(textboxModifyCommandInsertChar, mods, ch)
}

func (c *TextBox) cutSelected() {
	if c.ReadOnly() {
		return
	}

	if len(c.selectedLines()) == 0 {
		return
	}
	selectedText := c.SelectedText()
	if selectedText == "" {
		return
	}
	ClipboardSetText(selectedText)
	c.modifyText(textboxModifyCommandDelete, KeyModifiers{}, nil)
}

func (c *TextBox) copySelected() {
	if len(c.selectedLines()) == 0 {
		return
	}

	selectedText := c.SelectedText()

	if selectedText == "" {
		return
	}

	ClipboardSetText(selectedText)
}

func (c *TextBox) paste() {
	if c.ReadOnly() {
		return
	}

	text, err := ClipboardGetText()
	if err != nil {
		return
	}

	if text == "" {
		return
	}

	c.modifyText(textboxModifyCommandInsertString, KeyModifiers{}, text)
}

func (c *TextBox) KeyDown(key Key, mods KeyModifiers) bool {

	keyDownFunc := c.GetPropFunction("onkeydown")
	if keyDownFunc != nil {
		var ev EventTextboxKeyDown
		ev.Key = key
		ev.Mods = mods
		ev.Processed = false
		PushEvent(&ev)
		keyDownFunc()
		PopEvent()
		if ev.Processed {
			return true
		}
	}

	c.redraw()

	command := (mods.Ctrl || mods.Cmd) && !mods.Alt // Cmd on macOS
	if command && key == KeyA {
		c.SelectAllText()
		return true
	}

	if (mods.Ctrl || mods.Cmd) && !mods.Alt {
		switch {
		case key == KeyZ && !mods.Shift:
			c.Undo()
			return true
		case key == KeyY || (key == KeyZ && mods.Shift):
			c.Redo()
			return true
		}
	}

	if command && key == KeyX {
		c.cutSelected()
		return true
	}

	if command && key == KeyV {
		c.paste()
		return true
	}

	if command && key == KeyC {
		c.copySelected()
		return true
	}

	// Left and right go over a letter with its marks, a conjunct of an
	// Indic script, in the order of the text (in text written right to
	// left, Left goes back to its start, on the right)
	if (key == KeyArrowLeft || key == KeyArrowRight) && c.visualArrows {
		if x, ok := c.visualStep(key == KeyArrowRight); ok {
			c.moveCursor(x, c.cursorPosY, mods)
			return true
		}
	}

	if key == KeyArrowLeft {
		x := c.cursorPosX - 1
		if line := c.lines[c.cursorPosY]; x >= 0 && needsShaping(line) {
			x = prevGrapheme(line, c.cursorPosX)
		}
		c.moveCursor(x, c.cursorPosY, mods)
		return true
	}

	if key == KeyArrowRight {
		x := c.cursorPosX + 1
		if line := c.lines[c.cursorPosY]; needsShaping(line) {
			x = nextGrapheme(line, c.cursorPosX)
		}
		c.moveCursor(x, c.cursorPosY, mods)
		return true
	}

	if key == KeyArrowUp {
		c.moveCursorToLine(c.cursorPosY-1, mods)
		return true
	}

	if key == KeyArrowDown {
		c.moveCursorToLine(c.cursorPosY+1, mods)
		return true
	}

	if key == KeyHome {
		if (mods.Ctrl || mods.Cmd) && c.Multiline() {
			c.moveCursor(0, 0, mods)
			return true
		}
		c.moveCursor(0, c.cursorPosY, mods)
		return true
	}

	if key == KeyEnter {
		if !c.Multiline() && mods == (KeyModifiers{}) {
			c.textAtFocus = c.Text()
			c.OnEditingFinished.Emit(c.Text())
			if c.OnSubmit.Connected() {
				c.OnSubmit.Emit(c.Text())
				return true
			}
		}
		if c.ReadOnly() {
			return true
		}
		return c.insertReturn(mods)
	}

	if key == KeyEnd {
		if (mods.Ctrl || mods.Cmd) && c.Multiline() {
			last := len(c.lines) - 1
			c.moveCursor(utf8.RuneCountInString(c.lines[last]), last, mods)
			return true
		}
		if c.cursorPosY >= len(c.lines) {
			return true
		}
		c.moveCursor(utf8.RuneCountInString(c.lines[c.cursorPosY]), c.cursorPosY, mods)
		return true
	}

	if key == KeyBackspace {
		if c.ReadOnly() {
			return true
		}
		c.modifyText(textboxModifyCommandBackspace, mods, nil)
		return true
	}

	if key == KeyDelete {
		if c.ReadOnly() {
			return true
		}
		c.modifyText(textboxModifyCommandDelete, mods, nil)
		return true
	}

	return false
}

func (c *TextBox) KeyUp(key Key, mods KeyModifiers) {
}

func (c *TextBox) MouseDown(button MouseButton, x int, y int, mods KeyModifiers) {
	if button == MouseButtonLeft {
		c.redraw()
		c.mouseButtonPressed = true
		c.moveCursorNearPoint(x, y, mods)
		c.selectionLeftX = c.cursorPosX
		c.selectionLeftY = c.cursorPosY
		c.selectionRightX = c.cursorPosX
		c.selectionRightY = c.cursorPosY
		c.dragingCursor = true
		c.cursorVisible = true
		c.skipOneCursorBlinking = true
		c.form.Update()
	}
}

func (c *TextBox) MouseMove(x int, y int, mods KeyModifiers) {
	c.redraw()
	if c.mouseButtonPressed {
		c.moveCursorNearPoint(x, y, mods)
	}
	c.Update()
}

func (c *TextBox) moveCursorNearPoint(x, y int, modifiers KeyModifiers) {
	lines := c.lines

	_, textHeight, err := MeasureText(c.FontFamily(), c.FontSize(), "0")
	if err != nil {
		return
	}
	lineNumber := y / textHeight

	if lineNumber >= len(lines) {
		lineNumber = len(lines) - 1
	}

	if lineNumber < 0 {
		lineNumber = 0
	}

	// Shaped text: the caret nearest the point, at the start of a letter
	// with its marks
	if line := c.lineToPasswordChars(lines[lineNumber]); needsShaping(line) {
		i := TextIndexAt(c.FontFamily(), c.FontSize(), line, x-c.lineX(lineNumber))
		c.moveCursor(snapToGrapheme(line, i), lineNumber, modifiers)
		return
	}

	charPos, _ := GetCharPositions(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(lines[lineNumber]))
	for i := 0; i < len(charPos); i++ {
		charPos[i] = charPos[i] + c.lineX(lineNumber)
	}

	if len(charPos) == 1 {
		c.moveCursor(0, lineNumber, modifiers)
		return
	}

	if x < charPos[1]-(charPos[1]-charPos[0])/2 {
		c.moveCursor(0, lineNumber, modifiers)
	}

	for pos := 1; pos < len(charPos)-1; pos++ {
		left := charPos[pos] - (charPos[pos]-charPos[pos-1])/2
		right := charPos[pos] + (charPos[pos+1]-charPos[pos])/2
		if x >= left && x < right {
			c.moveCursor(pos, lineNumber, modifiers)
			break
		}
	}

	widthOfLastChar := 0
	if len(charPos) > 1 {
		widthOfLastChar = charPos[len(charPos)-1] - charPos[len(charPos)-2]
	}

	if x > charPos[len(charPos)-1]-widthOfLastChar/2 {
		c.moveCursor(len(charPos)-1, lineNumber, modifiers)
	}
}

func (c *TextBox) MouseUp(button MouseButton, x int, y int, mods KeyModifiers) {
	c.dragingCursor = false
	c.redraw()
	c.mouseButtonPressed = false
	c.form.Update()
}

func (c *TextBox) insertReturn(modifiers KeyModifiers) bool {
	if !c.Multiline() {
		return false
	}

	c.modifyText(textboxModifyCommandInsertReturn, modifiers, nil)
	return true
}

func (c *TextBox) selectionRange() TextBoxSelection {
	var result TextBoxSelection
	//var res1X, res1Y, res2X, res2Y int
	if c.selectionLeftY > c.selectionRightY {
		result.Y1 = c.selectionRightY
		result.Y2 = c.selectionLeftY
		result.X1 = c.selectionRightX
		result.X2 = c.selectionLeftX
	}

	if c.selectionLeftY < c.selectionRightY {
		result.Y2 = c.selectionRightY
		result.Y1 = c.selectionLeftY
		result.X2 = c.selectionRightX
		result.X1 = c.selectionLeftX
	}

	if c.selectionLeftY == c.selectionRightY {
		result.Y1 = c.selectionLeftY
		result.Y2 = c.selectionRightY

		if c.selectionLeftX > c.selectionRightX {
			result.X1 = c.selectionRightX
			result.X2 = c.selectionLeftX
		} else {
			result.X2 = c.selectionRightX
			result.X1 = c.selectionLeftX
		}
	}

	return result
}

func (c *TextBox) selectedLines() []int {
	var result []int
	result = make([]int, 0)
	selection := c.selectionRange()
	if selection.Y2 != selection.Y1 {
		for i := selection.Y1; i <= selection.Y2; i++ {
			result = append(result, i)
		}
	} else {
		if selection.X1 != selection.X2 {
			result = append(result, selection.Y1)
		}
	}
	return result
}

func (c *TextBox) moveCursor(posX int, posY int, modifiers KeyModifiers) {
	lines := c.lines

	if posY < 0 {
		return
	}

	if posY >= len(lines) {
		return
	}

	if posX < 0 {
		return
	}

	if n := utf8.RuneCountInString(lines[posY]); posX > n {
		posX = n
	}

	c.cursorPosX = posX
	c.cursorPosY = posY

	if !modifiers.Shift && !c.mouseButtonPressed {
		c.clearSelection()
	}

	if modifiers.Shift || c.dragingCursor {
		c.selectionRightX = c.cursorPosX
		c.selectionRightY = c.cursorPosY
	}

	c.ensureVisibleCursor()
	c.form.Update()
}

// caretX returns the x of the caret before the character x of the line y,
// from the start of the text: where the layout of the line puts it
func (c *TextBox) caretX(y, x int) int {
	return TextCaretX(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(c.lines[y]), x)
}

// moveCursorToLine moves the cursor up or down to the line, at the same
// character, the start of a letter with its marks there
func (c *TextBox) moveCursorToLine(y int, modifiers KeyModifiers) {
	x := c.cursorPosX
	if y >= 0 && y < len(c.lines) {
		if line := c.lines[y]; needsShaping(line) {
			x = snapToGrapheme(line, min(x, utf8.RuneCountInString(line)))
		}
	}
	c.moveCursor(x, y, modifiers)
}

func (c *TextBox) SelectedText() string {
	lines := c.lines
	selection := c.selectionRange()

	if selection.Y1 == selection.Y2 {
		runes1 := []rune(lines[selection.Y1])
		return string(runes1[selection.X1:selection.X2])
	}
	var b strings.Builder
	runes1 := []rune(lines[selection.Y1])
	b.WriteString(string(runes1[selection.X1:]))
	b.WriteByte('\n')
	for row := selection.Y1 + 1; row < selection.Y2; row++ {
		b.WriteString(lines[row])
		b.WriteByte('\n')
	}
	runes2 := []rune(lines[selection.Y2])
	b.WriteString(string(runes2[:selection.X2]))
	return b.String()
}

// setLine replaces the line i
func (c *TextBox) setLine(i int, s string) {
	c.lines[i] = s
	c.lineWidths[i] = -1
}

// insertLines puts the lines before the line i
func (c *TextBox) insertLines(i int, lines ...string) {
	c.lines = slices.Insert(c.lines, i, lines...)
	widths := make([]int, len(lines))
	for k := range widths {
		widths[k] = -1
	}
	c.lineWidths = slices.Insert(c.lineWidths, i, widths...)
}

// deleteLines removes the lines [i, j)
func (c *TextBox) deleteLines(i, j int) {
	c.lines = slices.Delete(c.lines, i, j)
	c.lineWidths = slices.Delete(c.lineWidths, i, j)
}

// deleteSelection removes the selected text and puts the cursor where it
// was; false if nothing is selected
func (c *TextBox) deleteSelection() bool {
	if len(c.selectedLines()) == 0 {
		return false
	}
	selection := c.selectionRange()
	runes1 := []rune(c.lines[selection.Y1])
	runes2 := []rune(c.lines[selection.Y2])
	c.setLine(selection.Y1, string(runes1[:selection.X1])+string(runes2[selection.X2:]))
	c.deleteLines(selection.Y1+1, selection.Y2+1)
	c.cursorPosX, c.cursorPosY = selection.X1, selection.Y1
	c.clearSelection()
	return true
}

func (c *TextBox) ensureVisibleCursor() {
	_, oneLineHeight, _ := MeasureText(c.FontFamily(), c.FontSize(), "Q")
	cursorPosInPixels := c.lineX(c.cursorPosY) + c.caretX(c.cursorPosY, c.cursorPosX)
	curX := cursorPosInPixels - (c.cursorWidth / 2)
	curY := c.cursorPosY * oneLineHeight
	// The column of the line numbers covers the left of the view
	c.ScrollEnsureVisible(max(0, curX-c.gutterWidth()-c.leftAndRightPadding), curY)
	c.ScrollEnsureVisible(curX+c.cursorWidth, curY+oneLineHeight)
}

func (c *TextBox) clearSelection() {
	c.selectionLeftX = c.cursorPosX
	c.selectionLeftY = c.cursorPosY
	c.selectionRightX = c.cursorPosX
	c.selectionRightY = c.cursorPosY
}

func (c *TextBox) modifyText(cmd textboxModifyCommand, modifiers KeyModifiers, data interface{}) {
	c.recordUndo(cmd)

	changed := false
	if cmd != textboxModifyCommandSetText {
		changed = c.deleteSelection()
	}
	curPosX, curPosY := c.cursorPosX, c.cursorPosY

	switch cmd {
	case textboxModifyCommandInsertChar:
		runes := []rune(c.lines[curPosY])
		c.setLine(curPosY, string(runes[:curPosX])+string(data.(rune))+string(runes[curPosX:]))
		curPosX++
		changed = true
	case textboxModifyCommandInsertReturn:
		runes := []rune(c.lines[curPosY])
		c.setLine(curPosY, string(runes[:curPosX]))
		c.insertLines(curPosY+1, string(runes[curPosX:]))
		curPosX = 0
		curPosY++
		changed = true
	case textboxModifyCommandBackspace:
		if changed { // the selection is what Backspace removes
			break
		}
		if curPosX > 0 {
			runes := []rune(c.lines[curPosY])
			c.setLine(curPosY, string(runes[:curPosX-1])+string(runes[curPosX:]))
			curPosX--
			changed = true
		} else if curPosY > 0 {
			curPosX = utf8.RuneCountInString(c.lines[curPosY-1])
			c.setLine(curPosY-1, c.lines[curPosY-1]+c.lines[curPosY])
			c.deleteLines(curPosY, curPosY+1)
			curPosY--
			changed = true
		}
	case textboxModifyCommandDelete:
		if changed {
			break
		}
		runes := []rune(c.lines[curPosY])
		if curPosX < len(runes) {
			// A letter with its marks, a conjunct, as one
			next := curPosX + 1
			if needsShaping(c.lines[curPosY]) {
				next = max(next, nextGrapheme(c.lines[curPosY], curPosX))
			}
			c.setLine(curPosY, string(runes[:curPosX])+string(runes[next:]))
			changed = true
		} else if curPosY < len(c.lines)-1 {
			c.setLine(curPosY, c.lines[curPosY]+c.lines[curPosY+1])
			c.deleteLines(curPosY+1, curPosY+2)
			changed = true
		}
	case textboxModifyCommandSetText:
		lines := splitLines(data.(string))
		changed = !slices.Equal(lines, c.lines)
		if changed {
			c.lines = lines
			c.lineWidths = make([]int, len(lines))
			for i := range c.lineWidths {
				c.lineWidths[i] = -1
			}
		}
		// The cursor stays where it was, within the new text
		curPosY = min(curPosY, len(c.lines)-1)
		curPosX = min(curPosX, utf8.RuneCountInString(c.lines[curPosY]))
	case textboxModifyCommandInsertString:
		// The whole text at once. (It used to be typed a character at a
		// time, each one rebuilding and measuring all the text: pasting
		// a few kilobytes froze the window for seconds.) Kept as typing
		// does: a line break only in a multiline box, no other control
		// characters.
		var b strings.Builder
		for _, ch := range data.(string) {
			if ch == '\n' {
				if c.Multiline() {
					b.WriteRune(ch)
				}
				continue
			}
			if ch < 32 {
				continue
			}
			b.WriteRune(ch)
		}
		if b.Len() == 0 {
			break
		}
		parts := strings.Split(b.String(), "\n")
		runes := []rune(c.lines[curPosY])
		left, right := string(runes[:curPosX]), string(runes[curPosX:])
		if len(parts) == 1 {
			c.setLine(curPosY, left+parts[0]+right)
			curPosX += utf8.RuneCountInString(parts[0])
		} else {
			last := parts[len(parts)-1]
			c.setLine(curPosY, left+parts[0])
			c.insertLines(curPosY+1, append(parts[1:len(parts)-1:len(parts)-1], last+right)...)
			curPosY += len(parts) - 1
			curPosX = utf8.RuneCountInString(last)
		}
		changed = true
	}

	if changed {
		c.textValid = false
	}
	c.updateInnerSize()
	c.moveCursor(curPosX, curPosY, modifiers)
	c.clearSelection()
	// The next typing at this place joins this undo step
	c.lastEditX, c.lastEditY = c.cursorPosX, c.cursorPosY

	if changed {
		c.textChanged(cmd != textboxModifyCommandSetText && !c.editByCode)
	}
	c.form.Update()
}

// textChanged tells the handlers the text changed, by the user or not
func (c *TextBox) textChanged(byUser bool) {
	if f := c.GetPropFunction("ontextchanged"); f != nil {
		PushEvent(nil)
		f()
		PopEvent()
	}
	// The text is joined for the handlers only
	if c.OnTextChanged.Connected() {
		c.OnTextChanged.Emit(c.Text())
	}
	if byUser && c.OnTextEdited.Connected() {
		c.OnTextEdited.Emit(c.Text())
	}
}

func (c *TextBox) SelectAllText() {
	last := len(c.lines) - 1
	c.selectionLeftX = 0
	c.selectionLeftY = 0
	c.selectionRightX = utf8.RuneCountInString(c.lines[last])
	c.selectionRightY = last
	c.form.Update()
}

// SelectRange selects the runes [from, to) of the first line and puts the
// cursor at the end of the selection (as file managers select a name
// without its extension)
func (c *TextBox) SelectRange(from, to int) {
	runes := []rune(c.lines[0])
	to = max(0, min(to, len(runes)))
	from = max(0, min(from, to))
	c.moveCursor(to, 0, KeyModifiers{})
	c.selectionLeftX, c.selectionLeftY = from, 0
	c.selectionRightX, c.selectionRightY = to, 0
	if c.form != nil {
		c.form.Update()
	}
}

func (c *TextBox) MoveCursorToEnd() {
	c.moveCursor(utf8.RuneCountInString(c.lines[c.cursorPosY]), c.cursorPosY, KeyModifiers{})
}

func (c *TextBox) ScrollToBegin() {
	c.ScrollEnsureVisible(0, 0)
	c.ScrollEnsureVisible(0, 1)
	//c.ScrollEnsureVisible(0, 0)
	//c.ScrollEnsureVisible(0, 1)
}

func (c *TextBox) OneLineHeight() int {
	_, fontHeight, _ := MeasureText(c.FontFamily(), c.FontSize(), "1Qg")
	return fontHeight
}

func (c *TextBox) MinHeight() int {
	return c.OneLineHeight() + 4
}

func (c *TextBox) AcceptsReturn() bool {
	return c.Multiline()
}

/*func (c *TextBox) FontFamily() string {
	return "robotomono"
}

func (c *TextBox) FontSize() float64 {
	return 16
}*/

// applyThemeMetrics measures the lines again: the font size or the scale
// of the screen (see Form.useTextScale) changed their widths
func (c *TextBox) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.remeasure()
}

// handlesKey: the text editing keys go to the text box before the shortcuts
// (see Form.AddShortcut), and Enter too when it starts a new line
func (c *TextBox) handlesKey(key Key, mods KeyModifiers) bool {
	return textEditingKey(key, mods) || (key == KeyEnter && c.Multiline())
}

// recordUndo saves the state before an edit. A run of typing (or of
// Backspace, Delete) at one place makes one step; the text set by the code
// (SetText) starts a new history.
func (c *TextBox) recordUndo(cmd textboxModifyCommand) {
	if cmd == textboxModifyCommandSetText {
		c.ClearUndo()
		return
	}
	grouped := false
	switch cmd {
	case textboxModifyCommandInsertChar, textboxModifyCommandBackspace, textboxModifyCommandDelete:
		sel := c.selectionRange()
		grouped = c.lastEditActive && c.lastEdit == cmd &&
			c.cursorPosX == c.lastEditX && c.cursorPosY == c.lastEditY &&
			sel.X1 == sel.X2 && sel.Y1 == sel.Y2 &&
			time.Since(c.lastEditTime) < textBoxUndoPause
	}
	if !grouped {
		c.undoStack = append(c.undoStack, c.state())
		if len(c.undoStack) > textBoxUndoLimit {
			c.undoStack = c.undoStack[1:]
		}
	}
	c.redoStack = nil
	c.lastEdit = cmd
	c.lastEditActive = true
	c.lastEditTime = time.Now()
	// Where the cursor ends is known after the edit: see afterEdit
	c.lastEditX, c.lastEditY = -1, -1
}

// Undo takes back the last edit of the user (Ctrl+Z)
func (c *TextBox) Undo() {
	if c.ReadOnly() || len(c.undoStack) == 0 {
		return
	}
	state := c.undoStack[len(c.undoStack)-1]
	c.undoStack = c.undoStack[:len(c.undoStack)-1]
	c.redoStack = append(c.redoStack, c.state())
	c.restoreState(state)
}

// Redo makes the edit taken back by Undo again (Ctrl+Y, Ctrl+Shift+Z)
func (c *TextBox) Redo() {
	if c.ReadOnly() || len(c.redoStack) == 0 {
		return
	}
	state := c.redoStack[len(c.redoStack)-1]
	c.redoStack = c.redoStack[:len(c.redoStack)-1]
	c.undoStack = append(c.undoStack, c.state())
	c.restoreState(state)
}

func (c *TextBox) CanUndo() bool {
	return len(c.undoStack) > 0
}

func (c *TextBox) CanRedo() bool {
	return len(c.redoStack) > 0
}

// ClearUndo forgets the undo history
func (c *TextBox) ClearUndo() {
	c.undoStack, c.redoStack = nil, nil
	c.lastEditActive = false
}

// state is the current state for the undo history: copies of the slices,
// the strings are shared
func (c *TextBox) state() textBoxState {
	return textBoxState{slices.Clone(c.lines), slices.Clone(c.lineWidths), c.measure, c.cursorPosX, c.cursorPosY}
}

// restoreState shows a state of the history, as an edit of the user. The
// state is the box's own from then on (taken off the history).
func (c *TextBox) restoreState(state textBoxState) {
	changed := !slices.Equal(state.lines, c.lines)
	c.lastEditActive = false
	c.lines, c.lineWidths = state.lines, state.widths
	if state.measure != c.measure {
		for i := range c.lineWidths {
			c.lineWidths[i] = -1
		}
	}
	c.textValid = false
	c.clearSelection()
	c.updateInnerSize()
	c.moveCursor(state.x, state.y, KeyModifiers{})
	c.clearSelection()
	c.ensureVisibleCursor()
	if changed {
		c.textChanged(true)
	}
	c.form.Update()
}

// The cursor, the selection and the editing from the code

// TextPos is a place in the text of a text box: the line and the column, in
// runes (characters), from 0
type TextPos struct {
	Line, Col int
}

// clampPos keeps the place within the text
func (c *TextBox) clampPos(pos TextPos) TextPos {
	pos.Line = max(0, min(pos.Line, len(c.lines)-1))
	pos.Col = max(0, min(pos.Col, utf8.RuneCountInString(c.lines[pos.Line])))
	return pos
}

// CursorPosition returns where the cursor is
func (c *TextBox) CursorPosition() TextPos {
	return TextPos{c.cursorPosY, c.cursorPosX}
}

// SetCursorPosition puts the cursor at the place (kept within the text),
// drops the selection and scrolls the place into view
func (c *TextBox) SetCursorPosition(pos TextPos) {
	pos = c.clampPos(pos)
	c.cursorPosX, c.cursorPosY = pos.Col, pos.Line
	c.clearSelection()
	c.ensureVisibleCursor()
	c.form.Update()
}

// Selection returns the selected part of the text, from <= to; from == to
// (the cursor) if nothing is selected
func (c *TextBox) Selection() (from, to TextPos) {
	sel := c.selectionRange()
	return TextPos{sel.Y1, sel.X1}, TextPos{sel.Y2, sel.X2}
}

// HasSelection reports whether a part of the text is selected
func (c *TextBox) HasSelection() bool {
	return len(c.selectedLines()) > 0
}

// SetSelection selects the text between anchor and cursor (in any order)
// and puts the cursor at cursor, scrolled into view: as Shift+arrows from
// anchor would
func (c *TextBox) SetSelection(anchor, cursor TextPos) {
	anchor, cursor = c.clampPos(anchor), c.clampPos(cursor)
	c.cursorPosX, c.cursorPosY = cursor.Col, cursor.Line
	c.selectionLeftX, c.selectionLeftY = anchor.Col, anchor.Line
	c.selectionRightX, c.selectionRightY = cursor.Col, cursor.Line
	c.ensureVisibleCursor()
	c.form.Update()
}

// InsertText puts the text at the cursor, replacing the selection, as a
// paste would: the edit can be undone (unlike SetText, which starts the
// history anew). OnTextChanged is emitted, OnTextEdited (of the user's
// edits) is not. To replace the whole text undoably:
//
//	box.SelectAllText()
//	box.InsertText(strings.ToUpper(box.Text()))
func (c *TextBox) InsertText(text string) {
	c.editByCode = true
	defer func() { c.editByCode = false }()
	c.modifyText(textboxModifyCommandInsertString, KeyModifiers{}, text)
	c.lastEditActive = false // the next typing is an undo step of its own
}

// Cut moves the selected text to the clipboard (Ctrl+X)
func (c *TextBox) Cut() {
	c.cutSelected()
}

// Copy copies the selected text to the clipboard (Ctrl+C)
func (c *TextBox) Copy() {
	c.copySelected()
}

// Paste puts the text of the clipboard at the cursor, replacing the
// selection (Ctrl+V)
func (c *TextBox) Paste() {
	c.paste()
}

// ScrollToCursor scrolls the cursor into view
func (c *TextBox) ScrollToCursor() {
	c.ensureVisibleCursor()
	c.form.Update()
}

// PosAt returns the place of the rune offset in the whole text (as Text()
// returns it, each line break one rune): for searching the text
func (c *TextBox) PosAt(offset int) TextPos {
	for i, line := range c.lines {
		n := utf8.RuneCountInString(line)
		if offset <= n {
			return TextPos{i, max(0, offset)}
		}
		offset -= n + 1
	}
	last := len(c.lines) - 1
	return TextPos{last, utf8.RuneCountInString(c.lines[last])}
}

// OffsetOf returns the rune offset of the place in the whole text, the
// inverse of PosAt
func (c *TextBox) OffsetOf(pos TextPos) int {
	pos = c.clampPos(pos)
	offset := 0
	for i := 0; i < pos.Line; i++ {
		offset += utf8.RuneCountInString(c.lines[i]) + 1
	}
	return offset + pos.Col
}

// Line numbers and highlighting: the text box as the editor of code

// TextSpan is a styled part of a line, see SetHighlighter: the runes
// [Start, End) of the line
type TextSpan struct {
	Start, End int
	Color      color.Color // the text color, nil for the box's
	Background color.Color // nil for none
	Bold       bool
	Italic     bool
}

// SetHighlighter colors the text: f gives the styled parts of a line (its
// index and text), for the lines shown only, at each paint - a highlighter
// of syntax, of the matches of a search. Parts may overlap: the later one
// wins. Bold and italic keep the places of the regular face: they fit
// best a monospaced font (FontFamilyMono), whose faces are as wide.
//
//	box.SetHighlighter(func(line int, text string) []ui.TextSpan {
//		if i := strings.Index(text, "//"); i >= 0 {
//			n := utf8.RuneCountInString(text[:i])
//			return []ui.TextSpan{{Start: n, End: utf8.RuneCountInString(text), Color: gray, Italic: true}}
//		}
//		return nil
//	})
func (c *TextBox) SetHighlighter(f func(line int, text string) []TextSpan) {
	c.highlighter = f
	c.form.Update()
}

// SetLineNumbers shows the numbers of the lines of a multiline box in a
// column at its left
func (c *TextBox) SetLineNumbers(show bool) {
	c.lineNumbers = show
	c.gutterDigits = 0
	c.updateInnerSize()
	c.form.Update()
}

func (c *TextBox) LineNumbers() bool {
	return c.lineNumbers
}

// TextAlign is where the lines of a TextBox are in it
type TextAlign int

const (
	// TextAlignLeft puts every line at the left (the default)
	TextAlignLeft TextAlign = iota
	// TextAlignRight puts every line at the right
	TextAlignRight
	// TextAlignCenter puts every line in the middle
	TextAlignCenter
	// TextAlignStart puts a line at the side its direction starts: a line
	// of Arabic or Hebrew at the right, the others at the left
	TextAlignStart
)

// SetTextAlign places the lines in the box: TextAlignStart for a box of
// Arabic or Hebrew text
func (c *TextBox) SetTextAlign(a TextAlign) {
	c.textAlign = a
	c.form.Update()
}

// TextAlign returns where the lines are placed
func (c *TextBox) TextAlign() TextAlign {
	return c.textAlign
}

// SetVisualCursorMovement makes Left and Right move the cursor on the
// screen - to the left and to the right - in text written right to left
// too; by default they go back and on in the order of the text
func (c *TextBox) SetVisualCursorMovement(on bool) {
	c.visualArrows = on
}

// lineX returns where the line y starts in the box, by the alignment
func (c *TextBox) lineX(y int) int {
	x := c.textX()
	if c.textAlign == TextAlignLeft || y < 0 || y >= len(c.lines) {
		return x
	}
	// The width the lines are aligned in: the view, or the widest line
	view := c.Width()
	if c.Multiline() {
		view, _ = c.viewportSize()
	}
	area := max(view, c.innerWidth) - x - c.leftAndRightPadding
	if c.lineWidths[y] < 0 {
		c.lineWidths[y], _, _ = MeasureText(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(c.lines[y]))
	}
	free := max(0, area-c.lineWidths[y])
	switch c.textAlign {
	case TextAlignRight:
		return x + free
	case TextAlignCenter:
		return x + free/2
	}
	if textIsRightToLeft(c.lines[y]) {
		return x + free
	}
	return x
}

// visualStep returns where Left (or Right) puts the cursor on the screen:
// the start of the grapheme whose caret is the nearest to the left (right)
// of the cursor's; false at the edge of the line
func (c *TextBox) visualStep(right bool) (int, bool) {
	line := c.lineToPasswordChars(c.lines[c.cursorPosY])
	if !needsShaping(line) {
		return 0, false // as the logical order
	}
	at := c.caretX(c.cursorPosY, c.cursorPosX)
	best, bestX := -1, 0
	for _, i := range graphemeBounds(line) {
		if i == c.cursorPosX {
			continue
		}
		x := c.caretX(c.cursorPosY, i)
		if (right && x > at && (best < 0 || x < bestX)) || (!right && x < at && (best < 0 || x > bestX)) {
			best, bestX = i, x
		}
	}
	if best < 0 {
		return c.cursorPosX, true
	}
	return best, true
}

// textX is where the text starts: after the inset, and the line numbers
func (c *TextBox) textX() int {
	return c.leftAndRightPadding + c.gutterWidth()
}

// gutterWidth is the width of the column of the line numbers, 0 without
func (c *TextBox) gutterWidth() int {
	if !c.lineNumbers || !c.Multiline() {
		return 0
	}
	digits := max(2, len(strconv.Itoa(len(c.lines))))
	if digits != c.gutterDigits {
		w, _, _ := MeasureText(c.FontFamily(), c.FontSize(), strings.Repeat("0", digits))
		c.gutterDigits, c.gutterW = digits, w+themeTextInset*2
	}
	return c.gutterW
}

// drawGutter draws the line numbers of the lines shown, in a column that
// stays at the left when the text scrolls sideways
func (c *TextBox) drawGutter(ctx *Canvas, first, last, yOffset, lineHeight int) {
	gw := c.gutterWidth()
	if gw == 0 {
		return
	}
	p := CurrentPalette()
	_, vh := c.viewportSize()
	x := c.scrollX
	ctx.FillRect(x, c.scrollY, gw, vh, p.Window)
	ctx.FillRect(x+gw-1, c.scrollY, 1, vh, p.Divider)
	ctx.SetFontFamily(c.FontFamily())
	ctx.SetFontSize(c.FontSize())
	ctx.SetHAlign(HAlignRight)
	ctx.SetVAlign(VAlignCenter)
	for i := first; i <= last; i++ {
		color := p.PlaceholderText
		if i == c.cursorPosY {
			color = p.Text
		}
		ctx.SetColor(color)
		ctx.DrawText(x, yOffset+i*lineHeight, gw-themeTextInset, lineHeight, strconv.Itoa(i+1))
	}
	ctx.SetHAlign(HAlignLeft)
}

// drawHighlighted draws a line in the styles of the highlighter
func (c *TextBox) drawHighlighted(ctx *Canvas, line int, text string, x, y, height int, textColor color.Color) {
	runes := []rune(text)
	spans := c.highlighter(line, text)
	if len(spans) == 0 || len(runes) == 0 {
		ctx.SetColor(textColor)
		ctx.DrawText(x, y, MaxInt/4, height, text)
		return
	}
	// The span of each rune, the later span winning
	of := make([]int, len(runes))
	for i := range of {
		of[i] = -1
	}
	for k, span := range spans {
		for i := max(0, span.Start); i < min(len(runes), span.End); i++ {
			of[i] = k
		}
	}
	if needsShaping(text) {
		c.drawHighlightedShaped(ctx, runes, spans, of, x, y, height, textColor)
		return
	}
	positions, err := GetCharPositions(c.FontFamily(), c.FontSize(), text)
	if err != nil {
		return
	}
	family := c.FontFamily()
	for start := 0; start < len(runes); {
		end := start + 1
		for end < len(runes) && of[end] == of[start] {
			end++
		}
		segColor, segFamily := textColor, family
		if k := of[start]; k >= 0 {
			span := spans[k]
			if span.Background != nil {
				ctx.FillRect(x+positions[start], y, positions[end]-positions[start], height, span.Background)
			}
			if span.Color != nil {
				segColor = span.Color
			}
			if span.Bold || span.Italic {
				segFamily = StyledFont(family, fontStyleOf(span.Bold, span.Italic))
			}
		}
		ctx.SetColor(segColor)
		ctx.SetFontFamily(segFamily)
		ctx.DrawText(x+positions[start], y, MaxInt/4, height, string(runes[start:end]))
		start = end
	}
	ctx.SetFontFamily(family)
}

// drawHighlightedShaped draws a line of shaped text in the styles of the
// highlighter: one layout of all of it, the parts in their styles, so
// letters join and the order is right across them. of is the part of
// each rune.
func (c *TextBox) drawHighlightedShaped(ctx *Canvas, runes []rune, spans []TextSpan, of []int, x, y, height int, textColor color.Color) {
	family, size := c.FontFamily(), c.FontSize()
	l := NewTextLayout("", TextStyle{Font: family, Size: size})
	type mark struct {
		from, to int // byte offsets
		col      color.Color
	}
	var backs []mark
	off := 0
	for start := 0; start < len(runes); {
		end := start + 1
		for end < len(runes) && of[end] == of[start] {
			end++
		}
		st := TextStyle{Font: family, Size: size, Color: colorToRGBA(textColor)}
		part := string(runes[start:end])
		if k := of[start]; k >= 0 {
			span := spans[k]
			if span.Color != nil {
				st.Color = colorToRGBA(span.Color)
			}
			st.Bold, st.Italic = span.Bold, span.Italic
			if span.Background != nil {
				backs = append(backs, mark{off, off + len(part), span.Background})
			}
		}
		l.Append(part, st)
		off += len(part)
		start = end
	}
	// The backgrounds as tall as the line, as the selection
	for _, b := range backs {
		for _, r := range l.SelectionRects(b.from, b.to) {
			x0, x1 := int(math.Round(r.MinX)), int(math.Round(r.MaxX))
			ctx.FillRect(x+x0, y, x1-x0, height, b.col)
		}
	}
	ctx.drawLayoutAsText(l, x, y, height, family, size)
}
