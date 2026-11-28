package ui

import "testing"

// Taking a widget's context menu away
func TestAuditSetContextMenuNil(t *testing.T) {
	l := NewLabel("x")
	newTestForm(t, l)
	l.SetContextMenu(NewContextMenu(nil))
	l.SetContextMenu(nil)
}

// Replacing a widget's context menu (e.g. rebuilt on every selection change)
func TestAuditReplaceContextMenuLeak(t *testing.T) {
	l := NewLabel("x")
	newTestForm(t, l)
	before := auditRegistrySize()
	for i := 0; i < 50; i++ {
		m := NewContextMenu(nil)
		m.AddItem("A", nil)
		l.SetContextMenu(m)
	}
	t.Logf("registry before %d after %d", before, auditRegistrySize())
	if auditRegistrySize() > before+5 {
		t.Errorf("registry grew from %d to %d after replacing the context menu 50 times", before, auditRegistrySize())
	}
}
