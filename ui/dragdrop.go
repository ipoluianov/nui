package ui

import (
	"fmt"
	"path/filepath"
)

// DragData is what is dragged: set the fields that apply. Files are set for
// files dragged from the system (e.g. from the file manager).
type DragData struct {
	// Text is shown next to the mouse while dragging, and may be the data
	// itself
	Text string
	// Files are paths
	Files []string
	// Value is any data of the application
	Value any
	// Source is the widget the drag started from, nil for system files
	Source Widgeter
}

// SetDragSource makes the widget a place to drag data from. When the user
// presses the left button on it and moves the mouse, f is called with the
// point the drag started at, in the widget's coordinates; it returns what is
// dragged, or nil for no drag there (e.g. outside the selected rows).
func (c *Widget) SetDragSource(f func(x, y int) *DragData) {
	c.dragSource = f
}

// SetDropTarget makes the widget a place to drop data on. While data is
// dragged over the widget, accept tells whether it can be dropped at (x, y),
// in the widget's coordinates; the widget is highlighted when it can. drop is
// called when the data is dropped there. A nil accept accepts everything.
// Files dragged from the system come to drop targets too (DragData.Files).
func (c *Widget) SetDropTarget(accept func(data *DragData, x, y int) bool, drop func(data *DragData, x, y int)) {
	c.dropAccept = accept
	c.dropHandler = drop
}

// dragDropWidget is implemented by every widget, through the embedded Widget
type dragDropWidget interface {
	dragSourceFunc() func(x, y int) *DragData
	dropTargetFuncs() (accept func(data *DragData, x, y int) bool, drop func(data *DragData, x, y int))
	ParentWidget() Widgeter
}

func (c *Widget) dragSourceFunc() func(x, y int) *DragData {
	return c.dragSource
}

func (c *Widget) dropTargetFuncs() (func(data *DragData, x, y int) bool, func(data *DragData, x, y int)) {
	return c.dropAccept, c.dropHandler
}

// SetOnFilesDropped sets the function called when files dragged from the
// system are dropped on the form outside every drop target that accepts them.
// x, y is the point in the form's client coordinates.
func (c *Form) SetOnFilesDropped(f func(files []string, x, y int)) {
	c.onFilesDropped = f
}

// dragDistance is how far the mouse moves with the button down before a drag starts
const dragDistance = 5

// dragCancelledPos is where a mouse release after a drag is reported to the
// widgets: outside all of them, so the release doesn't click the widget the
// drag started on
const dragCancelledPos = -1 << 20

// dragState is the drag in progress in a form
type dragState struct {
	// source is the widget with a drag source under the mouse press, until
	// the drag starts or the button is released
	source         Widgeter
	startX, startY int

	active    bool
	cancelled bool
	data      *DragData
	x, y      int
	// target accepts the data at the mouse, nil if nothing does
	target Widgeter
}

// dragSourceOf finds the widget or its nearest parent that is a drag source
func dragSourceOf(w Widgeter) Widgeter {
	for w != nil {
		d, ok := w.(dragDropWidget)
		if !ok {
			return nil
		}
		if d.dragSourceFunc() != nil {
			return w
		}
		w = d.ParentWidget()
	}
	return nil
}

// dropTargetAt finds the widget at the client point, or its nearest parent,
// that accepts the data there
func (c *Form) dropTargetAt(data *DragData, x, y int) Widgeter {
	for w := c.widgetUnderMouse(x, y); w != nil; {
		d, ok := w.(dragDropWidget)
		if !ok {
			return nil
		}
		if accept, drop := d.dropTargetFuncs(); drop != nil && w.Enabled() {
			wx, wy := w.RectClientAreaOnWindow()
			if accept == nil || accept(data, x-wx, y-wy) {
				return w
			}
		}
		w = d.ParentWidget()
	}
	return nil
}

// dragMouseDown remembers the drag source under a left button press
func (c *Form) dragMouseDown(pressed Widgeter, x, y int) {
	c.drag = dragState{source: dragSourceOf(pressed), startX: x, startY: y}
}

// dragMouseMove starts the drag once the mouse moved far enough, and follows
// it. Returns true while dragging: the widgets don't get the move then.
func (c *Form) dragMouseMove(x, y int) bool {
	d := &c.drag
	if d.cancelled {
		return true
	}
	if !d.active {
		if d.source == nil || !c.mouseLeftButtonPressed {
			return false
		}
		dx, dy := x-d.startX, y-d.startY
		if dx*dx+dy*dy < dragDistance*dragDistance {
			return false
		}
		source := d.source
		d.source = nil
		sx, sy := source.RectClientAreaOnWindow()
		data := source.(dragDropWidget).dragSourceFunc()(d.startX-sx, d.startY-sy)
		if data == nil {
			return false
		}
		if data.Source == nil {
			data.Source = source
		}
		d.active = true
		d.data = data
		c.tooltipSuppress()
	}
	d.x, d.y = x, y
	c.lastMouseX, c.lastMouseY = x, y
	d.target = c.dropTargetAt(d.data, x, y)
	c.Update()
	return true
}

// dragMouseUp drops the data, if a drag is in progress. Returns true if there
// was a drag: the release must not click the widgets then.
func (c *Form) dragMouseUp(x, y int) bool {
	d := c.drag
	c.drag = dragState{}
	if d.cancelled {
		c.Update()
		return true
	}
	if !d.active {
		return false
	}
	if target := c.dropTargetAt(d.data, x, y); target != nil {
		_, drop := target.(dragDropWidget).dropTargetFuncs()
		wx, wy := target.RectClientAreaOnWindow()
		drop(d.data, x-wx, y-wy)
	}
	c.Update()
	return true
}

// dragCancel ends the drag without a drop (Escape). Returns true if there was one.
func (c *Form) dragCancel() bool {
	if !c.drag.active {
		return false
	}
	c.drag = dragState{cancelled: true}
	c.Update()
	return true
}

// processFilesDropped delivers files dropped from the system to the drop
// target under them, or else to the form's SetOnFilesDropped function
func (c *Form) processFilesDropped(files []string, x, y int) {
	data := &DragData{Files: files}
	if len(files) == 1 {
		data.Text = filepath.Base(files[0])
	} else {
		data.Text = fmt.Sprintf("%d files", len(files))
	}
	if target := c.dropTargetAt(data, x, y); target != nil {
		_, drop := target.(dragDropWidget).dropTargetFuncs()
		wx, wy := target.RectClientAreaOnWindow()
		drop(data, x-wx, y-wy)
	} else if c.onFilesDropped != nil {
		c.onFilesDropped(files, x, y)
	}
	c.Update()
}

// dragPaint highlights the drop target and shows the dragged text at the mouse
func (c *Form) dragPaint(cnv *Canvas) {
	d := &c.drag
	if !d.active {
		return
	}
	p := CurrentPalette()
	if d.target != nil {
		x, y := d.target.RectClientAreaOnWindow()
		w, h := d.target.Width(), d.target.Height()
		cnv.FillRect(x, y, w, h, withAlpha(p.Highlight, 40))
		cnv.FillRect(x, y, w, 2, p.Highlight)
		cnv.FillRect(x, y+h-2, w, 2, p.Highlight)
		cnv.FillRect(x, y, 2, h, p.Highlight)
		cnv.FillRect(x+w-2, y, 2, h, p.Highlight)
	}

	text := d.data.Text
	if text == "" {
		return
	}
	text = truncateTextToWidth(ThemeFontFamily(), ThemeFontSize(), text, 300)
	tw, _, err := MeasureText(ThemeFontFamily(), ThemeFontSize(), text)
	if err != nil {
		return
	}
	w, h := tw+16, ThemeLineHeight()+8
	x, y := d.x+14, d.y+18
	fill, border := p.PopupBase, p.Border
	if d.target == nil {
		fill = MixColors(fill, p.Window, 0.5)
	}
	cnv.FillFrame(x, y, w, h, themeControlRadius, fill, border)
	cnv.SetHAlign(HAlignCenter)
	cnv.SetVAlign(VAlignCenter)
	cnv.SetColor(p.Text)
	cnv.SetFontFamily(ThemeFontFamily())
	cnv.SetFontSize(ThemeFontSize())
	cnv.DrawText(x, y, w, h, text)
}
