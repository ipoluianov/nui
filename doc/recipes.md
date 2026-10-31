# Recipes

Common tasks of an application on nui, each a short piece of code. The whole code of each is in [examples/recipes](../examples/recipes), built and tested with the library (`go test ./examples/recipes`), so it stays right. The pages of [ui/](ui/README.md) tell each widget in detail; [rules.md](rules.md) tells what the signatures do not.

- [The main function](#the-main-function)
- [The main window: menu, toolbar, status bar](#the-main-window-menu-toolbar-status-bar)
- [Ask before unsaved changes are lost](#ask-before-unsaved-changes-are-lost)
- [Open and save a file](#open-and-save-a-file)
- [A dialog with fields](#a-dialog-with-fields)
- [Work in the background with a progress dialog](#work-in-the-background-with-a-progress-dialog)
- [Data from a goroutine](#data-from-a-goroutine)
- [A list filtered as you type](#a-list-filtered-as-you-type)
- [A table of your own data](#a-table-of-your-own-data)
- [A widget of your own](#a-widget-of-your-own)
- [Files dropped on the window](#files-dropped-on-the-window)
- [Light and dark theme](#light-and-dark-theme)
- [Translations](#translations)
- [An application in the tray](#an-application-in-the-tray)
- [A test of a form](#a-test-of-a-form)

## The main function

One copy of the application, its settings, the theme and the language of last time ([app.go](../examples/recipes/app.go)):

```go
func main() {
	instance, ok := ui.AcquireSingleInstance("myapp", os.Args[1:])
	if !ok {
		return // the running copy got the arguments and came to the front
	}
	defer instance.Close()

	settings, _ := ui.OpenSettings("myapp") // empty on an error
	if settings.Bool("dark", false) {
		ui.ApplyDarkTheme()
	}
	if lang := settings.String("language", ""); lang != "" {
		ui.SetLanguage(lang) // else the system's
	}

	editor := NewEditor(settings)
	instance.OnActivate(func(args []string) { editor.Form.Activate() })
	editor.Form.Exec() // returns when the window closes
	settings.Save()
}
```

`Exec` runs the event loop in the main goroutine; everything that touches widgets runs there ([rules.md](rules.md#the-ui-thread)).

## The main window: menu, toolbar, status bar

A command is an `Action`: one for the menu item, the toolbar button and the shortcut. Its state follows the application through `SetEnabledFunc` ([editor.go](../examples/recipes/editor.go)):

```go
save := ui.NewAction("&Save", e.save).SetShortcut("Mod+S"). // Cmd on macOS, Ctrl elsewhere
	SetEnabledFunc(func() bool { return e.modified })

bar := ui.NewMenuBar()
file := bar.AddMenu("&File")
file.AddAction(open)
file.AddAction(save)
file.AddSeparator()
file.AddAction(quit)
form.SetMenuBar(bar)

tools := ui.NewToolBar()
tools.AddAction(open)
tools.AddAction(save)

text := ui.NewTextBox()
text.SetMultiline(true)
text.OnTextEdited.Connect(func(string) { e.setModified(true) }) // the user's edits only

status := ui.NewStatusBar()

p := form.Panel()
p.AddWidget(0, 0, tools)
p.AddWidget(1, 0, text) // a multiline box takes the free room
p.AddWidget(2, 0, status)
form.RememberState(settings, "editor") // the place and size of last time
```

See [actions.md](ui/content/actions.md), [menu_bar.md](ui/content/menu_bar.md), [application.md](ui/content/application.md).

## Ask before unsaved changes are lost

`OnClose` runs for the close button, `RequestClose` and the system's Quit; returning false keeps the window. The question's answers come in handlers:

```go
form.OnClose = func() bool {
	if !e.modified {
		return true
	}
	e.ask(func() { e.modified = false; form.Close() })
	return false
}

// ask runs then after the user agreed to lose the changes, or saved them
func (e *Editor) ask(then func()) {
	if !e.modified {
		then()
		return
	}
	ui.ShowQuestionMessageBoxYesNoCancel(form.Panel(), "Unsaved changes",
		"Save the changes to "+e.name()+"?",
		func() { // Yes
			if e.save(); !e.modified {
				then()
			}
		},
		then, // No
		nil)  // Cancel, Esc, the close button
}
```

The same `ask` guards New and Open. `form.Close()` closes without asking.

## Open and save a file

The file dialogs return at once; the result comes to the handler, an empty one when cancelled:

```go
form.ShowOpenFileDialog(ui.OpenFileDialogOptions{
	Filters: []ui.FileDialogFilter{{DisplayName: "Text files", Patterns: []string{"*.txt"}}},
}, func(paths []string, err error) {
	if err != nil || len(paths) == 0 {
		return
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		ui.ShowMessageBox(form.Panel(), "Open", err.Error())
		return
	}
	text.SetText(string(data))
})

form.ShowSaveFileDialog(ui.SaveFileDialogOptions{DefaultFileName: "untitled.txt"},
	func(path string, err error) { /* write to path */ })
```

## A dialog with fields

A dialog embeds `ui.DialogContent`. `DialogButtonBox` puts OK and Cancel in the order of the platform, Enter presses OK, Esc Cancel. The values are checked before the dialog closes ([widgets.go](../examples/recipes/widgets.go)):

```go
type ConnectionDialog struct {
	ui.DialogContent
	host *ui.TextBox
	port *ui.NumBox
	OnOK func(host string, port int)
}

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

// Shown over a form; ShowDialog returns at once
d := NewConnectionDialog("localhost", 8080)
d.OnOK = func(host string, port int) { connect(host, port) }
form.Panel().ShowDialog(d)
```

A dialog without a size gets the size its widgets need. See [dialog.md](ui/content/dialog.md), [layouts.md](ui/content/layouts.md); for a single value `ui.ShowInputDialog` ([standard_dialogs.md](ui/content/standard_dialogs.md)).

## Work in the background with a progress dialog

The work runs in a goroutine with a context, cancelled by the dialog's Cancel button or when the form closes; it reports its progress, and its result comes to the UI thread:

```go
ui.RunWithProgress(form.Panel(), "Counting", "Counting the files...",
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
	func(n int, err error) { // on the UI thread
		if errors.Is(err, context.Canceled) {
			return
		}
		status.ShowMessage(fmt.Sprint(n, " files"), 3*time.Second)
	})
```

Without a dialog: `ui.RunAsync`, with the progress in your own widgets ([async.md](ui/content/async.md)).

## Data from a goroutine

A goroutine never touches a widget; it hands the change to the UI thread:

```go
go func() {
	for v := range values {
		ui.Invoke(func() { label.SetText(fmt.Sprintf("%.1f", v)) })
	}
}()
```

The form repaints after the function. For a widget that paints data kept elsewhere, `widget.RequestRepaint()` from the goroutine is enough: many requests before a frame give one repaint. Live charts have their own sources ([timechart.md](ui/content/timechart.md)).

## A list filtered as you type

The filter runs once the typing pauses: a single-shot timer started again on each change.

```go
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
debounce := panel.NewTimer(300*time.Millisecond, filter)
debounce.SetSingleShot(true)
search.OnTextChanged.Connect(func(string) { debounce.Start() })

list.OnActivated.Connect(func(row int) { open(list.Model().Text(row, 0)) }) // double click, Enter
```

See [listview.md](ui/content/listview.md), [timer.md](ui/content/timer.md).

## A table of your own data

A model shows your slice as it is; the table asks only for the rows it draws, so a million rows cost what the visible ones do. `ColumnName` gives the headers, `ModelSignal` tells the table to show changes:

```go
type peopleModel struct {
	ui.ModelSignal
	people []Person
}

func (m *peopleModel) RowCount() int             { return len(m.people) }
func (m *peopleModel) ColumnCount() int          { return 2 }
func (m *peopleModel) ColumnName(col int) string { return [...]string{"Name", "Age"}[col] }
func (m *peopleModel) Text(row, col int) string {
	if col == 0 {
		return m.people[row].Name
	}
	return fmt.Sprint(m.people[row].Age)
}

model := &peopleModel{people: people}
table := ui.NewTable()
table.SetModel(model)
// later
model.people = append(model.people, p)
model.Changed() // the table shows the new row
```

For a quick one, `ui.NewFuncModel(rows, columns, text)`. Styles, editing and sorting of cells: [table.md](ui/content/table.md), [listview.md](ui/content/listview.md).

## A widget of your own

Embed `ui.Widget`, paint in `SetOnPaint`, take input from the signals of the base widget, call `Update()` when your data change: only your widget is painted again.

```go
type Dots struct {
	ui.Widget
	points  [][2]int
	OnAdded ui.Signal[int] // a signal of its own
}

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
		d.Update()
		d.OnAdded.Emit(len(d.points))
	})
	return d
}
```

Sizes are logical pixels: on a screen of scale 2 the canvas draws twice as dense by itself. See [widget.md](ui/content/widget.md), [canvas.md](ui/content/canvas.md), and [rules.md](rules.md#a-widget-of-ones-own) for what may be overridden.

## Files dropped on the window

```go
form.SetOnFilesDropped(func(files []string, x, y int) {
	for _, f := range files {
		open(f)
	}
})
```

Drop targets inside the window, and dragging files out of it: [dragdrop.md](ui/content/dragdrop.md).

## Light and dark theme

```go
func ToggleTheme(settings *ui.Settings) {
	dark := !settings.Bool("dark", false)
	if dark {
		ui.ApplyDarkTheme()
	} else {
		ui.ApplyLightTheme()
	}
	settings.Set("dark", dark)
}
```

The open forms repaint themselves. What the application colored itself, e.g. its icons, it recolors in `ui.OnThemeChanged` ([theme.md](ui/content/theme.md)).

## Translations

The texts are fields of a struct, a language is a value of it: a misspelled name is a compile error, a missing translation falls back to English. Texts set by functions follow `ui.SetLanguage` live:

```go
type Strings struct {
	Title string
	Files func(n int) string
}

var catalog = i18n.NewCatalog(Strings{
	Title: "Files",
	Files: func(n int) string { return fmt.Sprint(n, " ", i18n.Plural("en", n, "file", "files")) },
}, map[string]Strings{
	"ru": {
		Title: "Файлы",
		Files: func(n int) string { return fmt.Sprint(n, " ", i18n.Plural("ru", n, "файл", "файла", "файлов")) },
	},
})

func T() *Strings { return catalog.Get(ui.Language()) }

form.SetTitleFunc(func() string { return T().Title })
files.SetTextFunc(func() string { return T().Files(count()) })
```

The library's own texts (OK, Cancel, the calendar, the file and print dialogs) are built in for English, Russian, Chinese, Japanese, Korean, Turkish, Indonesian, Vietnamese and the main languages of Europe; `ui.RegisterUIStrings` adds others.

## An application in the tray

The close button hides the window, a click on the icon shows it, Quit in the icon's menu quits:

```go
tray, err := ui.NewTrayIcon()
if err != nil {
	return err // no tray here (ui.ErrTrayNotSupported): keep the window
}
quitting := false
tray.SetIcon(icon)
tray.SetOnClick(func() { form.Activate() })
tray.SetMenu(
	ui.TrayMenuItem{Text: "Show", OnClick: func() { form.Activate() }},
	ui.TrayMenuItem{Separator: true},
	ui.TrayMenuItem{Text: "Quit", OnClick: func() { quitting = true; tray.Close(); form.Close() }},
)
form.OnClose = func() bool {
	if quitting {
		return true
	}
	form.Hide()
	tray.ShowNotification("My app", "Still running in the tray")
	return false
}
```

See [trayicon.md](ui/content/trayicon.md), [toast.md](ui/content/toast.md) for messages inside the window.

## A test of a form

`ui/uitest` shows a form without a display and acts as the user ([recipes_test.go](../examples/recipes/recipes_test.go)):

```go
func TestEditor(t *testing.T) {
	e := NewEditor(nil)
	w := uitest.Show(t, e.Form) // closed at the end of the test
	w.Click(e.text)
	w.Type("hello")
	if w.Title() != "Untitled * - Editor" {
		t.Fatalf("title %q", w.Title())
	}
	w.RequestClose()        // the close button
	q := w.Dialog()         // the question
	q.Click(q.Find("No"))
	w.Settle()
	if !w.Closed() {
		t.Error("the window stays after No")
	}
}
```

Timers fire on `w.Wait(d)`, golden pictures with `w.MatchGolden(name)`: [testing.md](testing.md).
