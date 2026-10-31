# Space

A fixed-size empty widget, a gap of exact size between widgets.

```go
s := ui.NewSpace()
s.SetSize(16, 16) // fixed: the minimum and the maximum size, not expandable
row.AddWidget(0, 1, s)
```

For a gap that takes the free room use [HSpacer](hspacer.md) or [VSpacer](vspacer.md); in an `HBox`/`VBox`, `AddSpacing(px)` adds a fixed gap.
