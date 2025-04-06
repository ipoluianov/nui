package nui

/*
#include "window.h"
*/
import "C"
import (
	"fmt"
	"image"
	"strconv"
	"time"
	"unsafe"
)

//export go_on_paint
func go_on_paint(hwnd C.int, ptr unsafe.Pointer, width C.int, height C.int) {
	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), int(width*height*4)),
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowPaint(img)
	}
}

//export go_on_resize
func go_on_resize(windowId C.int, width C.int, height C.int) {
	if win, ok := hwnds[int(windowId)]; ok {
		win.windowResized(int(width), int(height))
	}
}

//export go_on_key_down
func go_on_key_down(hwnd C.int, code C.int) {
	fmt.Println("Key down", strconv.FormatInt(int64(code), 16))
	key := Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowKeyDown(key)
	}
}

//export go_on_key_up
func go_on_key_up(hwnd C.int, code C.int) {
	key := Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowKeyUp(key)
	}
}

//export go_on_modifier_change
func go_on_modifier_change(hwnd C.int, shift, ctrl, alt, cmd, caps, num, fnKey C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowKeyModifiersChanged(shift != 0, ctrl != 0, alt != 0, cmd != 0, caps != 0, num != 0, fnKey != 0)
	}
}

//export go_on_char
func go_on_char(hwnd C.int, codepoint C.int) {
	//fmt.Printf("Char typed: '%c' (U+%04X)\n", rune(codepoint), codepoint)
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowChar(rune(codepoint))
	}
}

func convertMacMouseButtons(button C.int) MouseButton {
	switch button {
	case 0:
		return MouseButtonLeft
	case 1:
		return MouseButtonRight
	case 2:
		return MouseButtonMiddle
	}
	return MouseButtonLeft
}

//export go_on_window_move
func go_on_window_move(hwnd C.int, x C.int, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowMoved(int(x), int(y))
	}
}

//export go_on_declare_draw_time
func go_on_declare_draw_time(hwnd C.int, dt C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowDeclareDrawTime(int(dt))
	}
}

//export go_on_mouse_down
func go_on_mouse_down(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDown(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

//export go_on_mouse_up
func go_on_mouse_up(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonUp(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

//export go_on_mouse_move
func go_on_mouse_move(hwnd C.int, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowMouseMove(int(x), int(y))
	}
}

//export go_on_mouse_scroll
func go_on_mouse_scroll(hwnd C.int, deltaX C.float, deltaY C.float) {
	if win, ok := hwnds[int(hwnd)]; ok {
		win.windowMouseWheel(float64(deltaX), float64(deltaY))
	}
}

//export go_on_mouse_enter
func go_on_mouse_enter(hwnd C.int) {
	//fmt.Println("Mouse entered")
	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnMouseEnter != nil {
			win.OnMouseEnter()
		}
	}
}

//export go_on_mouse_leave
func go_on_mouse_leave(hwnd C.int) {
	//fmt.Println("Mouse left")
	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnMouseLeave != nil {
			win.OnMouseLeave()
		}
	}
}

//export go_on_mouse_double_click
func go_on_mouse_double_click(hwnd C.int, button, x, y C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDblClick(convertMacMouseButtons(button), int(x), int(y))
		}
	}
}

var dtLastTimer = time.Now()

//export go_on_timer
func go_on_timer(hwnd C.int) {
	if win, ok := hwnds[int(hwnd)]; ok {
		dtNow := time.Now()
		dtDiff := dtNow.Sub(dtLastTimer)
		if dtDiff < time.Millisecond*50 {
			return
		}
		dtLastTimer = dtNow
		if win.OnTimer != nil {
			win.OnTimer()
		}
	}
}
