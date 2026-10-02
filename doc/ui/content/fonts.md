# Fonts

## Built-in fonts

By default the interface is drawn with **Noto Sans** (`ui.FontFamilySans`), built into the
application, so it looks the same everywhere. **JetBrains Mono** (`ui.FontFamilyMono`) is built in
for code and logs. Chinese, Japanese and Korean text falls back to a system CJK font.

```go
logView.SetFontFamily(ui.FontFamilyMono)
ui.ApplyBaseFontSize(16) // the theme font size, in pixels
```

`ui.RegisterFont(family, ttfBytes)` adds your own font file.

On **Windows** the built-in fonts and those of `RegisterFont` are drawn by GDI with ClearType, like
the system fonts: the font file is added to GDI for the application only, and GDI runs its hinting,
so the text is sharp at 100%. Text with characters the font lacks (e.g. Chinese, drawn with the
fallback font) is drawn by nui's rasterizer with the same line height and baseline. ClearType needs
opaque pixels under the text; on a transparent image the text is drawn in grayscale.

```go
ui.NativeFontRendering = false // before the first form: nui's rasterizer everywhere
```

## System fonts

A system font is a font of the operating system, not built into the application. It is drawn by the
system's text engine, so the text looks exactly like in the other applications:

- **Windows**: GDI with ClearType (subpixel antialiasing).
- **Linux**: FreeType with the desktop's fontconfig settings - subpixel order (RGB/BGR), hinting
  style, LCD filter. Characters the font lacks come from the fonts fontconfig picks for them.
- **macOS**: the system draws text in grayscale; the system font's file is drawn by nui.

```go
// The whole interface in the system's own font: "Segoe UI" on Windows,
// the desktop's font on Linux, the system font on macOS
if err := ui.UseSystemFont(ui.SystemUIFontName()); err != nil {
	log.Println(err) // ui.ErrSystemFontNotFound
}

// Any installed font for some widgets
ui.RegisterSystemFont("code", "Consolas")
editor.SetFontFamily("code")

ui.UseBuiltinFont() // back to Noto Sans
```

- `UseSystemFont(name)` makes the font the theme font and lays the open forms out again.
- `RegisterSystemFont(family, name)` makes `family` (any name you choose) drawn with the system font.
- `SystemUIFontName()` returns the name of the system's interface font.
- An unknown name gives `ErrSystemFontNotFound`: the system doesn't substitute another font.
- Sizes are the same as for the built-in fonts: the em size in pixels.

Subpixel antialiasing blends each color channel with what is under the text, so system fonts
are meant for text on opaque backgrounds, which all the controls have.
