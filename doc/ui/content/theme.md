# Theme

Theme helpers are in `ui/theme.go`.

## Apply light theme

```go
ui.ApplyLightTheme()
ui.UpdateMainForm()
```

## Theme getters used by widgets

- `ui.ThemeFontFamily() string`
- `ui.ThemeFontSize() float64`
- `ui.ThemeForegroundColor(role string) color.Color`
- `ui.ThemeBackgroundColor(elevation int, role string) color.Color`

Widgets typically use these via `Widget.FontFamily()`, `Widget.FontSize()`, `Widget.ForegroundColor()`, `Widget.BackgroundColor()`.

