# VSpacer

A vertical spacer: an empty widget that expands in Y, taking the free height of its column. Without one (or another Y-expandable widget) the free height is shared by all the rows.

```go
col := ui.NewPanel()
col.AddWidget(0, 0, nameBox)
col.AddWidget(1, 0, okButton)
col.AddVSpacer(2, 0) // the same as col.AddWidget(2, 0, ui.NewVSpacer())
```

In a `VBox`, `AddStretch()` does the same.
