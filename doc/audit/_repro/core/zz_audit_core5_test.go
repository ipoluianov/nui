package ui

import (
	"runtime"
	"testing"
	"weak"

	"github.com/ipoluianov/nui/internal/platforms"
)

func auditFreed(ref weak.Pointer[Form]) bool {
	for i := 0; i < 5 && ref.Value() != nil; i++ {
		runtime.GC()
	}
	return ref.Value() == nil
}

func TestAuditLeakBisect(t *testing.T) {
	platforms.SetHeadless()
	// 1: create the window directly
	ref := func() weak.Pointer[Form] {
		f := NewForm()
		f.createWindow(false)
		f.Close()
		return weak.Make(f)
	}()
	t.Logf("createWindow+Close freed: %v", auditFreed(ref))
	ref = func() weak.Pointer[Form] {
		f := NewForm()
		f.createWindow(false)
		f.wnd.Show()
		f.Close()
		return weak.Make(f)
	}()
	t.Logf("createWindow+wnd.Show+Close freed: %v", auditFreed(ref))
	ref = func() weak.Pointer[Form] {
		f := NewForm()
		f.Show()
		f.Close()
		return weak.Make(f)
	}()
	t.Logf("Show+Close freed: %v; open %d registry %d windows %d", auditFreed(ref), len(openForms()), auditRegistrySize(), len(platforms.HeadlessWindows()))
	// another window shown and closed: drops what the previous one left
	g := NewForm()
	g.Show()
	g.Close()
	t.Logf("after another form shown: first freed: %v", auditFreed(ref))
}
