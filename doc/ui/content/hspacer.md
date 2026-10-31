# HSpacer

A horizontal spacer: an empty widget that expands in X, taking the free width of its row. The widgets after it go to the right edge.

```go
row := ui.NewPanel()
row.AddWidget(0, 0, ui.NewLabel("Status"))
row.AddHSpacer(0, 1) // the same as row.AddWidget(0, 1, ui.NewHSpacer())
row.AddWidget(0, 2, ui.NewButton("Close"))
```

In an `HBox`, `AddStretch()` does the same.
