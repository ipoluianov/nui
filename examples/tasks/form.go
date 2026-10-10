package tasks

import (
	"fmt"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// Which tasks the list shows
const (
	showAll = iota
	showActive
	showDone
)

// planner is the state of the application
type planner struct {
	form  *ui.Form
	tasks []*Task

	// the filter
	calendar   *ui.Calendar
	allDates   *ui.ToggleSwitch
	show       int
	showRadios []*ui.RadioButton

	// the list: visible are the tasks of its rows
	table      *ui.Table
	visible    []*Task
	refreshing bool

	// the details pane
	grid     *ui.PropertyGrid
	details  *ui.Label
	selected *Task

	// the quick-add bar
	title    *ui.TextBox
	date     *ui.DatePicker
	time     *ui.TimePicker
	priority *ui.ComboBox

	// the status bar
	shown    *ui.Label
	progress *ui.ProgressBar

	tray           *ui.TrayIcon
	minimizeToTray *ui.ToggleSwitch
}

// NewForm is a to-do planner with reminders. A quick-add bar (TextBox with a
// hint, DatePicker, TimePicker, ComboBox; Enter adds) creates tasks; the
// Calendar, a ToggleSwitch and RadioButtons filter the Table, which colors
// the cells and shows images in them, toggles a task on a double click and
// has a context menu. A PropertyGrid (SetObject with struct tags) edits the
// selected task, the same in the Add task dialog. A ProgressBar in the status
// bar counts the done tasks. A form timer fires the reminders as toasts and
// tray notifications; the TrayIcon has a tooltip and a menu, and the window
// can be minimized to the tray.
func NewForm() *ui.Form {
	p := &planner{form: ui.NewForm(), tasks: seedTasks(time.Now())}
	p.form.SetTitle("Tasks")
	p.form.SetSize(1180, 720)
	p.form.SetIcon(appIcon(32))

	p.initTray()

	panel := p.form.Panel()
	panel.AddWidget(0, 0, p.quickAddBar())

	body := panel.AddPanel(1, 0)
	body.AddWidget(0, 0, p.filterPane())
	body.AddWidget(0, 1, p.taskList())
	body.AddWidget(0, 2, p.detailsPane())

	panel.AddWidget(2, 0, p.statusBar())

	p.form.Panel().AddTimer(1000, p.checkReminders)
	p.refresh(nil)
	return p.form
}

// quickAddBar is the row at the top that adds a task with a title, a due
// date and time, and a priority
func (p *planner) quickAddBar() ui.Widgeter {
	bar := ui.NewPanel()
	p.title = ui.NewTextBox()
	p.title.SetHint("New task…")
	p.title.SetXExpandable(true)
	// A single-line text box leaves Enter to its key handler
	p.title.SetOnKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
		if key == ui.KeyEnter {
			p.quickAdd()
			return true
		}
		return false
	})
	bar.AddWidget(0, 0, p.title)

	now := time.Now()
	p.date = ui.NewDatePicker()
	p.date.SetDate(now)
	bar.AddWidget(0, 1, p.date)
	p.time = ui.NewTimePicker()
	p.time.SetTime(now.Hour()+1, 0, 0)
	if now.Hour() == 23 {
		p.time.SetTime(23, 59, 0)
	}
	bar.AddWidget(0, 2, p.time)

	p.priority = ui.NewComboBox()
	for _, name := range []string{"Low", "Normal", "High"} {
		p.priority.AddItem(name, name)
	}
	p.priority.SetSelectedIndex(1)
	bar.AddWidget(0, 3, p.priority)

	bar.AddButton(0, 4, "Add", p.quickAdd)
	bar.AddButton(0, 5, "More…", func() { p.addTaskDialog() })
	return bar
}

// filterPane is the left column: the calendar and the filters
func (p *planner) filterPane() ui.Widgeter {
	pane := ui.NewPanel()
	pane.SetXExpandable(false)

	p.calendar = ui.NewCalendar()
	p.calendar.SetOnDateChanged(func(d time.Time) {
		// Choosing a day shows that day, and new tasks go there
		p.allDates.SetChecked(false)
		p.date.SetDate(d)
		p.refresh(p.selected)
	})
	pane.AddWidget(0, 0, p.calendar)

	p.allDates = ui.NewToggleSwitch("All dates")
	p.allDates.SetChecked(true)
	p.allDates.SetOnStateChanged(func() { p.refresh(p.selected) })
	pane.AddWidget(1, 0, p.allDates)

	show := ui.NewGroupBox("Show")
	pane.AddWidget(2, 0, show)
	for i, text := range []string{"All", "Active", "Done"} {
		radio := ui.NewRadioButton(text)
		radio.SetChecked(i == showAll)
		radio.SetOnStateChanged(func(btn *ui.RadioButton, checked bool) {
			if checked {
				p.show = i
				p.refresh(p.selected)
			}
		})
		show.AddWidget(i, 0, radio)
		p.showRadios = append(p.showRadios, radio)
	}

	pane.AddVSpacer(3, 0)
	p.minimizeToTray = ui.NewToggleSwitch("Minimize to tray")
	if p.tray == nil {
		p.minimizeToTray.SetEnabled(false)
	}
	pane.AddWidget(4, 0, p.minimizeToTray)
	return pane
}

// detailsPane is the right column: the properties of the selected task
func (p *planner) detailsPane() ui.Widgeter {
	pane := ui.NewPanel()
	pane.SetXExpandable(false)
	pane.SetMinWidth(320)
	pane.SetMaxWidth(340)
	p.details = pane.AddLabel(0, 0, "")
	p.grid = ui.NewPropertyGrid()
	p.grid.SetYExpandable(true)
	p.grid.SetOnChanged(func(prop *ui.Property) {
		task := p.selected
		if task == nil {
			return
		}
		if prop.Name() == "Due" || prop.Name() == "Reminder" {
			task.reminded = task.Due.Before(time.Now())
		}
		p.updateRow(task)
		p.updateStatus()
	})
	pane.AddWidget(1, 0, p.grid)
	return pane
}

// statusBar shows how many tasks are listed and how many are done
func (p *planner) statusBar() ui.Widgeter {
	bar := ui.NewPanel()
	p.shown = bar.AddLabel(0, 0, "")
	bar.AddHSpacer(0, 1)
	p.progress = ui.NewProgressBar(0, 1, 0)
	p.progress.SetMinWidth(240)
	p.progress.SetXExpandable(false)
	bar.AddWidget(0, 2, p.progress)
	return bar
}

// selectTask shows the task in the details pane (nil clears it)
func (p *planner) selectTask(task *Task) {
	p.selected = task
	if task == nil {
		p.grid.Clear()
		p.details.SetText("Select a task to see its details")
		return
	}
	p.details.SetText("Details")
	p.grid.SetObject(task)
}

// updateStatus updates the status bar and the tray tooltip
func (p *planner) updateStatus() {
	done, today := 0, 0
	now := time.Now()
	for _, t := range p.tasks {
		if t.Done {
			done++
		} else if sameDay(t.Due, now) {
			today++
		}
	}
	p.shown.SetText(plural(len(p.visible), "task") + " listed")
	p.progress.SetMaxValue(float64(max(len(p.tasks), 1)))
	p.progress.SetValue(float64(done))
	p.progress.SetText(fmt.Sprintf("%d of %d done", done, len(p.tasks)))
	if p.tray != nil {
		p.tray.SetTooltip(plural(today, "task") + " due today")
	}
}

// plural is "1 task", "3 tasks"
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
