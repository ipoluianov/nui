package tasks

import (
	"strings"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// taskDialog edits a copy of a task in a PropertyGrid; OK hands it over
type taskDialog struct {
	ui.DialogContent

	title string
	task  Task
	grid  *ui.PropertyGrid
	ok    *ui.Button

	OnOK func(task Task)
}

func newTaskDialog(title string, task Task) *taskDialog {
	c := &taskDialog{title: title, task: task}
	c.InitWidget()

	c.grid = ui.NewPropertyGrid()
	c.grid.SetObject(&c.task)
	c.grid.SetYExpandable(true)
	c.grid.SetOnChanged(func(*ui.Property) { c.ok.SetEnabled(strings.TrimSpace(c.task.Title) != "") })
	c.AddWidget(0, 0, c.grid)

	buttons := c.AddPanel(1, 0)
	buttons.AddHSpacer(0, 0)
	c.ok = buttons.AddButton(0, 1, "OK", func() {
		task := c.task
		c.Form().Close()
		c.RunInParent(func() {
			if c.OnOK != nil {
				c.OnOK(task)
			}
		})
	})
	c.ok.SetEnabled(strings.TrimSpace(c.task.Title) != "")
	cancel := buttons.AddButton(0, 2, "Cancel", func() { c.Form().Close() })

	c.OnDialogShow = func() {
		form := c.Form()
		form.SetTitle(c.title)
		form.SetSize(420, 520)
		form.MoveToCenterOfParent()
		form.SetAcceptButton(c.ok)
		form.SetCancelButton(cancel)
	}
	return c
}

// addTaskDialog adds a task with all its properties
func (p *planner) addTaskDialog() {
	due := time.Now().Add(time.Hour).Truncate(time.Hour)
	if !p.allDates.Checked() {
		// The day shown in the calendar, at the same hour
		d := p.calendar.Date()
		due = time.Date(d.Year(), d.Month(), d.Day(), due.Hour(), 0, 0, 0, time.Local)
	}
	dialog := newTaskDialog("Add task", Task{
		Project: "Inbox", Priority: "Normal", Due: due, Reminder: true, Color: projectColors["Inbox"],
	})
	dialog.OnOK = func(task Task) { p.addTask(&task) }
	p.form.Panel().ShowDialog(dialog)
}

// editTaskDialog edits the task in a dialog: the changes apply on OK
func (p *planner) editTaskDialog(t *Task) {
	if t == nil {
		return
	}
	dialog := newTaskDialog("Edit task", *t)
	dialog.OnOK = func(task Task) {
		if !task.Due.Equal(t.Due) {
			task.reminded = task.Due.Before(time.Now())
		}
		*t = task
		p.changed(t)
	}
	p.form.Panel().ShowDialog(dialog)
}
