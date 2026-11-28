package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
)

// A toast of a long text (an error with a log) is cut to toastMaxLines: its
// window used to be as tall as the whole text, millions of pixels
func TestToastLineLimit(t *testing.T) {
	f := NewForm()
	f.SetSize(400, 300)
	f.ShowToastFor(strings.Repeat("line\n", 100000), ToastError, time.Hour)
	if len(f.toasts) != 1 {
		t.Fatalf("%d toasts", len(f.toasts))
	}
	const maxLines = 10 // toastMaxLines
	toast := f.toasts[0]
	if len(toast.lines) != maxLines || !strings.HasSuffix(toast.lines[len(toast.lines)-1], "...") {
		t.Errorf("%d lines, the last %q", len(toast.lines), toast.lines[len(toast.lines)-1])
	}
	if want := maxLines*ThemeLineHeight() + toastPaddingY*2; toast.h != want {
		t.Errorf("toast height %d, want %d", toast.h, want)
	}
}

// The window of a tooltip of a huge text is no larger than the screen
func TestTooltipFitsScreen(t *testing.T) {
	platforms.SetHeadless()
	f := NewForm()
	f.SetSize(400, 300)
	p := NewPanel()
	p.SetMinSize(100, 100)
	p.SetTooltip(strings.Repeat("a long tooltip ", 2000))
	f.Panel().AddWidget(0, 0, p)
	f.Show()
	t.Cleanup(f.Close)
	platforms.RunHeadlessPosted()
	f.hoverWidget = p
	if !f.tooltipShowPopup() || !f.tooltip.popupVisible {
		t.Fatal("no tooltip popup")
	}
	_, _, aw, ah := f.wnd.ScreenWorkArea(0, 0)
	if f.tooltip.popupW > aw || f.tooltip.popupH > ah {
		t.Errorf("tooltip %dx%d on a screen of %dx%d", f.tooltip.popupW, f.tooltip.popupH, aw, ah)
	}
}
