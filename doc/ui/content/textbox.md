# TextBox

Single-line or multi-line text input.

## Create

```go
tb := ui.NewTextBox()
tb.SetHint("Type here")
```

## Single vs multi-line

```go
tb.SetMultiline(false) // default
// tb.SetMultiline(true)
```

## Read-only / password

```go
tb.SetReadOnly(true)
tb.SetIsPassword(true)
```

## Events

```go
tb.SetOnTextChanged(func() {
  fmt.Println(tb.Text())
})
```

## Undo and Redo

Ctrl+Z takes the last edit back, Ctrl+Y or Ctrl+Shift+Z makes it again (Cmd on macOS). A run of
typing, or of Backspace or Delete, at one place is one step; a paste is one step.

- `Undo()`, `Redo()`, `CanUndo()`, `CanRedo()` - e.g. for the Edit menu.
- `ClearUndo()` forgets the history; `SetText` from the code starts a new one.
