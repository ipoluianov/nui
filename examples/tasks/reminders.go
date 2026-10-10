package tasks

import (
	"time"

	"github.com/ipoluianov/nui/ui"
)

// initTray puts the icon in the tray. Without a tray (ErrTrayNotSupported,
// e.g. a desktop without StatusNotifierItem) the planner works without it.
func (p *planner) initTray() {
	tray, err := ui.NewTrayIcon()
	if err != nil {
		return
	}
	p.tray = tray
	tray.SetIcon(appIcon(32))
	tray.SetOnClick(p.showWindow)
	tray.SetMenu(
		ui.TrayMenuItem{Text: "Show window", OnClick: p.showWindow},
		ui.TrayMenuItem{Text: "Add task…", OnClick: func() {
			p.showWindow()
			p.addTaskDialog()
		}},
		ui.TrayMenuItem{Separator: true},
		ui.TrayMenuItem{Text: "Quit", OnClick: func() {
			// Close doesn't ask OnClose, which would hide the window again
			p.tray.Close()
			p.form.Close()
		}},
	)

	p.form.OnClose = func() bool {
		if p.minimizeToTray.Checked() {
			p.form.Hide()
			return false
		}
		p.tray.Close()
		return true
	}
}

func (p *planner) showWindow() {
	p.form.Show()
	p.form.RequestAttention()
}

// checkReminders runs every second: it reminds of the tasks that have
// become due
func (p *planner) checkReminders() {
	now := time.Now()
	for _, t := range p.tasks {
		if t.Done || !t.Reminder || t.reminded || t.Due.After(now) {
			continue
		}
		t.reminded = true
		p.form.ShowToastFor("Reminder: "+t.Title+" ("+t.Due.Format("15:04")+")", ui.ToastWarning, 8*time.Second)
		if p.tray != nil {
			// Not every desktop shows notifications: the toast is enough then
			_ = p.tray.ShowNotification("Task due", t.Title)
		}
		// The due times shown as overdue change color
		p.updateRow(t)
	}
}
