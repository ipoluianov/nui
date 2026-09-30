package ui

import (
	"image"
	"image/draw"

	"github.com/ipoluianov/nui/internal/platforms"
)

// TrayMenuItem is an item of the tray icon's menu: a text with OnClick, a
// Separator, a Checkable item, or a submenu (Items).
type TrayMenuItem = platforms.TrayMenuItem

// ErrTrayNotSupported is returned by NewTrayIcon when the system has no tray,
// e.g. a Linux desktop without StatusNotifierItem support
var ErrTrayNotSupported = platforms.ErrTrayNotSupported

// ErrNotificationsNotSupported is returned by ShowNotification where system
// notifications aren't available (macOS)
var ErrNotificationsNotSupported = platforms.ErrNotificationsNotSupported

// TrayIcon is an icon in the system tray: the notification area on Windows
// and Linux, the menu bar on macOS. It has a tooltip, a menu (a right click;
// on macOS any click) and a left-click action, and can show system
// notifications. Its callbacks run on the UI thread, like all the handlers;
// the application must be running a form's Exec.
//
//	tray, err := ui.NewTrayIcon()
//	if err == nil {
//		tray.SetIcon(icon)
//		tray.SetTooltip("My app")
//		tray.SetOnClick(func() { form.Show() })
//		tray.SetMenu(
//			ui.TrayMenuItem{Text: "Open", OnClick: func() { form.Show() }},
//			ui.TrayMenuItem{Separator: true},
//			ui.TrayMenuItem{Text: "Quit", OnClick: func() { form.Close() }},
//		)
//	}
//
// To keep the application in the tray when its window is closed, hide the
// form instead: form.OnClose = func() bool { form.Hide(); return false }
type TrayIcon struct {
	icon platforms.TrayIcon
}

// NewTrayIcon puts an icon in the system tray. Set its image with SetIcon:
// until then, it shows the application icon (see SetAppIcon) if there is one.
func NewTrayIcon() (*TrayIcon, error) {
	icon, err := platforms.CreateTrayIcon()
	if err != nil {
		return nil, err
	}
	t := &TrayIcon{icon: icon}
	if appIcon != nil {
		t.icon.SetIcon(appIcon)
	}
	return t, nil
}

// SetIcon sets the image; a square one of 32x32 or 64x64 pixels looks best
func (t *TrayIcon) SetIcon(img image.Image) {
	if img == nil {
		return
	}
	rgba, ok := img.(*image.RGBA)
	if !ok || rgba.Bounds().Min != (image.Point{}) {
		rgba = image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
		draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	}
	t.icon.SetIcon(rgba)
}

func (t *TrayIcon) SetTooltip(text string) {
	t.icon.SetTooltip(text)
}

// SetOnClick sets the function called on a left click on the icon (on macOS,
// only when the icon has no menu)
func (t *TrayIcon) SetOnClick(f func()) {
	t.icon.OnClick(f)
}

// SetMenu sets the menu of the icon; call it again to change the items, e.g.
// the state of a checkable one
func (t *TrayIcon) SetMenu(items ...TrayMenuItem) {
	t.icon.SetMenu(items)
}

// ShowNotification shows a system notification (a balloon on Windows, a
// desktop notification on Linux)
func (t *TrayIcon) ShowNotification(title, text string) error {
	return t.icon.ShowNotification(title, text)
}

// Close removes the icon from the tray
func (t *TrayIcon) Close() {
	t.icon.Close()
}
