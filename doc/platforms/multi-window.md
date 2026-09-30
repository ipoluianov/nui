# Multiple windows and threading rules

## One UI thread

All windows live on the **UI thread**: the main OS thread, where `main()` runs (the library locks the main goroutine to it at startup). A single event loop on that thread serves every window, and every callback (`OnPaint`, `OnKeyDown`, `OnTimer`, ...) of every window is called there, **one at a time**. This works the same on Linux, Windows and macOS.

As a result, a callback of one window can use another window's methods, and state shared between the callbacks of different windows needs no locking.

## Running several windows

```go
win1 := platforms.CreateWindow("Window 1", 100, 100, 500, 300, true, false)
win2 := platforms.CreateWindow("Window 2", 650, 100, 500, 300, true, false)

platforms.Run(win1, win2) // shows both, blocks until ALL of them are closed
```

Call `Run` (or `Exec`) from the main goroutine: that is what runs the event loop. Equivalent manual form:

```go
win1.Show()
win2.Show()
win1.Exec() // runs the event loop until win1 is closed
win2.Exec() // then until win2 is closed (returns at once if it already is)
```

Opening a window later, e.g. from a callback, needs only `Show()`: the running event loop serves it too.

```go
win.OnKeyDown(func(k platforms.Key, m platforms.KeyModifiers) bool {
	if k == platforms.KeyN {
		other := platforms.CreateWindow("New window", 0, 0, 300, 150, true, false)
		other.OnPaint(...)
		other.Show()
	}
	return true
})
```

`Exec()` called from a callback runs a nested event loop until that window is closed; the other windows keep working meanwhile.

## Modal dialogs

```go
dlg.ShowModal(parentWin)
```

Non-blocking on Linux/Windows: `parentWin` keeps repainting and running its timer, but gets no input while `dlg` is open.

- Linux: sets `WM_TRANSIENT_FOR` + `_NET_WM_STATE_MODAL`. The event loop also drops `parentWin`'s mouse/keyboard events itself, since not every window manager blocks pointer input.
- Windows: `dlg` is owned by `parentWin`, which is disabled until `dlg` closes.
- macOS: runs `dlg` as an app-modal window (`NSApp runModalForWindow:`). This **blocks the caller** until `dlg` closes: nesting a modal session must happen synchronously on the call stack for modal-on-modal to work reliably on Cocoa. It also blocks input to **all** of the app's windows, not just `parentWin`. Parent timers/repaint still run.

## Threading rules

1. **Call window methods on the UI thread**: from any window's callbacks, or before the event loop starts.
2. **Other goroutines hand work over to the UI thread**:
   - `platforms.Post(f)` runs `f` on the UI thread and returns at once. Called on the UI thread itself, `f` runs after the current callback.
   - `platforms.RunOnUIThread(f)` runs `f` on the UI thread and waits for it to return. Don't use it from a goroutine the UI thread is waiting for: that deadlocks.
   - `platforms.IsUIThread()` tells whether the caller is on the UI thread.
   - `CreateWindow` does the hand-over itself.
3. **Exceptions: `Update()`, `Close()` and `Exec()`** are safe from any goroutine.
   - `Close()` only requests the close. Off the UI thread it is handed over to it.
   - `Exec()` called from another goroutine just waits for the window to close.
4. **Don't block inside callbacks.** A blocked callback freezes *all* windows, since they share one thread. Long work belongs on its own goroutine; hand the results back with `Post`.

At the `ui` level, `ui.Invoke(f)` / `form.Invoke(f)` / `ui.InvokeSync(f)` are the same hand-over. `Form.Show`/`ShowModal` may be called from any goroutine: they open the window on the UI thread.

## How it works per platform

- **Linux (X11)**: all windows and popups share one X `Display` connection. The event loop waits on the X connection and on a pipe; `Post` writes to the pipe to wake it. The loop dispatches each X event to its window by id, then runs the timers and paints the windows that asked for it.
- **Windows**: every HWND is created on the UI thread, so a single `GetMessage` loop there gets the messages of all of them. `Post` sends a message to a hidden message-only window of the UI thread. Unlike a thread message, it isn't lost while a system modal loop runs (moving a window, a system dialog).
- **macOS (Cocoa)**: there is one `NSApplication` run loop for the process, on the main thread. The first `Exec()` starts it and later ones return at once. `Post` goes to the main dispatch queue.
