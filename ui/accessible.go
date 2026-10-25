package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
)

// Accessibility: what a screen reader (Orca, NVDA, VoiceOver) and the UI
// automation tools see of a form. Every widget is an accessible object of a
// role (a button, a text field, a list...), with a name, a value, states
// (focused, checked, disabled...) and actions (press, toggle); the layout
// containers (boxes, panels, splitters) are skipped, their widgets belong
// to the next object up. The platform reads the objects from the form (see
// Form.Accessible); on Linux they go to AT-SPI.
//
// The widgets of nui know their role, name and value: a button is named by
// its text, a text field by the label of its row in a FormLayout. A widget
// without a text (an icon button, a field without a label) is named with
// SetAccessibleName; a custom widget sets its role with SetAccessibleRole.

// AccessibleRole is what an accessible object is
type AccessibleRole int

const (
	// RoleNone: not an object of its own; a layout container. Its children
	// belong to its parent.
	RoleNone AccessibleRole = iota
	RoleWindow
	RoleDialog
	RoleGroup
	RoleButton
	RoleCheckBox
	RoleRadioButton
	RoleSwitch
	RoleLabel
	RoleLink
	RoleTextField
	RolePasswordField
	RoleTextArea
	RoleSpinButton
	RoleSlider
	RoleProgressBar
	RoleComboBox
	RoleList
	RoleListItem
	RoleTable
	RoleTableRow
	RoleTree
	RoleTreeItem
	RoleTabList
	RoleTab
	RoleMenuBar
	RoleMenu
	RoleMenuItem
	RoleToolBar
	RoleStatusBar
	RoleScrollBar
	RoleImage
	RoleCanvas
	RoleSplitter
	RoleCalendar
)

var roleNames = [...]string{"none", "window", "dialog", "group", "button", "check box", "radio button",
	"switch", "label", "link", "text", "password text", "text area", "spin button", "slider",
	"progress bar", "combo box", "list", "list item", "table", "table row", "tree", "tree item",
	"tab list", "tab", "menu bar", "menu", "menu item", "tool bar", "status bar", "scroll bar",
	"image", "canvas", "splitter", "calendar"}

func (r AccessibleRole) String() string {
	if int(r) < len(roleNames) {
		return roleNames[r]
	}
	return fmt.Sprintf("role %d", int(r))
}

// AccessibleNode is an accessible object: a snapshot of a widget, or of an
// item of one (a row of a list), made by Form.Accessible
type AccessibleNode struct {
	Role        AccessibleRole
	Name        string
	Description string
	// Value is the text of a text field, the value of a slider or a spin
	// button, the item chosen in a combo box
	Value string
	// Min, Max and Current are the range and the value of a slider, a spin
	// button, a progress bar
	Min, Max, Current float64
	// Rect is in the form's client area
	Rect image.Rectangle

	Focusable, Focused bool
	Enabled            bool
	Checkable, Checked bool
	Selectable         bool
	Selected           bool
	Editable           bool
	Expandable         bool
	Expanded           bool
	Multiline          bool
	// Caret is the position of the caret in Value, in runes
	Caret int
	// Level is the depth of a tree item, 0 at the top
	Level int

	Parent   *AccessibleNode
	Children []*AccessibleNode

	// Actions are the names of what DoAction does: "press", "toggle"...
	Actions []string
	doIt    func(action string) bool
	// setText, if set, replaces the text (Value) of an editable object
	setText func(text string) bool
	// setValue, if set, changes Current
	setValue func(v float64)
	focus    func()

	// Widget is the widget of the object, or the one whose item it is; nil
	// for the window
	Widget Widgeter
	form   *Form
	// Item is the index of the item of Widget (a row of a list); -1 for
	// the widget itself
	Item int
}

// DoAction does the action (one of Actions); false if the object has none
// of the name. On the UI thread.
func (n *AccessibleNode) DoAction(action string) bool {
	if n.doIt == nil || !slices.Contains(n.Actions, action) {
		return false
	}
	return n.doIt(action)
}

// SetText replaces the text of an editable object; false if it isn't one
func (n *AccessibleNode) SetText(text string) bool {
	if n.setText == nil || !n.Editable || !n.Enabled {
		return false
	}
	return n.setText(text)
}

// SetCurrent changes the value of a slider, a spin button, a scroll bar;
// false if it has none
func (n *AccessibleNode) SetCurrent(v float64) bool {
	if n.setValue == nil || !n.Enabled {
		return false
	}
	n.setValue(max(n.Min, min(n.Max, v)))
	return true
}

// GrabFocus focuses the object, if it can be
func (n *AccessibleNode) GrabFocus() bool {
	if n.focus == nil || !n.Focusable || !n.Enabled {
		return false
	}
	n.focus()
	return true
}

// Find returns the first object of the role and the name below n (n
// itself too), depth first; RoleNone and "" match any
func (n *AccessibleNode) Find(role AccessibleRole, name string) *AccessibleNode {
	if (role == RoleNone || n.Role == role) && (name == "" || n.Name == name) {
		return n
	}
	for _, c := range n.Children {
		if f := c.Find(role, name); f != nil {
			return f
		}
	}
	return nil
}

// FindAll returns the objects below n (n itself too) that match
func (n *AccessibleNode) FindAll(match func(*AccessibleNode) bool) []*AccessibleNode {
	var found []*AccessibleNode
	if match(n) {
		found = append(found, n)
	}
	for _, c := range n.Children {
		found = append(found, c.FindAll(match)...)
	}
	return found
}

// Focused returns the focused object below n, or nil
func (n *AccessibleNode) FocusedNode() *AccessibleNode {
	if f := n.FindAll(func(c *AccessibleNode) bool { return c.Focused }); len(f) > 0 {
		return f[len(f)-1]
	}
	return nil
}

// String is the tree below n, an object a line: the role, "name", the
// value and the states; for tests and debugging
func (n *AccessibleNode) String() string {
	var b strings.Builder
	n.write(&b, 0)
	return b.String()
}

func (n *AccessibleNode) write(b *strings.Builder, depth int) {
	b.WriteString(strings.Repeat("  ", depth))
	b.WriteString(n.Role.String())
	if n.Name != "" {
		fmt.Fprintf(b, " %q", n.Name)
	}
	if n.Value != "" {
		fmt.Fprintf(b, " value=%q", n.Value)
	}
	for _, s := range []struct {
		on   bool
		name string
	}{{n.Focused, "focused"}, {!n.Enabled, "disabled"}, {n.Checkable && n.Checked, "checked"},
		{n.Selected, "selected"}, {n.Expandable && n.Expanded, "expanded"}, {n.Expandable && !n.Expanded, "collapsed"}} {
		if s.on {
			b.WriteString(" " + s.name)
		}
	}
	b.WriteByte('\n')
	for _, c := range n.Children {
		c.write(b, depth+1)
	}
}

// SetAccessibleName names the widget for a screen reader: for a widget
// without a text of its own (an icon button, a text field without a label)
func (c *Widget) SetAccessibleName(name string) {
	c.accessibleName = name
}

// AccessibleName returns the name set by SetAccessibleName
func (c *Widget) AccessibleName() string {
	return c.accessibleName
}

// SetAccessibleDescription sets a longer text a screen reader tells after
// the name, e.g. what a field is for
func (c *Widget) SetAccessibleDescription(description string) {
	c.accessibleDescription = description
}

// SetAccessibleRole sets the role of a custom widget, e.g. RoleButton for
// a widget that draws a button itself; RoleNone makes it a container
func (c *Widget) SetAccessibleRole(role AccessibleRole) {
	c.accessibleRole = role
	c.accessibleRoleSet = true
}

// accessibleWidget is a widget that describes itself: its role, name,
// value and states, and the actions; n has the defaults of the widget
type accessibleWidget interface {
	accessibleInfo(n *AccessibleNode)
}

// accessibleItemsWidget is a widget whose items are objects of their own
// (the rows of a list, the tabs): they follow its widgets
type accessibleItemsWidget interface {
	accessibleItemCount() int
	// accessibleItem fills the item i; n has the defaults of its widget
	accessibleItem(n *AccessibleNode, i int)
}

// maxAccessibleItems is the most items of a widget in Form.Accessible: a
// table of a million rows is not copied whole
const maxAccessibleItems = 10000

// Accessible returns the accessible objects of the form: the window and,
// below it, the menu bar and the widgets. A snapshot, on the UI thread.
func (c *Form) Accessible() *AccessibleNode {
	root := c.accessibleRoot()
	var fill func(n *AccessibleNode)
	fill = func(n *AccessibleNode) {
		for _, child := range accessibleChildren(n) {
			child.Parent = n
			n.Children = append(n.Children, child)
			fill(child)
		}
	}
	fill(root)
	return root
}

// accessibleRoot is the object of the window, without its children
func (c *Form) accessibleRoot() *AccessibleNode {
	c.layoutNow()
	root := &AccessibleNode{Role: RoleWindow, Name: c.title, Enabled: true, Item: -1,
		Rect: image.Rect(0, 0, c.width, c.height), Focused: c.active, form: c}
	if c.parentForm != nil {
		root.Role = RoleDialog
	}
	return root
}

// accessibleChildren returns the objects right below n, without their own
// children: the objects of its widgets (the widgets of containers taken up)
// and its items
func accessibleChildren(n *AccessibleNode) []*AccessibleNode {
	var children []*AccessibleNode
	if n.Item >= 0 {
		return nil
	}
	if n.Widget == nil {
		// The window: the menu bar, then the widgets
		if bar := n.form.menuBar; bar != nil && !bar.native {
			children = appendAccessible(children, bar)
		}
		return appendAccessible(children, n.form.topWidget)
	}
	if !accessibleLeaf(n.Role) {
		for _, w := range accessibleWidgets(n.Widget) {
			children = appendAccessible(children, w)
		}
	}
	if items, ok := n.Widget.(accessibleItemsWidget); ok {
		count := min(items.accessibleItemCount(), maxAccessibleItems)
		for i := range count {
			children = append(children, accessibleItemNode(n.Widget, i))
		}
	}
	return children
}

// appendAccessible adds the object of the widget, or the objects of its
// widgets if it is a container (RoleNone)
func appendAccessible(list []*AccessibleNode, w Widgeter) []*AccessibleNode {
	if w == nil || !w.IsVisible() {
		return list
	}
	n := accessibleNodeOf(w)
	if n.Role != RoleNone {
		return append(list, n)
	}
	for _, child := range accessibleWidgets(n.Widget) {
		list = appendAccessible(list, child)
	}
	return list
}

// accessibleWidgets are the visible widgets of w in reading order: by
// rows, then left to right
func accessibleWidgets(w Widgeter) []Widgeter {
	var children []Widgeter
	for _, child := range w.widgetBase().widgets {
		if child.IsVisible() {
			children = append(children, child)
		}
	}
	slices.SortStableFunc(children, func(a, b Widgeter) int {
		if a.Y() != b.Y() {
			return a.Y() - b.Y()
		}
		return a.X() - b.X()
	})
	return children
}

// accessibleNodeOf is the object of the widget, without its children
func accessibleNodeOf(w Widgeter) *AccessibleNode {
	if registered, ok := lookupWidget(w.Id()); ok {
		w = registered
	}
	base := w.widgetBase()
	x, y := w.RectClientAreaOnWindow()
	n := &AccessibleNode{
		Widget:      w,
		Item:        -1,
		Rect:        image.Rect(x, y, x+w.Width(), y+w.Height()),
		Enabled:     w.Enabled(),
		Focusable:   base.canBeFocused,
		Focused:     base.IsFocused(),
		Description: base.accessibleDescription,
		focus:       w.Focus,
	}
	if a, ok := w.(accessibleWidget); ok {
		a.accessibleInfo(n)
	}
	if base.accessibleRoleSet {
		n.Role = base.accessibleRole
	}
	if base.accessibleName != "" {
		n.Name = base.accessibleName // set by the application: it wins
	}
	return n
}

// accessibleItemNode is the object of the item i of the widget
func accessibleItemNode(w Widgeter, i int) *AccessibleNode {
	n := &AccessibleNode{Widget: w, Item: i, Enabled: w.Enabled()}
	w.(accessibleItemsWidget).accessibleItem(n, i)
	return n
}

// accessibleLeaf: the objects of the role tell everything themselves; the
// widgets they are made of (a combo box's arrow) are not objects
func accessibleLeaf(role AccessibleRole) bool {
	switch role {
	case RoleButton, RoleCheckBox, RoleRadioButton, RoleSwitch, RoleLabel, RoleLink, RoleTextField,
		RolePasswordField, RoleTextArea, RoleSpinButton, RoleSlider, RoleProgressBar, RoleComboBox,
		RoleTab, RoleMenuItem, RoleScrollBar, RoleImage, RoleCalendar, RoleTabList, RoleList,
		RoleTable, RoleTree:
		return true
	}
	return false
}

// pressAction is the action of a button-like object
func pressAction(n *AccessibleNode, press func()) {
	n.Actions = []string{"press"}
	n.doIt = func(string) bool {
		if !n.Enabled {
			return false
		}
		press()
		return true
	}
}

// toggleAction is the action of a check box or a switch
func toggleAction(n *AccessibleNode, toggle func()) {
	n.Actions = []string{"toggle"}
	n.doIt = func(string) bool {
		if !n.Enabled {
			return false
		}
		toggle()
		return true
	}
}
