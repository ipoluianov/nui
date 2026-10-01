package platforms

import (
	"fmt"
	"math"
	"syscall"
	"unsafe"
)

// Printing on Windows: the system print dialog (PrintDlgW) gives a device
// context of the chosen printer with the user's settings (paper, orientation,
// copies); each page is drawn at up to MaxDPI and stretched onto the page.

var (
	procPrintDlgW         = comdlg32.NewProc("PrintDlgW")
	procStartDocW         = gdi32.NewProc("StartDocW")
	procEndDoc            = gdi32.NewProc("EndDoc")
	procAbortDoc          = gdi32.NewProc("AbortDoc")
	procStartPage         = gdi32.NewProc("StartPage")
	procEndPage           = gdi32.NewProc("EndPage")
	procGetDeviceCaps     = gdi32.NewProc("GetDeviceCaps")
	procStretchDIBits     = gdi32.NewProc("StretchDIBits")
	procSetStretchBltMode = gdi32.NewProc("SetStretchBltMode")
	procSetBrushOrgEx     = gdi32.NewProc("SetBrushOrgEx")
	procGlobalFree        = kernel32.NewProc("GlobalFree")
)

const (
	c_PD_RETURNDC                   = 0x00000100
	c_PD_NOSELECTION                = 0x00000004
	c_PD_PAGENUMS                   = 0x00000002
	c_PD_USEDEVMODECOPIESANDCOLLATE = 0x00040000

	c_LOGPIXELSX = 88
	c_LOGPIXELSY = 90

	c_HALFTONE       = 4
	c_DIB_RGB_COLORS = 0
	c_SRCCOPY        = 0x00CC0020
)

// PRINTDLGW on 64-bit Windows
type printDlgW struct {
	lStructSize         uint32
	hwndOwner           uintptr
	hDevMode            uintptr
	hDevNames           uintptr
	hDC                 uintptr
	flags               uint32
	nFromPage           uint16
	nToPage             uint16
	nMinPage            uint16
	nMaxPage            uint16
	nCopies             uint16
	hInstance           uintptr
	lCustData           uintptr
	lpfnPrintHook       uintptr
	lpfnSetupHook       uintptr
	lpPrintTemplateName uintptr
	lpSetupTemplateName uintptr
	hPrintTemplate      uintptr
	hSetupTemplate      uintptr
}

type docInfoW struct {
	cbSize       int32
	lpszDocName  *uint16
	lpszOutput   *uint16
	lpszDatatype *uint16
	fwType       uint32
}

// HasPrintDialog reports whether the system has a print dialog of its own
func HasPrintDialog() bool {
	return true
}

// PrintWithDialog shows the system print dialog and prints the document on
// the printer the user chose. UI thread only. ErrPrintCanceled if canceled.
func PrintWithDialog(owner Window, req PrintRequest) error {
	pd := printDlgW{
		flags:     c_PD_RETURNDC | c_PD_NOSELECTION | c_PD_USEDEVMODECOPIESANDCOLLATE,
		nFromPage: 1, nToPage: 1, nMinPage: 1, nMaxPage: 9999, nCopies: 1,
	}
	pd.lStructSize = uint32(unsafe.Sizeof(pd))
	if w, ok := owner.(*nativeWindow); ok && w != nil {
		pd.hwndOwner = uintptr(w.hwnd)
	}
	if ret, _, _ := procPrintDlgW.Call(uintptr(unsafe.Pointer(&pd))); ret == 0 {
		if code, _, _ := procCommDlgExtendedError.Call(); code != 0 {
			return fmt.Errorf("nui: print dialog error 0x%x", code)
		}
		return ErrPrintCanceled
	}
	defer func() {
		if pd.hDevMode != 0 {
			procGlobalFree.Call(pd.hDevMode)
		}
		if pd.hDevNames != 0 {
			procGlobalFree.Call(pd.hDevNames)
		}
	}()
	hdc := pd.hDC
	if hdc == 0 {
		return fmt.Errorf("%w: no printer device context", ErrNoPrinting)
	}
	defer procDeleteDC.Call(hdc)

	pageW, _, _ := procGetDeviceCaps.Call(hdc, c_HORZRES)
	pageH, _, _ := procGetDeviceCaps.Call(hdc, c_VERTRES)
	dpiX, _, _ := procGetDeviceCaps.Call(hdc, c_LOGPIXELSX)
	if pageW == 0 || pageH == 0 || dpiX == 0 {
		return fmt.Errorf("%w: the printer reports no page size", ErrNoPrinting)
	}
	// The pages are drawn at up to MaxDPI and stretched to the printer's
	scale := 1.0
	if req.MaxDPI > 0 && float64(dpiX) > req.MaxDPI {
		scale = req.MaxDPI / float64(dpiX)
	}
	w := int(math.Round(float64(pageW) * scale))
	h := int(math.Round(float64(pageH) * scale))
	dpi := float64(dpiX) * scale

	count := req.PageCount(w, h, dpi)
	from, to := 1, count
	if pd.flags&c_PD_PAGENUMS != 0 {
		from, to = max(int(pd.nFromPage), 1), min(int(pd.nToPage), count)
	}
	if from > to {
		return nil
	}

	title, _ := syscall.UTF16PtrFromString(req.Title)
	di := docInfoW{lpszDocName: title}
	di.cbSize = int32(unsafe.Sizeof(di))
	if job, _, _ := procStartDocW.Call(hdc, uintptr(unsafe.Pointer(&di))); int32(job) <= 0 {
		return fmt.Errorf("%w: StartDoc failed", ErrNoPrinting)
	}
	procSetStretchBltMode.Call(hdc, c_HALFTONE)
	procSetBrushOrgEx.Call(hdc, 0, 0, 0)

	for page := from - 1; page < to; page++ {
		if r, _, _ := procStartPage.Call(hdc); int32(r) <= 0 {
			procAbortDoc.Call(hdc)
			return fmt.Errorf("%w: StartPage failed", ErrNoPrinting)
		}
		img := req.Render(page, w, h, dpi)
		bgra := make([]byte, w*h*4)
		for y := 0; y < h; y++ {
			src := img.Pix[y*img.Stride : y*img.Stride+w*4]
			dst := bgra[y*w*4 : (y+1)*w*4]
			for x := 0; x < w*4; x += 4 {
				dst[x], dst[x+1], dst[x+2] = src[x+2], src[x+1], src[x]
			}
		}
		bi := bitmapInfoHeader{biWidth: int32(w), biHeight: -int32(h), biPlanes: 1, biBitCount: 32}
		bi.biSize = uint32(unsafe.Sizeof(bi))
		procStretchDIBits.Call(hdc, 0, 0, pageW, pageH, 0, 0, uintptr(w), uintptr(h),
			uintptr(unsafe.Pointer(&bgra[0])), uintptr(unsafe.Pointer(&bi)), c_DIB_RGB_COLORS, c_SRCCOPY)
		if r, _, _ := procEndPage.Call(hdc); int32(r) <= 0 {
			procAbortDoc.Call(hdc)
			return fmt.Errorf("%w: EndPage failed", ErrNoPrinting)
		}
	}
	procEndDoc.Call(hdc)
	return nil
}

// Printers, PrintFile: CUPS only (see print_cups.go)
func Printers() (names []string, defaultName string, err error) {
	return nil, "", ErrNoPrinting
}

func PrintFile(printer, path string, copies int, title string, options map[string]string) error {
	return ErrNoPrinting
}
