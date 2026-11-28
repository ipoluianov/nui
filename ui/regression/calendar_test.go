package regression

import (
	"fmt"
	"testing"
	"time"
	_ "time/tzdata" // the zones below on any machine

	"github.com/ipoluianov/nui/ui"
)

// dateZones are local time zones with a month around a change of the
// clocks: in Chile (Sep 6 2026) and in Brazil (Nov 4 2018) daylight saving
// time starts at midnight, so the day has no 00:00; the others change at
// another hour or not at all.
var dateZones = []struct {
	zone  string
	year  int
	month time.Month
}{
	{"America/Santiago", 2026, time.September},
	{"America/Santiago", 2026, time.April},
	{"America/Sao_Paulo", 2018, time.November},
	{"Europe/Moscow", 2026, time.March},
	{"UTC", 2026, time.February},
	{"America/New_York", 2026, time.March},
	{"America/New_York", 2026, time.November},
}

// setLocal makes the zone the local one for the test
func setLocal(t *testing.T, zone string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Skipf("no zone %s: %v", zone, err)
	}
	old := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = old })
	return loc
}

// civil is the calendar date of t, for comparing days
func civil(t time.Time) string {
	return t.Format("2006-01-02")
}

// checkStartOfDay checks that a date the calendar gives is the start of
// the day: DateOnly keeps it, and it's at midnight unless the clocks skip
// the midnight of that day
func checkStartOfDay(t *testing.T, what string, d time.Time) {
	t.Helper()
	if again := ui.DateOnly(d); !again.Equal(d) {
		t.Errorf("%s: DateOnly(%v) = %v, another instant", what, d, again)
	}
	y, m, day := d.Date()
	if midnight := time.Date(y, m, day, 0, 0, 0, 0, time.Local); midnight.Day() == day && !midnight.Equal(d) {
		t.Errorf("%s: %v is not at midnight", what, d)
	}
}

// Where the clocks skip midnight, every day of the month can still be
// selected and the arrow keys move one day at a time: dates kept as local
// midnight used to fall on 23:00 of the day before, so Sep 6 2026 in Chile
// could not be selected and Right got stuck on Sep 5.
func TestCalendarDatesAcrossClockChanges(t *testing.T) {
	for _, z := range dateZones {
		t.Run(fmt.Sprintf("%s-%d-%02d", z.zone, z.year, z.month), func(t *testing.T) {
			loc := setLocal(t, z.zone)
			cal := ui.NewCalendar()
			days := time.Date(z.year, z.month+1, 0, 12, 0, 0, 0, loc).Day()
			for d := 1; d <= days; d++ {
				want := time.Date(z.year, z.month, d, 12, 0, 0, 0, loc)
				cal.SetDate(want)
				if civil(cal.Date()) != civil(want) {
					t.Errorf("SetDate(%s): Date() = %v", civil(want), cal.Date())
				}
				checkStartOfDay(t, "SetDate", cal.Date())
				if got := ui.DateOnly(want); civil(got) != civil(want) {
					t.Errorf("DateOnly(%v) = %v", want, got)
				}
			}

			// Right from the last days of the month before to the first
			// days of the month after, then Left back
			start := time.Date(z.year, z.month, -2, 12, 0, 0, 0, loc)
			cal.SetDate(start)
			for i := 1; i <= days+6; i++ {
				cal.ProcessKeyDown(ui.KeyArrowRight, ui.KeyModifiers{})
				want := time.Date(z.year, z.month, -2+i, 12, 0, 0, 0, loc)
				if civil(cal.Date()) != civil(want) {
					t.Fatalf("Right #%d: %v, want %s", i, cal.Date(), civil(want))
				}
				checkStartOfDay(t, "Right", cal.Date())
			}
			for i := days + 5; i >= 0; i-- {
				cal.ProcessKeyDown(ui.KeyArrowLeft, ui.KeyModifiers{})
				want := time.Date(z.year, z.month, -2+i, 12, 0, 0, 0, loc)
				if civil(cal.Date()) != civil(want) {
					t.Fatalf("Left to %s: %v", civil(want), cal.Date())
				}
			}

			// A week down, the end of the month, a month on
			cal.SetDate(time.Date(z.year, z.month, 1, 12, 0, 0, 0, loc))
			for w := 1; w <= 3; w++ {
				cal.ProcessKeyDown(ui.KeyArrowDown, ui.KeyModifiers{})
				if want := time.Date(z.year, z.month, 1+7*w, 12, 0, 0, 0, loc); civil(cal.Date()) != civil(want) {
					t.Errorf("Down #%d: %v, want %s", w, cal.Date(), civil(want))
				}
			}
			cal.ProcessKeyDown(ui.KeyEnd, ui.KeyModifiers{})
			if want := time.Date(z.year, z.month, days, 12, 0, 0, 0, loc); civil(cal.Date()) != civil(want) {
				t.Errorf("End: %v, want %s", cal.Date(), civil(want))
			}
			cal.ProcessKeyDown(ui.KeyHome, ui.KeyModifiers{})
			cal.ProcessKeyDown(ui.KeyPageUp, ui.KeyModifiers{})
			cal.ProcessKeyDown(ui.KeyPageDown, ui.KeyModifiers{})
			if want := time.Date(z.year, z.month, 1, 12, 0, 0, 0, loc); civil(cal.Date()) != civil(want) {
				t.Errorf("Home, PageUp, PageDown: %v, want %s", cal.Date(), civil(want))
			}
			if m := cal.ShownMonth(); m.Month() != z.month || m.Day() != 1 {
				t.Errorf("shown month %v", m)
			}
		})
	}
}

// The DatePicker's Up and Down step one day across a midnight the clocks
// skip, too
func TestDatePickerDaysAcrossClockChanges(t *testing.T) {
	for _, z := range dateZones {
		t.Run(fmt.Sprintf("%s-%d-%02d", z.zone, z.year, z.month), func(t *testing.T) {
			loc := setLocal(t, z.zone)
			picker := ui.NewDatePicker()
			picker.SetDate(time.Date(z.year, z.month, 1, 12, 0, 0, 0, loc))
			for i := 1; i <= 31; i++ {
				picker.ProcessKeyDown(ui.KeyArrowUp, ui.KeyModifiers{})
				want := time.Date(z.year, z.month, 1+i, 12, 0, 0, 0, loc)
				if civil(picker.Date()) != civil(want) {
					t.Fatalf("Up #%d: %v, want %s", i, picker.Date(), civil(want))
				}
				checkStartOfDay(t, "Up", picker.Date())
			}
			for i := 30; i >= 0; i-- {
				picker.ProcessKeyDown(ui.KeyArrowDown, ui.KeyModifiers{})
				if want := time.Date(z.year, z.month, 1+i, 12, 0, 0, 0, loc); civil(picker.Date()) != civil(want) {
					t.Fatalf("Down to %s: %v", civil(want), picker.Date())
				}
			}
		})
	}
}
