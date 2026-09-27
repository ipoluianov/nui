# Dialog

Popup dialog widget shown on top of the main form.

## Create

```go
d := ui.NewDialog("Title", 500, 300)
d.ContentPanel().AddWidgetOnGrid(ui.NewLabel("Hello"), 0, 0)
d.ShowDialog()
```

## Accept / reject

```go
d.OnAccept = func() { /* ... */ }
d.OnReject = func() { /* ... */ }
```

You can also bind buttons:

```go
ok := ui.NewButton("OK")
cancel := ui.NewButton("Cancel")
d.SetAcceptButton(ok)
d.SetRejectButton(cancel)
```

