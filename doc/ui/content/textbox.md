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

## The cursor, the selection, the edits from the code

Places in the text are `ui.TextPos{Line, Col}`, in runes, from 0.

- `CursorPosition()`, `SetCursorPosition(pos)`, `ScrollToCursor()`
- `Selection() (from, to)`, `SetSelection(anchor, cursor)`, `HasSelection()`, `SelectedText()`, `SelectAllText()`
- `InsertText(s)`: at the cursor, replacing the selection; Undo takes it back (`SetText` starts the history anew). To replace the whole text undoably: `SelectAllText()`, then `InsertText(...)`.
- `Cut()`, `Copy()`, `Paste()`, `Undo()`, `Redo()`
- `LineCount()`, `Line(i)`, `Lines()`; `PosAt(offset)`, `OffsetOf(pos)`: the rune offsets in `Text()`, for searching it.
- Ctrl+Home and Ctrl+End go to the start and the end of a multiline text.

## A code editor

```go
code := ui.NewTextBox()
code.SetMultiline(true)
code.SetFontFamily(ui.FontFamilyMono)
code.SetLineNumbers(true)
code.SetHighlighter(func(line int, text string) []ui.TextSpan {
	// the styled parts of the line: runes [Start, End)
	return []ui.TextSpan{{Start: 0, End: 4, Color: keywordColor, Bold: true}}
})
```

The highlighter is asked for the lines shown only, at each paint: it may be a syntax highlighter or the matches of a search. Bold and italic keep the places of the regular face: they fit a monospaced font best.

## Signals

`OnTextChanged` (any change), `OnTextEdited` (the user's), `OnSubmit` (Enter in a single-line box), `OnEditingFinished` (Enter, or leaving the box after a change); see [events](../../events.md).
