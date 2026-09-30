package platforms

import (
	"sync/atomic"
	"syscall"
	"unsafe"
)

// c_WM_NUI_INVOKE asks the UI thread to run the posted functions (see Post)
const c_WM_NUI_INVOKE = c_WM_APP + 3

// hwndMessage is HWND_MESSAGE ((HWND)-3), the parent of message-only windows
const hwndMessage = ^uintptr(2)

var (
	uiThreadId uintptr

	// invokeHwnd is a hidden message-only window of the UI thread: posting
	// c_WM_NUI_INVOKE to it wakes the event loop to run the posted functions.
	// A window message, unlike a thread message, isn't lost while a modal
	// loop of the system (moving a window, a system dialog) runs.
	invokeHwnd uintptr

	// wakePending skips posting while a c_WM_NUI_INVOKE is already queued
	wakePending atomic.Bool
)

// initUIThread runs on the main OS thread, from the package init
func initUIThread() {
	uiThreadId, _, _ = procGetCurrentThreadId.Call()

	className, _ := syscall.UTF16PtrFromString("NUIInvokeWindow")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	wndClass := t_WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(t_WNDCLASSEXW{})),
		lpfnWndProc:   syscall.NewCallback(invokeWndProc),
		hInstance:     syscall.Handle(hInstance),
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))
	invokeHwnd, _, _ = procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0,
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		hInstance,
		0,
	)
	if invokeHwnd == 0 {
		panic("nui: unable to create the UI thread window")
	}
}

func isUIThread() bool {
	id, _, _ := procGetCurrentThreadId.Call()
	return id == uiThreadId
}

func wakeUIThread() {
	if wakePending.Swap(true) {
		return
	}
	procPostMessageW.Call(invokeHwnd, c_WM_NUI_INVOKE, 0, 0)
}

func invokeWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == c_WM_NUI_INVOKE {
		wakePending.Store(false)
		runPosted()
		return 0
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

// runLoopUntil is the message loop of all the windows, which all belong to
// the UI thread; it runs until done returns true. UI thread only. Loops may
// nest: Exec called from a callback runs one until its window is closed.
func runLoopUntil(done func() bool) {
	var msg t_MSG
	for !done() {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) == -1 {
			return
		}
		if ret == 0 {
			// WM_QUIT: posted again, so the outer loops end as well
			procPostQuitMessage.Call(msg.wParam)
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}
