package platforms

import (
	"image"
	"syscall"
	"unsafe"
)

var (
	procBitBlt               = gdi32.NewProc("BitBlt")
	procGetSystemDirectoryW  = kernel32.NewProc("GetSystemDirectoryW")
	procWTSRegisterSession   = wtsapi32.NewProc("WTSRegisterSessionNotification")
	procWTSUnRegisterSession = wtsapi32.NewProc("WTSUnRegisterSessionNotification")

	// wtsapi32.dll is not one of the KnownDLLs: loaded from the system
	// directory by its full path, not from the search path
	wtsapi32 = systemDLL("wtsapi32.dll")
)

const (
	c_WM_DISPLAYCHANGE        = 0x007E
	c_WM_WTSSESSION_CHANGE    = 0x02B1
	c_NOTIFY_FOR_THIS_SESSION = 0
)

// systemDLL is the DLL of the system directory
func systemDLL(name string) *syscall.LazyDLL {
	buf := make([]uint16, 512)
	n, _, _ := procGetSystemDirectoryW.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || int(n) >= len(buf) {
		return syscall.NewLazyDLL(name)
	}
	return syscall.NewLazyDLL(syscall.UTF16ToString(buf[:n]) + `\` + name)
}

// registerSessionNotification asks for WM_WTSSESSION_CHANGE: an RDP session
// connected again or the screen unlocked repaints the window
func registerSessionNotification(hwnd uintptr) {
	if procWTSRegisterSession.Find() == nil {
		procWTSRegisterSession.Call(hwnd, c_NOTIFY_FOR_THIS_SESSION)
	}
}

func unregisterSessionNotification(hwnd uintptr) {
	if procWTSUnRegisterSession.Find() == nil {
		procWTSUnRegisterSession.Call(hwnd)
	}
}

// backBuffer is the paint surface of a window: a 32-bit top-down DIB section
// selected into a memory DC, copied to the window with one BitBlt per paint.
//
// It replaces SetDIBitsToDevice from a Go buffer in bands, which some display
// drivers - those of virtual machines and of RDP sessions in particular -
// apply only in part, leaving pieces of the window unpainted. A BitBlt
// between two DCs is the operation every driver and the RDP protocol handle
// (and cache) best.
//
// Owned by the UI thread, like the window.
type backBuffer struct {
	dc     uintptr
	bitmap uintptr
	old    uintptr // the bitmap the memory DC came with, put back on release
	bits   unsafe.Pointer
	// The size of the DIB, at least the size painted
	width, height int32
}

// ensure makes the DIB at least width x height. It grows to the size asked
// and is made again smaller only when it is much larger than needed, so
// resizing a window doesn't make a new one on every step.
func (b *backBuffer) ensure(width, height int32) bool {
	if b.dc != 0 && width <= b.width && height <= b.height &&
		int64(width)*int64(height)*4 >= int64(b.width)*int64(b.height) {
		return true
	}
	b.release()

	dc, _, _ := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return false
	}
	bi := t_BITMAPINFO{
		Header: t_BITMAPINFOHEADER{
			Size:     uint32(unsafe.Sizeof(t_BITMAPINFOHEADER{})),
			Width:    width,
			Height:   -height, // top-down: the rows in the order of image.RGBA
			Planes:   1,
			BitCount: 32,
		},
	}
	var bits unsafe.Pointer
	bitmap, _, _ := procCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&bi)), c_DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		procDeleteDC.Call(dc)
		return false
	}
	old, _, _ := procSelectObject.Call(dc, bitmap)
	b.dc, b.bitmap, b.old, b.bits = dc, bitmap, old, bits
	b.width, b.height = width, height
	return true
}

// release frees the DIB and the memory DC
func (b *backBuffer) release() {
	if b.dc == 0 {
		return
	}
	procSelectObject.Call(b.dc, b.old)
	procDeleteObject.Call(b.bitmap)
	procDeleteDC.Call(b.dc)
	*b = backBuffer{}
}

// present copies the top-left width x height pixels of img to hdc at 0, 0.
// It reports false if the frame didn't get to the window: the caller paints
// it again later.
func (b *backBuffer) present(img *image.RGBA, hdc uintptr, width, height int32) bool {
	if width <= 0 || height <= 0 {
		return true
	}
	if b.ensure(width, height) {
		// GDI may still be writing into the DIB from a batched call
		procGdiFlush.Call()
		dst := unsafe.Slice((*uint32)(b.bits), int(b.width)*int(b.height))
		copyRGBAToBGRA(dst, int(b.width), img, int(width), int(height))
		ok, _, _ := procBitBlt.Call(hdc, 0, 0, uintptr(width), uintptr(height), b.dc, 0, 0, c_SRCCOPY)
		if ok != 0 {
			return true
		}
		// The DC of the window may not take a BitBlt from this one (e.g.
		// right after the display mode or the session changed): a DIB made
		// again next time matches the new state
		b.release()
	}
	// No DIB section (out of GDI resources): straight from memory instead
	return drawRGBAToHDC(img, hdc, width, height)
}

// copyRGBAToBGRA copies the top-left width x height pixels of img to dst,
// whose rows are dstStride pixels long, swapping red and blue. A pixel at a
// time as a little-endian uint32, as all the Windows targets are.
func copyRGBAToBGRA(dst []uint32, dstStride int, img *image.RGBA, width, height int) {
	for y := 0; y < height; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+width*4]
		src := unsafe.Slice((*uint32)(unsafe.Pointer(&row[0])), width)
		out := dst[y*dstStride : y*dstStride+width]
		for i, v := range src {
			out[i] = v&0xFF00FF00 | (v&0xFF)<<16 | (v>>16)&0xFF
		}
	}
}
