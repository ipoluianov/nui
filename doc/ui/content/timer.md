# Timer

Any widget can run periodic timers via `Widget.AddTimer(intervalMs, callback)`.

Example (blink, polling, animation):

```go
w := ui.NewPanel()
w.AddTimer(250, func() {
  ui.UpdateMainForm()
})
```

Timers are processed by the main form loop.

