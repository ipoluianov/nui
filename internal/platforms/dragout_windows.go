package platforms

import (
	"syscall"
	"unsafe"
)

// Files dragged out to other applications: the shell makes the data object of
// the files (the same one Explorer drags: CF_HDROP, the shell IDs...) and
// SHDoDragDrop runs the drag with the shell's drop source and drag image.

const (
	c_DROPEFFECT_COPY = 1
	c_DROPEFFECT_MOVE = 2
	c_DROPEFFECT_LINK = 4

	c_DRAGDROP_S_DROP = 0x00040100

	c_TYMED_HGLOBAL      = 1
	c_DVASPECT_CONTENT   = 1
	c_GMEM_MOVEABLE      = 0x0002
	c_IDataObjectSetData = 7 // the index of SetData in the IDataObject vtable
	c_IUnknownRelease    = 2
)

var (
	procOleInitialize               = ole32.NewProc("OleInitialize")
	procSHCreateDataObject          = shell32.NewProc("SHCreateDataObject")
	procSHDoDragDrop                = shell32.NewProc("SHDoDragDrop")
	procILCreateFromPathW           = shell32.NewProc("ILCreateFromPathW")
	procILClone                     = shell32.NewProc("ILClone")
	procILRemoveLastID              = shell32.NewProc("ILRemoveLastID")
	procILFindLastID                = shell32.NewProc("ILFindLastID")
	procILIsEqual                   = shell32.NewProc("ILIsEqual")
	procILFree                      = shell32.NewProc("ILFree")
	procGlobalAllocDrag             = kernel32.NewProc("GlobalAlloc")
	procGlobalLockDrag              = kernel32.NewProc("GlobalLock")
	procGlobalUnlockDrag            = kernel32.NewProc("GlobalUnlock")
	procRegisterClipboardFormatDrag = user32.NewProc("RegisterClipboardFormatW")
	procGetCaptureDrag              = user32.NewProc("GetCapture")

	iidIDataObject = syscall.GUID{Data1: 0x0000010E, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}

	oleInitialized bool
)

type formatEtc struct {
	cfFormat uint16
	ptd      uintptr
	dwAspect uint32
	lindex   int32
	tymed    uint32
}

type stgMedium struct {
	tymed          uint32
	hGlobal        uintptr
	pUnkForRelease uintptr
}

func comCall(obj uintptr, method int, args ...uintptr) uintptr {
	vtbl := *(*[16]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(obj))))
	ret, _, _ := syscall.SyscallN(vtbl[method], append([]uintptr{obj}, args...)...)
	return ret
}

func (c *nativeWindow) startFileDragSync(files []string) (effect DragEffect, dropped bool, started bool) {
	if len(files) == 0 {
		return DragEffectNone, false, false
	}
	if !oleInitialized {
		// DoDragDrop needs OLE on the thread of the window: the UI thread
		procOleInitialize.Call(0)
		oleInitialized = true
	}

	dataObject := newFilesDataObject(files)
	if dataObject == 0 {
		return DragEffectNone, false, false
	}
	defer comCall(dataObject, c_IUnknownRelease)

	var result uint32
	hr, _, _ := procSHDoDragDrop.Call(uintptr(c.hwnd), dataObject, 0,
		c_DROPEFFECT_COPY|c_DROPEFFECT_MOVE|c_DROPEFFECT_LINK, uintptr(unsafe.Pointer(&result)))

	// The drag took the button release: let the capture of the press go too
	if capture, _, _ := procGetCaptureDrag.Call(); capture == uintptr(c.hwnd) {
		procReleaseCapture.Call()
	}

	if hr != c_DRAGDROP_S_DROP {
		return DragEffectNone, false, true
	}
	switch {
	case result&c_DROPEFFECT_MOVE != 0:
		effect = DragEffectMove
	case result&c_DROPEFFECT_COPY != 0:
		effect = DragEffectCopy
	case result&c_DROPEFFECT_LINK != 0:
		effect = DragEffectLink
	}
	return effect, true, true
}

// newFilesDataObject makes the shell data object of the files, as Explorer
// drags them; 0 if it can't. Release it when done.
func newFilesDataObject(files []string) uintptr {
	// The absolute IDs of the files, and the IDs relative to their folder
	pidls := make([]uintptr, 0, len(files))
	defer func() {
		for _, p := range pidls {
			procILFree.Call(p)
		}
	}()
	for _, f := range files {
		path, err := syscall.UTF16PtrFromString(f)
		if err != nil {
			return 0
		}
		p, _, _ := procILCreateFromPathW.Call(uintptr(unsafe.Pointer(path)))
		if p == 0 {
			return 0
		}
		pidls = append(pidls, p)
	}
	folder := parentIDList(pidls[0])
	defer func() {
		if folder != 0 {
			procILFree.Call(folder)
		}
	}()
	children := make([]uintptr, len(pidls))
	for i, p := range pidls {
		parent := parentIDList(p)
		same, _, _ := procILIsEqual.Call(parent, folder)
		procILFree.Call(parent)
		if same == 0 {
			// Files of different folders: all of them relative to the desktop
			procILFree.Call(folder)
			folder = 0
			copy(children, pidls)
			break
		}
		children[i], _, _ = procILFindLastID.Call(p)
	}

	var dataObject uintptr
	hr, _, _ := procSHCreateDataObject.Call(folder, uintptr(len(children)), uintptr(unsafe.Pointer(&children[0])),
		0, uintptr(unsafe.Pointer(&iidIDataObject)), uintptr(unsafe.Pointer(&dataObject)))
	if int32(hr) < 0 || dataObject == 0 {
		return 0
	}
	setPreferredDropEffect(dataObject, c_DROPEFFECT_COPY)
	return dataObject
}

// parentIDList returns a copy of the ID list without its last ID (free it with ILFree)
func parentIDList(pidl uintptr) uintptr {
	parent, _, _ := procILClone.Call(pidl)
	procILRemoveLastID.Call(parent)
	return parent
}

// setPreferredDropEffect asks the targets to copy unless the user says else
// (Shift - move), as Explorer would move files dragged on the same drive
func setPreferredDropEffect(dataObject uintptr, dropEffect uint32) {
	name, _ := syscall.UTF16PtrFromString("Preferred DropEffect")
	cf, _, _ := procRegisterClipboardFormatDrag.Call(uintptr(unsafe.Pointer(name)))
	if cf == 0 {
		return
	}
	h, _, _ := procGlobalAllocDrag.Call(c_GMEM_MOVEABLE, 4)
	if h == 0 {
		return
	}
	if p, _, _ := procGlobalLockDrag.Call(h); p != 0 {
		*(*uint32)(unsafe.Pointer(p)) = dropEffect
		procGlobalUnlockDrag.Call(h)
	}
	fe := formatEtc{cfFormat: uint16(cf), dwAspect: c_DVASPECT_CONTENT, lindex: -1, tymed: c_TYMED_HGLOBAL}
	sm := stgMedium{tymed: c_TYMED_HGLOBAL, hGlobal: h}
	// fRelease: the data object owns the memory now
	if hr := comCall(dataObject, c_IDataObjectSetData, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&sm)), 1); int32(hr) < 0 {
		procGlobalFree.Call(h)
	}
}
