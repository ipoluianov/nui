// Package monitor is the Network Monitor example: a dashboard that pings a
// few simulated servers once a second and charts the results.
package monitor

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

// NewForm creates the Network Monitor: a table of hosts with their live
// status (cell images and colors), time charts of latency, jitter and traffic
// for the selected hosts - select several (Ctrl+click, Ctrl+A) to compare
// them - with areas, peak markers, candles, gaps and a live default range,
// a toolbar with tool buttons, toggle switches and a combo box, an "Add host"
// dialog, a confirmation message box, toasts for alerts, an event log and a
// status bar with a progress bar.
func NewForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Network Monitor")
	form.SetSize(1200, 760)
	form.Panel().AddWidget(0, 0, newMonitor())
	return form
}

// Window sizes the charts can show while live.
var windows = []struct {
	name string
	d    time.Duration
}{
	{"1 min", time.Minute},
	{"5 min", 5 * time.Minute},
	{"15 min", 15 * time.Minute},
	{"1 h", time.Hour},
}

type monitor struct {
	ui.Widget

	hosts      []*host
	lastSample time.Time
	window     time.Duration
	lastToast  time.Time

	btnRemove *ui.ToolButton
	swLive    *ui.ToggleSwitch
	swAlerts  *ui.ToggleSwitch
	tblHosts  *ui.Table
	charts    *charts
	events    *eventLog

	lblHosts     *ui.Label
	pbAvail      *ui.ProgressBar
	lblLastCheck *ui.Label
}

func newMonitor() *monitor {
	var c monitor
	c.InitWidget()
	c.window = 5 * time.Minute

	c.hosts = []*host{
		newHost("gateway.local", 2, profiles[0]),
		newHost("nas.local", 3, profiles[0]),
		newHost("api.example.de", 25, profiles[1]),
		newHost("mail.example.fr", 38, profiles[1]),
		newHost("cdn.example.com", 95, profiles[2]),
		newHost("backup.example.sg", 180, profiles[2]),
	}

	c.buildToolbar()

	c.tblHosts = ui.NewTable()
	c.tblHosts.SetColumnCount(5)
	for i, col := range []struct {
		name  string
		width int
	}{{"Status", 80}, {"Host", 150}, {"Ping", 70}, {"Avg 30 s", 70}, {"Loss 5 min", 80}} {
		c.tblHosts.SetColumnName(i, col.name)
		c.tblHosts.SetColumnWidth(i, col.width)
	}
	c.tblHosts.SetMultiselect(true)
	c.tblHosts.SetOnSelectionChanged(func(row, col int) { c.showSelected() })

	c.charts = newCharts()
	c.events = newEventLog()
	tabs := ui.NewTabWidget()
	tabs.AddPage("Latency", c.charts.latency)
	tabs.AddPage("Jitter", c.charts.jitter)
	tabs.AddPage("Traffic", c.charts.traffic)
	tabs.AddPage("Events", c.events.table)

	split := ui.NewHSplitter()
	split.SetWidgets(c.tblHosts, tabs)
	split.SetFirstSize(480)
	c.AddWidget(1, 0, split)

	c.buildStatusBar()

	// An hour of history, as if the monitor had been running for a while.
	now := time.Now().Truncate(sampleInterval)
	c.lastSample = now.Add(-historyLength)
	c.sampleUntil(now, false)

	c.updateHosts()
	c.tblHosts.SetCurrentCell2(0, 1)
	c.showSelected()
	c.updateStatus()

	c.AddTimer(250, c.onTimer)
	return &c
}

func (c *monitor) buildToolbar() {
	bar := c.AddPanel(0, 0)
	bar.SetPanelPadding(0)

	const btnSize = 36
	add := ui.NewToolButton(icons.Plus(24), "Add host...", c.addHost)
	add.SetButtonSize(btnSize, btnSize)
	bar.AddWidget(0, 0, add)
	c.btnRemove = ui.NewToolButton(icons.Cross(24), "Remove the selected hosts", c.removeHosts)
	c.btnRemove.SetButtonSize(btnSize, btnSize)
	bar.AddWidget(0, 1, c.btnRemove)
	bar.AddHSpacer(0, 2)

	c.swLive = ui.NewToggleSwitch("Live")
	c.swLive.SetChecked(true)
	c.swLive.SetTooltip("Pause or resume sampling")
	c.swLive.SetOnStateChanged(c.onLiveChanged)
	bar.AddWidget(0, 3, c.swLive)

	bar.AddLabel(0, 4, "Window:")
	cbWindow := ui.NewComboBox()
	for i, w := range windows {
		cbWindow.AddItem(w.name, w.d)
		if w.d == c.window {
			cbWindow.SetSelectedIndex(i)
		}
	}
	cbWindow.SetOnSelectedIndexChanged(func() {
		c.window = cbWindow.SelectedItemData().(time.Duration)
		c.charts.setRange(c.lastSample, c.window)
	})
	bar.AddWidget(0, 5, cbWindow)

	c.swAlerts = ui.NewToggleSwitch("Alerts")
	c.swAlerts.SetChecked(true)
	c.swAlerts.SetTooltip("Pop up a notification when a host goes down or responds slowly")
	bar.AddWidget(0, 6, c.swAlerts)
}

func (c *monitor) buildStatusBar() {
	bar := c.AddPanel(2, 0)
	bar.SetPanelPadding(0)
	c.lblHosts = bar.AddLabel(0, 0, "")
	bar.AddHSpacer(0, 1)
	c.lblLastCheck = bar.AddLabel(0, 2, "")
	bar.AddLabel(0, 3, "   Availability, 1 h:")
	c.pbAvail = ui.NewProgressBar(0, 100, 100)
	c.pbAvail.SetMinWidth(200)
	c.pbAvail.SetMaxWidth(200)
	bar.AddWidget(0, 4, c.pbAvail)
}

func (c *monitor) onTimer() {
	if !c.swLive.Checked() {
		return
	}
	before := c.lastSample
	c.sampleUntil(time.Now(), true)
	if c.lastSample.Equal(before) {
		return
	}
	c.charts.setRange(c.lastSample, c.window)
	c.updateHosts()
	c.updateStatus()
	c.Form().Update()
}

// sampleUntil takes a sample of every host for each interval up to t.
// live tells whether the samples are new, so alerts may pop up.
func (c *monitor) sampleUntil(t time.Time, live bool) {
	for next := c.lastSample.Add(sampleInterval); !next.After(t); next = next.Add(sampleInterval) {
		for _, h := range c.hosts {
			res := h.sample(next)
			switch {
			case res.wentDown:
				c.alert(live, next, h, severityError, "Host is not responding")
			case res.recovered:
				c.alert(live, next, h, severityOK, fmt.Sprintf("Host is back after %v", res.downFor))
			case res.spike:
				c.alert(live, next, h, severityWarning,
					fmt.Sprintf("Latency spike: %.0f ms (threshold %.0f ms)", h.lastPing, h.alertThreshold()))
			}
		}
		c.lastSample = next
	}
}

func (c *monitor) onLiveChanged() {
	if c.swLive.Checked() {
		// Nothing was measured while paused: break the lines there.
		now := time.Now().Truncate(sampleInterval)
		for _, h := range c.hosts {
			h.addGap(c.lastSample.Add(sampleInterval))
		}
		c.events.add(now, "", severityInfo, "Monitoring resumed")
		c.lastSample = now
	} else {
		c.events.add(c.lastSample, "", severityInfo, "Monitoring paused")
	}
	c.updateStatus()
}

// selectedHosts returns the hosts of the selected table rows.
func (c *monitor) selectedHosts() []*host {
	var result []*host
	for _, row := range c.tblHosts.SelectedRows() {
		if row >= 0 && row < len(c.hosts) {
			result = append(result, c.hosts[row])
		}
	}
	return result
}

func (c *monitor) showSelected() {
	selected := c.selectedHosts()
	c.btnRemove.SetEnabled(len(selected) > 0)
	c.charts.show(c.hosts, selected)
	c.charts.setRange(c.lastSample, c.window)
}

var (
	colorOK   = ui.ColorFromHex("#43A047")
	colorSlow = ui.ColorFromHex("#FB8C00")
	colorDown = ui.ColorFromHex("#E53935")

	dotOK   = statusDot("#43A047")
	dotSlow = statusDot("#FB8C00")
	dotDown = statusDot("#E53935")
)

const dotSize = 16

// statusDot is a colored circle for the Status column.
func statusDot(hex string) image.Image {
	return icons.Draw(dotSize, func(dc *gg.Context) {
		dc.DrawCircle(8, 8, 5)
		dc.SetHexColor(hex)
		dc.Fill()
	})
}

// updateHosts fills the table with the latest state of the hosts.
func (c *monitor) updateHosts() {
	t := c.tblHosts
	t.SetRowCount(len(c.hosts))
	for row, h := range c.hosts {
		status, col, dot := "OK", color.Color(colorOK), dotOK
		ping := fmt.Sprintf("%.1f", h.lastPing)
		switch {
		case !h.lastOK:
			status, col, dot, ping = "Down", colorDown, dotDown, "timeout"
		case h.lastPing > h.slowThreshold():
			status, col, dot = "Slow", colorSlow, dotSlow
		}
		t.SetCellImage(row, 0, dot, dotSize)
		t.SetCellText2(row, 0, status)
		// Only the problems stand out: a colored text is hard to read on
		// the selection, which the cell color overrides.
		if status == "OK" {
			col = nil
		}
		t.SetCellColor(row, 0, col)
		t.SetCellText2(row, 1, h.name)
		t.SetCellText2(row, 2, ping)
		t.SetCellText2(row, 3, fmt.Sprintf("%.1f", h.average()))
		t.SetCellText2(row, 4, fmt.Sprintf("%.1f %%", h.lossPercent(lossWindow)))
		for col := 2; col <= 4; col++ {
			t.SetCellHAlign(row, col, ui.HAlignRight)
		}
		if !h.lastOK {
			t.SetCellColor(row, 2, colorDown)
		} else {
			t.SetCellColor(row, 2, nil)
		}
	}
}

func (c *monitor) updateStatus() {
	up, slow, down := 0, 0, 0
	avail := 0.0
	for _, h := range c.hosts {
		switch {
		case !h.lastOK:
			down++
		case h.lastPing > h.slowThreshold():
			slow++
		default:
			up++
		}
		avail += 100 - h.lossPercent(maxReplies)
	}
	if len(c.hosts) > 0 {
		avail /= float64(len(c.hosts))
	} else {
		avail = 100
	}
	c.lblHosts.SetText(fmt.Sprintf("Hosts: %d up, %d slow, %d down", up, slow, down))
	c.pbAvail.SetValue(avail)
	c.pbAvail.SetText(fmt.Sprintf("%.2f %%", avail))
	if c.swLive.Checked() {
		c.lblLastCheck.SetText("Last check " + c.lastSample.Format("15:04:05"))
	} else {
		c.lblLastCheck.SetText("Paused at " + c.lastSample.Format("15:04:05"))
	}
}
