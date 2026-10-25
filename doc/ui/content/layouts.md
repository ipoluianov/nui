# Layouts

Every container (`Panel`, a form's `Panel()`, a dialog) lays its children out in a grid: `AddWidget(row, col, w)`. The boxes below are panels that fill the grid for the usual shapes.

## HBox, VBox

A row or a column of widgets, without the padding around:

```go
row := ui.NewHBox()
row.Add(ui.NewLabel("Find:"))
row.Add(findBox)         // an expandable widget takes the free width
row.AddSpacing(12)       // an empty space of 12 px
row.Add(findButton)

col := ui.NewVBox()
col.Add(row)
col.Add(results)
col.AddStretch()         // the rest of the height stays empty below
form.Panel().AddWidget(0, 0, col)
```

`Add` returns the widget. `AddStretch` puts an expandable empty space: the widgets after it go to the right (the bottom) edge.

## FormLayout

Labeled fields: the labels in a column, the fields taking the rest of the width.

```go
fields := ui.NewFormLayout()
fields.AddRow("Name:", nameBox)
fields.AddRow("Port:", portBox)
fields.AddRowWidget(ui.NewCheckbox("Remember the password")) // the whole width
```

## DialogButtonBox

The buttons of a dialog, at its right edge, in the order of the platform (OK Cancel on Windows and Linux, Cancel OK on macOS). In a form its accept button is pressed by Enter, the reject one by Esc.

```go
buttons := ui.NewDialogButtonBox()
buttons.AddOK(func() { apply(); c.Form().Close() })      // "OK" in the interface language
buttons.AddCancel(func() { c.Form().Close() })
buttons.AddButton("Apply", ui.ButtonOther, apply)
dialog.AddWidget(2, 0, buttons)
```

A dialog that sets no size in `OnDialogShow` gets the size its widgets need (see [Dialog](dialog.md)).

## The grid

- **Spans:** `AddWidgetSpan(row, col, rowSpan, colSpan, w)` or `w.SetGridSpan(rowSpan, colSpan)`. The widget takes the cells and the spacing between them; the columns get the width it needs.
- **Stretch weights:** `panel.SetColumnStretch(col, n)`, `panel.SetRowStretch(row, n)`. The free room is shared by the weights: a column of 2 grows twice as fast as one of 1. A weight makes the line expandable.
- **Expandable widgets** (`SetXExpandable`, `SetYExpandable`) take the free room; without one, the free room is shared by all the lines — put an `AddVSpacer` below the content.
- **Several widgets in a cell** lie on top of each other; the one added last gets the clicks.
- **Absolute placement:** `panel.SetAbsolutePositioning(true)`; the children keep the places of `SetPosition`/`SetSize`, and follow the edges of `SetAnchors` when the panel resizes.
