package platforms

import (
	"image"
	"os"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

// presenter gets the frames of a window to the X server.
//
// A frame is converted from the RGBA of the canvas into an XImage kept from
// frame to frame, with a GC also kept: nothing is allocated per frame.
//
// With the MIT-SHM extension the XImage is in memory shared with the X
// server, which reads the pixels from there instead of getting them through
// the connection. Without it - a remote display (ssh -X), a server without
// the extension or in another IPC namespace, NUI_X11_SHM=0 - the XImage is
// in ordinary memory and goes with XPutImage, as before.
//
// UI thread only, like the window.
type presenter struct {
	image  uintptr // XImage*, 0 - not made yet
	gc     uintptr
	pixels []uint32 // the pixels of image, a row of width pixels after another
	// The size of image, at least the size presented
	width, height int

	// MIT-SHM only: image's fields, the segment of image and its pixels
	shmHeader *xImageHeader
	shmInfo   *xShmSegmentInfo // in C memory: Xlib keeps the pointer in the XImage
	shmData   []byte
	// The X server may still be reading the pixels of the last frame
	shmPending bool
}

// XShmSegmentInfo, field order per X11/extensions/XShm.h
type xShmSegmentInfo struct {
	ShmSeg   uintptr
	ShmID    int32
	ShmAddr  uintptr
	ReadOnly int32
}

// xImageHeader is the beginning of XImage, field order per Xlib.h
type xImageHeader struct {
	Width, Height   int32
	XOffset, Format int32
	Data            uintptr
	ByteOrder       int32
	BitmapUnit      int32
	BitmapBitOrder  int32
	BitmapPad       int32
	Depth           int32
	BytesPerLine    int32
	BitsPerPixel    int32
}

const xLSBFirst = 0

// shmState is whether MIT-SHM is used: 0 - not known yet, 1 - yes, -1 - no.
// One failure turns it off for the process: the next window would fail the
// same way.
var shmState int

func shmUsable(display uintptr) bool {
	if shmState == 0 {
		shmState = -1
		if xShmLoaded && os.Getenv("NUI_X11_SHM") != "0" && xShmQueryExtension(display) != 0 {
			shmState = 1
		}
	}
	return shmState == 1
}

// present shows the top-left width x height pixels of img in window. A frame
// that can't be shown (out of memory) is dropped: the next one may be.
func (p *presenter) present(display, window uintptr, img *image.RGBA, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	if p.gc == 0 {
		p.gc = xCreateGC(display, window, 0, 0)
	}
	if !p.ensure(display, width, height) {
		return
	}
	// The pixels are not changed while the server reads them. Usually it is
	// done long ago: the frame was painted in the meantime.
	if p.shmPending {
		xSync(display, xFalse)
		p.shmPending = false
	}
	copyRGBAToXImage(p.pixels, p.width, img, width, height)
	if p.shmInfo != nil {
		xShmPutImage(display, window, p.gc, p.image, 0, 0, 0, 0, uint32(width), uint32(height), xFalse)
		p.shmPending = true
	} else {
		xPutImage(display, window, p.gc, p.image, 0, 0, 0, 0, uint32(width), uint32(height))
	}
}

// ensure makes the XImage at least width x height. As the back buffer on
// Windows, it is made again smaller only when it is much larger than needed,
// so resizing a window doesn't make a new one on every step.
func (p *presenter) ensure(display uintptr, width, height int) bool {
	if p.image != 0 && width <= p.width && height <= p.height &&
		width*height*4 >= p.width*p.height {
		return true
	}
	p.releaseImage(display)
	if shmUsable(display) && p.makeShmImage(display, width, height) {
		return true
	}
	return p.makeImage(display, width, height)
}

// makeShmImage makes the XImage in a shared memory segment
func (p *presenter) makeShmImage(display uintptr, width, height int) bool {
	visual := xDefaultVisual(display, xDefaultScreen(display))
	info := (*xShmSegmentInfo)(libcMalloc(unsafe.Sizeof(xShmSegmentInfo{})))
	if info == nil {
		return false
	}
	*info = xShmSegmentInfo{}
	ximage := xShmCreateImage(display, visual, 24, xZPixmap, 0, unsafe.Pointer(info), uint32(width), uint32(height))
	if ximage == nil {
		libcFree(uintptr(unsafe.Pointer(info)))
		return false
	}
	header := (*xImageHeader)(ximage)
	// The pixels are written as copyRGBAToXImage writes them, as for XCreateImage
	if header.BitsPerPixel != 32 || header.ByteOrder != xLSBFirst || int(header.BytesPerLine) != width*4 {
		xDestroyImage(uintptr(ximage))
		libcFree(uintptr(unsafe.Pointer(info)))
		shmState = -1
		return false
	}

	id, err := unix.SysvShmGet(unix.IPC_PRIVATE, width*height*4, unix.IPC_CREAT|0o600)
	if err != nil {
		xDestroyImage(uintptr(ximage))
		libcFree(uintptr(unsafe.Pointer(info)))
		return false // too large for the system's limits: ordinary memory for this size
	}
	data, err := unix.SysvShmAttach(id, 0, 0)
	if err != nil {
		unix.SysvShmCtl(id, unix.IPC_RMID, nil)
		xDestroyImage(uintptr(ximage))
		libcFree(uintptr(unsafe.Pointer(info)))
		return false
	}
	info.ShmID = int32(id)
	info.ShmAddr = uintptr(unsafe.Pointer(&data[0]))
	header.Data = info.ShmAddr

	// The server attaches the segment too. It can't if it is on another
	// machine or doesn't see this one's segments: that is an X error.
	attached := trapXErrors(display, func() { xShmAttach(display, unsafe.Pointer(info)) })
	// Removed once both have detached from it, also if the process dies
	unix.SysvShmCtl(id, unix.IPC_RMID, nil)
	if !attached {
		header.Data = 0
		xDestroyImage(uintptr(ximage))
		unix.SysvShmDetach(data)
		libcFree(uintptr(unsafe.Pointer(info)))
		shmState = -1
		return false
	}

	p.image, p.shmHeader, p.shmInfo, p.shmData = uintptr(ximage), header, info, data
	p.pixels = unsafe.Slice((*uint32)(unsafe.Pointer(&data[0])), width*height)
	p.width, p.height = width, height
	return true
}

// makeImage makes the XImage in ordinary memory
func (p *presenter) makeImage(display uintptr, width, height int) bool {
	// XDestroyImage frees the pixels through libX11's own free(), so they
	// come from the same libc malloc, not Go's allocator
	buf := libcMalloc(uintptr(width * height * 4))
	if buf == nil {
		return false
	}
	ximage := xCreateImage(display, xDefaultVisual(display, xDefaultScreen(display)), 24, xZPixmap, 0,
		buf, uint32(width), uint32(height), 32, 0)
	if ximage == 0 {
		libcFree(uintptr(buf))
		return false
	}
	p.image = ximage
	p.pixels = unsafe.Slice((*uint32)(buf), width*height)
	p.width, p.height = width, height
	return true
}

// releaseImage frees the XImage and its pixels
func (p *presenter) releaseImage(display uintptr) {
	if p.image == 0 {
		return
	}
	if p.shmInfo != nil {
		// The server detaches after the requests sent before, the last frame
		// included: the segment is gone once both have detached
		xShmDetach(display, unsafe.Pointer(p.shmInfo))
		// The pixels are not libc's: XDestroyImage must not free them
		p.shmHeader.Data = 0
		xDestroyImage(p.image)
		unix.SysvShmDetach(p.shmData)
		libcFree(uintptr(unsafe.Pointer(p.shmInfo)))
	} else {
		destroyXImage(p.image)
	}
	p.image, p.pixels, p.width, p.height = 0, nil, 0, 0
	p.shmHeader, p.shmInfo, p.shmData, p.shmPending = nil, nil, nil, false
}

// release frees everything, before the window is destroyed
func (p *presenter) release(display uintptr) {
	p.releaseImage(display)
	if p.gc != 0 {
		xFreeGC(display, p.gc)
		p.gc = 0
	}
}

// A frame of at least convertParallelPixels is converted in bands by up to
// convertMaxBands goroutines: one core doesn't use all the memory bandwidth
// the conversion is bound by. More than 3-4 gain nothing.
const (
	convertParallelPixels = 512 * 512
	convertMaxBands       = 4
)

// copyRGBAToXImage copies the top-left width x height pixels of img to dst,
// whose rows are dstStride pixels long, converting RGBA to the BGRA of the X
// image. It returns once all of it is copied.
func copyRGBAToXImage(dst []uint32, dstStride int, img *image.RGBA, width, height int) {
	bands := min(runtime.GOMAXPROCS(0), convertMaxBands, height)
	if width*height < convertParallelPixels || bands < 2 {
		copyRowsToXImage(dst, dstStride, img, width, 0, height)
		return
	}
	var wg sync.WaitGroup
	bandHeight := (height + bands - 1) / bands
	for y := 0; y < height; y += bandHeight {
		wg.Go(func() {
			copyRowsToXImage(dst, dstStride, img, width, y, min(y+bandHeight, height))
		})
	}
	wg.Wait()
}

// copyRowsToXImage is copyRGBAToXImage of the rows from y0 to y1. As little-
// endian integers, as all the Linux targets are: two pixels at a time as an
// uint64 where both rows are 8-byte aligned, else a pixel at a time.
func copyRowsToXImage(dst []uint32, dstStride int, img *image.RGBA, width, y0, y1 int) {
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
