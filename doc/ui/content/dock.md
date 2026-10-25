# DockLayout

Tool panels around the main area of a window, as in an IDE: a file tree at the left, properties at the right, a log at the bottom.

```go
docks := ui.NewDockLayout(editor)                     // the central widget
files := docks.AddDock("Files", fileTree, ui.DockLeft)
docks.AddDock("Outline", outline, ui.DockLeft)        // a second one in the area: tabs
docks.AddDock("Log", logView, ui.DockBottom)
viewMenu.AddAction(files.ToggleAction())              // View > Files, checked while shown
docks.RememberState(settings, "docks")
form.Panel().AddWidget(0, 0, docks)
```

- The areas are `DockLeft`, `DockRight` and `DockBottom`. The bottom one is under the central widget, between the left and the right ones.
- One panel in an area fills it; several are tabs.
- The user resizes an area with its splitter. Each panel has a title bar with a menu ("…": Dock Left / Right / Bottom, Close) and a close button "×".
- An area with no shown panels is hidden.
- `DockWidget`: `SetShown`, `Shown`, `MoveTo(area)`, `Area`, `Title`, `Content`, `ToggleAction`.
- `SetAreaSize(area, size)` and `AreaSize(area)` give the width of the left and right areas and the height of the bottom one.
- `RememberState(settings, key)` restores the saved layout and saves it when the window closes: the area sizes, plus the area of each panel and whether it is shown. Panels are matched by title, so a panel added in a newer version of the application keeps its default place.

Floating panels in their own windows are not supported.
