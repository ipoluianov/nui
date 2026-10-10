package platforms

// Pure-Go Cocoa/AppKit/CoreGraphics bindings for native_window_darwin.go,
// utils_darwin.go and filedialog_darwin.go, loaded at runtime via purego
// (dlopen/dlsym) and purego/objc (Objective-C runtime message sends, block
// creation, and runtime class registration) instead of cgo. This lets the
// package cross-compile with CGO_ENABLED=0 (no C compiler or Xcode command
// line tools needed at build time); AppKit itself still has to be present on
// the machine that actually runs the resulting binary, same as before (it's
// part of every macOS install). This is the darwin equivalent of
// xlib_linux.go, and replaces what used to be window.h/window.m.
//
// Two Objective-C classes are registered at runtime with objc.RegisterClass,
// replacing window.m's GoPaintView/AppDelegate: NUIPaintView (an NSView
// subclass hosting the software framebuffer and all input) and
// NUIAppDelegate (NSApplicationDelegate + NSWindowDelegate). Their methods
// are plain Go closures - since there's no cgo boundary anymore, they call
// straight into utils_darwin.go's go_on_* functions, no //export bridge
// needed.
//
// Object lifetime: this file follows plain MRC (manual retain/release, no
// ARC - purego doesn't have a compiler pass to insert retain/release calls).
// Anything alloc/init'd and kept in one of the cocoa* maps below owns the
// +1 reference that alloc/init returns; that reference is released exactly
// once, in nuiWindowWillClose/stopTimer/updateTrackingAreas, to avoid both
// leaks and over-release crashes. Autoreleased convenience-constructor
// objects (openPanel, arrowCursor, stringWithUTF8String:, ...) need no
// explicit handling since we never own them.
//
// Struct layouts (nsPoint/nsSize/nsRect) mirror NSGeometry.h/CGGeometry.h
// field-for-field (both NSRect and CGRect are the same layout on 64-bit),
// matching the same "field order must match the C ABI" rule xlib_linux.go's
// header comment calls out for X11 structs - purego supports struct-by-value
// args/returns on darwin amd64/arm64 as long as the Go struct mirrors it.

import (
	"math"
	"runtime"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

/////////////////////////////////////////////////////
// Geometry

type nsPoint struct{ X, Y float64 }
type nsSize struct{ Width, Height float64 }
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

func nsRectIntersection(a, b nsRect) nsRect {
	x1 := math.Max(a.Origin.X, b.Origin.X)
	y1 := math.Max(a.Origin.Y, b.Origin.Y)
	x2 := math.Min(a.Origin.X+a.Size.Width, b.Origin.X+b.Size.Width)
	y2 := math.Min(a.Origin.Y+a.Size.Height, b.Origin.Y+b.Size.Height)
	if x2 < x1 || y2 < y1 {
		return nsRect{}
	}
	return nsRect{nsPoint{x1, y1}, nsSize{x2 - x1, y2 - y1}}
}

/////////////////////////////////////////////////////
// Constants (AppKit/CoreGraphics headers)

const (
	nsApplicationActivationPolicyRegular = 0

	nsWindowStyleMaskTitled         = 1 << 0
	nsWindowStyleMaskClosable       = 1 << 1
	nsWindowStyleMaskMiniaturizable = 1 << 2
	nsWindowStyleMaskResizable      = 1 << 3

	nsBackingStoreBuffered = 2

	nsEventModifierFlagCapsLock   = 1 << 16
	nsEventModifierFlagShift      = 1 << 17
	nsEventModifierFlagControl    = 1 << 18
	nsEventModifierFlagOption     = 1 << 19
	nsEventModifierFlagCommand    = 1 << 20
	nsEventModifierFlagNumericPad = 1 << 21
	nsEventModifierFlagFunction   = 1 << 23

	nsTrackingMouseEnteredAndExited = 0x01
	nsTrackingMouseMoved            = 0x02
	nsTrackingActiveAlways          = 0x80
	nsTrackingInVisibleRect         = 0x200

	nsWindowZoomButton = 2

	nsModalResponseOK = 1

	nsDragOperationNone = 0
	nsDragOperationCopy = 1

	cgImageAlphaPremultipliedLast = 1
	cgBitmapByteOrder32Big        = 4 << 12
	cgRenderingIntentDefault      = 0
)

/////////////////////////////////////////////////////
// Selectors (safe as package-level var initializers: RegisterName only
// needs libobjc, which the objc package itself loads on import).

var (
	selAlloc   = objc.RegisterName("alloc")
	selInit    = objc.RegisterName("init")
	selRetain  = objc.RegisterName("retain")
	selRelease = objc.RegisterName("release")

	selSharedApplication         = objc.RegisterName("sharedApplication")
	selSetActivationPolicy       = objc.RegisterName("setActivationPolicy:")
	selSetDelegate               = objc.RegisterName("setDelegate:")
	selActivateIgnoringOtherApps = objc.RegisterName("activateIgnoringOtherApps:")
	selRun                       = objc.RegisterName("run")
	selTerminate                 = objc.RegisterName("terminate:")
	selModalWindow               = objc.RegisterName("modalWindow")
	selStopModal                 = objc.RegisterName("stopModal")
	selRunModalForWindow         = objc.RegisterName("runModalForWindow:")
	selSetApplicationIconImage   = objc.RegisterName("setApplicationIconImage:")

	selInitWithContentRectStyleMaskBackingDefer = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
	selSetContentView                           = objc.RegisterName("setContentView:")
	selContentView                              = objc.RegisterName("contentView")
	selSetTitle                                 = objc.RegisterName("setTitle:")
	selWindowNumber                             = objc.RegisterName("windowNumber")
	selMakeKeyAndOrderFront                     = objc.RegisterName("makeKeyAndOrderFront:")
	selClose                                    = objc.RegisterName("close")
	selFrame                                    = objc.RegisterName("frame")
	selSetFrameDisplayAnimate                   = objc.RegisterName("setFrame:display:animate:")
	selScreen                                   = objc.RegisterName("screen")
	selContentLayoutRect                        = objc.RegisterName("contentLayoutRect")
	selSetContentSize                           = objc.RegisterName("setContentSize:")
	selSetContentMinSize                        = objc.RegisterName("setContentMinSize:")
	selMiniaturize                              = objc.RegisterName("miniaturize:")
	selZoom                                     = objc.RegisterName("zoom:")
	selIsZoomed                                 = objc.RegisterName("isZoomed")
	selIsKeyWindow                              = objc.RegisterName("isKeyWindow")
	selStyleMask                                = objc.RegisterName("styleMask")
	selSetStyleMask                             = objc.RegisterName("setStyleMask:")
	selStandardWindowButton                     = objc.RegisterName("standardWindowButton:")
	selSetHidden                                = objc.RegisterName("setHidden:")

	selInitWithFrame                    = objc.RegisterName("initWithFrame:")
	selSetFrameSize                     = objc.RegisterName("setFrameSize:")
	selBounds                           = objc.RegisterName("bounds")
	selWindow                           = objc.RegisterName("window")
	selConvertRectFromView              = objc.RegisterName("convertRect:fromView:")
	selConvertPointFromView             = objc.RegisterName("convertPoint:fromView:")
	selAddTrackingArea                  = objc.RegisterName("addTrackingArea:")
	selRemoveTrackingArea               = objc.RegisterName("removeTrackingArea:")
	selInitWithRectOptionsOwnerUserInfo = objc.RegisterName("initWithRect:options:owner:userInfo:")
	selSetNeedsDisplay                  = objc.RegisterName("setNeedsDisplay:")

	selCharacters       = objc.RegisterName("characters")
	selLength           = objc.RegisterName("length")
	selCharacterAtIndex = objc.RegisterName("characterAtIndex:")
	selKeyCode          = objc.RegisterName("keyCode")
	selModifierFlags    = objc.RegisterName("modifierFlags")
	selClickCount       = objc.RegisterName("clickCount")
	selLocationInWindow = objc.RegisterName("locationInWindow")
	selDeltaX           = objc.RegisterName("deltaX")
	selDeltaY           = objc.RegisterName("deltaY")
	selObject           = objc.RegisterName("object")

	selStringWithUTF8String = objc.RegisterName("stringWithUTF8String:")
	selUTF8String           = objc.RegisterName("UTF8String")

	selMainScreen   = objc.RegisterName("mainScreen")
	selScreens      = objc.RegisterName("screens")
	selVisibleFrame = objc.RegisterName("visibleFrame")

	selArrowCursor           = objc.RegisterName("arrowCursor")
	selPointingHandCursor    = objc.RegisterName("pointingHandCursor")
	selResizeLeftRightCursor = objc.RegisterName("resizeLeftRightCursor")
	selResizeUpDownCursor    = objc.RegisterName("resizeUpDownCursor")
	selIBeamCursor           = objc.RegisterName("IBeamCursor")
	selSet                   = objc.RegisterName("set")

	selInitWithBitmapDataPlanes = objc.RegisterName("initWithBitmapDataPlanes:pixelsWide:pixelsHigh:bitsPerSample:samplesPerPixel:hasAlpha:isPlanar:colorSpaceName:bytesPerRow:bitsPerPixel:")
	selBitmapData               = objc.RegisterName("bitmapData")
	selInitWithSize             = objc.RegisterName("initWithSize:")
	selAddRepresentation        = objc.RegisterName("addRepresentation:")

	selCurrentContext = objc.RegisterName("currentContext")
	selCGContext      = objc.RegisterName("CGContext")

	selTimerWithTimeIntervalRepeatsBlock = objc.RegisterName("timerWithTimeInterval:repeats:block:")
	selInvalidate                        = objc.RegisterName("invalidate")
	selMainRunLoop                       = objc.RegisterName("mainRunLoop")
	selAddTimerForMode                   = objc.RegisterName("addTimer:forMode:")

	selIsMainThread = objc.RegisterName("isMainThread")

	selOpenPanel                  = objc.RegisterName("openPanel")
	selSavePanel                  = objc.RegisterName("savePanel")
	selSetCanChooseFiles          = objc.RegisterName("setCanChooseFiles:")
	selSetCanChooseDirectories    = objc.RegisterName("setCanChooseDirectories:")
	selSetCanCreateDirectories    = objc.RegisterName("setCanCreateDirectories:")
	selSetAllowsMultipleSelection = objc.RegisterName("setAllowsMultipleSelection:")
	selSetDirectoryURL            = objc.RegisterName("setDirectoryURL:")
	selSetNameFieldStringValue    = objc.RegisterName("setNameFieldStringValue:")
	selSetAllowedFileTypes        = objc.RegisterName("setAllowedFileTypes:")
	selRunModal                   = objc.RegisterName("runModal")
	selURLs                       = objc.RegisterName("URLs")
	selURL                        = objc.RegisterName("URL")
	selPath                       = objc.RegisterName("path")
	selCount                      = objc.RegisterName("count")
	selObjectAtIndex              = objc.RegisterName("objectAtIndex:")
	selAddObject                  = objc.RegisterName("addObject:")
	selFileURLWithPath            = objc.RegisterName("fileURLWithPath:")

	selApplicationShouldTerminateAfterLastWindowClosed = objc.RegisterName("applicationShouldTerminateAfterLastWindowClosed:")
	selWindowDidMove                                   = objc.RegisterName("windowDidMove:")
	selWindowShouldClose                               = objc.RegisterName("windowShouldClose:")
	selWindowWillClose                                 = objc.RegisterName("windowWillClose:")
	selWindowDidResignKey                              = objc.RegisterName("windowDidResignKey:")
	selWindowDidBecomeKey                              = objc.RegisterName("windowDidBecomeKey:")

	selAcceptsFirstResponder = objc.RegisterName("acceptsFirstResponder")
	selBecomeFirstResponder  = objc.RegisterName("becomeFirstResponder")
	selKeyUp                 = objc.RegisterName("keyUp:")
	selKeyDown               = objc.RegisterName("keyDown:")
	selFlagsChanged          = objc.RegisterName("flagsChanged:")
	selMouseDown             = objc.RegisterName("mouseDown:")
	selRightMouseDown        = objc.RegisterName("rightMouseDown:")
	selOtherMouseDown        = objc.RegisterName("otherMouseDown:")
	selMouseUp               = objc.RegisterName("mouseUp:")
	selRightMouseUp          = objc.RegisterName("rightMouseUp:")
	selOtherMouseUp          = objc.RegisterName("otherMouseUp:")
	selMouseMoved            = objc.RegisterName("mouseMoved:")
	selMouseDragged          = objc.RegisterName("mouseDragged:")
	selRightMouseDragged     = objc.RegisterName("rightMouseDragged:")
	selScrollWheel           = objc.RegisterName("scrollWheel:")
	selMouseEntered          = objc.RegisterName("mouseEntered:")
	selMouseExited           = objc.RegisterName("mouseExited:")
	selUpdateTrackingAreas   = objc.RegisterName("updateTrackingAreas")
	selIsFlipped             = objc.RegisterName("isFlipped")
	selDrawRect              = objc.RegisterName("drawRect:")

	selBackingScaleFactor             = objc.RegisterName("backingScaleFactor")
	selViewDidChangeBackingProperties = objc.RegisterName("viewDidChangeBackingProperties")

	selRegisterForDraggedTypes      = objc.RegisterName("registerForDraggedTypes:")
	selDraggingEntered              = objc.RegisterName("draggingEntered:")
	selDraggingUpdated              = objc.RegisterName("draggingUpdated:")
	selPerformDragOperation         = objc.RegisterName("performDragOperation:")
	selDraggingPasteboard           = objc.RegisterName("draggingPasteboard")
	selDraggingLocation             = objc.RegisterName("draggingLocation")
	selReadObjectsForClassesOptions = objc.RegisterName("readObjectsForClasses:options:")
	selIsFileURL                    = objc.RegisterName("isFileURL")
)

/////////////////////////////////////////////////////
// Classes and library handles (filled in by init(), NOT var initializers:
// GetClass needs AppKit dlopen'd first, which only happens inside init()).

var (
	clsNSApplication     objc.Class
	clsNSWindow          objc.Class
	clsNSString          objc.Class
	clsNSScreen          objc.Class
	clsNSCursor          objc.Class
	clsNSBitmapImageRep  objc.Class
	clsNSImage           objc.Class
	clsNSGraphicsContext objc.Class
	clsNSTimer           objc.Class
	clsNSRunLoop         objc.Class
	clsNSThread          objc.Class
	clsNSTrackingArea    objc.Class
	clsNSOpenPanel       objc.Class
	clsNSSavePanel       objc.Class
	clsNSURL             objc.Class
	clsNSMutableArray    objc.Class

	nuiAppDelegateClass objc.Class
	nuiPaintViewClass   objc.Class
)

var (
	cgColorSpaceCreateDeviceRGB  func() uintptr
	cgDataProviderCreateWithData func(info uintptr, data unsafe.Pointer, size uintptr, releaseData uintptr) uintptr
	cgImageCreate                func(width, height, bitsPerComponent, bitsPerPixel, bytesPerRow uintptr, space uintptr, bitmapInfo uint32, provider uintptr, decode uintptr, shouldInterpolate bool, intent int32) uintptr
	cgContextDrawImage           func(ctx uintptr, rect nsRect, image uintptr)
	cgContextSetRGBFillColor     func(ctx uintptr, r, g, b, a float64)
	cgContextFillRect            func(ctx uintptr, rect nsRect)
	cgContextSaveGState          func(ctx uintptr)
	cgContextRestoreGState       func(ctx uintptr)
	cgImageRelease               func(image uintptr)
	cgDataProviderRelease        func(provider uintptr)
	cgColorSpaceRelease          func(space uintptr)

	cgDataProviderCreateWithCFData func(data uintptr) uintptr
	cfDataCreateFn                 func(allocator uintptr, bytes unsafe.Pointer, length int) uintptr
	cfReleaseFn                    func(cf uintptr)

	dispatchMainQueue uintptr
	dispatchAsyncFn   func(queue uintptr, block uintptr)
	dispatchSyncFn    func(queue uintptr, block uintptr)

	objcAutoreleasePoolPushFn func() uintptr
	objcAutoreleasePoolPopFn  func(ctx uintptr)

	libcStrlen func(ptr uintptr) uintptr

	nsBeepFn func()
)

func init() {
	appKitHandle, err := purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&nsBeepFn, appKitHandle, "NSBeep")
	if _, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_NOW|purego.RTLD_GLOBAL); err != nil {
		panic(err)
	}
	cgHandle, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	libSystemHandle, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	libobjcHandle, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}

	purego.RegisterLibFunc(&cgColorSpaceCreateDeviceRGB, cgHandle, "CGColorSpaceCreateDeviceRGB")
	purego.RegisterLibFunc(&cgDataProviderCreateWithData, cgHandle, "CGDataProviderCreateWithData")
	purego.RegisterLibFunc(&cgImageCreate, cgHandle, "CGImageCreate")
	purego.RegisterLibFunc(&cgContextDrawImage, cgHandle, "CGContextDrawImage")
	purego.RegisterLibFunc(&cgContextSetRGBFillColor, cgHandle, "CGContextSetRGBFillColor")
	purego.RegisterLibFunc(&cgContextFillRect, cgHandle, "CGContextFillRect")
	purego.RegisterLibFunc(&cgContextSaveGState, cgHandle, "CGContextSaveGState")
	purego.RegisterLibFunc(&cgContextRestoreGState, cgHandle, "CGContextRestoreGState")
	purego.RegisterLibFunc(&cgImageRelease, cgHandle, "CGImageRelease")
	purego.RegisterLibFunc(&cgDataProviderRelease, cgHandle, "CGDataProviderRelease")
	purego.RegisterLibFunc(&cgColorSpaceRelease, cgHandle, "CGColorSpaceRelease")
	purego.RegisterLibFunc(&cgDataProviderCreateWithCFData, cgHandle, "CGDataProviderCreateWithCFData")

	cfHandle, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&cfDataCreateFn, cfHandle, "CFDataCreate")
	purego.RegisterLibFunc(&cfReleaseFn, cfHandle, "CFRelease")

	purego.RegisterLibFunc(&dispatchAsyncFn, libSystemHandle, "dispatch_async")
	purego.RegisterLibFunc(&dispatchSyncFn, libSystemHandle, "dispatch_sync")
	purego.RegisterLibFunc(&libcStrlen, libSystemHandle, "strlen")

	dispatchMainQueue, err = purego.Dlsym(libSystemHandle, "_dispatch_main_q")
	if err != nil {
		panic(err)
	}

	purego.RegisterLibFunc(&objcAutoreleasePoolPushFn, libobjcHandle, "objc_autoreleasePoolPush")
	purego.RegisterLibFunc(&objcAutoreleasePoolPopFn, libobjcHandle, "objc_autoreleasePoolPop")

	clsNSApplication = objc.GetClass("NSApplication")
	clsNSWindow = objc.GetClass("NSWindow")
	clsNSString = objc.GetClass("NSString")
	clsNSScreen = objc.GetClass("NSScreen")
	clsNSCursor = objc.GetClass("NSCursor")
	clsNSBitmapImageRep = objc.GetClass("NSBitmapImageRep")
	clsNSImage = objc.GetClass("NSImage")
	clsNSGraphicsContext = objc.GetClass("NSGraphicsContext")
	clsNSTimer = objc.GetClass("NSTimer")
	clsNSRunLoop = objc.GetClass("NSRunLoop")
	clsNSThread = objc.GetClass("NSThread")
	clsNSTrackingArea = objc.GetClass("NSTrackingArea")
	clsNSOpenPanel = objc.GetClass("NSOpenPanel")
	clsNSSavePanel = objc.GetClass("NSSavePanel")
	clsNSURL = objc.GetClass("NSURL")
	clsNSMutableArray = objc.GetClass("NSMutableArray")

	registerNuiClasses()
}

/////////////////////////////////////////////////////
// Registries - replace window.m's NSMutableDictionary windowMap/timers with
// plain Go maps, keyed the same way (by windowNumber, i.e. windowId).
// Mutated only from the real Cocoa main thread (either called directly from
// an AppKit callback, which AppKit always invokes on main, or funneled there
// via runOnMainSync/dispatchAsyncMain below), same invariant window.m relied
// on for its unsynchronized NSMutableDictionary.

var cocoaWindows = map[int]objc.ID{}           // windowId -> NSWindow
var cocoaDelegates = map[int]objc.ID{}         // windowId -> NUIAppDelegate (also the window's delegate)
var cocoaTimers = map[int]objc.ID{}            // windowId -> NSTimer
var cocoaTrackingAreas = map[objc.ID]objc.ID{} // NUIPaintView -> its current NSTrackingArea

func wndID(win objc.ID) windowId {
	if win == 0 {
		return -1
	}
	return windowId(objc.Send[int](win, selWindowNumber))
}

/////////////////////////////////////////////////////
// Main-thread dispatch and autorelease pools

// isMainThread mirrors window.m's [NSThread isMainThread] checks.
func isMainThread() bool {
	return objc.Send[bool](objc.ID(clsNSThread), selIsMainThread)
}

// runOnMainSync mirrors window.m's NUI_RunOnMainSync: runs fn() directly if
// already on the main thread (the common case - called from a Go callback
// that Cocoa itself invoked on main), else via dispatch_sync so a caller on
// another goroutine still blocks until fn() completes. dispatch_sync (unlike
// dispatch_async) doesn't need the block to outlive the call, so releasing
// our +1 reference right after it returns is correct.
func runOnMainSync(fn func()) {
	if isMainThread() {
		fn()
		return
	}
	block := objc.NewBlock(func(_ objc.Block) { fn() })
	dispatchSyncFn(dispatchMainQueue, uintptr(block))
	block.Release()
}

// dispatchAsyncMain mirrors window.m's dispatch_async(dispatch_get_main_queue(), ...)
// (used by QuitApp and ShowWindow's deferred resync). dispatch_async takes
// its own internal retain of the block for as long as it needs it and
// releases that when done, independent of our own +1 from NewBlock, so it's
// correct (and required, to avoid a leak) to release our own reference
// immediately after scheduling rather than waiting for it to run.
func dispatchAsyncMain(fn func()) {
	block := objc.NewBlock(func(_ objc.Block) { fn() })
	dispatchAsyncFn(dispatchMainQueue, uintptr(block))
	block.Release()
}

// newCGImageRGBA makes a CGImage of the RGBA pixels of buf. The pixels are
// copied into a CFData the image owns: CoreGraphics may read them after
// drawRect returns (deferred drawing), when buf - Go memory, referenced by
// nothing the GC sees - may be freed already. 0 if out of memory.
func newCGImageRGBA(buf []byte, width, height int, colorSpace uintptr) uintptr {
	data := cfDataCreateFn(0, unsafe.Pointer(&buf[0]), len(buf))
	runtime.KeepAlive(buf)
	if data == 0 {
		return 0
	}
	provider := cgDataProviderCreateWithCFData(data)
	cfReleaseFn(data) // the provider keeps it
	if provider == 0 {
		return 0
	}
	image := cgImageCreate(uintptr(width), uintptr(height), 8, 32, uintptr(width*4), colorSpace,
		cgImageAlphaPremultipliedLast|cgBitmapByteOrder32Big, provider, 0, false, cgRenderingIntentDefault)
	cgDataProviderRelease(provider) // the image keeps it
	return image
}

func withAutoreleasePool(fn func()) {
	ctx := objcAutoreleasePoolPushFn()
	defer objcAutoreleasePoolPopFn(ctx)
	fn()
}

/////////////////////////////////////////////////////
// String conversions (no cgo, so no C.GoString - decode UTF8String's
// returned const char* by hand instead)

func cGoString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	n := libcStrlen(ptr)
	b := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n)
	return string(b)
}

func nsStringToGo(s objc.ID) string {
	if s == 0 {
		return ""
	}
	return cGoString(objc.Send[uintptr](s, selUTF8String))
}

func goStringToNS(s string) objc.ID {
	return objc.ID(clsNSString).Send(selStringWithUTF8String, s)
}

/////////////////////////////////////////////////////
// Window geometry helpers (mirror window.m's NUI_WindowTopOriginY /
// NUI_ReportClientSizeToGo / NUI_DrawableRectInContentView)

// windowTopOriginY: top-down Y distance from screen top to window top
// (inverse of Cocoa's bottom-left frame.origin.y). Matches SetWindowPosition's input.
func windowTopOriginY(win objc.ID) int {
	if win == 0 {
		return -1
	}
	frame := objc.Send[nsRect](win, selFrame)
	screen := objc.Send[objc.ID](win, selScreen)
	if screen == 0 {
		screen = objc.ID(clsNSScreen).Send(selMainScreen)
	}
	screenFrame := objc.Send[nsRect](screen, selFrame)
	return int(screenFrame.Size.Height - frame.Origin.Y - frame.Size.Height)
}

// reportClientSizeToGo pushes NSWindow.contentLayoutRect size to Go's
// OnResize (same as CreateWindow width/height on Mac).
func reportClientSizeToGo(id windowId, win objc.ID) {
	if win == 0 {
		return
	}
	lr := objc.Send[nsRect](win, selContentLayoutRect)
	w := int(math.Max(1, math.Floor(lr.Size.Width)))
	h := int(math.Max(1, math.Floor(lr.Size.Height)))
	go_on_resize(id, w, h)
}

// drawableRectInContentView: region NUI actually paints - contentLayoutRect
// in contentView coords, clipped to the view's bounds.
func drawableRectInContentView(win, view objc.ID) nsRect {
	if view == 0 {
		return nsRect{}
	}
	if win == 0 {
		return objc.Send[nsRect](view, selBounds)
	}
	contentLayout := objc.Send[nsRect](win, selContentLayoutRect)
	converted := objc.Send[nsRect](view, selConvertRectFromView, contentLayout, objc.ID(0))
	bounds := objc.Send[nsRect](view, selBounds)
	drawable := nsRectIntersection(converted, bounds)
	if drawable.Size.Width < 1 || drawable.Size.Height < 1 {
		drawable = bounds
	}
	return drawable
}

/////////////////////////////////////////////////////
// NUIAppDelegate: NSApplicationDelegate + NSWindowDelegate. Cocoa dispatches
// delegate methods via respondsToSelector:, so formal protocol conformance
// isn't required for AppKit to call these.

// AppKit also asks this when the last visible window is ordered out, e.g. a
// form hidden to the tray: only quit once no form window is left open.
func nuiAppShouldTerminateAfterLastWindowClosed(self objc.ID, _ objc.SEL, _ objc.ID) bool {
	return len(cocoaWindows) == 0
}

func nuiWindowDidMove(self objc.ID, _ objc.SEL, notification objc.ID) {
	win := objc.Send[objc.ID](notification, selObject)
	id := wndID(win)
	frame := objc.Send[nsRect](win, selFrame)
	go_on_window_move(id, int(frame.Origin.X), windowTopOriginY(win))
}

func nuiWindowShouldClose(self objc.ID, _ objc.SEL, sender objc.ID) bool {
	return go_on_close_request(wndID(sender))
}

// nuiWindowWillClose ends the nested modal loop started by ShowModalWindow
// if this was the modal window, stops the window's timer, notifies Go, then
// drops the window and its delegate (self) from our own bookkeeping maps -
// see the file-level comment on object lifetime.
//
// Deliberately NOT released: both a synchronous and a next-run-loop-tick
// deferred release (dispatchAsyncMain, mirroring go_on_window_will_close's
// own QuitApp call) reliably segfaulted deep inside AppKit - reproducibly
// when closing a modal window that itself had already hosted and closed a
// nested modal child. AppKit apparently keeps its own internal references to
// a just-closed window/its delegate alive for longer than either of those
// windows, in ways not fully predictable from outside its private internals
// (window ordering lists, the responder chain, key-window bookkeeping, ...).
// Leaking one NSWindow + one delegate object per window ever opened for the
// lifetime of the process is an acceptable trade for not crashing - it
// matches (or is more conservative than) the original cgo/window.m
// implementation, which never released its own per-window AppDelegate
// instances either (nothing ever balanced their +1 alloc).
func nuiWindowWillClose(self objc.ID, _ objc.SEL, notification objc.ID) {
	win := objc.Send[objc.ID](notification, selObject)
	id := wndID(win)
	wid := int(id)

	nsApp := objc.ID(clsNSApplication).Send(selSharedApplication)
	if modalWin := objc.Send[objc.ID](nsApp, selModalWindow); modalWin == win {
		nsApp.Send(selStopModal)
	}

	stopTimer(id)
	go_on_window_will_close(id)

	delete(cocoaDelegates, wid)
	delete(cocoaWindows, wid)
}

// nuiWindowDidResignKey: the window lost keyboard focus to another window
// or application.
func nuiWindowDidResignKey(self objc.ID, _ objc.SEL, notification objc.ID) {
	win := objc.Send[objc.ID](notification, selObject)
	go_on_window_deactivate(wndID(win))
}

// nuiWindowDidBecomeKey: the window got keyboard focus.
func nuiWindowDidBecomeKey(self objc.ID, _ objc.SEL, notification objc.ID) {
	win := objc.Send[objc.ID](notification, selObject)
	go_on_window_activate(wndID(win))
}

/////////////////////////////////////////////////////
// NUIPaintView: full-window content NSView hosting the software framebuffer
// and input.

func nuiViewAcceptsFirstResponder(self objc.ID, _ objc.SEL) bool { return true }
func nuiViewBecomeFirstResponder(self objc.ID, _ objc.SEL) bool  { return true }
func nuiViewIsFlipped(self objc.ID, _ objc.SEL) bool             { return false }

// Window chrome can change without our Resize call; always resync client
// size to Go.
//
// setFrameSize:'s real signature takes one NSSize struct arg, but
// purego.NewCallback (used for the C-calls-Go IMP direction, unlike the
// Go-calls-C Send/SendSuper direction) doesn't support struct-typed
// parameters, so the two fields are received as separate float64s instead -
// NSSize is all-float, so it decomposes into the same pair of FP argument
// registers either way. Reassembled into a struct only to hand back to
// SendSuper, which (being outbound) does support struct args.
func nuiViewSetFrameSize(self objc.ID, cmd objc.SEL, width, height float64) {
	self.SendSuper(cmd, nsSize{width, height})
	win := objc.Send[objc.ID](self, selWindow)
	if win == 0 {
		return
	}
	reportClientSizeToGo(wndID(win), win)
}

func nuiViewKeyUp(self objc.ID, _ objc.SEL, event objc.ID) {
	chars := objc.Send[objc.ID](event, selCharacters)
	if objc.Send[int](chars, selLength) > 0 {
		win := objc.Send[objc.ID](self, selWindow)
		keyCode := int(objc.Send[uint16](event, selKeyCode))
		go_on_key_up(wndID(win), keyCode)
	}
}

// nuiViewKeyDown reports the key, then its character - unless the key was
// handled (e.g. a shortcut) or is a command with Cmd or Ctrl
func nuiViewKeyDown(self objc.ID, _ objc.SEL, event objc.ID) {
	win := objc.Send[objc.ID](self, selWindow)
	id := wndID(win)
	keyCode := int(objc.Send[uint16](event, selKeyCode))
	if go_on_key_down(id, keyCode) {
		return
	}
	flags := objc.Send[uint64](event, selModifierFlags)
	if flags&(nsEventModifierFlagCommand|nsEventModifierFlagControl) != 0 {
		return
	}
	chars := objc.Send[objc.ID](event, selCharacters)
	if objc.Send[int](chars, selLength) > 0 {
		ch := objc.Send[uint16](chars, selCharacterAtIndex, 0)
		go_on_char(id, int(ch))
	}
}

func nuiViewFlagsChanged(self objc.ID, _ objc.SEL, event objc.ID) {
	win := objc.Send[objc.ID](self, selWindow)
	syncEventModifiers(wndID(win), event)
}

// syncEventModifiers reports the modifier keys held during the event. Mouse
// events call it too: flagsChanged: only reaches the first responder of the
// key window, so a modifier pressed while it was elsewhere would be missed,
// and Shift/Ctrl+click wouldn't extend the selection.
func syncEventModifiers(id windowId, event objc.ID) {
	flags := objc.Send[uint64](event, selModifierFlags)
	go_on_modifier_change(id,
		flags&nsEventModifierFlagShift != 0,
		flags&nsEventModifierFlagControl != 0,
		flags&nsEventModifierFlagOption != 0,
		flags&nsEventModifierFlagCommand != 0,
		flags&nsEventModifierFlagCapsLock != 0,
		flags&nsEventModifierFlagNumericPad != 0,
		flags&nsEventModifierFlagFunction != 0,
	)
}

// backingScale returns how many pixels of the screen a point of the window
// takes: 2 on Retina
func backingScale(win objc.ID) float64 {
	if win == 0 {
		return 1
	}
	scale := objc.Send[float64](win, selBackingScaleFactor)
	if scale <= 0 {
		return 1
	}
	return scale
}

// scaledSize returns the size in points in pixels of the scale
func scaledSize(width, height int, scale float64) (int, int) {
	return max(1, int(math.Round(float64(width)*scale))), max(1, int(math.Round(float64(height)*scale)))
}

// nuiViewDidChangeBackingProperties: the window moved to a screen of another
// scale, so it's painted again in its pixels
func nuiViewDidChangeBackingProperties(self objc.ID, _ objc.SEL) {
	self.Send(selSetNeedsDisplay, true)
}

func eventLocationInView(self, event objc.ID) nsPoint {
	p := objc.Send[nsPoint](event, selLocationInWindow)
	return objc.Send[nsPoint](self, selConvertPointFromView, p, objc.ID(0))
}

// viewPointToInt rounds a view point to the whole point it lies in. The view
// isn't flipped and Go turns y into areaH-y, so y is rounded up: truncating
// it would move the point a whole point down after the flip.
func viewPointToInt(p nsPoint) (int, int) {
	return int(math.Floor(p.X)), int(math.Ceil(p.Y))
}

func nuiHandleMouseDown(self, event objc.ID, button int) {
	win := objc.Send[objc.ID](self, selWindow)
	id := wndID(win)
	syncEventModifiers(id, event)
	p := eventLocationInView(self, event)
	x, y := viewPointToInt(p)
	if objc.Send[int](event, selClickCount) == 2 {
		go_on_mouse_double_click(id, button, x, y)
	} else {
		go_on_mouse_down(id, button, x, y)
	}
}

func nuiViewMouseDown(self objc.ID, _ objc.SEL, event objc.ID) { nuiHandleMouseDown(self, event, 0) }
func nuiViewRightMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiHandleMouseDown(self, event, 1)
}
func nuiViewOtherMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiHandleMouseDown(self, event, 2)
}

func nuiHandleMouseUp(self, event objc.ID, button int) {
	win := objc.Send[objc.ID](self, selWindow)
	id := wndID(win)
	syncEventModifiers(id, event)
	x, y := viewPointToInt(eventLocationInView(self, event))
	go_on_mouse_up(id, button, x, y)
}

func nuiViewMouseUp(self objc.ID, _ objc.SEL, event objc.ID)      { nuiHandleMouseUp(self, event, 0) }
func nuiViewRightMouseUp(self objc.ID, _ objc.SEL, event objc.ID) { nuiHandleMouseUp(self, event, 1) }
func nuiViewOtherMouseUp(self objc.ID, _ objc.SEL, event objc.ID) { nuiHandleMouseUp(self, event, 2) }

// nuiViewMouseMoved backs mouseMoved:, mouseDragged: and rightMouseDragged:
// - all three do the same convert-point-and-report dance in window.m.
func nuiViewMouseMoved(self objc.ID, _ objc.SEL, event objc.ID) {
	win := objc.Send[objc.ID](self, selWindow)
	id := wndID(win)
	syncEventModifiers(id, event)
	x, y := viewPointToInt(eventLocationInView(self, event))
	go_on_mouse_move(id, x, y)
}

func nuiViewScrollWheel(self objc.ID, _ objc.SEL, event objc.ID) {
	dx := objc.Send[float64](event, selDeltaX)
	dy := objc.Send[float64](event, selDeltaY)
	if dx == 0 && dy == 0 {
		return
	}
	win := objc.Send[objc.ID](self, selWindow)
	go_on_mouse_scroll(wndID(win), dx, dy)
}

func nuiViewMouseEntered(self objc.ID, _ objc.SEL, _ objc.ID) {
	win := objc.Send[objc.ID](self, selWindow)
	go_on_mouse_enter(wndID(win))
}

func nuiViewMouseExited(self objc.ID, _ objc.SEL, _ objc.ID) {
	win := objc.Send[objc.ID](self, selWindow)
	go_on_mouse_leave(wndID(win))
}

func nuiViewUpdateTrackingAreas(self objc.ID, cmd objc.SEL) {
	self.SendSuper(cmd)

	if old, ok := cocoaTrackingAreas[self]; ok {
		self.Send(selRemoveTrackingArea, old)
		delete(cocoaTrackingAreas, self)
		old.Send(selRelease)
	}

	bounds := objc.Send[nsRect](self, selBounds)
	const opts = nsTrackingMouseEnteredAndExited | nsTrackingMouseMoved | nsTrackingActiveAlways | nsTrackingInVisibleRect
	area := objc.ID(clsNSTrackingArea).Send(selAlloc)
	area = area.Send(selInitWithRectOptionsOwnerUserInfo, bounds, opts, self, objc.ID(0))
	self.Send(selAddTrackingArea, area)
	cocoaTrackingAreas[self] = area
}

// drawRect: composites the Go-painted RGBA buffer via CoreGraphics. Buffer
// size/dest rect = drawable inside content view (not always full bounds).
// The buffer is an ordinary Go []byte, not C malloc'd: unlike window.m
// (which had to free() it from the same allocator XDestroyImage-style code
// on Linux uses) nothing outside this function call needs to free it, so a
// NULL release callback is passed to CGDataProviderCreateWithData and Go's
// GC reclaims it once unreferenced - the local `buf` keeps it alive for the
// duration of this synchronous call.
// drawRect:'s real signature takes one NSRect struct arg; flattened to four
// float64s (unused) for the same purego.NewCallback struct-arg limitation
// noted on nuiViewSetFrameSize above - the actual drawable rect is
// recomputed from contentLayoutRect/bounds instead of the dirty rect anyway.
func nuiViewDrawRect(self objc.ID, _ objc.SEL, _, _, _, _ float64) {
	start := time.Now()

	win := objc.Send[objc.ID](self, selWindow)
	if win == 0 {
		return
	}
	id := wndID(win)

	drawable := drawableRectInContentView(win, self)
	width := int(math.Max(1, math.Floor(drawable.Size.Width)))
	height := int(math.Max(1, math.Floor(drawable.Size.Height)))
	// Painted in the screen's pixels (2 per point on Retina) and drawn into
	// the rect in points, so the image isn't stretched
	scale := backingScale(win)
	pixelW, pixelH := scaledSize(width, height, scale)
	stride := pixelW * 4
	dataSize := stride * pixelH

	buf := make([]byte, dataSize)
	go_on_paint(id, unsafe.Pointer(&buf[0]), pixelW, pixelH, scale)

	ctx := objc.Send[uintptr](objc.ID(clsNSGraphicsContext).Send(selCurrentContext), selCGContext)
	colorSpace := cgColorSpaceCreateDeviceRGB()
	image := newCGImageRGBA(buf, pixelW, pixelH, colorSpace)

	bounds := objc.Send[nsRect](self, selBounds)
	dest := nsRect{
		nsPoint{math.Floor(drawable.Origin.X), math.Floor(drawable.Origin.Y)},
		nsSize{float64(width), float64(height)},
	}

	cgContextSaveGState(ctx)
	// Matches default darwin canvas clear in Go until SetBackgroundColor is bridged here.
	cgContextSetRGBFillColor(ctx, 0, 50.0/255.0, 0, 1)
	cgContextFillRect(ctx, bounds)
	cgContextDrawImage(ctx, dest, image)
	cgContextRestoreGState(ctx)

	cgImageRelease(image)
	cgColorSpaceRelease(colorSpace)

	go_on_declare_draw_time(id, int(time.Since(start).Microseconds()))
}

// Files dragged from Finder: the view is a dragging destination for file URLs
// (registered in initWindow).

func nuiViewDraggingEntered(self objc.ID, _ objc.SEL, sender objc.ID) uint {
	win := objc.Send[objc.ID](self, selWindow)
	if w, ok := hwnds[wndID(win)]; ok && w.onFilesDropped != nil {
		return nsDragOperationCopy
	}
	return nsDragOperationNone
}

func nuiViewPerformDragOperation(self objc.ID, _ objc.SEL, sender objc.ID) bool {
	win := objc.Send[objc.ID](self, selWindow)
	pasteboard := objc.Send[objc.ID](sender, selDraggingPasteboard)
	classes := objc.ID(clsNSMutableArray).Send(selAlloc).Send(selInit)
	classes.Send(selAddObject, objc.ID(clsNSURL))
	urls := objc.Send[objc.ID](pasteboard, selReadObjectsForClassesOptions, classes, objc.ID(0))
	classes.Send(selRelease)

	var files []string
	if urls != 0 {
		n := objc.Send[int](urls, selCount)
		for i := 0; i < n; i++ {
			u := objc.Send[objc.ID](urls, selObjectAtIndex, i)
			if objc.Send[bool](u, selIsFileURL) {
				files = append(files, nsStringToGo(objc.Send[objc.ID](u, selPath)))
			}
		}
	}
	if len(files) == 0 {
		return false
	}
	p := objc.Send[nsPoint](sender, selDraggingLocation)
	p = objc.Send[nsPoint](self, selConvertPointFromView, p, objc.ID(0))
	x, y := viewPointToInt(p)
	go_on_files_dropped(wndID(win), files, x, y)
	return true
}

func registerNuiClasses() {
	var err error

	nuiAppDelegateClass, err = objc.RegisterClass(
		"NUIAppDelegate",
		objc.GetClass("NSObject"),
		nil, nil,
		[]objc.MethodDef{
			{Cmd: selApplicationShouldTerminateAfterLastWindowClosed, Fn: nuiAppShouldTerminateAfterLastWindowClosed},
			{Cmd: selWindowDidMove, Fn: nuiWindowDidMove},
			{Cmd: selWindowShouldClose, Fn: nuiWindowShouldClose},
			{Cmd: selWindowWillClose, Fn: nuiWindowWillClose},
			{Cmd: selWindowDidResignKey, Fn: nuiWindowDidResignKey},
			{Cmd: selWindowDidBecomeKey, Fn: nuiWindowDidBecomeKey},
		},
	)
	if err != nil {
		panic(err)
	}

	nuiPaintViewClass, err = objc.RegisterClass(
		"NUIPaintView",
		objc.GetClass("NSView"),
		nil, nil,
		[]objc.MethodDef{
			{Cmd: selAcceptsFirstResponder, Fn: nuiViewAcceptsFirstResponder},
			{Cmd: selBecomeFirstResponder, Fn: nuiViewBecomeFirstResponder},
			{Cmd: selIsFlipped, Fn: nuiViewIsFlipped},
			{Cmd: selSetFrameSize, Fn: nuiViewSetFrameSize},
			{Cmd: selKeyUp, Fn: nuiViewKeyUp},
			{Cmd: selKeyDown, Fn: nuiViewKeyDown},
			{Cmd: selFlagsChanged, Fn: nuiViewFlagsChanged},
			{Cmd: selMouseDown, Fn: nuiViewMouseDown},
			{Cmd: selRightMouseDown, Fn: nuiViewRightMouseDown},
			{Cmd: selOtherMouseDown, Fn: nuiViewOtherMouseDown},
			{Cmd: selMouseUp, Fn: nuiViewMouseUp},
			{Cmd: selRightMouseUp, Fn: nuiViewRightMouseUp},
			{Cmd: selOtherMouseUp, Fn: nuiViewOtherMouseUp},
			{Cmd: selMouseMoved, Fn: nuiViewMouseMoved},
			{Cmd: selMouseDragged, Fn: nuiViewMouseMoved},
			{Cmd: selRightMouseDragged, Fn: nuiViewMouseMoved},
			{Cmd: selScrollWheel, Fn: nuiViewScrollWheel},
			{Cmd: selMouseEntered, Fn: nuiViewMouseEntered},
			{Cmd: selMouseExited, Fn: nuiViewMouseExited},
			{Cmd: selUpdateTrackingAreas, Fn: nuiViewUpdateTrackingAreas},
			{Cmd: selDrawRect, Fn: nuiViewDrawRect},
			{Cmd: selViewDidChangeBackingProperties, Fn: nuiViewDidChangeBackingProperties},
			{Cmd: selDraggingEntered, Fn: nuiViewDraggingEntered},
			{Cmd: selDraggingUpdated, Fn: nuiViewDraggingEntered},
			{Cmd: selPerformDragOperation, Fn: nuiViewPerformDragOperation},
		},
	)
	if err != nil {
		panic(err)
	}
}

/////////////////////////////////////////////////////
// Window creation and lifecycle (mirrors window.m's InitWindow/ShowWindow/
// RunEventLoop/CloseWindowById/QuitApp/ShowModalWindow)

// sharedAppDelegate is a single, permanent NUIAppDelegate instance used only
// for NSApplicationDelegate duties (applicationShouldTerminateAfterLast
// WindowClosed:). It is set as NSApp's delegate exactly once and never
// released.
//
// window.m's original InitWindow reassigned NSApp's delegate to every new
// window's own delegate object; that was harmless there because nothing was
// ever released (a plain leak). Once nuiWindowWillClose started releasing a
// closed window's delegate (see its comment), that pattern left NSApp.delegate
// - a non-retaining/weak reference - dangling at whatever window last closed:
// AppKit then segfaults the next time anything touches the app delegate,
// which happens while resuming a parent modal session after a nested child
// window closes. Using one delegate that's never released for the app-level
// role sidesteps the dangling-pointer risk entirely; each window still gets
// its own separate delegate instance for window-scoped notifications
// (windowDidMove:/windowShouldClose:/windowWillClose:), which is fine since
// nothing but that window itself ever consults its own .delegate.
var sharedAppDelegate objc.ID

// initWindow creates a new NSWindow + NUIPaintView + (window-scoped)
// NUIAppDelegate, mirroring window.m's InitWindow.
func initWindow() windowId {
	var wid windowId
	withAutoreleasePool(func() {
		nsApp := objc.ID(clsNSApplication).Send(selSharedApplication)
		nsApp.Send(selSetActivationPolicy, nsApplicationActivationPolicyRegular)

		if sharedAppDelegate == 0 {
			sharedAppDelegate = objc.ID(nuiAppDelegateClass).Send(selAlloc).Send(selInit)
			nsApp.Send(selSetDelegate, sharedAppDelegate)
		}

		delegate := objc.ID(nuiAppDelegateClass).Send(selAlloc).Send(selInit)

		frame := nsRect{nsPoint{100, 100}, nsSize{800, 600}}
		style := nsWindowStyleMaskTitled | nsWindowStyleMaskClosable | nsWindowStyleMaskResizable | nsWindowStyleMaskMiniaturizable

		win := objc.ID(clsNSWindow).Send(selAlloc)
		win = win.Send(selInitWithContentRectStyleMaskBackingDefer, frame, style, nsBackingStoreBuffered, false)

		view := objc.ID(nuiPaintViewClass).Send(selAlloc).Send(selInitWithFrame, frame)
		win.Send(selSetContentView, view)
		view.Send(selRelease) // window now owns it via setContentView:'s internal retain

		draggedTypes := objc.ID(clsNSMutableArray).Send(selAlloc).Send(selInit)
		draggedTypes.Send(selAddObject, goStringToNS("public.file-url"))
		view.Send(selRegisterForDraggedTypes, draggedTypes)
		draggedTypes.Send(selRelease)

		win.Send(selSetTitle, goStringToNS("NUI Window"))
		win.Send(selSetDelegate, delegate)

		nsApp.Send(selActivateIgnoringOtherApps, true)

		id := int(objc.Send[int](win, selWindowNumber))
		cocoaWindows[id] = win
		cocoaDelegates[id] = delegate
		wid = windowId(id)
	})
	return wid
}

func runEventLoop() {
	withAutoreleasePool(func() {
		objc.ID(clsNSApplication).Send(selSharedApplication).Send(selRun)
	})
}

// closeWindowById closes the window at once. "close", unlike "performClose:"
// (a simulated click on the close button), does not ask windowShouldClose:,
// so a programmatic Close does not run OnClose - the same as on Windows and
// Linux. windowWillClose: is still sent, ending a modal session as usual.
func closeWindowById(id windowId) {
	if win, ok := cocoaWindows[int(id)]; ok {
		win.Send(selClose)
	}
}

// quitApp is deferred a tick (dispatch_async) so it doesn't reenter while
// the caller's own window-close sequence is still on the stack (called from
// go_on_window_will_close, i.e. from nuiWindowWillClose).
func quitApp() {
	dispatchAsyncMain(func() {
		objc.ID(clsNSApplication).Send(selSharedApplication).Send(selTerminate, objc.ID(0))
	})
}

// showWindow runs directly when already on the main thread (e.g. the first
// window, shown before RunEventLoop starts - dispatch_sync would deadlock).
// A non-modal window's own goroutine calls this from a background thread,
// where runOnMainSync's dispatch_sync is safe.
func showWindow(id windowId) {
	runOnMainSync(func() {
		win, ok := cocoaWindows[int(id)]
		if !ok {
			return
		}
		win.Send(selMakeKeyAndOrderFront, objc.ID(0))
		objc.ID(clsNSApplication).Send(selSharedApplication).Send(selActivateIgnoringOtherApps, true)

		// Deferred so layout/tab bar etc. settle; fires one client-size sync to Go.
		dispatchAsyncMain(func() {
			w, ok := cocoaWindows[int(id)]
			if !ok {
				return
			}
			reportClientSizeToGo(id, w)
		})
	})
}

// showModalWindow shows an app-modal dialog. runModalForWindow: is called
// synchronously (nesting a modal session directly on the call stack) rather
// than deferred: a dispatched call can stall/misorder while an outer modal
// session is already running, which broke modal-on-modal (the intermediate
// window stayed key/frontmost and activatable). This makes ShowModal a
// blocking call on macOS only - it returns once the dialog closes. Like
// window.m, this assumes it's already called from the right thread (no
// runOnMainSync wrapper).
func showModalWindow(id, parentID windowId) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	win.Send(selMakeKeyAndOrderFront, objc.ID(0))
	nsApp := objc.ID(clsNSApplication).Send(selSharedApplication)
	nsApp.Send(selActivateIgnoringOtherApps, true)
	reportClientSizeToGo(id, win)
	nsApp.Send(selRunModalForWindow, win)
}

func hideWindow(id windowId) {
	runOnMainSync(func() {
		if win, ok := cocoaWindows[int(id)]; ok {
			win.Send(selOrderOut, objc.ID(0))
		}
	})
}

func updateWindow(id windowId) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	view := objc.Send[objc.ID](win, selContentView)
	view.Send(selSetNeedsDisplay, true)
}

/////////////////////////////////////////////////////
// Window appearance

func setWindowTitle(id windowId, title string) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	win.Send(selSetTitle, goStringToNS(title))
}

func setAppIconFromRGBA(pix []byte, width, height int) {
	if len(pix) == 0 || width <= 0 || height <= 0 {
		return
	}
	withAutoreleasePool(func() {
		image := newNSImageFromRGBA(pix, width, height, float64(width), float64(height))
		if image == 0 {
			return
		}
		objc.ID(clsNSApplication).Send(selSharedApplication).Send(selSetApplicationIconImage, image)
		image.Send(selRelease)
	})
}

// newNSImageFromRGBA makes an NSImage (+1 retained, the caller releases it)
// of the pixels, drawn at the size in points
func newNSImageFromRGBA(pix []byte, width, height int, pointsW, pointsH float64) objc.ID {
	colorSpaceName := goStringToNS("NSCalibratedRGBColorSpace")
	bitmapRep := objc.ID(clsNSBitmapImageRep).Send(selAlloc)
	bitmapRep = bitmapRep.Send(selInitWithBitmapDataPlanes,
		uintptr(0), width, height, 8, 4, true, false, colorSpaceName, width*4, 32)
	if bitmapRep == 0 {
		return 0
	}

	dataPtr := objc.Send[uintptr](bitmapRep, selBitmapData)
	dst := unsafe.Slice((*byte)(unsafe.Pointer(dataPtr)), width*height*4)
	copy(dst, pix)

	image := objc.ID(clsNSImage).Send(selAlloc).Send(selInitWithSize, nsSize{pointsW, pointsH})
	image.Send(selAddRepresentation, bitmapRep)
	bitmapRep.Send(selRelease)
	return image
}

// setMacCursor: cursorType mirrors native_window_darwin.go's macSetMouseCursor switch.
func setMacCursor(cursorType int) {
	var cursor objc.ID
	switch cursorType {
	case 1:
		cursor = objc.ID(clsNSCursor).Send(selArrowCursor)
	case 2:
		cursor = objc.ID(clsNSCursor).Send(selPointingHandCursor)
	case 3:
		cursor = objc.ID(clsNSCursor).Send(selResizeLeftRightCursor)
	case 4:
		cursor = objc.ID(clsNSCursor).Send(selResizeUpDownCursor)
	case 5:
		cursor = objc.ID(clsNSCursor).Send(selIBeamCursor)
	default:
		cursor = objc.ID(clsNSCursor).Send(selArrowCursor)
	}
	cursor.Send(selSet)
}

/////////////////////////////////////////////////////
// Window position and size

func setWindowPosition(id windowId, x, y int) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	frame := objc.Send[nsRect](win, selFrame)
	screen := objc.Send[objc.ID](win, selScreen)
	screenFrame := objc.Send[nsRect](screen, selFrame)
	newY := screenFrame.Size.Height - float64(y) - frame.Size.Height
	newFrame := nsRect{nsPoint{float64(x), newY}, frame.Size}
	win.Send(selSetFrameDisplayAnimate, newFrame, true, false)
}

func setWindowSize(id windowId, width, height int) {
	win, ok := cocoaWindows[int(id)]
	if !ok || width <= 0 || height <= 0 {
		return
	}
	win.Send(selSetContentSize, nsSize{float64(width), float64(height)})
}

// setWindowMinSize keeps the user from making the content of the window
// smaller than width x height
func setWindowMinSize(id windowId, width, height int) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	win.Send(selSetContentMinSize, nsSize{float64(max(width, 1)), float64(max(height, 1))})
}

func minimizeWindow(id windowId) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	objc.ID(clsNSApplication).Send(selSharedApplication).Send(selActivateIgnoringOtherApps, true)
	win.Send(selMakeKeyAndOrderFront, objc.ID(0))
	win.Send(selMiniaturize, objc.ID(0))
}

func maximizeWindow(id windowId) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	if !objc.Send[bool](win, selIsZoomed) {
		win.Send(selZoom, objc.ID(0))
	}
}

// setWindowAlwaysOnTop puts the window on the floating level (above normal windows)
func setWindowAlwaysOnTop(id windowId, onTop bool) {
	runOnMainSync(func() {
		win, ok := cocoaWindows[int(id)]
		if !ok {
			return
		}
		level := 0 // NSNormalWindowLevel
		if onTop {
			level = 3 // NSFloatingWindowLevel
		}
		win.Send(objc.RegisterName("setLevel:"), level)
	})
}

// setWindowDarkMode gives the window the dark or light (Aqua) appearance,
// which is what its title bar is drawn with.
func setWindowDarkMode(id windowId, dark bool) {
	runOnMainSync(func() {
		win, ok := cocoaWindows[int(id)]
		if !ok {
			return
		}
		name := "NSAppearanceNameAqua"
		if dark {
			name = "NSAppearanceNameDarkAqua"
		}
		withAutoreleasePool(func() {
			appearance := objc.ID(objc.GetClass("NSAppearance")).Send(objc.RegisterName("appearanceNamed:"), goStringToNS(name))
			win.Send(objc.RegisterName("setAppearance:"), appearance)
		})
	})
}

// requestUserAttention bounces the dock icon once (NSInformationalRequest)
func requestUserAttention() {
	runOnMainSync(func() {
		objc.ID(clsNSApplication).Send(selSharedApplication).Send(objc.RegisterName("requestUserAttention:"), 10)
	})
}

func systemBeep() {
	runOnMainSync(func() { nsBeepFn() })
}

// restoreWindow un-zooms a zoomed window; zoom: toggles, so it is only sent when zoomed
func restoreWindow(id windowId) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return
	}
	if objc.Send[bool](win, selIsZoomed) {
		win.Send(selZoom, objc.ID(0))
	}
}

// isKeyWindow: the window gets the keyboard input (it is active and its
// application is in front)
func isKeyWindow(id windowId) bool {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return false
	}
	return objc.Send[bool](win, selIsKeyWindow)
}

func isWindowZoomed(id windowId) bool {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return false
	}
	return objc.Send[bool](win, selIsZoomed)
}

// setWindowAllowMinimize toggles NSWindowStyleMaskMiniaturizable, which
// governs both the miniaturize button's presence AND whether Cmd+M/the
// Window menu can miniaturize.
func setWindowAllowMinimize(id windowId, allow bool) {
	runOnMainSync(func() {
		win, ok := cocoaWindows[int(id)]
		if !ok {
			return
		}
		mask := objc.Send[uint64](win, selStyleMask)
		if allow {
			mask |= nsWindowStyleMaskMiniaturizable
		} else {
			mask &^= uint64(nsWindowStyleMaskMiniaturizable)
		}
		win.Send(selSetStyleMask, mask)
	})
}

// setWindowAllowMaximize: unlike miniaturize, AppKit ties the zoom button's
// enabled state to NSWindowStyleMaskResizable - there's no bit for "just the
// zoom button" that leaves edge-drag resizing alone - so this just hides it.
func setWindowAllowMaximize(id windowId, allow bool) {
	runOnMainSync(func() {
		win, ok := cocoaWindows[int(id)]
		if !ok {
			return
		}
		btn := objc.Send[objc.ID](win, selStandardWindowButton, nsWindowZoomButton)
		if btn != 0 {
			btn.Send(selSetHidden, !allow)
		}
	})
}

//////////////////////////////////////////////////
// Window information

func getWindowPositionX(id windowId) int {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return -1
	}
	frame := objc.Send[nsRect](win, selFrame)
	return int(frame.Origin.X)
}

func getWindowPositionY(id windowId) int {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return -1
	}
	return windowTopOriginY(win)
}

// getWindowWidth/Height: client/layout content size (same numbers as Resize
// / WM_SIZE parity on Windows).
func getWindowWidth(id windowId) int {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return -1
	}
	lr := objc.Send[nsRect](win, selContentLayoutRect)
	return int(math.Floor(lr.Size.Width))
}

func getWindowHeight(id windowId) int {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return -1
	}
	lr := objc.Send[nsRect](win, selContentLayoutRect)
	return int(math.Floor(lr.Size.Height))
}

// getClientOrigin: top-left of the client area (contentLayoutRect, the same
// rect mouse coordinates are relative to) in global top-down coordinates
// (see cocoaToTopDownY).
func getClientOrigin(id windowId) (int, int) {
	win, ok := cocoaWindows[int(id)]
	if !ok {
		return 0, 0
	}
	frame := objc.Send[nsRect](win, selFrame)
	lr := objc.Send[nsRect](win, selContentLayoutRect)
	top := frame.Origin.Y + lr.Origin.Y + lr.Size.Height
	return int(frame.Origin.X + lr.Origin.X), int(cocoaToTopDownY(top))
}

// primaryScreenHeight is the height of the screen with the menu bar
// (screens[0]): Cocoa's global coordinates have their origin at its
// bottom-left corner, on every monitor.
func primaryScreenHeight() float64 {
	screens := objc.ID(clsNSScreen).Send(selScreens)
	if objc.Send[int](screens, selCount) == 0 {
		return float64(getScreenHeight())
	}
	primary := objc.Send[objc.ID](screens, selObjectAtIndex, 0)
	return objc.Send[nsRect](primary, selFrame).Size.Height
}

// cocoaToTopDownY flips a global Cocoa Y (up from the primary screen's
// bottom) to top-down, like on Windows/X11. The same formula converts back.
func cocoaToTopDownY(y float64) float64 {
	return primaryScreenHeight() - y
}

// getScreenWorkArea: visibleFrame (without menu bar and dock) of the screen
// containing the top-down point (x, y), or of the main screen.
func getScreenWorkArea(x, y int) (int, int, int, int) {
	pt := nsPoint{float64(x), cocoaToTopDownY(float64(y))}
	screen := objc.ID(clsNSScreen).Send(selMainScreen)
	screens := objc.ID(clsNSScreen).Send(selScreens)
	for i := 0; i < objc.Send[int](screens, selCount); i++ {
		s := objc.Send[objc.ID](screens, selObjectAtIndex, i)
		f := objc.Send[nsRect](s, selFrame)
		if pt.X >= f.Origin.X && pt.X < f.Origin.X+f.Size.Width &&
			pt.Y >= f.Origin.Y && pt.Y < f.Origin.Y+f.Size.Height {
			screen = s
			break
		}
	}
	vf := objc.Send[nsRect](screen, selVisibleFrame)
	top := cocoaToTopDownY(vf.Origin.Y + vf.Size.Height)
	return int(vf.Origin.X), int(top), int(vf.Size.Width), int(vf.Size.Height)
}

// getClientAreaWidth/Height: identical to getWindowWidth/Height on Darwin.
func getClientAreaWidth(id windowId) int  { return getWindowWidth(id) }
func getClientAreaHeight(id windowId) int { return getWindowHeight(id) }

// getScreenWidth/Height: NSScreen.mainScreen frame in points (used for
// MoveToCenter in Go).
func getScreenWidth() int {
	screen := objc.ID(clsNSScreen).Send(selMainScreen)
	frame := objc.Send[nsRect](screen, selFrame)
	return int(frame.Size.Width)
}

func getScreenHeight() int {
	screen := objc.ID(clsNSScreen).Send(selMainScreen)
	frame := objc.Send[nsRect](screen, selFrame)
	return int(frame.Size.Height)
}

/////////////////////////////////////////////////////
// Timers

func startTimer(id windowId, intervalMs float64) {
	stopTimer(id)

	block := objc.NewBlock(func(_ objc.Block, _ objc.ID) {
		go_on_timer(id)
	})
	timer := objc.ID(clsNSTimer).Send(selTimerWithTimeIntervalRepeatsBlock, intervalMs/1000.0, true, uintptr(block))
	timer.Send(selRetain) // NSTimer's own block-copy is independent of ours; see dispatchAsyncMain's comment.
	block.Release()

	mainRunLoop := objc.ID(clsNSRunLoop).Send(selMainRunLoop)
	// Timers scheduled only in the default run loop mode pause during modal/
	// tracking loops (e.g. window dragging/resizing); common modes doesn't.
	mainRunLoop.Send(selAddTimerForMode, timer, goStringToNS("kCFRunLoopCommonModes"))

	cocoaTimers[int(id)] = timer
}

func stopTimer(id windowId) {
	wid := int(id)
	if timer, ok := cocoaTimers[wid]; ok {
		delete(cocoaTimers, wid)
		timer.Send(selInvalidate)
		timer.Send(selRelease)
	}
}

/////////////////////////////////////////////////////
// File dialogs

func applyPanelCommonOptions(panel objc.ID, parentID windowId, title, dir string) {
	if title != "" {
		panel.Send(selSetTitle, goStringToNS(title))
	}
	if dir != "" {
		url := objc.ID(clsNSURL).Send(selFileURLWithPath, goStringToNS(dir))
		panel.Send(selSetDirectoryURL, url)
	}
	// runModal is app-modal (not attached to a specific window), but bring the
	// logical owner frontmost first so the panel doesn't appear behind it.
	if parentID >= 0 {
		if owner, ok := cocoaWindows[int(parentID)]; ok {
			owner.Send(selMakeKeyAndOrderFront, objc.ID(0))
		}
	}
	objc.ID(clsNSApplication).Send(selSharedApplication).Send(selActivateIgnoringOtherApps, true)
}

func setPanelAllowedExtensions(panel objc.ID, exts []string) {
	if len(exts) == 0 {
		return
	}
	arr := objc.ID(clsNSMutableArray).Send(selAlloc).Send(selInit)
	for _, e := range exts {
		arr.Send(selAddObject, goStringToNS(e))
	}
	panel.Send(selSetAllowedFileTypes, arr)
	arr.Send(selRelease)
}

func showOpenFileDialog(parentID windowId, title, dir string, exts []string, allowMultiple bool) []string {
	var result []string
	runOnMainSync(func() {
		withAutoreleasePool(func() {
			panel := objc.ID(clsNSOpenPanel).Send(selOpenPanel)
			panel.Send(selSetCanChooseFiles, true)
			panel.Send(selSetCanChooseDirectories, false)
			panel.Send(selSetAllowsMultipleSelection, allowMultiple)
			applyPanelCommonOptions(panel, parentID, title, dir)
			setPanelAllowedExtensions(panel, exts)

			if objc.Send[int](panel, selRunModal) == nsModalResponseOK {
				urls := objc.Send[objc.ID](panel, selURLs)
				count := objc.Send[int](urls, selCount)
				for i := 0; i < count; i++ {
					u := objc.Send[objc.ID](urls, selObjectAtIndex, i)
					p := objc.Send[objc.ID](u, selPath)
					result = append(result, nsStringToGo(p))
				}
			}
		})
	})
	return result
}

func showSaveFileDialog(parentID windowId, title, dir, defaultName string, exts []string) string {
	var result string
	runOnMainSync(func() {
		withAutoreleasePool(func() {
			panel := objc.ID(clsNSSavePanel).Send(selSavePanel)
			panel.Send(selSetCanCreateDirectories, true)
			applyPanelCommonOptions(panel, parentID, title, dir)
			if defaultName != "" {
				panel.Send(selSetNameFieldStringValue, goStringToNS(defaultName))
			}
			setPanelAllowedExtensions(panel, exts)

			if objc.Send[int](panel, selRunModal) == nsModalResponseOK {
				u := objc.Send[objc.ID](panel, selURL)
				p := objc.Send[objc.ID](u, selPath)
				result = nsStringToGo(p)
			}
		})
	})
	return result
}

func showSelectDirectoryDialog(parentID windowId, title, dir string) string {
	var result string
	runOnMainSync(func() {
		withAutoreleasePool(func() {
			panel := objc.ID(clsNSOpenPanel).Send(selOpenPanel)
			panel.Send(selSetCanChooseFiles, false)
			panel.Send(selSetCanChooseDirectories, true)
			panel.Send(selSetCanCreateDirectories, true)
			panel.Send(selSetAllowsMultipleSelection, false)
			applyPanelCommonOptions(panel, parentID, title, dir)

			if objc.Send[int](panel, selRunModal) == nsModalResponseOK {
				u := objc.Send[objc.ID](panel, selURL)
				p := objc.Send[objc.ID](u, selPath)
				result = nsStringToGo(p)
			}
		})
	})
	return result
}
