package nui

/*
#include "window.h"
*/
import "C"
import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
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
func go_on_paint(ptr unsafe.Pointer, width C.int, height C.int) {
	// read the embedded png image

	imgTest := GetRGBATestImage()

	fmt.Println("imgTest", imgTest.Bounds())

	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), int(width*height*4)),
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	draw.Draw(img, img.Rect, imgTest, image.Point{0, 0}, draw.Src)
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
