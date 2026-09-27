# EditableComboBox

A text field with a list of suggestions. Typing shows the items containing the typed text - the
ones that start with it first, the matched part highlighted. The arrow button, Alt+Down or F4 shows
all the items. Any text can be entered, not only an item.

```go
city := ui.NewEditableComboBox()
city.SetHint("City")
city.SetItems([]string{"Amsterdam", "Berlin", "Bern", "London"})
city.SetOnItemSelected(func(index int, text string) { ... })
city.SetOnAccept(func(text string) { search(text) })
```

| Key | Action |
|---|---|
| typing | suggestions (with `SetAutoComplete(true)`, the default) |
| Down / Up, PageDown / PageUp | move in the list (Down opens it) |
| Enter | take the chosen item; without one - `SetOnAccept` |
| Escape | close the list |
| Alt+Down, F4, the arrow button | show / hide all the items |

The keyboard focus stays in the field while the list is open.

- `SetItems([]string)`, `AddItem(text)`, `Items()`
- `Text()` / `SetText(text)` - `SetText` shows no suggestions and calls no callback.
- `SetHint(text)`, `SetAutoComplete(bool)`, `TextBox()` - the field itself.
- `SetOnTextChanged(func())`, `SetOnItemSelected(func(index int, text string))`, `SetOnAccept(func(text string))`
- `IsPopupOpen()`
