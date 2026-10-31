# Timer

A `ui.Timer` runs a function on the UI thread at an interval, while its widget is in an open form. The function may change the widgets directly.

After the function the timer's widget is repainted, not the whole form (a timer of the form's `Panel()` repaints all of it). The setters of the other widgets ask for their own repaint; a widget that draws data of its own needs its `Update()` (see [Widget](widget.md#painting)).

```go
// Every second, the first time right away; the timer is returned to stop it
refresh := panel.AddTimer(1000, func() { status.SetText(load()) })
refresh.Stop()
refresh.Start() // again, the next run a full interval from now

// Once, after a delay
panel.AfterFunc(2*time.Second, func() { hint.SetVisible(false) })

// A stopped timer, started when needed
blink := panel.NewTimer(500*time.Millisecond, toggleCursor)
blink.Start()
```

## Debounce

`Start` on a running timer starts its interval anew. A single-shot timer restarted on every change runs once the changes pause:

```go
search := box.NewTimer(300*time.Millisecond, func() { runSearch(box.Text()) })
search.SetSingleShot(true)
box.OnTextChanged.Connect(func(string) { search.Start() })
```

## Methods

| Method | |
|---|---|
| `Widget.AddTimer(intervalMs, f) *Timer` | a running timer; the first run at the next tick |
| `Widget.NewTimer(interval, f) *Timer` | a stopped timer |
| `Widget.AfterFunc(d, f) *Timer` | runs `f` once after `d` |
| `Start()`, `Stop()`, `Running()` | start (restart), stop |
| `SetInterval(d)`, `Interval()` | |
| `SetSingleShot(bool)`, `SingleShot()` | run once a `Start` |

The timers are ticked by the form's event loop, every few milliseconds: an interval shorter than that runs at each tick. A timer of a widget that is not in an open form doesn't run. To wake the UI thread from another goroutine, use `ui.Invoke`.
