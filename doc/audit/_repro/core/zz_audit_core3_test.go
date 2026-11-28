package ui

import (
	"context"
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/ipoluianov/nui/internal/platforms"
)

func auditHeadless(t *testing.T) *Form {
	platforms.SetHeadless()
	f := NewForm()
	f.SetSize(400, 300)
	f.Show()
	t.Cleanup(func() { f.Close() })
	platforms.RunHeadlessPosted()
	return f
}

func auditWnd(f *Form) *platforms.HeadlessWindow {
	return f.SystemHandle().(*platforms.HeadlessWindow)
}

// ShowModal of a dialog that is already open
func TestAuditShowModalTwice(t *testing.T) {
	parent := auditHeadless(t)
	dlg := NewForm()
	dlg.SetSize(200, 100)
	n0 := len(platforms.HeadlessWindows())
	dlg.ShowModal(parent)
	dlg.ShowModal(parent)
	t.Logf("windows: %d -> %d", n0, len(platforms.HeadlessWindows()))
	dlg.Close()
	if auditWnd(parent).Blocked() {
		t.Errorf("parent still blocked after the dialog closed; open windows %d (was %d)", len(platforms.HeadlessWindows()), n0)
	}
}

// ShowMaximized of a form that is already shown
func TestAuditShowMaximizedTwice(t *testing.T) {
	f := auditHeadless(t)
	n0 := len(platforms.HeadlessWindows())
	f.ShowMaximized()
	n1 := len(platforms.HeadlessWindows())
	f.Close()
	n2 := len(platforms.HeadlessWindows())
	t.Logf("windows: %d -> %d -> after Close %d", n0, n1, n2)
	if n2 != n0-1 {
		t.Errorf("a window is left open after Close: %d windows (started with %d incl. this form)", n2, n0)
	}
}

// Enter in a Yes/No message box answers once
func TestAuditMessageBoxEnterOnce(t *testing.T) {
	parent := auditHeadless(t)
	yes, no := 0, 0
	ShowQuestionMessageBoxYesNo(parent.Panel(), "Q", "Sure?", func() { yes++ }, func() { no++ })
	var box *Form
	for _, f := range openForms() {
		if f != parent {
			box = f
		}
	}
	if box == nil {
		t.Fatal("no box")
	}
	bw := auditWnd(box)
	bw.KeyDown(KeyEnter, KeyModifiers{})
	bw.KeyUp(KeyEnter, KeyModifiers{})
	platforms.RunHeadlessPosted()
	if yes != 1 || no != 0 {
		t.Errorf("yes %d no %d", yes, no)
	}
}

// The form closed from a button's click, in the middle of the mouse handling
func TestAuditCloseInClick(t *testing.T) {
	f := auditHeadless(t)
	clicked := 0
	b := f.Panel().AddButton(0, 0, "Close", func() { clicked++; f.Close() })
	f.Panel().AddVSpacer(1, 0)
	f.UpdateLayout()
	w := auditWnd(f)
	x, y := b.RectClientAreaOnWindow()
	w.MouseDown(MouseButtonLeft, x+5, y+5)
	w.MouseUp(MouseButtonLeft, x+5, y+5)
	platforms.RunHeadlessPosted()
	if clicked != 1 || !w.Closed() {
		t.Errorf("clicked %d closed %v", clicked, w.Closed())
	}
}

// Timers stopped/started from their callbacks; Close from a timer
func TestAuditTimerCallbacks(t *testing.T) {
	f := auditHeadless(t)
	p := f.Panel()
	runs := 0
	var t1 *Timer
	t1 = p.AddTimer(0, func() {
		runs++
		t1.Stop()
		t1.Start()
		t1.Stop()
	})
	other := 0
	t2 := p.NewTimer(time.Millisecond, func() { other++ })
	t2.SetSingleShot(true)
	t2.Start()
	t3 := p.AfterFunc(0, func() { t2.Stop(); f.Close() })
	_ = t3
	for i := 0; i < 3; i++ {
		platforms.TickHeadlessTimers()
	}
	t.Logf("runs %d other %d", runs, other)
}

// RunAsync whose form closes: done isn't called; the closed form is freed
func TestAuditRunAsyncFormClosed(t *testing.T) {
	platforms.SetHeadless()
	start := make(chan struct{})
	finished := make(chan struct{})
	called := false
	ref := func() weak.Pointer[Form] {
		f := NewForm()
		f.Show()
		RunAsync(f.Panel(), func(ctx context.Context, p *Progress) (int, error) {
			close(start)
			<-ctx.Done()
			p.Report(1, "x")
			close(finished)
			return 1, ctx.Err()
		}, func(int, error) { called = true })
		<-start
		f.Close()
		return weak.Make(f)
	}()
	<-finished
	time.Sleep(20 * time.Millisecond)
	platforms.RunHeadlessPosted()
	if called {
		t.Error("done called after close")
	}
	// the last window made stays referenced somewhere (bounded): drop it
	g := NewForm()
	g.Show()
	g.Close()
	for i := 0; i < 5 && ref.Value() != nil; i++ {
		runtime.GC()
	}
	if ref.Value() != nil {
		t.Error("closed form kept in memory")
	}
}
