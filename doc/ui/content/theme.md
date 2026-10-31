# Theme

Theme helpers are in `ui/theme.go`. The dark theme is applied at start.

## Light, dark, own colors

```go
ui.ApplyLightTheme() // the open forms repaint in the new colors themselves
ui.ApplyDarkTheme()

// A palette of one's own: a copy of a built-in one with another accent
p := ui.LightPalette
p.Highlight = ui.ColorFromHex("#2e7d32")
ui.ApplyPalette(p, false) // false: a light theme
```

`ui.ThemePalette` holds the colors by their role, as `QPalette` in Qt: `Window`/`WindowText` (forms and panels), `Base`/`Text` (input fields, lists, tables), `Button`/`ButtonText`, `Border`, `Divider`, `Highlight`/`HighlightedText` (the accent), `Selection`, `DisabledText`, `PlaceholderText`, `PopupBase`, `ToolTipBase`/`ToolTipText`, `Link`, `Error`, `Warning`, `Success`. `ui.CurrentPalette()` returns the colors in use, and `ui.IsDarkTheme` tells whether the theme is dark - for a widget that paints itself:

```go
p := ui.CurrentPalette()
cnv.FillRect(0, 0, w, h, p.Base)
```

`ui.OnThemeChanged` is emitted after a theme is applied: connect a handler to recolor what the application colored itself, e.g. its icons.

```go
ui.OnThemeChanged.Connect(func(dark bool) { saveAction.SetIcon(saveIcon(dark)) })
```

## Font size

`ui.ApplyBaseFontSize(16)` sets the theme font size, in pixels (`ui.DefaultFontSize`, 14, by default); the heights of the controls follow it. See [Fonts](fonts.md).

## Theme getters used by widgets

- `ui.ThemeFontFamily() string`
- `ui.ThemeFontSize() float64`
- `ui.ThemeForegroundColor(role string) color.RGBA`
- `ui.ThemeBackgroundColor(elevation int, role string) color.RGBA`

Widgets typically use these via `Widget.FontFamily()`, `Widget.FontSize()`, `Widget.ForegroundColor()`, `Widget.BackgroundColor()`.
