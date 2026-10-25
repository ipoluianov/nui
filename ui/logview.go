package ui

import (
	"slices"
	"strings"
	"sync"
	"unicode/utf8"
)

// LogView shows a growing log: lines are appended at the end, the oldest
// dropped past a limit, and the view follows the end while the user is at
// it (scrolled up, it stays where the user is). It is a read-only text box:
// the text can be selected, copied, searched (TextBox's API).
//
//	logs := ui.NewLogView()
//	logs.AppendLine("started")
//	log.SetOutput(logs) // an io.Writer: from any goroutine
type LogView struct {
	TextBox
	maxLines int

	mu      sync.Mutex
	pending []byte // written, not shown yet (Write)
	posted  bool
}

// logViewDefaultMaxLines is how many lines a log view keeps by default
const logViewDefaultMaxLines = 100_000

// NewLogView makes an empty log view in the monospaced font
func NewLogView() *LogView {
	c := &LogView{maxLines: logViewDefaultMaxLines}
	c.initTextBox()
	c.SetTypeName("LogView")
	c.SetMultiline(true)
	c.SetReadOnly(true)
	c.SetFontFamily(FontFamilyMono)
	return c
}

// SetMaxLines sets how many lines the view keeps; the oldest go
func (c *LogView) SetMaxLines(n int) {
	c.maxLines = max(1, n)
	c.trim()
}

// AppendLine adds a line at the end; on the UI thread (see Write for the
// other goroutines). A text with line breaks adds several.
func (c *LogView) AppendLine(text string) {
	c.appendLines(splitLines(text))
}

// AppendLines adds the lines at the end
func (c *LogView) appendLines(lines []string) {
	follow := c.atEnd()
	if len(c.lines) == 1 && c.lines[0] == "" {
		c.lines, c.lineWidths = c.lines[:0], c.lineWidths[:0]
	}
	for _, line := range lines {
		c.lines = append(c.lines, line)
		c.lineWidths = append(c.lineWidths, -1)
	}
	c.textValid = false
	c.trim()
	c.updateSize()
	if follow {
		c.scrollToEnd()
	}
	c.form.Update()
	if c.OnTextChanged.Connected() {
		c.OnTextChanged.Emit(c.Text())
	}
}

// Clear removes all the lines
func (c *LogView) Clear() {
	c.SetText("")
}

// trim drops the oldest lines past the limit; by tenths of it, so a full log
// doesn't move all its lines on every append
func (c *LogView) trim() {
	extra := len(c.lines) - c.maxLines
	if extra <= 0 {
		return
	}
	extra = min(len(c.lines)-1, extra+c.maxLines/10)
	c.lines = slices.Delete(c.lines, 0, extra)
	c.lineWidths = slices.Delete(c.lineWidths, 0, extra)
	c.textValid = false
	// The cursor and the selection stay on their lines, or go to the start
	c.cursorPosY = max(0, c.cursorPosY-extra)
	c.cursorPosX = min(c.cursorPosX, utf8.RuneCountInString(c.lines[c.cursorPosY]))
	c.clearSelection()
	c.updateInnerSize()
	c.setScrollY(max(0, c.scrollY-extra*c.OneLineHeight()))
}

// updateSize measures the new lines only: the width of the content grows to
// the widest line seen
func (c *LogView) updateSize() {
	lineHeight := c.OneLineHeight()
	maxWidth := c.innerWidth - c.textX() - c.leftAndRightPadding*2
	for i := len(c.lineWidths) - 1; i >= 0 && c.lineWidths[i] < 0; i-- {
		w, _, _ := MeasureText(c.FontFamily(), c.FontSize(), c.lines[i])
		c.lineWidths[i] = w
		maxWidth = max(maxWidth, w)
	}
	c.innerWidth = maxWidth + c.textX() + c.leftAndRightPadding*2
	c.innerHeight = lineHeight * len(c.lines)
	c.checkScrolls()
}

// atEnd reports whether the view shows the last line: it follows the new
// ones then
func (c *LogView) atEnd() bool {
	_, vh := c.viewportSize()
	return c.scrollY+vh >= c.innerHeight-c.OneLineHeight()
}

func (c *LogView) scrollToEnd() {
	_, vh := c.viewportSize()
	c.setScrollY(max(0, c.innerHeight-vh))
	c.checkScrolls()
}

// Write appends the text written, as lines, from any goroutine: an
// io.Writer for log.SetOutput and the like. A line shows once its line
// break is written; many writes before a frame make one update.
func (c *LogView) Write(p []byte) (int, error) {
	c.mu.Lock()
	c.pending = append(c.pending, p...)
	post := !c.posted
	c.posted = true
	c.mu.Unlock()
	if post {
		Invoke(c.flush)
	}
	return len(p), nil
}

// flush shows the complete lines written
func (c *LogView) flush() {
	c.mu.Lock()
	text := string(c.pending)
	i := strings.LastIndexByte(text, '\n')
	if i < 0 {
		c.posted = false
		c.mu.Unlock()
		return
	}
	c.pending = append(c.pending[:0], text[i+1:]...)
	c.posted = false
	c.mu.Unlock()
	c.appendLines(splitLines(text[:i]))
}
