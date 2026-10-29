//go:build linux || windows

package platforms

import (
	"image"
	"runtime"
	"sync"
	"unsafe"
)

// A frame of at least convertParallelPixels is converted in bands by up to
// convertMaxBands goroutines: one core doesn't use all the memory bandwidth
// the conversion is bound by. More than 3-4 gain nothing.
const (
	convertParallelPixels = 512 * 512
	convertMaxBands       = 4
)

// copyRGBAToBGRA copies the top-left width x height pixels of img to dst,
// whose rows are dstStride pixels long, converting RGBA to BGRA: the pixels
// of an XImage and of a Windows DIB. It returns once all of it is copied.
func copyRGBAToBGRA(dst []uint32, dstStride int, img *image.RGBA, width, height int) {
	bands := min(runtime.GOMAXPROCS(0), convertMaxBands, height)
	if width*height < convertParallelPixels || bands < 2 {
		copyRowsToBGRA(dst, dstStride, img, width, 0, height)
		return
	}
	var wg sync.WaitGroup
	bandHeight := (height + bands - 1) / bands
	for y := 0; y < height; y += bandHeight {
		wg.Go(func() {
			copyRowsToBGRA(dst, dstStride, img, width, y, min(y+bandHeight, height))
		})
	}
	wg.Wait()
}

// copyRowsToBGRA is copyRGBAToBGRA of the rows from y0 to y1. As little-
// endian integers, as all the Linux and Windows targets are: two pixels at a
// time as an uint64 where both rows are 8-byte aligned, else a pixel at a
// time.
func copyRowsToBGRA(dst []uint32, dstStride int, img *image.RGBA, width, y0, y1 int) {
	for y := y0; y < y1; y++ {
		row := img.Pix[y*img.Stride : y*img.Stride+width*4]
		src := unsafe.Slice((*uint32)(unsafe.Pointer(&row[0])), width)
		out := dst[y*dstStride : y*dstStride+width]
		i := 0
		if (uintptr(unsafe.Pointer(&src[0]))|uintptr(unsafe.Pointer(&out[0])))%8 == 0 {
			pairs := width / 2
			src2 := unsafe.Slice((*uint64)(unsafe.Pointer(&src[0])), pairs)
			out2 := unsafe.Slice((*uint64)(unsafe.Pointer(&out[0])), pairs)
			for j, v := range src2 {
				out2[j] = v&0xFF00FF00FF00FF00 | (v&0x000000FF000000FF)<<16 | (v>>16)&0x000000FF000000FF
			}
			i = pairs * 2
		}
		for ; i < width; i++ {
			v := src[i]
			out[i] = v&0xFF00FF00 | (v&0xFF)<<16 | (v>>16)&0xFF
		}
	}
}
