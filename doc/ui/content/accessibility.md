# Accessibility

Screen readers (Orca, NVDA, VoiceOver) and UI automation tools see a form as a tree of accessible objects. Each object has:

- a role (button, text field, list...);
- a name, and a value where the role has one;
- states (focused, checked, disabled...);
- actions (press, toggle, select).

Layout containers (boxes, panels, splitters) are not objects: their widgets belong to the next object up.

The widgets of nui describe themselves:

- a button, a check box or a label is named by its text;
- a text field, a slider or a combo box is named by the label of its row in a `FormLayout`, or by a label given with `Label.SetBuddy`;
- the value is the text of a field, the number of a slider or spin box, the item chosen in a combo box;
- the items of a list, the rows of a table, the nodes of a tree shown now and the tabs of a `TabWidget` are objects of their own.

A widget without a text needs a name:

```go
refresh := ui.NewToolButton(icons.Refresh, "Refresh", reload) // named by the tooltip
search := ui.NewTextBox()
search.SetAccessibleName("Search")
search.SetAccessibleDescription("Finds the files by their names")
nameLabel.SetBuddy(nameField) // "Name: text"
custom.SetAccessibleRole(ui.RoleButton) // a widget that draws a button itself
```

## In tests

`Form.Accessible()` (or `uitest.Window.Accessible()`) returns the tree. `String()` prints it, and `Find(role, name)` finds an object to check or to act on:

```go
root := w.Accessible()
// window "Settings"
//   text "Name" value="Ann" focused
//   check box "Remember me"
//   button "OK"
root.Find(ui.RoleButton, "OK").DoAction("press")
root.Find(ui.RoleTextField, "Name").SetText("Bob")
root.Find(ui.RoleSlider, "Speed").SetCurrent(5)
```

## Platforms

- **Linux:** AT-SPI.
  - The objects are exported on the accessibility bus with the Accessible, Component, Action, Text, EditableText and Value interfaces.
  - A focus change is sent as an event.
  - As in Qt, the bridge starts only when the desktop has accessibility on: a screen reader runs, or the setting is on. `NUI_ACCESSIBILITY=1` starts it always and `NUI_ACCESSIBILITY=0` never.
- **Windows (UI Automation) and macOS (NSAccessibility):** not yet. The tree is the same on every system, and only the bridges to them are missing.
