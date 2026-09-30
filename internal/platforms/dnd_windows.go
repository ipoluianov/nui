package platforms

import (
	"syscall"
	"unsafe"
)

// Files dropped from Explorer: DragAcceptFiles makes Windows send the window
// WM_DROPFILES with the list of the dropped files.

const c_WM_DROPFILES = 0x0233

var (
	procDragAcceptFiles = shell32.NewProc("DragAcceptFiles")
	procDragQueryFileW  = shell32.NewProc("DragQueryFileW")
	procDragQueryPoint  = shell32.NewProc("DragQueryPoint")
	procDragFinish      = shell32.NewProc("DragFinish")
)

func (c *nativeWindow) enableFileDrop() {
	procDragAcceptFiles.Call(uintptr(c.hwnd), 1)
}

// processDropFiles handles WM_DROPFILES: hDrop holds the files and the point
// of the drop, in client coordinates
func (c *nativeWindow) processDropFiles(hDrop uintptr) {
	defer procDragFinish.Call(hDrop)
	if c.onFilesDropped == nil {
		return
	}
	count, _, _ := procDragQueryFileW.Call(hDrop, 0xFFFFFFFF, 0, 0)
	files := make([]string, 0, count)
	for i := uintptr(0); i < count; i++ {
		n, _, _ := procDragQueryFileW.Call(hDrop, i, 0, 0)
		if n == 0 {
			continue
		}
		buf := make([]uint16, n+1)
		procDragQueryFileW.Call(hDrop, i, uintptr(unsafe.Pointer(&buf[0])), n+1)
		files = append(files, syscall.UTF16ToString(buf))
	}
	var pt struct{ x, y int32 }
	procDragQueryPoint.Call(hDrop, uintptr(unsafe.Pointer(&pt)))
	if len(files) > 0 {
		c.onFilesDropped(files, int(pt.x), int(pt.y))
	}
}
