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
	"unsafe"
)

func GetRGBATestImage() *image.RGBA {
	rgba, err := loadPngFromBytes(TestPng)
	if err != nil {
		panic(err)
	}
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
func go_on_paint(ptr unsafe.Pointer, width C.int, height C.int, hwnd C.int) {
	// read the embedded png image

	imgTest := GetRGBATestImage()

	fmt.Println("imgTest", imgTest.Bounds())

	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), int(width*height*4)),
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	if win, ok := hwnds[int(hwnd)]; ok {
		win.OnPaint(img)
	}

	//mainNativeWindow.OnPaint(img)

	//draw.Draw(img, img.Rect, imgTest, image.Point{0, 0}, draw.Src)
	_ = imgTest

	/*for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			if x > 100 && x < 200 && y > 100 && y < 200 {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				img.Set(x, y, color.RGBA{0, 255, 0, 255})
			}
		}
	}*/
}

//export go_on_key_down
func go_on_key_down(code C.int) {
	fmt.Println("Key down:", code)
	for _, win := range hwnds {
		if win.OnKeyDown != nil {
			win.OnKeyDown(Key(code))
		}
	}
}

//export go_on_key_up
func go_on_key_up(code C.int) {
	fmt.Println("Key up:", code)
}

//export go_on_modifier_change
func go_on_modifier_change(shift, ctrl, alt, cmd C.int) {
	fmt.Printf("Modifiers: Shift=%v Ctrl=%v Alt=%v Cmd=%v\n", shift != 0, ctrl != 0, alt != 0, cmd != 0)
}

//export go_on_char
func go_on_char(codepoint C.int) {
	fmt.Printf("Char typed: '%c' (U+%04X)\n", rune(codepoint), codepoint)
}

//export go_on_mouse_down
func go_on_mouse_down(button, x, y C.int) {
	fmt.Printf("Mouse down: button=%d at (%d,%d)\n", button, x, y)
}

//export go_on_mouse_up
func go_on_mouse_up(button, x, y C.int) {
	fmt.Printf("Mouse up: button=%d at (%d,%d)\n", button, x, y)
}

//export go_on_mouse_move
func go_on_mouse_move(x, y C.int) {
	fmt.Printf("Mouse move: (%d,%d)\n", x, y)
}

//export go_on_mouse_scroll
func go_on_mouse_scroll(delta C.int) {
	fmt.Printf("Scroll: delta=%d\n", delta)
}

//export go_on_mouse_enter
func go_on_mouse_enter() {
	fmt.Println("Mouse entered")
}

//export go_on_mouse_leave
func go_on_mouse_leave() {
	fmt.Println("Mouse left")
}

//export go_on_mouse_double_click
func go_on_mouse_double_click(button, x, y C.int) {
	fmt.Printf("Mouse double click: button=%d at (%d,%d)\n", button, x, y)
}
