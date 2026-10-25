//go:build darwin

package platforms

import (
	"github.com/ebitengine/purego/objc"
)

// Files dragged out to other applications: an AppKit dragging session of
// the files' URLs, started from the view in its mouseDragged: (the drag
// leaving the form). The view is the session's source (NSDraggingSource):
// AppKit asks it which operations it allows and tells it when the session
// ends, with the operation the target did. The session runs in the event
// loop after StartFileDrag returns: done is called at its end.

const (
	nsDragOperationLink = 2
	nsDragOperationMove = 16
)

var (
	selBeginDraggingSession     = objc.RegisterName("beginDraggingSessionWithItems:event:source:")
	selSourceOperationMask      = objc.RegisterName("draggingSession:sourceOperationMaskForDraggingContext:")
	selDraggingSessionEnded     = objc.RegisterName("draggingSession:endedAtPoint:operation:")
	selInitWithPasteboardWriter = objc.RegisterName("initWithPasteboardWriter:")
	selSetDraggingFrameContents = objc.RegisterName("setDraggingFrame:contents:")
	selSharedWorkspace          = objc.RegisterName("sharedWorkspace")
	selIconForFile              = objc.RegisterName("iconForFile:")
	selCurrentEvent             = objc.RegisterName("currentEvent")

	clsNSDraggingItem = objc.GetClass("NSDraggingItem")
	clsNSWorkspace    = objc.GetClass("NSWorkspace")
)

// dragOutDone is the done of the drag going on, by the view that started
// it; one drag at a time, on the UI thread
var dragOutDone = map[objc.ID]func(effect DragEffect, dropped bool){}

func (c *nativeWindow) StartFileDrag(files []string, done func(effect DragEffect, dropped bool)) bool {
	win, ok := cocoaWindows[int(c.hwnd)]
	if !ok || len(files) == 0 {
		return false
	}
	started := false
	withAutoreleasePool(func() {
		view := objc.Send[objc.ID](win, selContentView)
		nsApp := objc.ID(clsNSApplication).Send(selSharedApplication)
		event := objc.Send[objc.ID](nsApp, selCurrentEvent)
		if view == 0 || event == 0 {
			return
		}
		// The icons of the files under the mouse, one behind another
		p := eventLocationInView(view, event)
		items := objc.ID(clsNSMutableArray).Send(selAlloc).Send(selInit)
		defer items.Send(selRelease)
		workspace := objc.ID(clsNSWorkspace).Send(selSharedWorkspace)
		for i, f := range files {
			url := objc.ID(clsNSURL).Send(selFileURLWithPath, goStringToNS(f))
			item := objc.ID(clsNSDraggingItem).Send(selAlloc).Send(selInitWithPasteboardWriter, url)
			icon := workspace.Send(selIconForFile, goStringToNS(f))
			offset := float64(min(i, 4) * 4)
			frame := nsRect{Origin: nsPoint{p.X - 16 + offset, p.Y - 16 - offset}, Size: nsSize{32, 32}}
			item.Send(selSetDraggingFrameContents, frame, icon)
			items.Send(selAddObject, item)
			item.Send(selRelease)
		}
		session := view.Send(selBeginDraggingSession, items, event, view)
		if session == 0 {
			return
		}
		dragOutDone[view] = done
		started = true
	})
	return started
}

// nuiViewSourceOperationMask: the files may be copied, moved or linked
func nuiViewSourceOperationMask(self objc.ID, _ objc.SEL, session objc.ID, context int) uint {
	return nsDragOperationCopy | nsDragOperationMove | nsDragOperationLink
}

// nuiViewDraggingSessionEnded gives the result to the done of the drag. The
// point (an NSPoint) comes as its two doubles: they are passed the same way
// on arm64 and amd64.
func nuiViewDraggingSessionEnded(self objc.ID, _ objc.SEL, session objc.ID, x, y float64, operation uint) {
	done := dragOutDone[self]
	delete(dragOutDone, self)
	if done == nil {
		return
	}
	switch {
	case operation&nsDragOperationMove != 0:
		done(DragEffectMove, true)
	case operation&nsDragOperationCopy != 0:
		done(DragEffectCopy, true)
	case operation&nsDragOperationLink != 0:
		done(DragEffectLink, true)
	case operation != nsDragOperationNone:
		done(DragEffectCopy, true)
	default:
		done(DragEffectNone, false)
	}
}
