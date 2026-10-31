package recipes

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// ConnectionDialog is a dialog with fields, checked before it closes: OK
// is pressed by Enter, Cancel by Esc, the buttons in the order of the
// platform.
type ConnectionDialog struct {
	ui.DialogContent
	host *ui.TextBox
	port *ui.NumBox
	// OnOK gets the values the user accepted
	OnOK func(host string, port int)
}

// NewConnectionDialog makes the dialog with the values to edit.
func NewConnectionDialog(host string, port int) *ConnectionDialog {
	d := &ConnectionDialog{}
	d.InitWidget()
	d.host = ui.NewTextBox()
	d.host.SetText(host)
	d.port = ui.NewNumBox()
	d.port.SetMin(1)
	d.port.SetMax(65535)
	d.port.SetValue(float64(port))

	fields := ui.NewFormLayout()
	fields.AddRow("Host:", d.host)
	fields.AddRow("Port:", d.port)
	d.AddWidget(0, 0, fields)
	d.AddVSpacer(1, 0)

	buttons := ui.NewDialogButtonBox()
	buttons.AddOK(func() {
		host := strings.TrimSpace(d.host.Text())
		if host == "" {
			ui.ShowMessageBox(d, "Connection", "Enter the host.")
			d.host.Focus()
			return // the dialog stays open
		}
		port := int(d.port.Value())
		d.RunInParent(func() { d.OnOK(host, port) }) // after the dialog closes
		d.Form().Close()
	})
	buttons.AddCancel(func() { d.Form().Close() })
	d.AddWidget(2, 0, buttons)

	d.OnDialogShow = func() {
		d.Form().SetTitle("Connection")
		d.host.Focus()
	}
	return d
}

// ShowConnectionDialog shows the dialog over the form.
func ShowConnectionDialog(form *ui.Form, apply func(host string, port int)) {
	d := NewConnectionDialog("localhost", 8080)
	d.OnOK = apply
	form.Panel().ShowDialog(d)
}

// FilterList is a list filtered as the user types: the filter runs once
// the typing pauses, not on every key.
func FilterList(items []string) ui.Widgeter {
	panel := ui.NewPanel()
	search := ui.NewTextBox()
	list := ui.NewListView()
	list.SetItems(items)
	list.SetEmptyText("Nothing found")

	filter := func() {
		q := strings.ToLower(search.Text())
		var found []string
		for _, it := range items {
			if strings.Contains(strings.ToLower(it), q) {
				found = append(found, it)
			}
		}
		list.SetItems(found)
	}
	// Start again on each change: it fires 300 ms after the last one
	debounce := panel.NewTimer(300*time.Millisecond, filter)
	debounce.SetSingleShot(true)
	search.OnTextChanged.Connect(func(string) { debounce.Start() })

	list.OnActivated.Connect(func(row int) {
		ui.ShowMessageBox(panel, "Open", list.Model().Text(row, 0))
	})
	panel.AddWidget(0, 0, search)
	panel.AddWidget(1, 0, list)
	return panel
}

// Person is a row of PeopleTable
type Person struct {
	Name string
	Age  int
}

// peopleModel shows a slice of the application's own type in a table,
// with column headers; the table asks only for the visible rows.
type peopleModel struct {
	ui.ModelSignal
	people []Person
}

func (m *peopleModel) RowCount() int    { return len(m.people) }
func (m *peopleModel) ColumnCount() int { return 2 }
func (m *peopleModel) ColumnName(col int) string {
	return [...]string{"Name", "Age"}[col]
}
func (m *peopleModel) Text(row, col int) string {
	if col == 0 {
		return m.people[row].Name
	}
	return fmt.Sprint(m.people[row].Age)
}

// PeopleTable is a table of people; add appends one and shows it.
func PeopleTable(people []Person) (table *ui.Table, add func(Person)) {
	model := &peopleModel{people: people}
	table = ui.NewTable()
	table.SetModel(model)
	return table, func(p Person) {
		model.people = append(model.people, p)
		model.Changed() // the table shows the new row
	}
}

// CountFiles is work in the background with a progress dialog and a
// Cancel button; the result comes to done on the UI thread.
func CountFiles(owner ui.Widgeter, dirs []string, done func(n int)) *ui.Task {
	return ui.RunWithProgress(owner, "Counting", "Counting the files...",
		func(ctx context.Context, p *ui.Progress) (int, error) {
			n := 0
			for i, dir := range dirs {
				if ctx.Err() != nil {
					return n, ctx.Err() // cancelled
				}
				n += countIn(dir)
				p.Report(float64(i+1)/float64(len(dirs)), dir)
			}
			return n, nil
		},
		func(n int, err error) {
			if errors.Is(err, context.Canceled) {
				return
			}
			done(n)
		})
}

func countIn(dir string) int {
	time.Sleep(10 * time.Millisecond) // the slow part, off the UI thread
	return len(dir)
}

// LiveLabel shows a value that a goroutine produces: the goroutine never
// touches the widget, it hands the value to the UI thread.
func LiveLabel(values <-chan float64) *ui.Label {
	label := ui.NewLabel("-")
	go func() {
		for v := range values {
			ui.Invoke(func() { label.SetText(fmt.Sprintf("%.1f", v)) })
		}
	}()
	return label
}

// Dots is a widget of its own: it paints itself, takes the mouse, and
// asks for a repaint when its data change.
type Dots struct {
	ui.Widget
	points [][2]int
	// OnAdded fires with the number of dots
	OnAdded ui.Signal[int]
}

// NewDots makes the widget; a click puts a dot.
func NewDots() *Dots {
	d := &Dots{}
	d.InitWidget()
	d.SetMinSize(200, 150)
	d.SetXExpandable(true)
	d.SetYExpandable(true)
	d.SetOnPaint(func(cnv *ui.Canvas) {
		cnv.FillRect(0, 0, d.Width(), d.Height(), color.RGBA{0xFA, 0xFA, 0xFA, 0xFF})
		for _, p := range d.points {
			cnv.FillCircle(float64(p[0]), float64(p[1]), 4, color.RGBA{0x20, 0x70, 0xD0, 0xFF})
		}
	})
	d.OnMouseDown.Connect(func(e *ui.MouseEvent) {
		if e.Button != ui.MouseButtonLeft {
			return
		}
		d.points = append(d.points, [2]int{e.X, e.Y})
		d.Update() // only this widget is painted again
		d.OnAdded.Emit(len(d.points))
	})
	return d
}

// AcceptFiles opens the files dropped on the window from the system.
func AcceptFiles(form *ui.Form, open func(path string)) {
	form.SetOnFilesDropped(func(files []string, x, y int) {
		for _, f := range files {
			open(f)
		}
	})
}
