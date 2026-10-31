package ui

import (
	"image"
	"slices"
	"sync"
	"time"
)

// Action is a command of the application - Save, Undo, Bold - with its text,
// icon, shortcut and state, shown by any number of menu items and toolbar
// buttons (ContextMenu.AddAction, ToolBar.AddAction): they all show its
// state and run it. Disable the action, and all its items are disabled.
//
//	save := ui.NewAction("&Save", doSave).
//		SetIcon(icons.Save(16)).SetShortcut("Mod+S").
//		SetEnabledFunc(func() bool { return doc.Modified() })
//	fileMenu.AddAction(save)
//	toolbar.AddAction(save)
//
// An enabled function is asked again before a menu opens and every few
// tenths of a second, so the items follow the state of the application
// without code that updates them.
type Action struct {
	text        string
	icon        image.Image
	shortcut    string
	tooltip     string
	enabled     bool
	enabledFunc func() bool
	checkable   bool
	checked     bool
	run         func()
	shown       int // items of it in open forms (see show)

	// OnTriggered: the action ran (after its function)
	OnTriggered Signal[*Action]
	// OnChanged: the text, the icon, the enabled or the checked state
	// changed: the items of the action follow it
	OnChanged Signal[*Action]
}

// NewAction makes an enabled action running f. The text may have a
// mnemonic: "&Save".
func NewAction(text string, f func()) *Action {
	return &Action{text: text, enabled: true, run: f}
}

func (a *Action) changed() {
	a.OnChanged.Emit(a)
}

// SetText changes the text
func (a *Action) SetText(text string) *Action {
	a.text = text
	a.changed()
	return a
}

func (a *Action) Text() string { return a.text }

// SetIcon sets the icon of the items
func (a *Action) SetIcon(icon image.Image) *Action {
	a.icon = icon
	a.changed()
	return a
}

func (a *Action) Icon() image.Image { return a.icon }

// SetShortcut sets the shortcut, e.g. "Mod+S" (see ParseShortcut): shown by
// the menu items; pressed, it runs the action from a menu of the menu bar,
// or of a form the action is added to (Form.AddAction)
func (a *Action) SetShortcut(shortcut string) *Action {
	MustParseShortcut(shortcut)
	a.shortcut = shortcut
	a.changed()
	return a
}

func (a *Action) Shortcut() string { return a.shortcut }

// SetTooltip sets the tooltip of the toolbar buttons; without it, the text
// and the shortcut
func (a *Action) SetTooltip(tooltip string) *Action {
	a.tooltip = tooltip
	a.changed()
	return a
}

// Tooltip returns the tooltip of the toolbar buttons
func (a *Action) Tooltip() string {
	if a.tooltip != "" {
		return a.tooltip
	}
	text, _, _ := parseMnemonic(a.text)
	if a.shortcut != "" {
		text += " (" + MustParseShortcut(a.shortcut).String() + ")"
	}
	return text
}

// SetEnabled enables or disables the action and its items
func (a *Action) SetEnabled(enabled bool) *Action {
	if a.enabled != enabled {
		a.enabled = enabled
		a.changed()
	}
	return a
}

// SetEnabledFunc makes the action enabled when f returns true: f is asked
// again before a menu opens and every few tenths of a second
func (a *Action) SetEnabledFunc(f func() bool) *Action {
	a.enabledFunc = f
	a.Update()
	return a
}

// Enabled reports whether the action runs
func (a *Action) Enabled() bool {
	return a.enabled
}

// Update asks the enabled function again
func (a *Action) Update() {
	if a.enabledFunc != nil {
		a.SetEnabled(a.enabledFunc())
	}
}

// SetCheckable makes the action a toggle: running it checks or unchecks it,
// its menu items show a check mark, its buttons a pressed look
func (a *Action) SetCheckable(checkable bool) *Action {
	a.checkable = checkable
	a.changed()
	return a
}

func (a *Action) Checkable() bool { return a.checkable }

// SetChecked checks or unchecks a checkable action
func (a *Action) SetChecked(checked bool) *Action {
	if a.checked != checked {
		a.checked = checked
		a.changed()
	}
	return a
}

func (a *Action) Checked() bool { return a.checked }

// Trigger runs the action as its items do: nothing if it is disabled; a
// checkable one toggles first
func (a *Action) Trigger() {
	a.Update()
	if !a.enabled {
		return
	}
	if a.checkable {
		a.SetChecked(!a.checked)
	}
	if a.run != nil {
		a.run()
	}
	a.OnTriggered.Emit(a)
}

// The actions shown by items in open forms: their enabled functions are
// asked again from time to time. An action whose items are all gone (their
// forms closed) is not kept here, nor asked.
var (
	dynamicActionsMu   sync.Mutex
	dynamicActions     []*Action
	lastActionsUpdated time.Time
)

// show counts an item of the action shown (+1) or gone (-1)
func (a *Action) show(delta int) {
	dynamicActionsMu.Lock()
	defer dynamicActionsMu.Unlock()
	a.shown += delta
	switch {
	case a.shown == 1 && delta > 0:
		dynamicActions = append(dynamicActions, a)
	case a.shown == 0:
		dynamicActions = slices.DeleteFunc(dynamicActions, func(o *Action) bool { return o == a })
	}
}

// linkItem makes the item w follow the action with sync while w is in an
// open form (see Widget.linkWhileShown): an action kept longer than the
// item's form doesn't keep it
func (a *Action) linkItem(w *Widget, sync func(*Action)) {
	w.linkWhileShown(func() func() {
		a.show(1)
		a.Update()
		sync(a)
		conn := a.OnChanged.Connect(sync)
		return func() {
			conn.Disconnect()
			a.show(-1)
		}
	})
}

// updateDynamicActions asks the enabled functions, at most every 150 ms;
// UI thread
func updateDynamicActions() {
	if time.Since(lastActionsUpdated) < 150*time.Millisecond {
		return
	}
	lastActionsUpdated = time.Now()
	dynamicActionsMu.Lock()
	actions := append([]*Action(nil), dynamicActions...)
	dynamicActionsMu.Unlock()
	for _, a := range actions {
		a.Update()
	}
}

// AddAction adds an item showing the action: its text, icon, shortcut,
// enabled and checked state, following its changes; a click runs it
func (c *ContextMenu) AddAction(a *Action) *ContextMenuItem {
	item := c.AddItem(a.text, a.Trigger)
	item.action = a
	sync := func(*Action) {
		item.SetText(a.text)
		item.SetImage(a.icon)
		item.SetShortcut(a.shortcut)
		item.SetCheckable(a.checkable)
		item.SetChecked(a.checked)
		item.SetEnabled(a.enabled)
	}
	sync(a)
	a.linkItem(&item.Widget, sync)
	return item
}

// AddAction makes the action's shortcut work in the form, whatever has the
// focus (an action in a menu of the menu bar works from there already)
func (c *Form) AddAction(a *Action) {
	if a.shortcut != "" {
		c.AddShortcut(a.shortcut, a.Trigger)
	}
}
