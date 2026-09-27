package ui

import ()

const splitterDefaultHandleSize = 6

// Splitter shows two widgets side by side (or one above the other)
// with a handle between them that can be dragged to resize them.
//
// One of the panes is "fixed": it keeps its size when the splitter is resized,
// the other one takes the rest. SetFirstSize makes the first pane fixed,
// SetSecondSize makes the second pane fixed.
// If one of the panes is hidden, the other one takes the whole area.
type Splitter struct {
	Widget

	vertical   bool
	first      Widgeter
	second     Widgeter
	handle     *splitterHandle
	handleSize int

	fixedSecond bool
	fixedSize   int // -1 - not set yet, the area is split in half

	onSplitChanged func()
}

// NewHSplitter creates a splitter with the panes on the left and on the right
func NewHSplitter() *Splitter {
	return newSplitter(false)
}

// NewVSplitter creates a splitter with the panes on the top and on the bottom
func NewVSplitter() *Splitter {
	return newSplitter(true)
}

func newSplitter(vertical bool) *Splitter {
	var c Splitter
	c.InitWidget()
	c.SetTypeName("Splitter")
	c.SetPanelPadding(0)
	c.SetCellPadding(0)
	c.vertical = vertical
	c.handleSize = splitterDefaultHandleSize
	c.fixedSize = -1
	c.handle = newSplitterHandle(&c)
	c.Widget.AddWidget(0, 1, c.handle)
	return &c
}

// SetWidgets sets the panes: left/top and right/bottom. nil leaves the pane empty.
func (c *Splitter) SetWidgets(first, second Widgeter) {
	// RemoveWidget does not detach from the form, and AddWidget ignores attached widgets
	for _, w := range []Widgeter{c.first, c.second} {
		if w != nil {
			c.Widget.RemoveWidget(w)
			w.attachToForm(w, nil)
		}
	}
	c.first = first
	c.second = second
	if first != nil {
		c.Widget.AddWidget(0, 0, first)
	}
	if second != nil {
		c.Widget.AddWidget(0, 2, second)
	}
	c.layoutPanes()
}

func (c *Splitter) First() Widgeter {
	return c.first
}

func (c *Splitter) Second() Widgeter {
	return c.second
}

func (c *Splitter) IsVertical() bool {
	return c.vertical
}

// SetHandleSize sets the thickness of the handle between the panes
func (c *Splitter) SetHandleSize(size int) {
	c.handleSize = size
	c.layoutPanes()
}

func (c *Splitter) HandleSize() int {
	return c.handleSize
}

// SetFirstSize sets the size of the first pane and keeps it when the splitter is resized
func (c *Splitter) SetFirstSize(size int) {
	c.fixedSecond = false
	c.fixedSize = size
	c.layoutPanes()
}

// SetSecondSize sets the size of the second pane and keeps it when the splitter is resized
func (c *Splitter) SetSecondSize(size int) {
	c.fixedSecond = true
	c.fixedSize = size
	c.layoutPanes()
}

// FirstSize returns the current size of the first pane
func (c *Splitter) FirstSize() int {
	if c.first == nil {
		return 0
	}
	return c.paneSize(c.first)
}

// SecondSize returns the current size of the second pane
func (c *Splitter) SecondSize() int {
	if c.second == nil {
		return 0
	}
	return c.paneSize(c.second)
}

// SetOnSplitChanged sets the callback called when the user has dragged the handle
func (c *Splitter) SetOnSplitChanged(fn func()) {
	c.onSplitChanged = fn
}

// The splitter always fills the space given to it
func (c *Splitter) XExpandable() bool {
	return true
}

func (c *Splitter) YExpandable() bool {
	return true
}

func (c *Splitter) MinWidth() int {
	return c.minSize(false)
}

func (c *Splitter) MinHeight() int {
	return c.minSize(true)
}

// minSize sums the panes along the split direction and takes the largest across it
func (c *Splitter) minSize(vertical bool) int {
	first, second := c.visiblePane(c.first), c.visiblePane(c.second)
	result := 0
	if vertical == c.vertical {
		result = c.paneMinSize(first) + c.paneMinSize(second)
		if first != nil && second != nil {
			result += c.handleSize
		}
	} else {
		result = max(c.paneMinSize(first), c.paneMinSize(second))
	}
	if vertical {
		return max(result, c.minHeight)
	}
	return max(result, c.minWidth)
}

func (c *Splitter) SetSize(w, h int) {
	c.w = w
	c.h = h
	c.layoutPanes()
}

func (c *Splitter) updateLayout(oldWidth, oldHeight, newWidth, newHeight int) {
	c.layoutPanes()
}

func (c *Splitter) visiblePane(w Widgeter) Widgeter {
	if w == nil || !w.IsVisible() {
		return nil
	}
	return w
}

// paneMinSize returns the minimum size of the pane along the split direction
func (c *Splitter) paneMinSize(w Widgeter) int {
	if w == nil {
		return 0
	}
	if c.vertical {
		return w.MinHeight()
	}
	return w.MinWidth()
}

func (c *Splitter) paneSize(w Widgeter) int {
	if c.vertical {
		return w.Height()
	}
	return w.Width()
}

// length is the size of the splitter along the split direction
func (c *Splitter) length() int {
	if c.vertical {
		return c.h
	}
	return c.w
}

// placePane puts a widget at the offset along the split direction, spanning the whole splitter across it
func (c *Splitter) placePane(w Widgeter, offset, size int) {
	if c.vertical {
		w.SetPosition(0, offset)
		w.SetSize(c.w, size)
	} else {
		w.SetPosition(offset, 0)
		w.SetSize(size, c.h)
	}
}

// clampFixedSize limits the size of the fixed pane so that both panes fit their minimum sizes
func (c *Splitter) clampFixedSize(size int) int {
	fixed, other := c.first, c.second
	if c.fixedSecond {
		fixed, other = other, fixed
	}
	available := c.length() - c.handleSize
	size = min(size, available-c.paneMinSize(other))
	size = max(size, c.paneMinSize(fixed))
	return max(0, min(size, available))
}

func (c *Splitter) layoutPanes() {
	if c.form == nil || c.form.layoutingBlockStack > 0 {
		return
	}

	first, second := c.visiblePane(c.first), c.visiblePane(c.second)
	for _, w := range c.widgets {
		if w != first && w != second {
			w.SetPosition(0, 0)
			w.SetSize(0, 0)
		}
	}

	switch {
	case first != nil && second != nil:
		if c.fixedSize < 0 {
			c.fixedSize = (c.length() - c.handleSize) / 2
		}
		fixedSize := c.clampFixedSize(c.fixedSize)
		firstSize := fixedSize
		if c.fixedSecond {
			firstSize = c.length() - c.handleSize - fixedSize
		}
		c.placePane(first, 0, firstSize)
		c.placePane(c.handle, firstSize, c.handleSize)
		c.placePane(second, firstSize+c.handleSize, c.length()-firstSize-c.handleSize)
	case first != nil:
		c.placePane(first, 0, c.length())
	case second != nil:
		c.placePane(second, 0, c.length())
	}

	c.innerWidth = c.w
	c.innerHeight = c.h
}

// dragHandleTo moves the handle to the offset along the split direction
func (c *Splitter) dragHandleTo(offset int) {
	fixedSize := offset
	if c.fixedSecond {
		fixedSize = c.length() - c.handleSize - offset
	}
	fixedSize = c.clampFixedSize(fixedSize)
	if fixedSize == c.clampFixedSize(c.fixedSize) {
		return
	}
	c.fixedSize = fixedSize
	c.layoutPanes()
	c.form.Update()
	if c.onSplitChanged != nil {
		c.onSplitChanged()
	}
}

// splitterHandle is the draggable bar between the panes of a Splitter
type splitterHandle struct {
	Widget

	splitter   *Splitter
	dragging   bool
	grabOffset int
}

func newSplitterHandle(splitter *Splitter) *splitterHandle {
	var c splitterHandle
	c.InitWidget()
	c.SetTypeName("SplitterHandle")
	c.splitter = splitter
	if splitter.vertical {
		c.SetMouseCursor(MouseCursorResizeVer)
	} else {
		c.SetMouseCursor(MouseCursorResizeHor)
	}
	c.SetOnPaint(c.draw)
	c.SetOnMouseDown(c.processMouseDown)
	c.SetOnMouseMove(c.processMouseMove)
	c.SetOnMouseUp(c.processMouseUp)
	return &c
}

// offset returns the coordinate along the split direction
func (c *splitterHandle) offset(x, y int) int {
	if c.splitter.vertical {
		return y
	}
	return x
}

func (c *splitterHandle) processMouseDown(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft {
		return false
	}
	c.dragging = true
	c.grabOffset = c.offset(x, y)
	c.form.Update()
	return true
}

// While the button is pressed the form sends mouse moves to the handle
// even outside of it, in the handle's coordinates
func (c *splitterHandle) processMouseMove(x int, y int, mods KeyModifiers) bool {
	if !c.dragging {
		return false
	}
	c.splitter.dragHandleTo(c.offset(c.X(), c.Y()) + c.offset(x, y) - c.grabOffset)
	return true
}

func (c *splitterHandle) processMouseUp(button MouseButton, x int, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft {
		return false
	}
	c.dragging = false
	c.form.Update()
	return true
}

func (c *splitterHandle) draw(cnv *Canvas) {
	p := CurrentPalette()
	backColor := colorToRGBA(c.BackgroundColor())
	if c.dragging {
		backColor = pressedColor(backColor, p.WindowText)
	} else if c.IsHovered() {
		backColor = hoverColor(backColor, p.WindowText)
	}
	cnv.FillRect(0, 0, c.Width(), c.Height(), backColor)

	// Grip: a short line in the middle of the handle
	gripColor := p.Border
	const gripLength = 24
	if c.splitter.vertical {
		cnv.FillRect((c.Width()-gripLength)/2, c.Height()/2, gripLength, 1, gripColor)
	} else {
		cnv.FillRect(c.Width()/2, (c.Height()-gripLength)/2, 1, gripLength, gripColor)
	}
}
