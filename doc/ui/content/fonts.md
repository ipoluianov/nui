# Fonts

## Built-in fonts

By default the interface is drawn with **Noto Sans** (`ui.FontFamilySans`), built into the
application, so it looks the same everywhere. **JetBrains Mono** (`ui.FontFamilyMono`) is built in
for code and logs. Characters Noto Sans lacks come from the fallback fonts: JetBrains Mono for
symbols such as the keys of the shortcuts on macOS (⌘ ⇧ ⌃ ⌥) and arrows, a system CJK font for
Chinese, Japanese and Korean text, and a system font for each other script the built-in fonts lack
(Arabic, Hebrew, Devanagari and the other Indic scripts, Thai, Armenian, Georgian...), looked up the
first time a character of it is drawn: by fontconfig on Linux, among the fonts of Windows and
macOS that have many scripts (see [Limitations](#limitations)). `ui.SystemFallbackFonts = false`
keeps to the fonts the application registers.

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

### nui's rasterizer

Elsewhere - the built-in fonts on Linux and macOS, every font with `NativeFontRendering` off,
`TextLayout` and `DrawTextF` - nui draws the glyphs itself, the way FreeType draws the system's:

- **Hinting.** A light vertical hinter, as FreeType's "slight": the height of the small letters is
  scaled to whole pixels, the baseline, the tops of the small letters, of the capitals and of the
  ascenders lie on the edges of the pixels, and the horizontal strokes (the bars of e, H, t) take
  whole rows. Across, the glyphs are not moved: the widths of the text and the layout stay as they
  are. `ui.TextHinting = false` draws the outlines as they are, as before (the classic drawing).
- **Subpixels.** On an LCD screen the text is drawn with the red, green and blue stripes of its
  pixels - three times as many columns - filtered as FreeType's default LCD filter, so stems have no
  colored fringes. The order of the stripes is the system's: fontconfig's `rgba` on Linux, ClearType
  on Windows; macOS draws in gray, as it does itself. `ui.TextSubpixel` sets it
  (`ui.SubpixelAuto`, `SubpixelNone`, `SubpixelRGB`, `SubpixelBGR`).
- Only the windows have subpixels: pictures (`NewImageCanvas`, export, printing), the headless mode
  of the tests and text over what is not opaque are drawn in gray, so they do not depend on the
  screen.

The masks of the glyphs are kept, made once for a font, a size and a place: a line of 25 characters
takes about 13 µs in gray and 20 µs in subpixels (11 µs the classic drawing). On Linux the text
drawn so has the same weight and the same heights as FreeType's, to a tenth of a percent of its ink.

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

## Limitations

- **Emoji are drawn as boxes.** The emoji fonts are color fonts (Apple Color Emoji keeps images,
  `sbix`; Noto Color Emoji keeps `CBDT` bitmaps or `COLR` layers), and nui's rasterizer draws outline
  fonts only, in one color. An emoji is typed, copied, stored and measured as one character: only
  its picture is a box (the `.notdef` glyph of the font), and the cursor stays right around it.
- **Shaping.** A line that needs it - Arabic, Hebrew, the Indic scripts, Thai and the other
  scripts whose letters join or combine, combining marks, the marks of direction - is shaped and
  laid out by the bidi algorithm wherever it is drawn: `DrawText`, the labels, buttons, menus,
  tables, the text box (see [textbox.md](textbox.md)). Other text is drawn a character after
  another, as quick as before. `MeasureText` measures a shaped line as it is drawn;
  `GetCharPositions` gives the carets of its characters, which in right-to-left text are not in
  order - `TextCaretX`, `TextIndexAt` and `TextSelectionSpans` are the caret, the click and the
  selection for any text.
- **Symbols below Hebrew.** A character before U+0590 the built-in fonts lack (rare Latin, IPA) is
  not looked up among the system fonts: register a font with it and add it to the chain,
  `ui.RegisterFont(name, data)`, then `ui.AddFallbackFont(name)`.
