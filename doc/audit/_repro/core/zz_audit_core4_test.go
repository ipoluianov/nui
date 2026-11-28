package ui

import (
	"runtime"
	"testing"
	"weak"

	"github.com/ipoluianov/nui/internal/platforms"
)

// A shown and closed form, nothing else, is freed
func TestAuditShownFormFreed(t *testing.T) {
	platforms.SetHeadless()
	ref := func() weak.Pointer[Form] {
		f := NewForm()
		f.Show()
		f.Close()
		return weak.Make(f)
	}()
	platforms.RunHeadlessPosted()
	for i := 0; i < 5 && ref.Value() != nil; i++ {
		runtime.GC()
	}
	if ref.Value() != nil {
		t.Error("closed form kept in memory")
	}
}

func TestAuditMessageBoxEnterOnce2(t *testing.T) {
	parent := auditHeadless(t)
	yes, no := 0, 0
	ShowQuestionMessageBoxYesNo(parent.Panel(), "Q", "Sure?", func() { yes++ }, func() { no++ })
	var box *Form
	for _, f := range openForms() {
		t.Logf("open form %q wnd %v", f.title, f.wnd != nil)
		if f != parent && f.wnd != nil {
			box = f
		}
	}
	if box == nil {
		t.Fatal("no box")
	}
	auditWnd(box).KeyDown(KeyEnter, KeyModifiers{})
	auditWnd(box).KeyUp(KeyEnter, KeyModifiers{})
	platforms.RunHeadlessPosted()
	if yes != 1 || no != 0 {
		t.Errorf("yes %d no %d", yes, no)
	}
}
