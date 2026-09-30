# TrayIcon

An icon in the system tray - the notification area on Windows and Linux, the menu bar on macOS -
with a tooltip, a menu and a click action. It can also show system notifications.

```go
tray, err := ui.NewTrayIcon()
if err != nil {
	// ui.ErrTrayNotSupported: the desktop has no tray
}
tray.SetIcon(img) // a square image, 32x32 or 64x64
tray.SetTooltip("My app")
tray.SetOnClick(func() { form.Show() })
tray.SetMenu(
	ui.TrayMenuItem{Text: "Open", OnClick: func() { form.Show() }},
	ui.TrayMenuItem{Text: "Mute", Checkable: true, Checked: muted, OnClick: toggleMute},
	ui.TrayMenuItem{Text: "More", Items: []ui.TrayMenuItem{{Text: "About", OnClick: about}}},
	ui.TrayMenuItem{Separator: true},
	ui.TrayMenuItem{Text: "Quit", OnClick: func() { form.Close() }},
)
tray.ShowNotification("Done", "The export has finished")
```

- The callbacks run on the UI thread like all the handlers; the application must be running a
  form's `Exec`.
- The menu opens on a right click; on macOS any click opens it, and `SetOnClick` works only
  without a menu. Call `SetMenu` again to change the items (e.g. a check mark).
- `Close()` removes the icon.

Keeping the application in the tray when its window is closed:

```go
form.OnClose = func() bool {
	form.Hide() // Show() brings it back
	return false
}
```

Platforms:

- Windows: `Shell_NotifyIcon`; notifications are balloons (toasts on Windows 10+).
- Linux: a StatusNotifierItem on D-Bus - KDE Plasma, GNOME with the AppIndicator extension,
  XFCE, Cinnamon. Notifications go to `org.freedesktop.Notifications`.
- macOS: `NSStatusItem`. `ShowNotification` returns `ErrNotificationsNotSupported` (macOS allows
  notifications only from signed application bundles).
