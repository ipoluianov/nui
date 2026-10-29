//go:build windows

package platforms

import (
	"syscall"
	"unsafe"
)

// The input method (Chinese, Japanese, Korean) on Windows: DefWindowProc
// shows the window of the composition and the candidates; nui tells it
// where the caret is, else the windows open in a corner of the screen.

var (
	imm32 = syscall.NewLazyDLL("imm32.dll")

	procImmGetContext           = imm32.NewProc("ImmGetContext")
	procImmReleaseContext       = imm32.NewProc("ImmReleaseContext")
	procImmSetCompositionWindow = imm32.NewProc("ImmSetCompositionWindow")
	procImmSetCandidateWindow   = imm32.NewProc("ImmSetCandidateWindow")
)

const (
	c_WM_IME_STARTCOMPOSITION = 0x010D

	c_CFS_POINT   = 0x0002
	c_CFS_EXCLUDE = 0x0080
)

type imePoint struct{ x, y int32 }

type imeRect struct{ left, top, right, bottom int32 }

// COMPOSITIONFORM
type compositionForm struct {
	style uint32
	pos   imePoint
	area  imeRect
}

// CANDIDATEFORM
type candidateForm struct {
	index uint32
	style uint32
	pos   imePoint
	area  imeRect
}

// SetTextInputRect keeps the caret and moves the input method's windows
// there, if a composition is going on
// SetTextInputActive: the input method of Windows is the system's own, on
// in every window
func (c *nativeWindow) SetTextInputActive(active bool) {}

func (c *nativeWindow) SetTextInputRect(x, y, width, height int) {
	c.platform.textInputRect = [4]int{x, y, width, height}
	c.applyTextInputRect()
}

// applyTextInputRect places the composition at the caret and the
// candidates under it, not over the caret's line
func (c *nativeWindow) applyTextInputRect() {
	if procImmGetContext.Find() != nil || c.hwnd == 0 {
		return
	}
	himc, _, _ := procImmGetContext.Call(uintptr(c.hwnd))
	if himc == 0 {
		return
	}
	defer procImmReleaseContext.Call(uintptr(c.hwnd), himc)
	r := c.platform.textInputRect
	x, y := int32(c.toPhysical(r[0])), int32(c.toPhysical(r[1]))
	right, bottom := int32(c.toPhysical(r[0]+r[2])), int32(c.toPhysical(r[1]+r[3]))
	comp := compositionForm{style: c_CFS_POINT, pos: imePoint{x, y}}
	procImmSetCompositionWindow.Call(himc, uintptr(unsafe.Pointer(&comp)))
	cand := candidateForm{style: c_CFS_EXCLUDE, pos: imePoint{x, bottom}, area: imeRect{x, y, right, bottom}}
	procImmSetCandidateWindow.Call(himc, uintptr(unsafe.Pointer(&cand)))
}
