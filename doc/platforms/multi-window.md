# Multiple windows and threading rules

## Running several windows

```go
win1 := platforms.CreateWindow("Window 1", 100, 100, 500, 300, true, false)
win2 := platforms.CreateWindow("Window 2", 650, 100, 500, 300, true, false)

platforms.Run(win1, win2) // blocks until ALL windows are closed
```

`platforms.Run` runs the first window's `Exec()` on the calling goroutine and starts one goroutine per remaining window (`go win.Exec()`), then waits for all to finish. Equivalent manual form:

```go
go win1.Exec()
go win2.Exec()
// ... wait yourself, e.g. sync.WaitGroup
```

Opening a window later (e.g. from a button click) works the same way:

```go
win.OnKeyDown(func(k platforms.Key, m platforms.KeyModifiers) bool {
	if k == platforms.KeyN {
		other := platforms.CreateWindow("New window", 0, 0, 300, 150, true, false)
		other.OnPaint(...)
		go other.Exec()
	}
	return true
})
```

Note: the very *first* window's `Exec()` (or your `platforms.Run(...)` call) must never be wrapped in `go` — on macOS the real event loop has to start on the process's original goroutine/thread.

## Modal dialogs

```go
dlg.ShowModal(parentWin)
```

Non-blocking on Linux/Windows: `parentWin`'s own event loop, timers and repaint keep running.

- Linux: sets `WM_TRANSIENT_FOR` + `_NET_WM_STATE_MODAL`; the window manager blocks mouse/keyboard input to `parentWin` while `dlg` is open (GNOME/KDE; not enforced by all window managers, e.g. openbox).
- macOS: runs `dlg` as an app-modal window (`NSApp runModalForWindow:`). This **blocks the calling goroutine** until `dlg` closes (unlike other platforms) — nesting a modal session must happen synchronously on the call stack for modal-on-modal to work reliably on Cocoa; a deferred/async call let an intermediate dialog stay key and clickable. It also blocks input to **all** of the app's windows, not just `parentWin` — Cocoa has no native "block only this one parent" dialog style that keeps its own title bar. Parent timers/repaint still run since all windows share one process-wide run loop.
- Windows: not implemented yet (behaves like a non-modal window).

## Threading rules (must follow)

1. **One goroutine per window.** Never call `Exec()`/`EventLoop()`/`Show()` for the same window twice or from two goroutines at once.
2. **Call a window's own methods/setters only from that window's own callbacks** (its `On*` handlers), or before its `Exec()`/goroutine has started. Do not call `win2.SetTitle(...)`, `win2.Resize(...)`, etc. from inside `win1`'s callback.
3. **Exception: `Close()`.** `win.Close()` is safe to call from any goroutine, including another window's callback (e.g. a parent closing a child dialog). It only requests the close; the actual teardown always runs on the window's own event-loop goroutine.
4. **Don't block inside callbacks** (`OnPaint`, `OnKeyDown`, `OnTimer`, ...). A blocked callback freezes that window's repaint/timer/input until it returns. Long work belongs on its own goroutine; hand results back via a channel and call `win.Update()`.
5. **Shared state read/written from multiple windows' callbacks needs your own synchronization** (mutex/channel) — the library does not add any for you.

## Why this works safely on Linux (X11)

- Each window opens its own X11 `Display` connection; `XInitThreads()` is called once at startup, so Xlib supports being driven from independent goroutines/threads.
- Each window has its own paint buffer — no shared global framebuffer between windows.
- `Close()` defers the actual `XDestroyWindow`/`XCloseDisplay` to the window's own event-loop goroutine, so a cross-window `Close()` call can never race with that window's own event pump.

## Why this works safely on macOS (Cocoa)

- Unlike X11/Win32, there is exactly **one** `NSApplication` / one `[NSApp run]` loop for the whole process, shared by every window, and it must start on the process's original ("main") OS thread.
- Only the first window to reach `EventLoop()` actually calls `[NSApp run]`; every later window (secondary window or modal dialog) just registers/shows itself and returns immediately — the already-running shared loop services it too. This is why `platforms.Run()`'s first window must run on the calling goroutine instead of a spawned one (see above).
- `ShowModal` doesn't spawn its own event loop: it shows the dialog and calls `NSApp runModalForWindow:` synchronously, right there on the calling goroutine, nesting a new modal session directly on the call stack — the only reliable way to stack a modal on top of another already-open modal on Cocoa. That call is what actually restricts input to the dialog until it's closed, and it's why `ShowModal` blocks its caller on macOS (see above).

