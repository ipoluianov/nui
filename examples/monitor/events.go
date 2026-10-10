package monitor

import (
	"image/color"
	"time"

	"github.com/ipoluianov/nui/ui"
)

type severity int

const (
	severityInfo severity = iota
	severityOK
	severityWarning
	severityError
)

func (s severity) String() string {
	return [...]string{"Info", "OK", "Warning", "Error"}[s]
}

func (s severity) color() color.Color {
	switch s {
	case severityOK:
		return colorOK
	case severityWarning:
		return colorSlow
	case severityError:
		return colorDown
	}
	return nil
}

const (
	maxEvents  = 1000
	toastEvery = 5 * time.Second // at most one alert toast per this time
)

type event struct {
	dt       time.Time
	host     string
	severity severity
	text     string
}

// eventLog is the Events tab: a table of the events, the newest on top.
type eventLog struct {
	table  *ui.Table
	events []event
}

func newEventLog() *eventLog {
	c := eventLog{table: ui.NewTable()}
	c.table.SetColumnCount(4)
	for i, col := range []struct {
		name  string
		width int
	}{{"Time", 140}, {"Host", 150}, {"Severity", 80}, {"Event", 380}} {
		c.table.SetColumnName(i, col.name)
		c.table.SetColumnWidth(i, col.width)
	}
	return &c
}

func (c *eventLog) add(dt time.Time, host string, s severity, text string) {
	c.events = append(c.events, event{dt, host, s, text})
	if len(c.events) > maxEvents {
		c.events = c.events[1:]
	}
	c.refresh()
}

func (c *eventLog) refresh() {
	t := c.table
	t.SetRowCount(len(c.events))
	for row := range c.events {
		e := c.events[len(c.events)-1-row]
		t.SetCellText2(row, 0, e.dt.Format("2006-01-02 15:04:05"))
		t.SetCellText2(row, 1, e.host)
		t.SetCellText2(row, 2, e.severity.String())
		t.SetCellColor(row, 2, e.severity.color())
		t.SetCellText2(row, 3, e.text)
	}
}

// alert logs the event and, for a live one with alerts on, pops up a toast.
// Toasts are rate-limited, so an outage of several hosts does not flood the
// screen; the log keeps everything.
func (c *monitor) alert(live bool, dt time.Time, h *host, s severity, text string) {
	c.events.add(dt, h.name, s, text)
	if !live || !c.swAlerts.Checked() || time.Since(c.lastToast) < toastEvery {
		return
	}
	c.lastToast = time.Now()
	kind := ui.ToastInfo
	switch s {
	case severityOK:
		kind = ui.ToastSuccess
	case severityWarning:
		kind = ui.ToastWarning
	case severityError:
		kind = ui.ToastError
	}
	c.Form().ShowToast(h.name+": "+text, kind)
}
