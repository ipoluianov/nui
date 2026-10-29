package examples

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// TestSnapshots saves screenshots of the applications and of a gallery of
// the widgets, in both themes and at the scales 1, 1.5 and 2, into the
// folder NUI_SNAPSHOT_DIR. Taken before and after a change of the drawing,
// they are compared by cmd/snapdiff (see doc/testing.md). Without the
// variable the test is skipped.
//
// The pictures are the same from run to run, except those of the network
// monitor (its data are random and its time axis is the clock) - it is left
// out - and the "today" mark of the calendars, which moves at midnight.
func TestSnapshots(t *testing.T) {
	dir := os.Getenv("NUI_SNAPSHOT_DIR")
	if dir == "" {
		t.Skip("NUI_SNAPSHOT_DIR not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer uitest.SetScreen(1920, 1080, 1)
	defer ui.ApplyDarkTheme()

	type entry struct {
		name    string
		newForm func() *ui.Form
	}
	entries := []entry{{"gallery", snapshotGallery}}
	for _, app := range Apps {
		if app.Name == "tasks" || app.Name == "monitor" {
			continue // a tray icon needs the UI thread; random data
		}
		entries = append(entries, entry{app.Name, app.NewForm})
	}

	for _, theme := range []string{"dark", "light"} {
		if theme == "dark" {
			ui.ApplyDarkTheme()
		} else {
			ui.ApplyLightTheme()
		}
		for _, scale := range []float64{1, 1.5, 2} {
			for _, e := range entries {
				name := fmt.Sprintf("%s-%s-%v", e.name, theme, scale)
				uitest.SetScreen(4000, 3000, scale)
				form := e.newForm()
				form.SetSize(1200, 900)
				w := uitest.Show(t, form)
				w.Settle()
				saveSnapshot(t, dir, name, w)

				switch e.name {
				case "notepad": // an open menu
					w.Click(w.Find("File"))
					w.Settle()
					saveSnapshot(t, dir, name+"-menu", w)
					w.Key(ui.KeyEsc, ui.KeyModifiers{})
				case "gallery": // the link of the markdown under the mouse
					w.Move(uitest.Point{X: 620, Y: 363})
					w.Settle()
					saveSnapshot(t, dir, name+"-hover", w)
				}
				form.Close()
				uitest.RunPosted()
			}
		}
	}
}

func saveSnapshot(t *testing.T, dir, name string, w *uitest.Window) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.Screenshot()); err != nil {
		t.Fatal(err)
	}
}

// snapshotGallery is a form with every kind of widget, in a few states
func snapshotGallery() *ui.Form {
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.Local)
	form := ui.NewForm()
	form.SetTitle("Gallery")
	p := form.Panel()

	col1 := p.AddPanel(0, 0)
	col1.AddLabel(0, 0, "Label")
	col1.AddLabel(1, 0, "Underlined label").SetUnderline(true)
	col1.AddButton(2, 0, "Button", nil)
	col1.AddButton(3, 0, "Disabled", nil).SetEnabled(false)
	checked := ui.NewCheckbox("Checked")
	checked.SetChecked(true)
	col1.AddWidget(4, 0, checked)
	col1.AddWidget(5, 0, ui.NewCheckbox("Unchecked"))
	radio := ui.NewRadioButton("Radio on")
	radio.SetChecked(true)
	col1.AddWidget(6, 0, radio)
	col1.AddWidget(7, 0, ui.NewRadioButton("Radio off"))
	toggle := ui.NewToggleSwitch("Toggle")
	toggle.SetChecked(true)
	col1.AddWidget(8, 0, toggle)
	combo := ui.NewComboBox()
	combo.SetItems([]string{"First item", "Second"})
	combo.SetSelectedIndex(0)
	col1.AddWidget(9, 0, combo)
	slider := ui.NewSlider()
	slider.SetValue(30)
	col1.AddWidget(10, 0, slider)
	col1.AddWidget(11, 0, ui.NewProgressBar(0, 100, 60))
	num := ui.NewNumBox()
	num.SetValue(42)
	col1.AddWidget(12, 0, num)
	text := ui.NewTextBox()
	text.SetText("Selected text")
	text.SelectAllText()
	col1.AddWidget(13, 0, text)
	col1.AddWidget(14, 0, ui.NewLink("A link", nil))
	datePicker := ui.NewDatePicker()
	datePicker.SetDate(date)
	col1.AddWidget(15, 0, datePicker)
	col1.AddWidget(16, 0, ui.NewTimePicker())
	col1.AddWidget(17, 0, ui.NewEditableComboBox())

	col2 := p.AddPanel(0, 1)
	group := ui.NewGroupBox("Group box")
	group.AddLabel(0, 0, "Inside the group")
	col2.AddWidget(0, 0, group)
	expander := ui.NewExpander("Expander")
	expander.Content().AddLabel(0, 0, "Expanded content")
	expander.SetExpanded(true)
	col2.AddWidget(1, 0, expander)
	tabs := ui.NewTabWidget()
	tabs.AddPage("Tab one", ui.NewLabel("Page one"))
	tabs.AddPage("Tab two", ui.NewLabel("Page two"))
	col2.AddWidget(2, 0, tabs)
	markdown := ui.NewMarkdownView()
	markdown.SetMarkdown("# Title\n\nSome **bold**, *italic*, `code` and a [link](http://example.com).\n\n- one\n- two\n\n```\ncode block\n```\n")
	col2.AddWidget(3, 0, markdown)
	tree := ui.NewTreeView()
	root := tree.AddNode(nil, "Root")
	tree.AddNode(root, "Child 1")
	child := tree.AddNode(root, "Child 2")
	tree.AddNode(child, "Grandchild")
	tree.ExpandAll()
	col2.AddWidget(4, 0, tree)
	calendar := ui.NewCalendar()
	calendar.SetDate(date)
	col2.AddWidget(5, 0, calendar)

	col3 := p.AddPanel(0, 2)
	table := ui.NewTable()
	table.SetColumnCount(3)
	table.SetRowCount(20)
	for r := 0; r < 20; r++ {
		for c := 0; c < 3; c++ {
			table.SetCellText2(r, c, fmt.Sprint(r, ":", c))
		}
	}
	col3.AddWidget(0, 0, table)
	list := ui.NewListView()
	list.SetModel(ui.NewListModel("Alpha", "Beta", "Gamma", "Delta"))
	col3.AddWidget(1, 0, list)
	props := ui.NewPropertyGrid()
	props.AddString("General", "Name", "Value")
	props.AddInt("General", "Count", 5, 0, 10)
	props.AddFloat("Other", "Ratio", 0.5, 2)
	col3.AddWidget(2, 0, props)
	chart := ui.NewChart()
	var points []ui.ChartPoint
	for i := 0; i < 50; i++ {
		x := float64(i) / 5
		points = append(points, ui.ChartPoint{X: x, Y: x * x / 10})
	}
	chart.SetData(points)
	col3.AddWidget(3, 0, chart)
	log := ui.NewLogView()
	log.AppendLine("log line one")
	log.AppendLine("log line two")
	col3.AddWidget(4, 0, log)
	col3.AddWidget(5, 0, ui.NewColorPicker())
	return form
}
