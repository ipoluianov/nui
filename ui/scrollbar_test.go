package ui

import (
	"testing"
	"time"
)

// newScrollTestWidget makes a scrolling widget of the size w x h with the
// content of innerW x innerH, laid out by hand
func newScrollTestWidget(t *testing.T, w, h, innerW, innerH int) *Panel {
	t.Helper()
	p := NewPanel()
	p.SetAbsolutePositioning(true)
	p.SetAllowScroll(true, true)
	newTestForm(t, p)
	p.SetSize(w, h)
	p.SetInnerSize(innerW, innerH)
	return p
}

func TestScrollBarsVisibility(t *testing.T) {
	const b = scrollBarSize
	tests := []struct {
		name           string
		innerW, innerH int
		wantX, wantY   bool
	}{
		{"fits", 100, 100, false, false},
		{"smaller", 50, 50, false, false},
		{"taller", 100 - b, 101, false, true},
		{"wider", 101, 100 - b, true, false},
		// The vertical bar takes its room from the width, then the content
		// doesn't fit the width either
		{"taller, then wider", 100 - b + 1, 101, true, true},
		// ...and the other way round
		{"wider, then taller", 101, 100 - b + 1, true, true},
		{"taller, fits the width left", 100 - b, 150, false, true},
		{"both", 200, 200, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newScrollTestWidget(t, 100, 100, tt.innerW, tt.innerH)
			barX, barY := p.scrollBarsVisible()
			if barX != tt.wantX || barY != tt.wantY {
				t.Fatalf("bars = %v, %v, want %v, %v", barX, barY, tt.wantX, tt.wantY)
			}
			wantW, wantH := 100, 100
			if tt.wantY {
				wantW -= b
			}
			if tt.wantX {
				wantH -= b
			}
			if p.ViewportWidth() != wantW || p.ViewportHeight() != wantH {
				t.Errorf("viewport = %dx%d, want %dx%d", p.ViewportWidth(), p.ViewportHeight(), wantW, wantH)
			}
		})
	}
}

// Without scrolling the view is the whole widget, whatever the content
func TestScrollBarsNotAllowed(t *testing.T) {
	p := newScrollTestWidget(t, 100, 100, 500, 500)
	p.SetAllowScroll(false, false)
	if p.ViewportWidth() != 100 || p.ViewportHeight() != 100 {
		t.Fatalf("viewport = %dx%d, want 100x100", p.ViewportWidth(), p.ViewportHeight())
	}
	if p.inScrollBars(99, 99) {
		t.Error("the corner is taken by scroll bars")
	}
}

// The scroll bars of a framed widget are inside the frame; the arrow
// buttons are square at the ends, the track between them
func TestScrollBarsInset(t *testing.T) {
	const b = scrollBarSize
	p := newScrollTestWidget(t, 100, 100, 200, 200)
	p.scrollBarInset = 1
	view := 100 - b - 1
	if p.ViewportWidth() != view || p.ViewportHeight() != view {
		t.Fatalf("viewport = %dx%d, want %dx%d", p.ViewportWidth(), p.ViewportHeight(), view, view)
	}
	for _, bar := range []scrollBar{p.scrollBarX(), p.scrollBarY()} {
		if bar.barPos != 1 || bar.barLen != view-1 || bar.crossPos != view || bar.btnLen != b ||
			bar.trackPos != 1+b || bar.trackLen != view-1-2*b {
			t.Errorf("bar = %+v", bar)
		}
	}
}

// The content scrolls to its end in the view, not in the whole widget
func TestScrollRangeIsTheView(t *testing.T) {
	const b = scrollBarSize
	p := newScrollTestWidget(t, 100, 100, 300, 300)
	p.SetScrollX(1000)
	p.SetScrollY(1000)
	if end := 300 - (100 - b); p.ScrollX() != end || p.ScrollY() != end {
		t.Fatalf("scroll = %d, %d, want %d", p.ScrollX(), p.ScrollY(), end)
	}
	p.SetScrollY(-5)
	if p.ScrollY() != 0 {
		t.Fatalf("scroll = %d, want 0", p.ScrollY())
	}

	// The content gets smaller: the scroll stays within it
	p.SetScrollY(200)
	p.SetInnerSize(300, 150)
	p.checkScrolls()
	if end := 150 - (100 - b); p.ScrollY() != end {
		t.Fatalf("scroll = %d, want %d", p.ScrollY(), end)
	}
}

func TestScrollEnsureVisibleInTheView(t *testing.T) {
	const b = scrollBarSize
	p := newScrollTestWidget(t, 100, 100, 300, 300)
	p.ScrollEnsureVisible(0, 150)
	if want := 150 - (100 - b); p.ScrollY() != want {
		t.Fatalf("scroll = %d, want %d: the point just above the horizontal bar", p.ScrollY(), want)
	}
}

func TestScrollBarMouse(t *testing.T) {
	const b = scrollBarSize
	p := newScrollTestWidget(t, 100, 100, 100-b, 400)
	by := p.scrollBarY()
	if !by.visible || p.scrollBarX().visible {
		t.Fatal("only the vertical bar is expected")
	}
	trackLen := 100 - 2*b
	if by.thumbPos != b || by.thumbLen != max(scrollBarMinThumb, trackLen/4) {
		t.Fatalf("thumb at %d, %d long", by.thumbPos, by.thumbLen)
	}
	x := by.crossPos + 2
	click := func(y int) {
		p.ProcessMouseDown(MouseButtonLeft, x, y, KeyModifiers{})
		p.ProcessMouseUp(MouseButtonLeft, x, y, KeyModifiers{}, "")
	}

	// The down arrow scrolls a line, the up arrow back
	click(100 - b/2)
	if p.ScrollY() != scrollBarLineStep {
		t.Fatalf("scroll = %d, want a line", p.ScrollY())
	}
	click(b / 2)
	if p.ScrollY() != 0 {
		t.Fatalf("scroll = %d, want 0", p.ScrollY())
	}
	// Nothing above the top
	click(b / 2)
	if p.ScrollY() != 0 {
		t.Fatalf("scroll = %d, want 0", p.ScrollY())
	}

	// A press on the track between the thumb and the down arrow scrolls a
	// page down, between the up arrow and the thumb a page up
	click(100 - b - 2)
	if p.ScrollY() != 100 {
		t.Fatalf("scroll = %d, want a page", p.ScrollY())
	}
	click(b + 1)
	if p.ScrollY() != 0 {
		t.Fatalf("scroll = %d, want 0", p.ScrollY())
	}

	// Dragging the thumb to the end scrolls to the end
	by = p.scrollBarY()
	p.ProcessMouseDown(MouseButtonLeft, x, by.thumbPos+1, KeyModifiers{})
	if !p.scrollingY {
		t.Fatal("the press on the thumb doesn't start dragging it")
	}
	p.ProcessMouseMove(x, by.thumbPos+1+100, KeyModifiers{})
	p.ProcessMouseUp(MouseButtonLeft, x, 0, KeyModifiers{}, "")
	if p.ScrollY() != 300 {
		t.Fatalf("scroll = %d, want 300", p.ScrollY())
	}

	// The content under the bar doesn't get the press
	got := false
	p.SetOnMouseDown(func(button MouseButton, x, y int, mods KeyModifiers) bool {
		got = true
		return true
	})
	p.ProcessMouseDown(MouseButtonRight, x, 50, KeyModifiers{})
	if got {
		t.Error("a press on the bar reached the content")
	}
	p.ProcessMouseDown(MouseButtonLeft, 10, 50, KeyModifiers{})
	if !got {
		t.Error("a press in the view didn't reach the content")
	}
}

// A held arrow button repeats; the held track pages until the thumb comes
// to the mouse; the release stops it
func TestScrollBarRepeat(t *testing.T) {
	const b = scrollBarSize
	p := newScrollTestWidget(t, 100, 100, 100-b, 1000)
	x := p.scrollBarY().crossPos + 2
	p.form.mouseLeftButtonPressed = true
	tick := func() {
		p.scrollPressNext = time.Now().Add(-time.Millisecond)
		p.ProcessTimer()
	}

	p.ProcessMouseDown(MouseButtonLeft, x, 100-b/2, KeyModifiers{})
	tick()
	tick()
	if p.ScrollY() != 3*scrollBarLineStep {
		t.Fatalf("scroll = %d, want 3 lines", p.ScrollY())
	}
	// Moved off the button: no more steps until it comes back
	p.ProcessMouseMove(10, 50, KeyModifiers{})
	tick()
	if p.ScrollY() != 3*scrollBarLineStep {
		t.Fatalf("scroll = %d, want 3 lines", p.ScrollY())
	}
	p.ProcessMouseUp(MouseButtonLeft, x, 0, KeyModifiers{}, "")
	tick()
	if p.ScrollY() != 3*scrollBarLineStep || p.scrollPressPart != scrollPartNone {
		t.Fatalf("scroll = %d after the release", p.ScrollY())
	}

	// The track just above the down arrow: pages until the thumb is there
	p.SetScrollY(0)
	y := 100 - b - 1
	p.ProcessMouseDown(MouseButtonLeft, x, y, KeyModifiers{})
	for i := 0; i < 50; i++ {
		tick()
	}
	if by := p.scrollBarY(); by.partAt(x, y) == scrollPartPageForward {
		t.Fatalf("the thumb at %d didn't come to the mouse at %d", by.thumbPos, y)
	}
	stopped := p.ScrollY()
	tick()
	if p.ScrollY() != stopped {
		t.Fatal("the paging goes on past the mouse")
	}
	p.ProcessMouseUp(MouseButtonLeft, x, y, KeyModifiers{}, "")

	// The release out of the window stops it too
	p.SetScrollY(0)
	p.ProcessMouseDown(MouseButtonLeft, x, 100-b/2, KeyModifiers{})
	p.form.mouseLeftButtonPressed = false
	tick()
	if p.ScrollY() != scrollBarLineStep || p.scrollPressPart != scrollPartNone {
		t.Fatalf("scroll = %d, the repeat goes on without the button", p.ScrollY())
	}
}

// A quick double click on an arrow scrolls two lines
func TestScrollBarDoubleClick(t *testing.T) {
	const b = scrollBarSize
	p := newScrollTestWidget(t, 100, 100, 100-b, 1000)
	x := p.scrollBarY().crossPos + 2
	p.ProcessMouseDown(MouseButtonLeft, x, 100-b/2, KeyModifiers{})
	p.ProcessMouseUp(MouseButtonLeft, x, 100-b/2, KeyModifiers{}, "")
	p.ProcessMouseDblClick(MouseButtonLeft, x, 100-b/2, KeyModifiers{})
	p.ProcessMouseUp(MouseButtonLeft, x, 100-b/2, KeyModifiers{}, "")
	if p.ScrollY() != 2*scrollBarLineStep {
		t.Fatalf("scroll = %d, want 2 lines", p.ScrollY())
	}
}

// A grid container lays its children out in the view: an expandable child
// is as wide as the view, so the vertical bar doesn't bring the horizontal
// one
func TestScrollAreaLayoutInView(t *testing.T) {
	const b = scrollBarSize
	area := NewScrollArea()
	area.SetPanelPadding(0)
	newTestForm(t, area)
	area.SetSize(200, 100)

	child := NewPanel()
	child.SetXExpandable(true)
	child.SetMinSize(50, 300)
	area.AddWidget(0, 0, child)
	area.SetSize(200, 100)

	barX, barY := area.scrollBarsVisible()
	if barX || !barY {
		t.Fatalf("bars = %v, %v, want only the vertical one", barX, barY)
	}
	if child.Width() != 200-b {
		t.Errorf("child width = %d, want %d: the view", child.Width(), 200-b)
	}

	// Higher than the content: no bars, the child takes the whole width
	area.SetSize(200, 400)
	if barX, barY := area.scrollBarsVisible(); barX || barY {
		t.Fatalf("bars = %v, %v, want none", barX, barY)
	}
	if child.Width() != 200 {
		t.Errorf("child width = %d, want 200", child.Width())
	}

	// Narrower than the child: both bars
	child.SetMinSize(250, 300)
	area.ClearLayoutCache()
	area.SetSize(200, 100)
	if barX, barY := area.scrollBarsVisible(); !barX || !barY {
		t.Fatalf("bars = %v, %v, want both", barX, barY)
	}
	if area.ViewportWidth() != 200-b || area.ViewportHeight() != 100-b {
		t.Errorf("viewport = %dx%d, want %dx%d", area.ViewportWidth(), area.ViewportHeight(), 200-b, 100-b)
	}
}

// The content only just fits until the vertical bar appears: the grid is
// laid out again in the narrower view and the horizontal bar appears too
func TestScrollAreaCascade(t *testing.T) {
	const b = scrollBarSize
	area := NewScrollArea()
	area.SetPanelPadding(0)
	newTestForm(t, area)

	child := NewPanel()
	child.SetMinSize(200-b+5, 300)
	area.AddWidget(0, 0, child)
	area.SetSize(200, 100)

	if barX, barY := area.scrollBarsVisible(); !barX || !barY {
		t.Fatalf("bars = %v, %v, want both", barX, barY)
	}
	area.SetScrollX(1000)
	if area.ScrollX() != 5 {
		t.Errorf("scroll = %d, want 5", area.ScrollX())
	}
}

// A container that scrolls only vertically asks for the room of the bar
func TestScrollMinWidthHasBarRoom(t *testing.T) {
	p := NewPanel()
	p.SetPanelPadding(0)
	p.SetAllowScroll(false, true)
	child := NewPanel()
	child.SetMinSize(100, 20)
	p.AddWidget(0, 0, child)
	if p.MinWidth() != 100+scrollBarSize {
		t.Fatalf("min width = %d, want %d", p.MinWidth(), 100+scrollBarSize)
	}
}

// The stretched last column fills the view only: the vertical bar doesn't
// bring the horizontal one
func TestTableStretchedColumnNoHorizontalBar(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	table.SetColumnCount(1)
	table.SetColumnWidth(0, 50)
	table.SetStretchLastColumn(true)
	table.SetRowCount(1000)
	table.SetSize(300, 200)

	barX, barY := table.scrollBarsVisible()
	if barX || !barY {
		t.Fatalf("bars = %v, %v, want only the vertical one", barX, barY)
	}
	if got, want := table.columnWidth(0), table.ViewportWidth(); got != want {
		t.Errorf("stretched column = %d, want the view %d", got, want)
	}
}

func TestTreeViewStretchedColumnNoHorizontalBar(t *testing.T) {
	tree, _ := newTestTree(t)
	for i := 0; i < 100; i++ {
		tree.AddNode(nil, "more")
	}
	tree.SetColumnWidth(0, 50)
	tree.SetColumnWidth(1, 50)
	tree.SetSize(300, 200)
	tree.ensureRows()

	barX, barY := tree.scrollBarsVisible()
	if barX || !barY {
		t.Fatalf("bars = %v, %v, want only the vertical one", barX, barY)
	}
	if got := tree.columnOffset(1) + tree.columnWidth(1); got != tree.ViewportWidth() {
		t.Errorf("columns = %d, want the view %d", got, tree.ViewportWidth())
	}
}

// A scroll area keeps its content at the minimum size and scrolls it
func TestScrollAreaDoesNotShrinkContent(t *testing.T) {
	area := NewScrollArea()
	area.SetPanelPadding(0)
	newTestForm(t, area)
	child := NewPanel()
	child.SetMinSize(500, 500)
	area.AddWidget(0, 0, child)
	area.SetSize(200, 100)
	if child.Width() != 500 || child.Height() != 500 {
		t.Fatalf("child = %dx%d, want 500x500", child.Width(), child.Height())
	}
}
