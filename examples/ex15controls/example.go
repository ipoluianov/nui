package ex15controls

import (
	"fmt"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows the input controls: Slider, ToggleSwitch, Link,
// GroupBox, EditableComboBox with suggestions, DatePicker, Calendar and
// TimePicker. The status line tells what changed.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Controls")
	form.SetSize(760, 560)
	panel := form.Panel()

	status := panel.AddLabel(0, 0, "Change something")
	setStatus := func(format string, args ...any) { status.SetText(fmt.Sprintf(format, args...)) }

	columns := panel.AddPanel(1, 0)
	left := columns.AddPanel(0, 0)
	right := columns.AddPanel(0, 1)

	// Sliders
	sliders := ui.NewGroupBox("Slider")
	left.AddWidget(0, 0, sliders)
	volumeValue := sliders.AddLabel(0, 1, "50")
	volume := ui.NewSlider()
	volume.SetRange(0, 100)
	volume.SetStep(1)
	volume.SetValue(50)
	volume.SetTickInterval(10)
	volume.SetOnValueChanged(func() {
		volumeValue.SetText(fmt.Sprint(volume.Value()))
		setStatus("Volume: %v", volume.Value())
	})
	sliders.AddWidget(0, 0, volume)
	disabled := ui.NewSlider()
	disabled.SetValue(30)
	disabled.SetEnabled(false)
	sliders.AddWidget(1, 0, disabled)
	vertical := ui.NewSlider()
	vertical.SetVertical(true)
	vertical.SetMinHeight(100)
	vertical.SetValue(70)
	vertical.SetOnValueChanged(func() { setStatus("Vertical: %.1f", vertical.Value()) })
	sliders.AddWidget(0, 2, vertical)

	// Switches and links
	switches := ui.NewGroupBox("ToggleSwitch and Link")
	left.AddWidget(1, 0, switches)
	wifi := ui.NewToggleSwitch("Wi-Fi")
	wifi.SetChecked(true)
	wifi.SetOnStateChanged(func() { setStatus("Wi-Fi: %v", wifi.Checked()) })
	switches.AddWidget(0, 0, wifi)
	bluetooth := ui.NewToggleSwitch("Bluetooth")
	bluetooth.SetOnStateChanged(func() { setStatus("Bluetooth: %v", bluetooth.Checked()) })
	switches.AddWidget(1, 0, bluetooth)
	airplane := ui.NewToggleSwitch("Airplane mode (disabled)")
	airplane.SetEnabled(false)
	switches.AddWidget(2, 0, airplane)
	switches.AddWidget(3, 0, ui.NewLink("Show a message", func() { setStatus("The link was clicked") }))
	site := ui.NewLink("Open go.dev in the browser", nil)
	site.SetURL("https://go.dev")
	switches.AddWidget(4, 0, site)

	// Suggestions
	combos := ui.NewGroupBox("EditableComboBox")
	left.AddWidget(2, 0, combos)
	city := ui.NewEditableComboBox()
	city.SetHint("Type a city: \"ber\", \"lon\"...")
	city.SetItems([]string{"Amsterdam", "Athens", "Barcelona", "Berlin", "Bern", "Brussels", "Budapest",
		"Copenhagen", "Dublin", "Helsinki", "Lisbon", "London", "Madrid", "Oberhausen", "Oslo",
		"Paris", "Prague", "Rome", "Stockholm", "Vienna", "Warsaw", "Zurich"})
	city.SetOnItemSelected(func(index int, text string) { setStatus("City chosen: %s", text) })
	city.SetOnAccept(func(text string) { setStatus("Entered: %q", text) })
	combos.AddWidget(0, 0, city)

	left.AddVSpacer(3, 0)

	// Date and time
	dates := ui.NewGroupBox("DatePicker, TimePicker, Calendar")
	right.AddWidget(0, 0, dates)
	dates.AddLabel(0, 0, "Date")
	due := ui.NewDatePicker()
	due.SetOnDateChanged(func(d time.Time) { setStatus("Date: %s", due.Text()) })
	dates.AddWidget(0, 1, due)
	dates.AddLabel(1, 0, "Time")
	at := ui.NewTimePicker()
	at.SetTime(9, 30, 0)
	at.SetOnTimeChanged(func() { setStatus("Time: %s", at.Text()) })
	dates.AddWidget(1, 1, at)
	dates.AddLabel(2, 0, "With seconds")
	precise := ui.NewTimePicker()
	precise.SetShowSeconds(true)
	precise.SetTime(23, 59, 30)
	precise.SetOnTimeChanged(func() { setStatus("Time: %s", precise.Text()) })
	dates.AddWidget(2, 1, precise)

	calendars := ui.NewGroupBox("Calendar (double click sets the date above)")
	right.AddWidget(1, 0, calendars)
	cal := ui.NewCalendar()
	cal.SetOnDateChanged(func(d time.Time) { setStatus("Calendar: %s", d.Format("2006-01-02")) })
	cal.SetOnDateActivated(func(d time.Time) { due.SetDate(d); setStatus("Date set from the calendar") })
	calendars.AddWidget(0, 0, cal)
	calendars.AddHSpacer(0, 1)
	right.AddVSpacer(2, 0)

	lang := ui.NewToggleSwitch("Русский язык")
	lang.SetOnStateChanged(func() {
		if lang.Checked() {
			ui.SetLanguage("ru")
		} else {
			ui.SetLanguage("en")
		}
	})
	right.AddWidget(3, 0, lang)

	return form
}
