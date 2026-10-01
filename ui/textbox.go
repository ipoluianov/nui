package ui

import (
	"strings"
	"time"
	"unicode/utf8"
)

type TextBox struct {
	Widget

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

	blockUpdate bool

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

// textBoxState is a step of the undo history: the text and the cursor
type textBoxState struct {
	text string
	x, y int
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

	c.SetOnKeyDown(func(key Key, mods KeyModifiers) bool {
		return c.KeyDown(key, mods)
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

func (c *TextBox) Lines() []string {
	text := c.Text()
	lines := strings.Split(strings.Replace(text, "\r", "", -1), "\n")
	return lines
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

func (c *TextBox) SetProp(key string, value any) {
	c.Widget.SetProp(key, value)
	if key == "text" {
		c.setText(c.GetPropString("text", ""), false)
	}
}

func (c *TextBox) setText(text string, updateProp bool) {
	c.redraw()
	var modifiers KeyModifiers
	c.modifyText(textboxModifyCommandSetText, modifiers, text)
	c.updateInnerSize()
	c.ScrollToBegin()
	c.form.Update()

	if updateProp {
		c.SetProp("text", text)
	}
}

func (c *TextBox) SetText(text string) {
	c.setText(text, true)
}

func (c *TextBox) Text() string {
	return c.GetPropString("text", "")
}

func (c *TextBox) SetHint(text string) {
	c.SetProp("hint", text)
}

func (c *TextBox) Hint() string {
	return c.GetPropString("hint", "")
}

func (c *TextBox) SetMultiline(multiline bool) {
	c.SetProp("multiline", multiline)
}

func (c *TextBox) Multiline() bool {
	return c.GetPropBool("multiline", false)
}

// ///////////////////////////////////////////////////////////////////////////////
// Props
func (c *TextBox) ProcessPropChange(key string, value interface{}) {
	if c.propIsProcessing {
		return
	}
	c.propIsProcessing = true

	multiline := c.GetPropBool("multiline", false)
	if multiline {
		c.allowScrollX = true
		c.allowScrollY = true
		c.SetXExpandable(true)
		c.SetYExpandable(true)
		//c.verticalScrollVisible.SetOwnValue(true)
		//c.horizontalScrollVisible.SetOwnValue(true)
	} else {
		c.SetXExpandable(true)
		c.SetYExpandable(false)
	}
	c.updateInnerSize()
	c.form.Update()

	c.propIsProcessing = false
}

func (c *TextBox) AssemblyText(lines []string) string {
	result := ""
	for pos, line := range lines {
		result += line
		if pos < len(lines)-1 {
			result += "\n"
		}
	}
	return result
}

func (c *TextBox) updateInnerSize() {
	lines := c.Lines()

	_, textHeight, err := MeasureText(c.FontFamily(), c.FontSize(), "0")
	if err != nil {
		return
	}
	c.innerHeight = textHeight * len(lines)

	var maxTextWidth int
	for _, line := range lines {
		textWidth, _, err := MeasureText(c.FontFamily(), c.FontSize(), line)
		if err != nil {
			return
		}
		if textWidth > maxTextWidth {
			maxTextWidth = textWidth
		}
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
		lenOfLine := utf8.RuneCountInString(line)
		line = ""
		for i := 0; i < lenOfLine; i++ {
			line += "*"
		}
	}
	return line
}

func (c *TextBox) Draw(ctx *Canvas, width, height int) {
	p := CurrentPalette()

	// The frame doesn't scroll with the text
	fill, border := inputFrameColors(&c.Widget)
	ctx.FillFrame(c.scrollX, c.scrollY, c.Width(), c.Height(), themeControlRadius, fill, border)

	textColor := colorToRGBA(c.ForegroundColor())
	if !c.Enabled() {
		textColor = p.DisabledText
	}

	lines := c.Lines()

	oneLineHeight := c.OneLineHeight()

	var yStaticOffset int
	if c.Multiline() {
		yStaticOffset = 1
	} else {
		yStaticOffset = (c.Height() - oneLineHeight) / 2
		//yStaticOffset = 0
	}

	_ = yStaticOffset

	// Selection
	if len(c.selectedLines()) > 0 {
		selection := c.selectionRange()
		for selY := selection.Y1; selY <= selection.Y2; selY++ {
			lineCharPos, err := GetCharPositions(c.FontFamily(), c.FontSize(), lines[selY])

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
	yOffset := 0

	for _, line := range lines {
		line = c.lineToPasswordChars(line)
		ctx.SetColor(textColor)
		_, textHeightInLine, err := MeasureText(c.FontFamily(), c.FontSize(), line)
		ctx.SetHAlign(HAlignLeft)
		ctx.SetVAlign(VAlignCenter)
		ctx.SetFontFamily(c.FontFamily())
		ctx.SetFontSize(c.FontSize())
		ctx.DrawText(c.leftAndRightPadding, yStaticOffset+yOffset, width-c.leftAndRightPadding*2, textHeightInLine, line)

		if err != nil {
			return
		}
		yOffset += oneLineHeight
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

	if c.Text() == "" && c.Hint() != "" && !focus {
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

	if mods.Ctrl && key == KeyA {
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

	if mods.Ctrl && key == KeyX {
		c.cutSelected()
		return true
	}

	if mods.Ctrl && key == KeyV {
		c.paste()
		return true
	}

	if mods.Ctrl && key == KeyC {
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
		c.moveCursor(0, c.cursorPosY, mods)
		return true
	}

	if key == KeyEnter {
		if c.ReadOnly() {
			return true
		}
		return c.insertReturn(mods)
	}

	if key == KeyEnd {
		lines := c.Lines()
		if c.cursorPosY >= len(lines) {
			return true
		}
		runes := []rune(lines[c.cursorPosY])
		c.moveCursor(len(runes), c.cursorPosY, mods)
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
	lines := c.Lines()

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

	charPos, _ := GetCharPositions(c.FontFamily(), c.FontSize(), lines[lineNumber])
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
	lines := c.Lines()

	if posY < 0 {
		return
	}

	if posY >= len(lines) {
		return
	}

	runes := []rune(lines[posY])

	if posX < 0 {
		return
	}

	if posX > len(runes) {
		posX = len(runes)
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

	if !c.blockUpdate {
		c.ensureVisibleCursor()
	}
	c.form.Update()
}

func (c *TextBox) SelectedText() string {
	lines := c.Lines()
	result := ""

	//lines := make([]string, 0)
	selection := c.selectionRange()

	if selection.Y1 == selection.Y2 {
		runes1 := []rune(lines[selection.Y1])
		result += string(runes1[selection.X1:selection.X2])
	} else {
		runes1 := []rune(lines[selection.Y1])
		result += string(runes1[selection.X1:])
		result += "\n"

		if selection.Y2-selection.Y1 > 1 {
			for row := selection.Y1 + 1; row < selection.Y2; row++ {
				result += lines[row]
				result += "\n"
			}
		}

		runes2 := []rune(lines[selection.Y2])
		result += string(runes2[0:selection.X2])
	}

	return result
}

func (c *TextBox) removeSelectedText(modifiers KeyModifiers) (bool, []string, int, int) {
	oldLines := c.Lines()
	lines := make([]string, 0)
	modified := false
	selection := c.selectionRange()
	curPosX := c.cursorPosX
	curPosY := c.cursorPosY
	if len(c.selectedLines()) > 0 {
		lines = append(lines, oldLines[0:selection.Y1]...)
		runes1 := []rune(oldLines[selection.Y1])
		runes2 := []rune(oldLines[selection.Y2])
		lines = append(lines, string(runes1[0:selection.X1])+string(runes2[selection.X2:]))
		lines = append(lines, oldLines[selection.Y2+1:]...)
		modified = true
		curPosX = selection.X1
		curPosY = selection.Y1
	} else {
		lines = append(lines, oldLines...)
	}

	return modified, lines, curPosX, curPosY
}

func (c *TextBox) ensureVisibleCursor() {
	lines := c.Lines()
	_, oneLineHeight, _ := MeasureText(c.FontFamily(), c.FontSize(), "Q")
	charPos, err := GetCharPositions(c.FontFamily(), c.FontSize(), lines[c.cursorPosY])
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
	c.redraw()
	// Pasting inserts the text a character at a time, through here again
	// (blockUpdate): the whole paste is one step
	if !c.blockUpdate {
		c.recordUndo(cmd)
	}

	valid := true
	selectedTextRemoved, lines, curPosX, curPosY := c.removeSelectedText(modifiers)

	switch cmd {
	case textboxModifyCommandInsertChar:
		{
			out := []rune(lines[curPosY])
			left := string(out[0:curPosX])
			right := string(out[curPosX:])
			lines[curPosY] = left + string(data.(rune)) + right
			curPosX += 1
		}
	case textboxModifyCommandInsertReturn:
		{
			runes := []rune(lines[curPosY])
			left := string(runes[0:curPosX])
			right := string(runes[curPosX:])
			linesBefore := lines[0:curPosY]
			linesAfter := lines[curPosY:]
			lines = append(linesBefore, right)
			lines = append(lines, linesAfter...)
			lines[curPosY] = left
			curPosX = 0
			curPosY++
		}
	case textboxModifyCommandBackspace:
		{
			runes := []rune(lines[curPosY])
			if !selectedTextRemoved {
				if curPosX > 0 {
					left := string(runes[0 : curPosX-1])
					right := string(runes[curPosX:])
					lines[curPosY] = left + right
					curPosX = curPosX - 1
				} else {
					if curPosY > 0 {
						runes := []rune(lines[curPosY-1])
						newCursorPosX := len(runes)
						linesTemp := make([]string, 0)
						linesTemp = append(linesTemp, lines[0:curPosY]...)
						linesTemp[curPosY-1] += lines[curPosY]
						linesTemp = append(linesTemp, lines[curPosY+1:]...)
						lines = linesTemp
						curPosX = newCursorPosX
						curPosY = curPosY - 1
					}
				}
			}
		}
	case textboxModifyCommandDelete:
		{
			runes := []rune(lines[curPosY])
			if !selectedTextRemoved {
				if curPosX < len(runes) {
					left := string(runes[0:curPosX])
					right := string(runes[curPosX+1:])
					lines[curPosY] = left + right
				} else {
					if curPosY < len(lines)-1 {
						linesTemp := make([]string, 0)
						linesTemp = append(linesTemp, lines[0:curPosY+1]...)
						linesTemp[curPosY] += lines[curPosY+1]
						linesTemp = append(linesTemp, lines[curPosY+2:]...)
						lines = linesTemp
					}
				}
			}
		}
	case textboxModifyCommandSetText:
		{
			lines = strings.Split(strings.Replace(data.(string), "\r", "", -1), "\n")

			// Ensure cursor is in a valid position
			if curPosY >= len(lines) {
				curPosY = len(lines) - 1
			}
			runes := []rune(lines[curPosY])
			if curPosX > len(runes) {
				curPosX = len(runes)
			}
		}
	case textboxModifyCommandInsertString:
		{
			c.blockUpdate = true
			runes := string(data.(string))
			for _, ch := range runes {
				if ch < 32 {
					if ch == 10 {
						c.insertReturn(modifiers)
					}
				}

				c.KeyChar(ch, modifiers)
			}
			lines = c.Lines()
			curPosX = c.cursorPosX
			curPosY = c.cursorPosY
			c.blockUpdate = false
		}
	}

	if valid {
		newText := c.AssemblyText(lines)
		c.Widget.SetProp("text", newText)
		c.updateInnerSize()
		c.moveCursor(curPosX, curPosY, modifiers)

		if !c.blockUpdate {
			c.clearSelection()
			c.updateInnerSize()
			// The next typing at this place joins this undo step
			c.lastEditX, c.lastEditY = c.cursorPosX, c.cursorPosY

			f := c.GetPropFunction("ontextchanged")
			if f != nil {
				PushEvent(nil)
				f()
				PopEvent()
			}
		}

	}

	c.form.Update()
}

func (c *TextBox) SelectAllText() {
	lines := c.Lines()
	runesLast := []rune(lines[len(lines)-1])
	c.selectionLeftX = 0
	c.selectionLeftY = 0
	c.selectionRightX = len(runesLast)
	c.selectionRightY = len(lines) - 1
}

func (c *TextBox) MoveCursorToEnd() {
	lines := c.Lines()
	runes := []rune(lines[c.cursorPosY])
	c.moveCursor(len(runes), c.cursorPosY, KeyModifiers{})
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
		c.undoStack = append(c.undoStack, textBoxState{c.Text(), c.cursorPosX, c.cursorPosY})
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
	c.redoStack = append(c.redoStack, textBoxState{c.Text(), c.cursorPosX, c.cursorPosY})
	c.restoreState(state)
}

// Redo makes the edit taken back by Undo again (Ctrl+Y, Ctrl+Shift+Z)
func (c *TextBox) Redo() {
	if c.ReadOnly() || len(c.redoStack) == 0 {
		return
	}
	state := c.redoStack[len(c.redoStack)-1]
	c.redoStack = c.redoStack[:len(c.redoStack)-1]
	c.undoStack = append(c.undoStack, textBoxState{c.Text(), c.cursorPosX, c.cursorPosY})
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

// restoreState shows a state of the history, as an edit of the user
func (c *TextBox) restoreState(state textBoxState) {
	c.lastEditActive = false
	c.Widget.SetProp("text", state.text)
	c.updateInnerSize()
	c.moveCursor(state.x, state.y, KeyModifiers{})
	c.clearSelection()
	c.ensureVisibleCursor()
	if f := c.GetPropFunction("ontextchanged"); f != nil {
		PushEvent(nil)
		f()
		PopEvent()
	}
	c.form.Update()
}
