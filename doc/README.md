# Documentation

For applications:

- [recipes.md](recipes.md): common tasks in short code (the main window, dialogs, background work, tables, translations, the tray, tests).
- [ui/](ui/README.md): forms, widgets, canvas, charts: a page for each (package `ui`, plus `ui/i18n` and `ui/uitest`).
- [rules.md](rules.md): what the signatures do not show: the UI thread, repaint, events, dialogs, widgets of your own.
- [testing.md](testing.md): tests of forms without a display, golden pictures, screenshots before and after, benchmarks.

For contributors:

- [platforms/](platforms/README.md): the internal native layer: windows, the event loop, keyboard, mouse (`internal/platforms`, `internal/canvas`; not importable by applications).
- [todo.md](todo.md): what is known to be missing.
- [platform-checks.md](platform-checks.md): checks by hand on Windows, macOS and Linux.
- Build scripts: [scripts/](scripts/); the check of the applications next to nui: [../scripts/check-clients.sh](../scripts/check-clients.sh) (see [testing.md](testing.md)).
