package platforms

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

var (
	procSHParseDisplayName = shell32.NewProc("SHParseDisplayName")
	procSHBindToParent     = shell32.NewProc("SHBindToParent")

	iidIShellFolder = syscall.GUID{Data1: 0x000214E6, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIDropTarget  = syscall.GUID{Data1: 0x00000122, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

const (
	c_CF_HDROP             = 15
	c_IDataObjectGetData   = 3
	c_IShellFolderGetUIObj = 10
	c_IDropTargetDragEnter = 3
	c_IDropTargetDragLeave = 5
	c_MK_LBUTTON           = 1
	c_MK_SHIFT             = 4
)

// explorerDropTarget is the drop target Explorer gives for the directory
func explorerDropTarget(t *testing.T, dir string) uintptr {
	path, _ := syscall.UTF16PtrFromString(dir)
	var pidl uintptr
	if hr, _, _ := procSHParseDisplayName.Call(uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&pidl)), 0, 0); int32(hr) < 0 {
		t.Fatalf("SHParseDisplayName: %x", hr)
	}
	defer procILFree.Call(pidl)
	var folder, child uintptr
	if hr, _, _ := procSHBindToParent.Call(pidl, uintptr(unsafe.Pointer(&iidIShellFolder)), uintptr(unsafe.Pointer(&folder)), uintptr(unsafe.Pointer(&child))); int32(hr) < 0 {
		t.Fatalf("SHBindToParent: %x", hr)
	}
	defer comCall(folder, c_IUnknownRelease)
	var target uintptr
	if hr := comCall(folder, c_IShellFolderGetUIObj, 0, 1, uintptr(unsafe.Pointer(&child)), uintptr(unsafe.Pointer(&iidIDropTarget)), 0, uintptr(unsafe.Pointer(&target))); int32(hr) < 0 {
		t.Fatalf("GetUIObjectOf: %x", hr)
	}
	return target
}

// dragEnter returns the effect the target picks for the data with the keys held
func dragEnter(target, dataObject uintptr, keys uintptr) uint32 {
	effect := uint32(c_DROPEFFECT_COPY | c_DROPEFFECT_MOVE | c_DROPEFFECT_LINK)
	comCall(target, c_IDropTargetDragEnter, dataObject, keys, 0, uintptr(unsafe.Pointer(&effect)))
	comCall(target, c_IDropTargetDragLeave)
	return effect
}

func TestFilesDataObject(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procOleInitialize.Call(0)

	src, dst := t.TempDir(), t.TempDir() // on the same drive: Explorer would move
	files := []string{filepath.Join(src, "a.txt"), filepath.Join(src, "sub")}
	os.WriteFile(files[0], []byte("a"), 0644)
	os.Mkdir(files[1], 0755)

	dataObject := newFilesDataObject(files)
	if dataObject == 0 {
		t.Fatal("no data object")
	}
	defer comCall(dataObject, c_IUnknownRelease)

	// The files are there as Explorer's drop (WM_DROPFILES) gives them
	fe := formatEtc{cfFormat: c_CF_HDROP, dwAspect: c_DVASPECT_CONTENT, lindex: -1, tymed: c_TYMED_HGLOBAL}
	var sm stgMedium
	if hr := comCall(dataObject, c_IDataObjectGetData, uintptr(unsafe.Pointer(&fe)), uintptr(unsafe.Pointer(&sm))); int32(hr) < 0 {
		t.Fatalf("GetData(CF_HDROP): %x", hr)
	}
	count, _, _ := procDragQueryFileW.Call(sm.hGlobal, 0xFFFFFFFF, 0, 0)
	if int(count) != len(files) {
		t.Fatalf("CF_HDROP has %d files", count)
	}
	for i := range files {
		buf := make([]uint16, 1024)
		procDragQueryFileW.Call(sm.hGlobal, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), 1024)
		if got := syscall.UTF16ToString(buf); got != files[i] {
			t.Errorf("file %d: %q, want %q", i, got, files[i])
		}
	}

	target := explorerDropTarget(t, dst)
	defer comCall(target, c_IUnknownRelease)
	if effect := dragEnter(target, dataObject, c_MK_LBUTTON); effect != c_DROPEFFECT_COPY {
		t.Errorf("Explorer picks effect %d, want copy", effect)
	}
	if effect := dragEnter(target, dataObject, c_MK_LBUTTON|c_MK_SHIFT); effect != c_DROPEFFECT_MOVE {
		t.Errorf("Explorer picks effect %d with Shift, want move", effect)
	}
}
