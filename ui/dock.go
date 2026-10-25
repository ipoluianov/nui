package ui

// Docking: tool panels around the main area of a window, as in IDEs - a
// file tree at the left, a properties panel at the right, a log at the
// bottom. The user resizes the areas with their splitters, closes a panel,
// moves it to another area from its menu; several panels in one area are
// tabs. The layout is saved and restored with the settings.
//
//	docks := ui.NewDockLayout(editor)
//	files := docks.AddDock("Files", fileTree, ui.DockLeft)
//	docks.AddDock("Log", logView, ui.DockBottom)
//	viewMenu.AddAction(files.ToggleAction()) // Show or hide it
//	docks.RememberState(settings, "docks")
//	form.Panel().AddWidget(0, 0, docks)

// DockArea is where a dock panel is
type DockArea int

const (
	DockLeft DockArea = iota
	DockRight
	DockBottom
)

var dockAreaNames = []string{"Left", "Right", "Bottom"}

// DockLayout is the main area of a window with the dock panels around it
type DockLayout struct {
	Panel
	central Widgeter
	docks   []*DockWidget

	outer, inner, middle *Splitter
	areas                [3]*Panel

	// pendingSave saves the state when the form closes (RememberState),
	// once the layout is in a form
	pendingSave func()
	saveHooked  *Form
}

// DockWidget is a panel of a DockLayout: a title bar with a menu and a
// close button, and the content
type DockWidget struct {
	layout  *DockLayout
	title   string
	content Widgeter
	area    DockArea
	shown   bool
	frame   *VBox
	toggle  *Action
}

// NewDockLayout makes the layout with the central widget and no panels
func NewDockLayout(central Widgeter) *DockLayout {
	c := &DockLayout{central: central}
	c.InitWidget()
	c.SetTypeName("DockLayout")
	c.SetPanelPadding(0)
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	for i := range c.areas {
		c.areas[i] = NewPanel()
		c.areas[i].SetPanelPadding(0)
		c.areas[i].SetVisible(false)
	}
	c.middle = NewVSplitter()
	c.middle.SetWidgets(central, c.areas[DockBottom])
	c.middle.SetSecondSize(180)
	c.inner = NewHSplitter()
	c.inner.SetWidgets(c.middle, c.areas[DockRight])
	c.inner.SetSecondSize(240)
	c.outer = NewHSplitter()
	c.outer.SetWidgets(c.areas[DockLeft], c.inner)
	c.outer.SetFirstSize(240)
	c.AddWidget(0, 0, c.outer)
	return c
}

// AddDock adds a panel with the title and the content in the area, shown
func (c *DockLayout) AddDock(title string, content Widgeter, area DockArea) *DockWidget {
	d := &DockWidget{layout: c, title: title, content: content, area: area, shown: true}
	d.frame = NewVBox()
	d.frame.SetCellPadding(0)
	d.frame.Add(d.titleBar())
	d.frame.Add(content)
	d.frame.SetRowStretch(1, 1) // the title bar keeps its height
	d.toggle = NewAction(title, nil).SetCheckable(true).SetChecked(true)
	d.toggle.OnTriggered.Connect(func(a *Action) { d.SetShown(a.Checked()) })
	c.docks = append(c.docks, d)
	c.rebuild()
	return d
}

// Docks returns the panels
func (c *DockLayout) Docks() []*DockWidget {
	return c.docks
}

// SetAreaSize sets the width of the left or the right area, or the height
// of the bottom one
func (c *DockLayout) SetAreaSize(area DockArea, size int) {
	switch area {
	case DockLeft:
		c.outer.SetFirstSize(size)
	case DockRight:
		c.inner.SetSecondSize(size)
	case DockBottom:
		c.middle.SetSecondSize(size)
	}
}

// AreaSize returns the size SetAreaSize sets
func (c *DockLayout) AreaSize(area DockArea) int {
	switch area {
	case DockLeft:
		return c.outer.FirstSize()
	case DockRight:
		return c.inner.SecondSize()
	}
	return c.middle.SecondSize()
}

// rebuild puts the shown panels in their areas: one alone, more as tabs
func (c *DockLayout) rebuild() {
	for _, panel := range c.areas {
		for _, w := range panel.Widgets() {
			panel.RemoveWidget(w)
			w.attachToForm(w, nil)
		}
	}
	for _, d := range c.docks {
		d.frame.attachToForm(d.frame, nil)
	}
	for i, panel := range c.areas {
		var docks []*DockWidget
		for _, d := range c.docks {
			if d.shown && d.area == DockArea(i) {
				docks = append(docks, d)
			}
		}
		switch len(docks) {
		case 0:
			panel.SetVisible(false)
			continue
		case 1:
			panel.AddWidget(0, 0, docks[0].frame)
		default:
			tabs := NewTabWidget()
			for _, d := range docks {
				tabs.AddPage(d.title, d.frame)
			}
			panel.AddWidget(0, 0, tabs)
		}
		panel.SetVisible(true)
	}
	c.form.UpdateLayout()
}

// Title returns the title of the panel
func (d *DockWidget) Title() string { return d.title }

// Content returns the widget in the panel
func (d *DockWidget) Content() Widgeter { return d.content }

// Area returns where the panel is
func (d *DockWidget) Area() DockArea { return d.area }

// Shown reports whether the panel is shown
func (d *DockWidget) Shown() bool { return d.shown }

// SetShown shows or hides the panel
func (d *DockWidget) SetShown(shown bool) {
	if d.shown == shown {
		return
	}
	d.shown = shown
	d.toggle.SetChecked(shown)
	d.layout.rebuild()
}

// MoveTo moves the panel to the area, shown
func (d *DockWidget) MoveTo(area DockArea) {
	d.area = area
	d.shown = true
	d.toggle.SetChecked(true)
	d.layout.rebuild()
}

// ToggleAction returns the checkable action showing and hiding the panel:
// for the View menu
func (d *DockWidget) ToggleAction() *Action {
	return d.toggle
}

// titleBar is the title, the menu of the areas and the close button
func (d *DockWidget) titleBar() Widgeter {
	bar := NewHBox()
	bar.SetCellPadding(0)
	bar.SetAutoFillBackground(true)
	bar.SetRole("surface")
	title := NewLabel(" " + d.title)
	title.SetFontBold(true)
	bar.Add(title)
	bar.AddStretch()
	menu := NewContextMenu(nil)
	for i, name := range dockAreaNames {
		area := DockArea(i)
		menu.AddItem("Dock "+name, func() { d.MoveTo(area) })
	}
	menu.AddSeparator()
	menu.AddItem("Close", func() { d.SetShown(false) })
	more := d.titleButton("…", nil)
	more.SetOnClick(func() {
		x, y := more.RectClientAreaOnWindow()
		menu.attachToForm(menu, more.Form())
		menu.ShowMenu(x, y+more.Height())
	})
	bar.Add(more)
	bar.Add(d.titleButton("×", func() { d.SetShown(false) }))
	return bar
}

func (d *DockWidget) titleButton(text string, onClick func()) *Button {
	b := NewButton(text)
	size := ThemeControlHeight()
	b.SetMinSize(size, size)
	b.SetMaxSize(size, size)
	if onClick != nil {
		b.SetOnClick(onClick)
	}
	return b
}

// dockState is what RememberState saves
type dockState struct {
	Areas [3]int
	Docks map[string]dockWidgetState
}

type dockWidgetState struct {
	Area  DockArea
	Shown bool
}

// RememberState restores the layout saved in the settings under the key,
// if any - the sizes of the areas, where each panel is (by its title) and
// whether shown - and saves it there when the window closes
func (c *DockLayout) RememberState(settings *Settings, key string) {
	var state dockState
	if settings.Get(key, &state) {
		for i, size := range state.Areas {
			if size > 0 {
				c.SetAreaSize(DockArea(i), size)
			}
		}
		for _, d := range c.docks {
			if s, ok := state.Docks[d.title]; ok {
				d.area, d.shown = s.Area, s.Shown
				d.toggle.SetChecked(s.Shown)
			}
		}
		c.rebuild()
	}
	save := func() {
		state := dockState{Docks: map[string]dockWidgetState{}}
		for i := range state.Areas {
			state.Areas[i] = c.AreaSize(DockArea(i))
		}
		for _, d := range c.docks {
			state.Docks[d.title] = dockWidgetState{d.area, d.shown}
		}
		settings.Set(key, state)
		settings.Save()
	}
	c.pendingSave = save
	c.hookSave()
}

// hookSave makes the form save the state when it closes
func (c *DockLayout) hookSave() {
	if c.pendingSave != nil && c.form != nil && c.saveHooked != c.form {
		c.saveHooked = c.form
		c.form.onClosing(c.pendingSave)
	}
}

func (c *DockLayout) attachToForm(self Widgeter, form *Form) {
	c.Panel.attachToForm(self, form)
	c.hookSave()
}
