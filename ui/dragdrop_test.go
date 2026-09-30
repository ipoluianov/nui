package ui

import "testing"

func TestDragAndDrop(t *testing.T) {
	form := NewForm()
	form.processResize(600, 400)
	src := NewButton("Source")
	clicks := 0
	src.SetOnClick(func() { clicks++ })
	src.SetDragSource(func(x, y int) *DragData { return &DragData{Text: "item", Value: 42} })
	target := NewPanel()
	target.SetMinHeight(100)
	var dropped *DragData
	target.SetDropTarget(func(d *DragData, x, y int) bool { return d.Value == 42 }, func(d *DragData, x, y int) { dropped = d })
	form.Panel().AddWidget(0, 0, src)
	form.Panel().AddWidget(1, 0, target)
	form.Panel().AddVSpacer(2, 0)
	form.UpdateLayout()

	sx, sy := src.RectClientAreaOnWindow()
	tx, ty := target.RectClientAreaOnWindow()
	press := func() { form.processMouseDown(MouseButtonLeft, sx+5, sy+5) }

	// A small move is still a click
	press()
	form.processMouseMove(sx+7, sy+6)
	form.processMouseUp(MouseButtonLeft, sx+7, sy+6)
	if clicks != 1 || form.drag.active {
		t.Fatalf("small move: clicks %d, drag %v", clicks, form.drag.active)
	}

	// A drag onto the target drops, and doesn't click the source
	press()
	form.processMouseMove(sx+5, sy+40)
	form.processMouseMove(tx+20, ty+20)
	if !form.drag.active || form.drag.target == nil || form.drag.target.Id() != target.Id() {
		t.Fatalf("dragging over the target: active %v, target %v", form.drag.active, form.drag.target)
	}
	form.processMouseUp(MouseButtonLeft, tx+20, ty+20)
	if dropped == nil || dropped.Text != "item" || dropped.Source == nil || dropped.Source.Id() != src.Id() || clicks != 1 {
		t.Fatalf("drop: %+v, clicks %d", dropped, clicks)
	}

	// Dropped back on the source itself: no target, no click
	dropped = nil
	press()
	form.processMouseMove(sx+30, sy+5)
	form.processMouseUp(MouseButtonLeft, sx+10, sy+5)
	if dropped != nil || clicks != 1 {
		t.Errorf("drop on the source: dropped %v, clicks %d", dropped, clicks)
	}

	// Escape cancels: the release drops nothing
	press()
	form.processMouseMove(tx+20, ty+20)
	form.processKeyDown(KeyEsc, KeyModifiers{})
	form.processMouseMove(tx+25, ty+25)
	form.processMouseUp(MouseButtonLeft, tx+25, ty+25)
	if dropped != nil || form.drag.active || form.drag.cancelled {
		t.Errorf("Escape: dropped %v, state %+v", dropped, form.drag)
	}

	// System files go to a target that accepts them, else to the form
	var formFiles []string
	form.SetOnFilesDropped(func(files []string, x, y int) { formFiles = files })
	form.processFilesDropped([]string{"/tmp/a.txt"}, tx+20, ty+20)
	if len(formFiles) != 1 {
		t.Errorf("files not taken by the target should reach the form: %v", formFiles)
	}
	target.SetDropTarget(nil, func(d *DragData, x, y int) { dropped = d })
	form.processFilesDropped([]string{"/tmp/b.txt"}, tx+20, ty+20)
	if dropped == nil || dropped.Files[0] != "/tmp/b.txt" || dropped.Text != "b.txt" {
		t.Errorf("files to the target: %+v", dropped)
	}
}
