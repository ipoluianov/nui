package platforms

import (
	"math"
	"syscall"
	"unsafe"
)

// HiDPI on Windows: the process declares itself per-monitor DPI aware, so
// Windows doesn't stretch the windows' bitmaps (which blurs them) on a
// screen scaled to 125%, 150%... The windows get their real pixels instead,
// and everything given to nui - sizes, positions, mouse coordinates - is
// converted to logical pixels: physical ones divided by the scale (the
// screen's DPI / 96). At 100% the scale is 1 and nothing changes.

var (
	shcore = syscall.NewLazyDLL("shcore.dll")

	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDpiAwareness        = shcore.NewProc("SetProcessDpiAwareness")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	procGetDpiForSystem               = user32.NewProc("GetDpiForSystem")
	procGetDC                         = user32.NewProc("GetDC")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
)

const (
	c_WM_DPICHANGED = 0x02E0

	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	c_DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = ^uintptr(3)
	c_PROCESS_PER_MONITOR_DPI_AWARE              = 2
)

func init() {
	enableDpiAwareness()
}

// enableDpiAwareness makes the process DPI aware, the best way the system
// supports: per monitor v2 (Windows 10 1703+; the title bars scale too),
// per monitor (8.1) or system-wide (Vista). It must run before any window
// is created.
func enableDpiAwareness() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		if ok, _, _ := procSetProcessDpiAwarenessContext.Call(c_DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2); ok != 0 {
			return
		}
	}
	if procSetProcessDpiAwareness.Find() == nil {
		if hr, _, _ := procSetProcessDpiAwareness.Call(c_PROCESS_PER_MONITOR_DPI_AWARE); hr == 0 {
			return
		}
	}
	if procSetProcessDPIAware.Find() == nil {
		procSetProcessDPIAware.Call()
	}
}

// dpiScale converts a DPI to the scale: 96 DPI is 100%
func dpiScale(dpi uintptr) float64 {
	if dpi == 0 {
		return 1
	}
	return float64(dpi) / 96
}

// systemScale returns the scale of the primary screen, where new windows open
func systemScale() float64 {
	if procGetDpiForSystem.Find() == nil {
		dpi, _, _ := procGetDpiForSystem.Call()
		return dpiScale(dpi)
	}
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return 1
	}
	defer procReleaseDC.Call(0, hdc)
	dpi, _, _ := procGetDeviceCaps.Call(hdc, c_LOGPIXELSX)
	return dpiScale(dpi)
}

// hwndScale returns the scale of the screen the window is on
func hwndScale(hwnd uintptr) float64 {
	if procGetDpiForWindow.Find() == nil {
		if dpi, _, _ := procGetDpiForWindow.Call(hwnd); dpi != 0 {
			return dpiScale(dpi)
		}
	}
	return systemScale()
}

// toPhysical converts logical pixels to the window's physical ones
func (c *nativeWindow) toPhysical(v int) int {
	return scaleToPhysical(v, c.Scale())
}

// toLogical converts a physical position (e.g. of the mouse) to the logical
// pixel it's in
func (c *nativeWindow) toLogical(v int) int {
	return scaleToLogical(v, c.Scale())
}

// toLogicalSize converts a physical size to a logical one that covers it whole
func (c *nativeWindow) toLogicalSize(v int) int {
	s := c.Scale()
	if s == 1 {
		return v
	}
	return int(math.Ceil(float64(v)/s - 1e-9))
}

// toLogicalScreen converts a screen coordinate to the logical one that
// toPhysical turns back into it
func (c *nativeWindow) toLogicalScreen(v int) int {
	s := c.Scale()
	if s == 1 {
		return v
	}
	return int(math.Round(float64(v) / s))
}

func scaleToPhysical(v int, s float64) int {
	if s == 1 {
		return v
	}
	return int(math.Round(float64(v) * s))
}

func scaleToLogical(v int, s float64) int {
	if s == 1 {
		return v
	}
	return int(math.Floor(float64(v) / s))
}

// dpiChanged handles WM_DPICHANGED: the window moved to a screen of another
// scale. Windows suggests the rectangle that keeps its logical size there.
func (c *nativeWindow) dpiChanged(wParam, lParam uintptr) {
	c.scale = dpiScale(wParam & 0xFFFF)
	if lParam != 0 {
		r := (*rect)(unsafe.Pointer(lParam))
		procSetWindowPos.Call(uintptr(c.hwnd), 0,
			uintptr(r.left), uintptr(r.top), uintptr(r.right-r.left), uintptr(r.bottom-r.top),
			c_SWP_NOZORDER|c_SWP_NOACTIVATE)
	}
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
}
