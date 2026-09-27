package ui

import "testing"

// A hidden item takes no place in the menu, and the menu is as wide as the
// items shown
func TestContextMenuHiddenItem(t *testing.T) {
	menu := NewContextMenu(nil)
	shown := menu.AddItem("Edit", nil)
	hidden := menu.AddItem("A much longer item that would widen the menu", nil)
	menu.rebuildVisualElements()
	fullHeight, fullWidth := menu.menuHeight, menu.menuWidth

	hidden.SetVisible(false)
	menu.rebuildVisualElements()
	if menu.menuHeight != shown.height() {
		t.Errorf("height with a hidden item = %d, want %d", menu.menuHeight, shown.height())
	}
	if menu.menuHeight >= fullHeight || menu.menuWidth >= fullWidth {
		t.Errorf("hidden item still counted: %dx%d, all shown %dx%d", menu.menuWidth, menu.menuHeight, fullWidth, fullHeight)
	}

	hidden.SetVisible(true)
	menu.rebuildVisualElements()
	if menu.menuHeight != fullHeight {
		t.Errorf("height after showing the item again = %d, want %d", menu.menuHeight, fullHeight)
	}
}
