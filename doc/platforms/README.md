# platforms documentation (internal native layer)

Contributor docs. These packages live under `internal/` and can only be
imported from inside this module; applications use `ui`.

Native GUI library for Go. Windows, keyboard, mouse, 2D canvas. Talks to OS native APIs directly - cgo on macOS (Cocoa), [purego](https://github.com/ebitengine/purego) on Linux (Xlib, no cgo), plain `syscall` on Windows (Win32).

## Contents

- [getting-started.md](getting-started.md) — build, run, minimal app
- [window.md](window.md) — `Window` interface: creation, properties, events
- [multi-window.md](multi-window.md) — multiple windows, modal dialogs, goroutine rules
- [canvas.md](canvas.md) — 2D drawing API (`internal/canvas`)
- [keyboard.md](keyboard.md) — key codes
- [mouse.md](mouse.md) — mouse buttons/cursors

## Packages

| Package | Import path | Purpose |
|---|---|---|
| `platforms` | `github.com/ipoluianov/nui/internal/platforms` | windows, event loop, app entry points, `Key`/`Mouse*` types |
| `canvas` | `github.com/ipoluianov/nui/internal/canvas` | draw into an `image.RGBA` (lines, rects, circles, text) |

## Supported OS

Linux (X11), Windows, macOS.
