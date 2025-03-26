package nui

/*
#include "window.h"
*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"time"
	"unsafe"
)

var rgbaTest *image.RGBA

func GetRGBATestImage() *image.RGBA {
	if rgbaTest != nil {
		return rgbaTest
	}
	rgba, err := loadPngFromBytes(TestPng)
	if err != nil {
		panic(err)
	}
	rgbaTest = rgba
	return rgba
}

func loadPngFromBytes(bs []byte) (*image.RGBA, error) {
	img, err := png.Decode(bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}

	rgba := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return rgba, nil
}

//export go_on_paint
func go_on_paint(hwnd C.int, ptr unsafe.Pointer, width C.int, height C.int) {
	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), int(width*height*4)),
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	imgDataSize := img.Rect.Dx() * img.Rect.Dy() * 4
	copy(img.Pix[:imgDataSize], canvasBufferBackground)

	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnPaint != nil {
			win.OnPaint(img)
		}
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
	key := Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnKeyDown != nil {
			win.OnKeyDown(key)
		}
	}
}

//export go_on_key_up
func go_on_key_up(hwnd C.int, code C.int) {
	key := Key(ConvertMacOSKeyToNuiKey(int(code)))
	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnKeyUp != nil {
			win.OnKeyUp(key)
		}
	}
}

//export go_on_modifier_change
func go_on_modifier_change(hwnd C.int, shift, ctrl, alt, cmd C.int) {
	fmt.Printf("Modifiers: Shift=%v Ctrl=%v Alt=%v Cmd=%v\n", shift != 0, ctrl != 0, alt != 0, cmd != 0)
}

//export go_on_char
func go_on_char(hwnd C.int, codepoint C.int) {
	//fmt.Printf("Char typed: '%c' (U+%04X)\n", rune(codepoint), codepoint)
	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnChar != nil {
			win.OnChar(rune(codepoint))
		}
	}
}

//export go_on_mouse_down
func go_on_mouse_down(hwnd C.int, button, x, y C.int) {
	fmt.Printf("Mouse down: button=%d at (%d,%d)\n", button, x, y)
	if win, ok := hwnds[int(hwnd)]; ok {
		if button == 0 {
			if win.OnMouseDownLeftButton != nil {
				win.OnMouseDownLeftButton(int(x), int(y))
			}
		} else if button == 1 {
			if win.OnMouseDownRightButton != nil {
				win.OnMouseDownRightButton(int(x), int(y))
			}
		} else if button == 2 {
			if win.OnMouseDownMiddleButton != nil {
				win.OnMouseDownMiddleButton(int(x), int(y))
			}
		}
	}
}

//export go_on_mouse_up
func go_on_mouse_up(hwnd C.int, button, x, y C.int) {
	//fmt.Printf("Mouse up: button=%d at (%d,%d)\n", button, x, y)
	if win, ok := hwnds[int(hwnd)]; ok {
		if button == 0 {
			if win.OnMouseUpLeftButton != nil {
				win.OnMouseUpLeftButton(int(x), int(y))
			}
		} else if button == 1 {
			if win.OnMouseUpRightButton != nil {
				win.OnMouseUpRightButton(int(x), int(y))
			}
		} else if button == 2 {
			if win.OnMouseUpMiddleButton != nil {
				win.OnMouseUpMiddleButton(int(x), int(y))
			}
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
	dt := time.Now()
	dtStr := dt.Format("15:04:05.000")

	fmt.Println("Scroll: delta=", dtStr, deltaX, deltaY)
	deltaX = deltaX * 2
	deltaY = deltaY * 2

	if win, ok := hwnds[int(hwnd)]; ok {
		if win.OnMouseWheel != nil {
			win.OnMouseWheel(float64(deltaX), float64(deltaY))
		}
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
	//fmt.Printf("Mouse double click: button=%d at (%d,%d)\n", button, x, y)
	if win, ok := hwnds[int(hwnd)]; ok {
		if button == 0 {
			if win.OnMouseDoubleClickLeftButton != nil {
				win.OnMouseDoubleClickLeftButton(int(x), int(y))
			}
		} else if button == 1 {
			if win.OnMouseDoubleClickRightButton != nil {
				win.OnMouseDoubleClickRightButton(int(x), int(y))
			}
		} else if button == 2 {
			if win.OnMouseDoubleClickMiddleButton != nil {
				win.OnMouseDoubleClickMiddleButton(int(x), int(y))
			}
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
