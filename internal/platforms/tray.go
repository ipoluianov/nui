package platforms

import (
	"errors"
	"image"
)

// ErrTrayNotSupported is returned when the system has no tray (notification
// area) to put an icon in
var ErrTrayNotSupported = errors.New("nui: the system tray is not available")

// ErrNotificationsNotSupported is returned by ShowNotification where the
// platform can't show system notifications
var ErrNotificationsNotSupported = errors.New("nui: system notifications are not available")

// TrayMenuItem is an item of the tray icon's menu
type TrayMenuItem struct {
	Text      string
	OnClick   func()
	Disabled  bool
	Separator bool
	// Checkable items show a check box, checked when Checked is set
	Checkable bool
	Checked   bool
	// Items makes the item a submenu
	Items []TrayMenuItem
}

// TrayIcon is an icon in the system tray (the notification area of the
// taskbar, or the menu bar on macOS) with a tooltip and a menu. Its callbacks
// are called on the UI thread; call its methods there too.
type TrayIcon interface {
	SetIcon(icon *image.RGBA)
	SetTooltip(text string)
	// SetMenu sets the menu shown on a right click (on macOS, on any click)
	SetMenu(items []TrayMenuItem)
	// OnClick sets the function called on a left click
	OnClick(f func())
	// ShowNotification shows a system notification from the application
	ShowNotification(title, text string) error
	// Close removes the icon
	Close()
}

// CreateTrayIcon puts an icon in the system tray
func CreateTrayIcon() (TrayIcon, error) {
	var icon TrayIcon
	var err error
	RunOnUIThread(func() {
		icon, err = createTrayIcon()
	})
	return icon, err
}

// trayMenuIndex numbers the menu items (depth first, from 1) so the
// platforms can refer to them by id
func trayMenuIndex(items []TrayMenuItem) map[int32]*TrayMenuItem {
	index := make(map[int32]*TrayMenuItem)
	var next int32 = 1
	var walk func(items []TrayMenuItem)
	walk = func(items []TrayMenuItem) {
		for i := range items {
			index[next] = &items[i]
			next++
			walk(items[i].Items)
		}
	}
	walk(items)
	return index
}
