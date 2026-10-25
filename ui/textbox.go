package ui

import (
	"fmt"
	"slices"
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
	text       string // the joined text, if textValid
	textValid  bool
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

	dragingCursor bool

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
	x, y   int
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

	return &c
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
	c.innerWidth = maxTextWidth + c.leftAndRightPadding*3
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

	// Selection
	if len(c.selectedLines()) > 0 {
		selection := c.selectionRange()
		for selY := max(selection.Y1, first); selY <= min(selection.Y2, last); selY++ {
			lineCharPos, err := GetCharPositions(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(lines[selY]))

			if err != nil {
				return
			}
			for i := 0; i < len(lineCharPos); i++ {
				lineCharPos[i] = lineCharPos[i] + c.leftAndRightPadding
			}

			selXBegin := 0
			selXWidth := lineCharPos[len(lineCharPos)-1]
			if selY == selection.Y1 {
				selXBegin = lineCharPos[selection.X1]
				selXWidth = lineCharPos[len(lineCharPos)-1] - selXBegin
			}
			if selY == selection.Y2 {
				if selection.X2 < len(lineCharPos) {
					selXWidth = lineCharPos[selection.X2] - selXBegin
				}
			}

			rectY := selY * oneLineHeight

			if !c.Multiline() {
				rectY = yStaticOffset
			}

			ctx.FillRect(selXBegin, rectY, selXWidth, oneLineHeight, p.Selection)
		}
	}

	// Text
	ctx.SetColor(textColor)
	ctx.SetHAlign(HAlignLeft)
	ctx.SetVAlign(VAlignCenter)
	ctx.SetFontFamily(c.FontFamily())
	ctx.SetFontSize(c.FontSize())
	for i := first; i <= last; i++ {
		ctx.DrawText(c.leftAndRightPadding, yStaticOffset+i*oneLineHeight, width-c.leftAndRightPadding*2, oneLineHeight, c.lineToPasswordChars(lines[i]))
	}

	// Compare by id via IsFocused() rather than "c.form.focusedWidget == c":
	// for a TextBox nested inside a custom composite widget, the interface
	// value stored as focusedWidget can carry the promoted *Widget type
	// instead of *TextBox, so a direct pointer/interface comparison against
	// c never matches even though the widget is genuinely focused.
	focus := c.IsFocused()

	// Cursor
	if focus && c.cursorVisible {
		charPos, err := GetCharPositions(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(lines[c.cursorPosY]))
		for i := 0; i < len(charPos); i++ {
			charPos[i] = charPos[i] + c.leftAndRightPadding
		}
		if err != nil {
			return
		}
		if c.cursorPosX < len(charPos) {
			cursorPosInPixels := charPos[c.cursorPosX]
			curX := cursorPosInPixels - (c.cursorWidth / 2)
			curY := yStaticOffset + c.cursorPosY*oneLineHeight
			ctx.FillRect(curX, curY, c.cursorWidth, oneLineHeight, textColor)
		}
	}

	if len(lines) == 1 && lines[0] == "" && c.Hint() != "" && !focus {
		ctx.SetHAlign(HAlignLeft)
		if c.Multiline() {
			ctx.SetVAlign(VAlignTop)
		} else {
			ctx.SetVAlign(VAlignCenter)
		}
		ctx.SetColor(p.PlaceholderText)
		ctx.DrawText(c.leftAndRightPadding, 0, c.w, c.h, c.Hint())
	}
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

	if key == KeyArrowLeft {
		c.moveCursor(c.cursorPosX-1, c.cursorPosY, mods)
		return true
	}

	if key == KeyArrowRight {
		c.moveCursor(c.cursorPosX+1, c.cursorPosY, mods)
		return true
	}

	if key == KeyArrowUp {
		c.moveCursor(c.cursorPosX, c.cursorPosY-1, mods)
		return true
	}

	if key == KeyArrowDown {
		c.moveCursor(c.cursorPosX, c.cursorPosY+1, mods)
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
	c.form.Update()
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

	charPos, _ := GetCharPositions(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(lines[lineNumber]))
	for i := 0; i < len(charPos); i++ {
		charPos[i] = charPos[i] + c.leftAndRightPadding
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
	lines := c.lines
	_, oneLineHeight, _ := MeasureText(c.FontFamily(), c.FontSize(), "Q")
	charPos, err := GetCharPositions(c.FontFamily(), c.FontSize(), c.lineToPasswordChars(lines[c.cursorPosY]))
	for i := 0; i < len(charPos); i++ {
		charPos[i] = charPos[i] + c.leftAndRightPadding
	}
	if err != nil {
		return
	}
	cursorPosInPixels := charPos[c.cursorPosX]
	curX := cursorPosInPixels - (c.cursorWidth / 2)
	curY := c.cursorPosY * oneLineHeight
	// ctx.FillRect(curX, curY, c.cursorWidth, oneLineHeight)
	//c.ScrollEnsureVisible(curX, curY)
	//c.ScrollEnsureVisible(curX+c.cursorWidth, curY+oneLineHeight)
	c.ScrollEnsureVisible(curX, curY)
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
			c.setLine(curPosY, string(runes[:curPosX])+string(runes[curPosX+1:]))
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

func (c *TextBox) applyThemeMetrics() {
	c.Widget.applyThemeMetrics()
	c.updateInnerSize()
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
	return textBoxState{slices.Clone(c.lines), slices.Clone(c.lineWidths), c.cursorPosX, c.cursorPosY}
}

// restoreState shows a state of the history, as an edit of the user. The
// state is the box's own from then on (taken off the history).
func (c *TextBox) restoreState(state textBoxState) {
	changed := !slices.Equal(state.lines, c.lines)
	c.lastEditActive = false
	c.lines, c.lineWidths = state.lines, state.widths
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
