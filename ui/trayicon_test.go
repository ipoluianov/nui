package ui

import "testing"

// A click on the tray icon calls the old setter's function and the signal
func TestTrayIconClick(t *testing.T) {
	tray := &TrayIcon{}
	setter, signal := 0, 0
	tray.SetOnClick(func() { setter++ })
	tray.OnClick.Connect(func(icon *TrayIcon) {
		if icon == tray {
			signal++
		}
	})
	tray.clicked()
	if setter != 1 || signal != 1 {
		t.Errorf("setter %d, signal %d", setter, signal)
	}
}
