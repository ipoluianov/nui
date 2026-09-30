package platforms

import (
	"image"
	"syscall"
	"unsafe"
)

// The tray icon is a notification area icon (Shell_NotifyIconW). Its mouse
// messages go to a hidden top-level window of the UI thread (trayHwnd), so
// the callbacks run on the UI thread. Not a message-only window: those get
// no TaskbarCreated broadcast and can't be the foreground window, which the
// native popup menu needs to close on a click elsewhere.

var (
	procShellNotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procDestroyIcon            = user32.NewProc("DestroyIcon")
	procRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
)

const (
	c_WM_NUI_TRAY = c_WM_APP + 4

	c_NIM_ADD    = 0
	c_NIM_MODIFY = 1
	c_NIM_DELETE = 2

	c_NIF_MESSAGE = 0x01
	c_NIF_ICON    = 0x02
	c_NIF_TIP     = 0x04
	c_NIF_INFO    = 0x10
	c_NIIF_INFO   = 0x01

	c_MF_STRING    = 0x0000
	c_MF_GRAYED    = 0x0001
	c_MF_CHECKED   = 0x0008
	c_MF_POPUP     = 0x0010
	c_MF_SEPARATOR = 0x0800

	c_TPM_RIGHTBUTTON = 0x0002
	c_TPM_NONOTIFY    = 0x0080
	c_TPM_RETURNCMD   = 0x0100

	c_WM_NULL = 0x0000
)

// NOTIFYICONDATAW (Vista and later)
type notifyIconData struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

type trayIcon struct {
	id       uint32
	hIcon    uintptr
	tooltip  string
	onClick  func()
	menu     []TrayMenuItem
	menuByID map[int32]*TrayMenuItem
}

var (
	trayIcons      = map[uint32]*trayIcon{}
	nextTrayID     uint32
	taskbarCreated uintptr
	trayHwnd       uintptr
)

// createTrayWindow creates the hidden window that gets the tray messages
func createTrayWindow() {
	name, _ := syscall.UTF16PtrFromString("TaskbarCreated")
	taskbarCreated, _, _ = procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(name)))

	className, _ := syscall.UTF16PtrFromString("NUITrayWindow")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	wndClass := t_WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(t_WNDCLASSEXW{})),
		lpfnWndProc:   syscall.NewCallback(trayWindowProc),
		hInstance:     syscall.Handle(hInstance),
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))
	trayHwnd, _, _ = procCreateWindowExW.Call(
		c_WS_EX_TOOLWINDOW,
		uintptr(unsafe.Pointer(className)),
		0,
		c_WS_POPUP,
		0, 0, 0, 0,
		0,
		0,
		hInstance,
		0,
	)
}

func createTrayIcon() (TrayIcon, error) {
	if trayHwnd == 0 {
		createTrayWindow()
		if trayHwnd == 0 {
			return nil, ErrTrayNotSupported
		}
	}
	nextTrayID++
	t := &trayIcon{id: nextTrayID}
	if !t.notify(c_NIM_ADD, c_NIF_MESSAGE) {
		return nil, ErrTrayNotSupported
	}
	trayIcons[t.id] = t
	return t, nil
}

func (t *trayIcon) data(flags uint32) notifyIconData {
	d := notifyIconData{
		hWnd:             trayHwnd,
		uID:              t.id,
		uFlags:           flags,
		uCallbackMessage: c_WM_NUI_TRAY,
		hIcon:            t.hIcon,
	}
	d.cbSize = uint32(unsafe.Sizeof(d))
	if t.hIcon != 0 {
		d.uFlags |= c_NIF_ICON
	}
	if t.tooltip != "" {
		d.uFlags |= c_NIF_TIP
		copyUTF16(d.szTip[:], t.tooltip)
	}
	return d
}

func (t *trayIcon) notify(message uintptr, flags uint32) bool {
	d := t.data(flags)
	ret, _, _ := procShellNotifyIconW.Call(message, uintptr(unsafe.Pointer(&d)))
	return ret != 0
}

// copyUTF16 copies s into the fixed buffer, cut to fit with the terminating 0
func copyUTF16(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) {
		u = u[:len(dst)]
		u[len(u)-1] = 0
	}
	copy(dst, u)
}

func (t *trayIcon) SetIcon(icon *image.RGBA) {
	old := t.hIcon
	t.hIcon = uintptr(createHICONFromRGBA(icon))
	t.notify(c_NIM_MODIFY, c_NIF_MESSAGE)
	if old != 0 {
		procDestroyIcon.Call(old)
	}
}

func (t *trayIcon) SetTooltip(text string) {
	t.tooltip = text
	t.notify(c_NIM_MODIFY, c_NIF_MESSAGE)
}

func (t *trayIcon) OnClick(f func()) {
	t.onClick = f
}

func (t *trayIcon) SetMenu(items []TrayMenuItem) {
	t.menu = items
	t.menuByID = trayMenuIndex(items)
}

// ShowNotification shows a balloon (a toast on Windows 10 and later) from the icon
func (t *trayIcon) ShowNotification(title, text string) error {
	d := t.data(c_NIF_INFO)
	copyUTF16(d.szInfoTitle[:], title)
	copyUTF16(d.szInfo[:], text)
	d.dwInfoFlags = c_NIIF_INFO
	ret, _, _ := procShellNotifyIconW.Call(c_NIM_MODIFY, uintptr(unsafe.Pointer(&d)))
	if ret == 0 {
		return ErrNotificationsNotSupported
	}
	return nil
}

func (t *trayIcon) Close() {
	if _, ok := trayIcons[t.id]; !ok {
		return
	}
	t.notify(c_NIM_DELETE, 0)
	delete(trayIcons, t.id)
	if t.hIcon != 0 {
		procDestroyIcon.Call(t.hIcon)
		t.hIcon = 0
	}
}

func trayWindowProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	if taskbarCreated != 0 && uintptr(msg) == taskbarCreated {
		// Explorer restarted: the icons are gone from the new taskbar
		for _, t := range trayIcons {
			t.notify(c_NIM_ADD, c_NIF_MESSAGE)
		}
		return 0
	}
	if msg != c_WM_NUI_TRAY {
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}
	t, ok := trayIcons[uint32(wParam)]
	if !ok {
		return 0
	}
	switch uint32(lParam) {
	case c_WM_LBUTTONUP:
		if t.onClick != nil {
			t.onClick()
		}
	case c_WM_RBUTTONUP:
		t.showMenu()
	}
	return 0
}

func (t *trayIcon) showMenu() {
	if len(t.menu) == 0 {
		return
	}
	var id int32 = 1
	menu := buildWin32Menu(t.menu, &id)
	defer procDestroyMenu.Call(menu)

	var pt struct{ x, y int32 }
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	// Without the foreground the menu doesn't close on a click elsewhere
	procSetForegroundWindow.Call(trayHwnd)
	cmd, _, _ := procTrackPopupMenu.Call(menu, c_TPM_RETURNCMD|c_TPM_RIGHTBUTTON|c_TPM_NONOTIFY,
		uintptr(pt.x), uintptr(pt.y), 0, trayHwnd, 0)
	procPostMessageW.Call(trayHwnd, c_WM_NULL, 0, 0)

	if item, ok := t.menuByID[int32(cmd)]; ok && item.OnClick != nil && !item.Disabled {
		item.OnClick()
	}
}

// buildWin32Menu numbers the items the same way as trayMenuIndex
func buildWin32Menu(items []TrayMenuItem, id *int32) uintptr {
	menu, _, _ := procCreatePopupMenu.Call()
	for i := range items {
		item := &items[i]
		itemID := *id
		*id++
		if item.Separator {
			procAppendMenuW.Call(menu, c_MF_SEPARATOR, 0, 0)
			continue
		}
		text, _ := syscall.UTF16PtrFromString(item.Text)
		flags := uintptr(c_MF_STRING)
		if item.Disabled {
			flags |= c_MF_GRAYED
		}
		if item.Checkable && item.Checked {
			flags |= c_MF_CHECKED
		}
		if len(item.Items) > 0 {
			submenu := buildWin32Menu(item.Items, id)
			procAppendMenuW.Call(menu, flags|c_MF_POPUP, submenu, uintptr(unsafe.Pointer(text)))
			continue
		}
		procAppendMenuW.Call(menu, flags, uintptr(itemID), uintptr(unsafe.Pointer(text)))
	}
	return menu
}
