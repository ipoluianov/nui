package tasks

import (
	"image/color"
	"sort"
	"time"
)

// Task is one to-do item. The details pane and the task dialog edit it with
// a PropertyGrid (SetObject), so the struct tags describe the editors.
type Task struct {
	Title    string     `category:"Task"`
	Project  string     `category:"Task" options:"Inbox, Work, Home, Health, Personal"`
	Priority string     `category:"Task" options:"Low, Normal, High"`
	Due      time.Time  `category:"Schedule" desc:"The day the task is due (the time of day is kept)"`
	Reminder bool       `category:"Schedule" desc:"Show a notification when the task is due"`
	Done     bool       `category:"Status"`
	Progress int        `category:"Status" prop:"Progress, %" min:"0" max:"100"`
	Color    color.RGBA `category:"Status" prop:"Label" desc:"The color of the mark next to the title"`
	Notes    string     `category:"Notes"`

	// reminded is set once the reminder has been shown
	reminded bool
}

var (
	colorBlue   = color.RGBA{0x1E, 0x88, 0xE5, 0xFF}
	colorGreen  = color.RGBA{0x43, 0xA0, 0x47, 0xFF}
	colorOrange = color.RGBA{0xFB, 0x8C, 0x00, 0xFF}
	colorPurple = color.RGBA{0x8E, 0x24, 0xAA, 0xFF}
	colorGray   = color.RGBA{0x90, 0xA4, 0xAE, 0xFF}
)

// projectColors gives a new task the label color of its project
var projectColors = map[string]color.RGBA{
	"Inbox": colorGray, "Work": colorBlue, "Home": colorOrange,
	"Health": colorGreen, "Personal": colorPurple,
}

// seedTasks returns the demo tasks, around today. One of them is due 20
// seconds after the start, with a reminder, so the reminder fires in the demo.
func seedTasks(now time.Time) []*Task {
	day := func(offset, hour, minute int) time.Time {
		return time.Date(now.Year(), now.Month(), now.Day()+offset, hour, minute, 0, 0, time.Local)
	}
	list := []*Task{
		{Title: "Team stand-up", Project: "Work", Priority: "Normal", Due: day(0, 9, 30), Done: true, Progress: 100},
		{Title: "Reply to the client's email", Project: "Work", Priority: "High", Due: day(0, 10, 0), Done: true, Progress: 100},
		{Title: "Stand up and stretch", Project: "Health", Priority: "Normal", Due: now.Add(20 * time.Second).Truncate(time.Second),
			Reminder: true, Notes: "Due 20 seconds after the start: watch the reminder."},
		{Title: "Code review for the login page", Project: "Work", Priority: "Normal", Due: day(0, 16, 0), Progress: 60},
		{Title: "Buy groceries", Project: "Home", Priority: "Normal", Due: day(0, 18, 0), Reminder: true,
			Notes: "Milk, eggs, bread, apples, coffee"},
		{Title: "Call the plumber", Project: "Home", Priority: "High", Due: day(-1, 11, 0),
			Notes: "The kitchen tap is leaking again"},
		{Title: "Pay the electricity bill", Project: "Home", Priority: "High", Due: day(-2, 12, 0), Done: true, Progress: 100},
		{Title: "Prepare the quarterly report", Project: "Work", Priority: "High", Due: day(1, 12, 0), Reminder: true, Progress: 40},
		{Title: "Morning run", Project: "Health", Priority: "Low", Due: day(1, 7, 0), Reminder: true},
		{Title: "Update the project roadmap", Project: "Work", Priority: "Normal", Due: day(2, 15, 0), Progress: 20},
		{Title: "Dentist appointment", Project: "Health", Priority: "Normal", Due: day(3, 14, 30), Reminder: true,
			Notes: "Dr. Miller, 2nd floor"},
		{Title: "Plan the weekend trip", Project: "Personal", Priority: "Low", Due: day(5, 20, 0), Done: true, Progress: 100},
	}
	for _, t := range list {
		t.Color = projectColors[t.Project]
		// The reminders of the past don't fire at the start
		t.reminded = t.Due.Before(now)
	}
	return list
}

// sameDay reports whether a and b are the same calendar day
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// sortByDue orders the tasks by their due time
func sortByDue(list []*Task) {
	sort.SliceStable(list, func(i, j int) bool { return list[i].Due.Before(list[j].Due) })
}

// dueText is the due time for the list: "Today 18:00", "Mon, Oct 12 09:30"
func dueText(due, now time.Time) string {
	switch {
	case sameDay(due, now):
		return "Today " + due.Format("15:04")
	case sameDay(due, now.AddDate(0, 0, 1)):
		return "Tomorrow " + due.Format("15:04")
	case sameDay(due, now.AddDate(0, 0, -1)):
		return "Yesterday " + due.Format("15:04")
	}
	return due.Format("Mon, Jan 2 15:04")
}
