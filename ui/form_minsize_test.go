package ui

import (
	"testing"

	"github.com/ipoluianov/nui/internal/platforms"
)

// fakeMinSizeWindow records what the form asks of its window about the size
type fakeMinSizeWindow struct {
	platforms.Window // the rest isn't used

	minW, minH       int
	resizes          [][2]int
	areaW, areaH     int
	workAreaRequests int
}

func (w *fakeMinSizeWindow) SetMinSize(width, height int) { w.minW, w.minH = width, height }
func (w *fakeMinSizeWindow) Resize(width, height int) {
	w.resizes = append(w.resizes, [2]int{width, height})
}
func (w *fakeMinSizeWindow) Update()                              {}
func (w *fakeMinSizeWindow) Scale() float64                       { return 1 }
func (w *fakeMinSizeWindow) SetMouseCursor(platforms.MouseCursor) {}
func (w *fakeMinSizeWindow) PosX() int                            { return 0 }
func (w *fakeMinSizeWindow) PosY() int                            { return 0 }
func (w *fakeMinSizeWindow) ScreenWorkArea(x, y int) (int, int, int, int) {
	w.workAreaRequests++
	return 0, 0, w.areaW, w.areaH
}

// newMinSizeForm makes a form of 200x100 with a widget of the minimum size
// of minW x minH in it
func newMinSizeForm(t *testing.T, minW, minH int) (*Form, *fakeMinSizeWindow, *Panel) {
	t.Helper()
	form := NewForm()
	form.Panel().SetPanelPadding(0)
	form.processResize(200, 100)
	wnd := &fakeMinSizeWindow{areaW: 5000, areaH: 5000}
	form.wnd = wnd
	child := NewPanel()
	child.SetMinSize(minW, minH)
	form.Panel().AddWidget(0, 0, child)
	return form, wnd, child
}

func TestFormMinSize(t *testing.T) {
	form, wnd, child := newMinSizeForm(t, 150, 80)
	if w, h := form.MinSize(); w != 150 || h != 80 {
		t.Fatalf("min size = %dx%d, want 150x80", w, h)
	}
	if wnd.minW != 150 || wnd.minH != 80 {
		t.Fatalf("window min size = %dx%d, want 150x80", wnd.minW, wnd.minH)
	}
	if len(wnd.resizes) != 0 {
		t.Fatalf("the window fits, but it is resized: %v", wnd.resizes)
	}

	// The content grows beyond the window: the window grows to it
	child.SetMinSize(300, 120)
	form.UpdateLayout()
	if wnd.minW != 300 || wnd.minH != 120 {
		t.Fatalf("window min size = %dx%d, want 300x120", wnd.minW, wnd.minH)
	}
	if n := len(wnd.resizes); n == 0 || wnd.resizes[n-1] != [2]int{300, 120} {
		t.Fatalf("resizes = %v, want to 300x120", wnd.resizes)
	}

	// SetSize doesn't make the window smaller than the content
	form.SetSize(100, 50)
	if w, h := form.Size(); w != 300 || h != 120 {
		t.Fatalf("size = %dx%d, want 300x120", w, h)
	}
	form.SetSize(400, 50)
	if w, h := form.Size(); w != 400 || h != 120 {
		t.Fatalf("size = %dx%d, want 400x120", w, h)
	}
}

// A minimum larger than the screen is cut to it
func TestFormMinSizeScreen(t *testing.T) {
	form, wnd, child := newMinSizeForm(t, 100, 50)
	wnd.areaW, wnd.areaH = 800, 600
	child.SetMinSize(1000, 700)
	form.UpdateLayout()
	if wnd.minW != 800 || wnd.minH != 600 {
		t.Fatalf("window min size = %dx%d, want the screen 800x600", wnd.minW, wnd.minH)
	}

	// The screen is asked again only when the need changes
	n := wnd.workAreaRequests
	form.UpdateLayout()
	form.UpdateLayout()
	if wnd.workAreaRequests != n {
		t.Errorf("the screen is asked %d times more", wnd.workAreaRequests-n)
	}
}

// The tab widget needs the room of its largest page, not only of the
// current one: the window doesn't change when another tab is chosen
func TestTabWidgetMinSizeOfAllPages(t *testing.T) {
	small := NewPanel()
	small.SetMinSize(50, 30)
	large := NewPanel()
	large.SetMinSize(400, 300)
	tabs := NewTabWidget()
	newTestForm(t, tabs)
	tabs.AddPage("small", small)
	tabs.AddPage("large", large)
	tabs.onTabChanged(0)

	wantW := large.MinWidth()
	if tabs.MinWidth() < wantW || tabs.MinHeight() < 300 {
		t.Fatalf("min size = %dx%d, want at least %dx300", tabs.MinWidth(), tabs.MinHeight(), wantW)
	}
	w, h := tabs.MinWidth(), tabs.MinHeight()
	tabs.onTabChanged(1)
	tabs.ClearLayoutCache()
	if tabs.MinWidth() != w || tabs.MinHeight() != h {
		t.Errorf("min size changed with the tab: %dx%d, was %dx%d", tabs.MinWidth(), tabs.MinHeight(), w, h)
	}
}
