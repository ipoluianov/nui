package platforms

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var (
	comdlg32 = syscall.NewLazyDLL("comdlg32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procGetOpenFileNameW     = comdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW     = comdlg32.NewProc("GetSaveFileNameW")
	procCommDlgExtendedError = comdlg32.NewProc("CommDlgExtendedError")

	procSHBrowseForFolderW   = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = shell32.NewProc("SHGetPathFromIDListW")

	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
	procCoTaskMemFree  = ole32.NewProc("CoTaskMemFree")
)

// OPENFILENAMEW (Windows 2000+ layout, with the pvReserved/dwReserved/FlagsEx
// tail). Field order and types mirror the Win32 struct exactly so Go's
// natural alignment on amd64 lines up with the C layout - do not reorder.
type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       uintptr
	lpstrCustomFilter uintptr
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         uintptr
	nMaxFile          uint32
	lpstrFileTitle    uintptr
	nMaxFileTitle     uint32
	lpstrInitialDir   uintptr
	lpstrTitle        uintptr
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       uintptr
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    uintptr
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

// BROWSEINFOW, used by SHBrowseForFolderW (the classic "Select Folder" dialog).
type browseInfoW struct {
	hwndOwner      uintptr
	pidlRoot       uintptr
	pszDisplayName uintptr
	lpszTitle      uintptr
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

const (
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnExplorer         = 0x00080000
	ofnOverwritePrompt  = 0x00000002
	ofnAllowMultiSelect = 0x00000200
	ofnNoChangeDir      = 0x00000008
	ofnHideReadOnly     = 0x00000004

	// Buffer for the (possibly multi-select) result path list, in UTF-16 units.
	ofnFileBufSize = 1 << 16

	bifReturnOnlyFSDirs = 0x00000001
	bifNewDialogStyle   = 0x00000040

	bffmInitialized   = 1
	bffmSetSelection  = 0x400 + 103 // BFFM_SETSELECTIONW
	coinitApartment   = 0x2         // COINIT_APARTMENTTHREADED
	maxPidlPathBufLen = 32768
)

// selectDirBrowseCallbackPtr is created once at init (syscall.NewCallback
// must not be called repeatedly/concurrently) and reused for every
// SelectDirectoryDialog call - fine since dialogs in this package are shown
// synchronously, one at a time.
var selectDirBrowseCallbackPtr = syscall.NewCallback(selectDirBrowseCallback)

// selectDirBrowseCallback pre-selects DefaultDirectory: on BFFM_INITIALIZED,
// lpData carries a UTF-16 path pointer set by the caller, forwarded to the
// dialog via BFFM_SETSELECTIONW.
func selectDirBrowseCallback(hwnd syscall.Handle, uMsg uint32, lParam, lpData uintptr) uintptr {
	if uMsg == bffmInitialized && lpData != 0 {
		procSendMessageW.Call(uintptr(hwnd), bffmSetSelection, 1, lpData)
	}
	return 0
}

func ptrOrZero(p *uint16) uintptr {
	if p == nil {
		return 0
	}
	return uintptr(unsafe.Pointer(p))
}

func ownerHwnd(parent Window) uintptr {
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		return uintptr(p.hwnd)
	}
	return 0
}

// makeFilterW builds the OPENFILENAME filter string: pairs of
// "Display Name (pattern)\0pattern\0", double-NUL terminated.
func makeFilterW(filters []FileDialogFilter) []uint16 {
	if len(filters) == 0 {
		return utf16FilterParts("All Files (*.*)", "*.*")
	}
	parts := make([]string, 0, len(filters)*2)
	for _, f := range filters {
		pattern := strings.Join(f.Patterns, ";")
		name := pattern
		if f.DisplayName != "" {
			name = f.DisplayName + " (" + pattern + ")"
		}
		parts = append(parts, name, pattern)
	}
	return utf16FilterParts(parts...)
}

func utf16FilterParts(parts ...string) []uint16 {
	var buf []uint16
	for _, p := range parts {
		u, _ := syscall.UTF16FromString(p) // already NUL-terminated
		buf = append(buf, u...)
	}
	return append(buf, 0) // second, closing NUL
}

// parseMultiSelectBuffer decodes an OFN_EXPLORER|OFN_ALLOWMULTISELECT result:
// a single string is already a full path; multiple strings are "dir", then
// file names to join under it (see GetOpenFileName's documented format).
func parseMultiSelectBuffer(buf []uint16) []string {
	var strs []string
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] != 0 {
			continue
		}
		if i == start {
			break
		}
		strs = append(strs, syscall.UTF16ToString(buf[start:i]))
		start = i + 1
	}
	if len(strs) <= 1 {
		return strs
	}

	dir := strs[0]
	if !strings.HasSuffix(dir, "\\") {
		dir += "\\"
	}
	result := make([]string, 0, len(strs)-1)
	for _, name := range strs[1:] {
		result = append(result, dir+name)
	}
	return result
}

func openFileDialog(parent Window, opts OpenFileDialogOptions) ([]string, error) {
	fileBuf := make([]uint16, ofnFileBufSize)
	filter := makeFilterW(opts.Filters)

	var titlePtr, initialDirPtr *uint16
	if opts.Title != "" {
		titlePtr, _ = syscall.UTF16PtrFromString(opts.Title)
	}
	if opts.DefaultDirectory != "" {
		initialDirPtr, _ = syscall.UTF16PtrFromString(opts.DefaultDirectory)
	}

	flags := uint32(ofnExplorer | ofnPathMustExist | ofnFileMustExist | ofnNoChangeDir | ofnHideReadOnly)
	if opts.AllowMultiple {
		flags |= ofnAllowMultiSelect
	}

	var pin runtime.Pinner
	defer pin.Unpin()
	pinAll(&pin, &filter[0], &fileBuf[0], titlePtr, initialDirPtr)

	ofn := openFileNameW{
		lStructSize:     uint32(unsafe.Sizeof(openFileNameW{})),
		hwndOwner:       ownerHwnd(parent),
		lpstrFilter:     uintptr(unsafe.Pointer(&filter[0])),
		lpstrFile:       uintptr(unsafe.Pointer(&fileBuf[0])),
		nMaxFile:        uint32(len(fileBuf)),
		lpstrInitialDir: ptrOrZero(initialDirPtr),
		lpstrTitle:      ptrOrZero(titlePtr),
		flags:           flags,
	}

	ret, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return nil, commDlgError("GetOpenFileName")
	}

	return parseMultiSelectBuffer(fileBuf), nil
}

func saveFileDialog(parent Window, opts SaveFileDialogOptions) (string, error) {
	fileBuf := make([]uint16, ofnFileBufSize)
	if opts.DefaultFileName != "" {
		name, _ := syscall.UTF16FromString(opts.DefaultFileName)
		copy(fileBuf, name)
	}

	filter := makeFilterW(opts.Filters)

	var titlePtr, initialDirPtr *uint16
	if opts.Title != "" {
		titlePtr, _ = syscall.UTF16PtrFromString(opts.Title)
	}
	if opts.DefaultDirectory != "" {
		initialDirPtr, _ = syscall.UTF16PtrFromString(opts.DefaultDirectory)
	}

	var pin runtime.Pinner
	defer pin.Unpin()
	pinAll(&pin, &filter[0], &fileBuf[0], titlePtr, initialDirPtr)

	ofn := openFileNameW{
		lStructSize:     uint32(unsafe.Sizeof(openFileNameW{})),
		hwndOwner:       ownerHwnd(parent),
		lpstrFilter:     uintptr(unsafe.Pointer(&filter[0])),
		lpstrFile:       uintptr(unsafe.Pointer(&fileBuf[0])),
		nMaxFile:        uint32(len(fileBuf)),
		lpstrInitialDir: ptrOrZero(initialDirPtr),
		lpstrTitle:      ptrOrZero(titlePtr),
		flags:           ofnExplorer | ofnOverwritePrompt | ofnPathMustExist | ofnNoChangeDir,
	}

	ret, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return "", commDlgError("GetSaveFileName")
	}

	return syscall.UTF16ToString(fileBuf), nil
}

// pinAll pins the buffers a dialog structure refers to by uintptr fields
// (nil ones are skipped). The dialog runs a modal loop calling back into Go
// (paint, timers): without the pin a buffer on the goroutine stack could
// move as the stack grows, and one on the heap - referenced by nothing the
// GC sees - could be freed, while the dialog still reads or writes it.
func pinAll(pin *runtime.Pinner, ptrs ...*uint16) {
	for _, p := range ptrs {
		if p != nil {
			pin.Pin(p)
		}
	}
}

func commDlgError(api string) error {
	// A zero CommDlgExtendedError means the user simply cancelled.
	code, _, _ := procCommDlgExtendedError.Call()
	if code == 0 {
		return nil
	}
	return fmt.Errorf("nui: %s failed (CommDlgExtendedError=0x%x)", api, code)
}

func selectDirectoryDialog(parent Window, opts SelectDirectoryDialogOptions) (string, error) {
	// Best-effort: SHBrowseForFolderW works fine even if this reports
	// RPC_E_CHANGED_MODE because some other component already initialized
	// COM in a different apartment mode, so the result is intentionally ignored.
	procCoInitializeEx.Call(0, coinitApartment)

	var titlePtr *uint16
	if opts.Title != "" {
		titlePtr, _ = syscall.UTF16PtrFromString(opts.Title)
	}

	var pin runtime.Pinner
	defer pin.Unpin()

	var lpData uintptr
	if opts.DefaultDirectory != "" {
		initialDir, _ := syscall.UTF16PtrFromString(opts.DefaultDirectory)
		pinAll(&pin, initialDir)
		lpData = uintptr(unsafe.Pointer(initialDir))
	}

	displayName := make([]uint16, 4096)
	pinAll(&pin, &displayName[0], titlePtr)

	bi := browseInfoW{
		hwndOwner:      ownerHwnd(parent),
		pszDisplayName: uintptr(unsafe.Pointer(&displayName[0])),
		lpszTitle:      ptrOrZero(titlePtr),
		ulFlags:        bifReturnOnlyFSDirs | bifNewDialogStyle,
		lpfn:           selectDirBrowseCallbackPtr,
		lParam:         lpData,
	}

	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		// The user cancelled - SHBrowseForFolderW has no extended-error API.
		return "", nil
	}
	defer procCoTaskMemFree.Call(pidl)

	pathBuf := make([]uint16, maxPidlPathBufLen)
	ret, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))
	if ret == 0 {
		return "", errors.New("nui: SHGetPathFromIDListW failed")
	}

	return syscall.UTF16ToString(pathBuf), nil
}
