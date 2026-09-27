# ColorPicker

A field showing a color (swatch + hex code). A click, or Enter / Space / Down / F4, opens a panel:

- saturation/brightness square and hue bar (drag in them);
- opacity bar, with `SetAlphaEnabled(true)`;
- old | new comparison (a click on the old half returns to it) and a hex field (`#RGB`, `#RRGGBB`, `#RRGGBBAA`);
- palette and the colors chosen recently in any picker of the application.

A palette/recent swatch or Enter in the hex field chooses and closes; a click outside keeps the
new color; **Escape returns to the color the panel was opened with**.

```go
picker := ui.NewColorPicker()
picker.SetColor(ui.ColorFromHex("#1E88E5"))
picker.SetOnColorChanged(func(col color.RGBA) {
	preview.SetBackgroundColor(col) // live, also while dragging
})
```

## API

- `Color() color.RGBA` / `SetColor(color.Color)` - `SetColor` doesn't call `SetOnColorChanged`'s function.
- `SetOnColorChanged(func(col color.RGBA))` - every change made by the user, including while dragging and on Escape.
- `SetAlphaEnabled(bool)` / `AlphaEnabled()` - without it colors are opaque and the code has 6 digits.
- `SetPalette([]color.RGBA)` / `Palette()` - `nil` hides the palette; the default is `ui.DefaultColorPickerPalette`.
- `OpenPopup()`, `ClosePopup()`, `IsPopupOpen()`
- `ui.RecentColors()` - the colors chosen recently, newest first.
- `ui.ParseHexColor(s) (color.RGBA, bool)` - parses `#RGB`, `#RRGGBB`, `#RRGGBBAA`, with or without `#`.

Colors are `color.RGBA` with a straight (not premultiplied) alpha, as `ui.ColorFromHex` returns them;
other `color.Color` values passed to `SetColor` are converted.

See `examples/ex14colorpicker`.

## Popups of your own

The panel cancels on Escape through `ui.PopupCanceler`: a popup widget (see `Form.OpenPopup`)
implementing `CancelPopup()` gets it called when Escape closes it, before it closes.
