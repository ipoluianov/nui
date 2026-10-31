# To do

What is known to be missing, kept short. The checks on Windows and macOS
are in [platform-checks.md](platform-checks.md).

## Widgets and layouts

- Signals (`ui.Signal`) of TimeChart, PropertyGrid and EditableComboBox;
  of the form: moved, window state changed.
- `TabWidget.SetPageVisible`.
- Alignment of a widget within its grid cell; a margin of a widget.
- Sorting and filtering over an `ItemModel`; a table cell painter that
  gets the context of the cell.
- A message box with buttons and an icon of the application's choice.
- Word wrap and folding of blocks in TextBox.
- Dock panels floating in windows of their own, and dragged by the mouse.
- Public `ElideText` and `WrapText`.

## Theme

- Color roles of widgets (`SetForegroundRole`), a color that follows the
  theme, monochrome icons tinted by the theme.

## Application and windows

- `IsMinimized`, the screens and their geometry.
- The state of Splitter and of table headers in `RememberState` (now kept
  by hand with `Settings.Set`).
- `ShowModal` on macOS blocks until the dialog is closed
  (`runModalForWindow`); on Windows and Linux it does not. A modal session
  of Cocoa (`beginModalSessionForWindow`) would make it the same.

## Text

- On Windows a line with a character the built-in font lacks is drawn
  whole by the rasterizer of nui, without ClearType: it is to be split in
  runs by font (GDI for its own characters), with the measuring, the
  caret and the baseline agreeing.

## Examples

- The welcome text of notepad says "Ctrl+O", "Ctrl+F"..., though the
  shortcuts are `Mod+` (⌘ on macOS).

## Accessibility

- Bridges to UI Automation on Windows and NSAccessibility on macOS (the
  tree of `Form.Accessible` and the AT-SPI bridge on Linux exist). They
  are to be written and debugged on those systems: a fault in a COM server
  or in an NSAccessibility object breaks the application on every call of
  a screen reader.
- Objects of open menus and popups; events of changed text.
