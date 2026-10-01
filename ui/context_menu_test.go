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

// A menu command runs on the release, not on the press: on the item under
// the mouse, which may be another item than the pressed one
func TestContextMenuItemClicksOnRelease(t *testing.T) {
	form := NewForm()
	form.processResize(400, 300)
	menu := NewContextMenu(form.Panel())
	var clicked []string
	copyItem := menu.AddItem("Copy", func() { clicked = append(clicked, "copy") })
	menu.AddSeparator()
	pasteItem := menu.AddItem("Paste", func() { clicked = append(clicked, "paste") })
	menu.attachToForm(menu, form)
	menu.rebuildVisualElements()
	menu.SetPosition(10, 20)
	// The menu is open in its popup window
	form.popupHosts = append(form.popupHosts, &popupHost{widget: menu})
	form.topWidget.PopupWidgets = append(form.topWidget.PopupWidgets, menu)

	w, h := copyItem.Width(), copyItem.Height()
	if w == 0 || h == 0 {
		t.Fatalf("item size %dx%d", w, h)
	}
	copyItem.mouseDownHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	if len(clicked) != 0 {
		t.Fatal("the press ran the command")
	}
	copyItem.mouseUpHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	if len(clicked) != 1 || clicked[0] != "copy" {
		t.Fatalf("the release: %v", clicked)
	}

	// Pressed on Copy, released on Paste: Paste runs
	copyItem.mouseDownHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	copyItem.mouseUpHandler(MouseButtonLeft, 5, pasteItem.Y()-copyItem.Y()+5, KeyModifiers{})
	if len(clicked) != 2 || clicked[1] != "paste" {
		t.Fatalf("released on another item: %v", clicked)
	}

	// Released off the menu, on the separator, without a press, or with the
	// right button: nothing runs
	copyItem.mouseDownHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	copyItem.mouseUpHandler(MouseButtonLeft, w+50, 5, KeyModifiers{})
	copyItem.mouseDownHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	copyItem.mouseUpHandler(MouseButtonLeft, 5, h+ContextMenuSeparatorHeight/2, KeyModifiers{})
	copyItem.mouseUpHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	copyItem.mouseDownHandler(MouseButtonRight, 5, 5, KeyModifiers{})
	copyItem.mouseUpHandler(MouseButtonRight, 5, 5, KeyModifiers{})
	if len(clicked) != 2 {
		t.Errorf("clicks %v, want 2", clicked)
	}
}

func TestComboBoxItemPickedOnRelease(t *testing.T) {
	item := newComboBoxPopupItem(2, "Two")
	newTestForm(t, item)
	item.SetSize(100, 20)
	picked := -1
	item.OnClick = func(i int) { picked = i }
	item.mouseDownHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	if picked != -1 {
		t.Fatal("picked on the press")
	}
	item.mouseUpHandler(MouseButtonLeft, 200, 5, KeyModifiers{})
	if picked != -1 {
		t.Fatal("picked on a release off the item")
	}
	item.mouseDownHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	item.mouseUpHandler(MouseButtonLeft, 5, 5, KeyModifiers{})
	if picked != 2 {
		t.Errorf("picked %d, want 2", picked)
	}
}
