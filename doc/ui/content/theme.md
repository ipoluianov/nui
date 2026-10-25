# Theme

Theme helpers are in `ui/theme.go`.

## Apply light theme

```go
ui.ApplyLightTheme() // the open forms repaint in the new colors themselves
```

`ui.OnThemeChanged` is emitted after a theme is applied: connect a handler to recolor what the application colored itself, e.g. its icons.

```go
ui.OnThemeChanged.Connect(func(dark bool) { toolbar.SetIcons(iconsFor(dark)) })
```

## Theme getters used by widgets

- `ui.ThemeFontFamily() string`
- `ui.ThemeFontSize() float64`
- `ui.ThemeForegroundColor(role string) color.Color`
- `ui.ThemeBackgroundColor(elevation int, role string) color.Color`

Widgets typically use these via `Widget.FontFamily()`, `Widget.FontSize()`, `Widget.ForegroundColor()`, `Widget.BackgroundColor()`.

