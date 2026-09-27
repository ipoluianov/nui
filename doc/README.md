# Documentation

`github.com/ipoluianov/nui` consists of two layers:

- [ui/](ui/README.md) — forms and widgets, the public API
  (package `ui`, plus `ui/i18n`).
- [platforms/](platforms/README.md) — internal native layer: windows, event loop, keyboard, mouse
  (packages `internal/platforms`, `internal/canvas`; not importable by applications).

Build scripts for all platforms: [scripts/](scripts/).
