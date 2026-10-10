package tasks

import (
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

// The columns of the task list
const (
	colDone = iota
	colTitle
	colProject
	colPriority
	colDue
)

var (
	colorHigh    = color.RGBA{0xE5, 0x39, 0x35, 0xFF}
	colorLow     = color.RGBA{0x8A, 0x8A, 0x8A, 0xFF}
	colorOverdue = color.RGBA{0xE5, 0x39, 0x35, 0xFF}
)

// taskList is the table of the tasks with its context menu
func (p *planner) taskList() ui.Widgeter {
	p.table = ui.NewTable()
	p.table.SetXExpandable(true)
	p.table.SetYExpandable(true)
	p.table.SetColumnCount(5)
	for col, name := range []string{"", "Title", "Project", "Priority", "Due"} {
		p.table.SetColumnName(col, name)
	}
	for col, width := range []int{30, 270, 80, 70, 140} {
		p.table.SetColumnWidth(col, width)
	}
	p.table.SetStretchLastColumn(true)
	p.table.SetSelectingRows(true)
	p.table.SetHotTracking(true)

	p.table.SetOnSelectionChanged(func(int, int) {
		if !p.refreshing {
			p.selectTask(p.current())
		}
	})
	p.table.SetOnCellMouseDblClick(func() { p.toggleDone(p.current()) })
	p.table.SetOnKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
		switch key {
		case ui.KeySpace:
			p.toggleDone(p.current())
		case ui.KeyDelete:
			p.deleteTask(p.current())
		default:
			return false
		}
		return true
	})

	menu := ui.NewContextMenu(p.table)
	markDone := menu.AddItem("Mark done", func() { p.toggleDone(p.current()) }).SetImage(drawCheck(icons.Size, true))
	menu.AddItem("Edit…", func() { p.editTaskDialog(p.current()) }).SetImage(pencilIcon())
	menu.AddItem("Postpone 1 day", func() { p.postpone(p.current()) })
	menu.AddSeparator()
	menu.AddItem("Delete", func() { p.deleteTask(p.current()) }).SetImage(crossIcon())
	menu.SetOnShow(func() {
		if t := p.current(); t != nil && t.Done {
			markDone.SetText("Mark not done")
		} else {
			markDone.SetText("Mark done")
		}
	})
	p.table.SetContextMenu(menu)
	return p.table
}

// current is the task of the current row, nil if none
func (p *planner) current() *Task {
	row := p.table.CurrentRow()
	if row < 0 || row >= len(p.visible) {
		return nil
	}
	return p.visible[row]
}

// matches reports whether the filter shows the task
func (p *planner) matches(t *Task) bool {
	if !p.allDates.Checked() && !sameDay(t.Due, p.calendar.Date()) {
		return false
	}
	switch p.show {
	case showActive:
		return !t.Done
	case showDone:
		return t.Done
	}
	return true
}

// refresh fills the table with the tasks the filter shows, sorted by the due
// time, and selects keep if it is among them, else the first one
func (p *planner) refresh(keep *Task) {
	sortByDue(p.tasks)
	p.visible = p.visible[:0]
	for _, t := range p.tasks {
		if p.matches(t) {
			p.visible = append(p.visible, t)
		}
	}

	p.refreshing = true
	p.table.ClearRows()
	p.table.SetRowCount(len(p.visible))
	selected := -1
	for row, t := range p.visible {
		p.fillRow(row, t)
		if t == keep {
			selected = row
		}
	}
	if selected < 0 && len(p.visible) > 0 {
		selected = 0
	}
	if selected >= 0 {
		p.table.SetCurrentCell2(selected, colTitle)
		p.table.ScrollToCell2(selected, colTitle)
	}
	p.refreshing = false

	if selected >= 0 {
		p.selectTask(p.visible[selected])
	} else {
		p.selectTask(nil)
	}
	p.updateStatus()
}

// fillRow shows the task in the row
func (p *planner) fillRow(row int, t *Task) {
	p.table.SetCellImage(row, colDone, checkIcon(t.Done), 16)
	p.table.SetCellImage(row, colTitle, labelIcon(t.Color), 12)
	title := t.Title
	if t.Progress > 0 && t.Progress < 100 && !t.Done {
		title += "  (" + strconv.Itoa(t.Progress) + "%)"
	}
	p.table.SetCellText2(row, colTitle, title)
	p.table.SetCellContraction(row, colTitle, true)
	p.table.SetCellText2(row, colProject, t.Project)
	p.table.SetCellText2(row, colPriority, t.Priority)
	p.table.SetCellText2(row, colDue, dueText(t.Due, time.Now()))

	var priorityColor, dueColor color.Color
	switch t.Priority {
	case "High":
		priorityColor = colorHigh
	case "Low":
		priorityColor = colorLow
	}
	if t.Done {
		priorityColor = colorLow
	} else if t.Due.Before(time.Now()) {
		dueColor = colorOverdue
	}
	p.table.SetCellColor(row, colPriority, priorityColor)
	p.table.SetCellColor(row, colDue, dueColor)
}

// updateRow shows the changes of the task in its row, without filtering and
// sorting again (it is called while the task is being edited)
func (p *planner) updateRow(task *Task) {
	for row, t := range p.visible {
		if t == task {
			p.fillRow(row, t)
		}
	}
}

// changed refreshes the list and the details after a change from the code
func (p *planner) changed(task *Task) {
	p.refresh(task)
}

func (p *planner) toggleDone(t *Task) {
	if t == nil {
		return
	}
	t.Done = !t.Done
	if t.Done {
		t.Progress = 100
		p.form.ShowToast("Done: "+t.Title, ui.ToastSuccess)
	} else if t.Progress == 100 {
		t.Progress = 0
	}
	p.changed(t)
}

func (p *planner) postpone(t *Task) {
	if t == nil {
		return
	}
	t.Due = t.Due.AddDate(0, 0, 1)
	t.reminded = t.Due.Before(time.Now())
	p.form.ShowToast("Postponed to "+t.Due.Format("Mon, Jan 2"), ui.ToastInfo)
	p.changed(t)
}

func (p *planner) deleteTask(t *Task) {
	if t == nil {
		return
	}
	for i, task := range p.tasks {
		if task == t {
			p.tasks = append(p.tasks[:i], p.tasks[i+1:]...)
			break
		}
	}
	p.form.ShowToast("Deleted: "+t.Title, ui.ToastInfo)
	p.changed(nil)
}

// addTask adds the task and shows it: the filter changes if it would hide it
func (p *planner) addTask(t *Task) {
	t.reminded = t.Due.Before(time.Now())
	p.tasks = append(p.tasks, t)
	if !p.matches(t) {
		p.allDates.SetChecked(true)
		if t.Done != (p.show == showDone) {
			p.show = showAll
			p.showRadios[showAll].SetChecked(true)
		}
	}
	p.refresh(t)
	p.form.ShowToast("Added: "+t.Title, ui.ToastSuccess)
}

// quickAdd adds a task from the quick-add bar
func (p *planner) quickAdd() {
	title := strings.TrimSpace(p.title.Text())
	if title == "" {
		p.title.Focus()
		return
	}
	d := p.date.Date()
	due := time.Date(d.Year(), d.Month(), d.Day(), p.time.Hour(), p.time.Minute(), 0, 0, time.Local)
	p.addTask(&Task{
		Title: title, Project: "Inbox", Priority: p.priority.SelectedItemText(),
		Due: due, Reminder: true, Color: projectColors["Inbox"],
	})
	p.title.SetText("")
	p.title.Focus()
}
