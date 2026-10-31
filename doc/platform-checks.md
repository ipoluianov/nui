# Manual checks

What the automatic tests do not cover: it needs a real system, a mouse, a keyboard, a monitor, a printer. Each item says what to do, what must come out and, when it was checked already, on what.

Reporting: remove an item when all is right, or write down what came out. Give the version of the system and the scale of the screen once, at the start of a section.

Programs for the checks (delete them with this file when the items are done):
- [cmd/fontcheck](../cmd/fontcheck/main.go): the styles and widths of fonts; `-system` draws the interface with the system font;
- [cmd/clipcheck](../cmd/clipcheck/main.go): the clipboard between two copies, `-copy` and `-paste`;
- [cmd/dragcheck](../cmd/dragcheck/main.go): dragging files out of a window and into it.

---

## Partial repaint (all systems)

A frame repaints only the rectangles of the widgets that changed, over the last one (see [testing.md](testing.md#partial-repaint)). Run the examples with `NUI_CHECK_REPAINT=1` and watch the console: each line `nui: repaint check: …` is a bug, with the area and the widgets.

Checked automatically, with no difference: macOS 27.0.1 Retina (`TestRepaintMonkey`, viewer3d in a window); Linux Debian 13 KDE X11 (all the examples in real windows through XTest at `NUI_SCALE` 1, 1.5 and 2; covering, iconifying and resizing give the same picture to the pixel).

- **No traces of an old picture** (Windows; on macOS and Linux, with the mouse by hand). In the launcher, gallery, notepad, contacts, cad, chartgallery, viewer3d, monitor: move the mouse, turn the wheel, drag, open menus and lists, resize the window, iconify it, cover it with another window. No `repaint check` in the console, no traces of a hover, a caret, a crosshair, a tooltip on the screen.
- **The background of the window** (Windows, macOS; Linux in the light theme). The edge of a window being resized and an empty form are of the background color of the theme, light and dark.
- **The window after being covered and iconified** is painted whole (Windows, macOS).
- **Scales 125-200 %** (Windows; Linux by eye): hovering over buttons and lists leaves no stripes of a pixel at the edges of the highlighted items.

## Text and fonts

- **The caret in a TextBox** (Linux and Windows at 125, 150 and 200 %; macOS Retina checked). A click between letters puts the caret right in the gap; a mouse selection covers exactly the letters selected; the caret at the end of a long line is right after the last character. At the start, the middle and the end of a line.
- **Chinese in bold text** (macOS; Windows at 100 % checked by a screenshot). `中文` in the bold lines of `fontcheck` is bold and on the baseline of the Latin letters.
- **No delay at the first bold text** (all systems). No pause is seen when a window is first shown: each style is unpacked when first used, about 9 ms for Noto Sans.
- **Windows: Noto Sans installed in the system.** GDI sees two fonts of one name: the built-in one loaded in memory, and the system one. Bold and italic in `fontcheck` stay real: `native-regular` of `notosans:bold` is well below 39 (a synthesized bold is wider by exactly 1 px a character), the italic is cursive (the Cyrillic «д» looks like «g», «т» like «m»).
- **Windows: scales 150 % and 200 %.** As above: bold and italic are real, the italic at 40 px is not cut off.

## Windows

- **`RememberState`** (Windows and Linux). Maximize a window, close it with the close button, open it: it is maximized, and Restore gives the last place and size. Windows with one monitor is checked. With two monitors: close the window on the second, unplug it, open: the window is on the first.
- **macOS: a background of each window.** Two forms with different `SetBackgroundColor`: each is filled with its own color while resized. A window over a whole external 5K or 6K monitor: the picture is not cut off at the right and the bottom.
- **macOS: a second monitor of another scale** (Retina and ordinary). Drag the window of notepad to the other screen: the text is sharp, the caret at the end of a line is in place; the window opens on the monitor where it was closed.
- **macOS: printing through CUPS.** `Printers()` returns the printers of System Settings → Printers and the default one. Printing to a real printer, or to PDF through a virtual printer, works. Without printers there is a clear error, not a crash.

## Linux

- **Dragging files out of a window into Dolphin, Nautilus, Thunar** (`dragcheck`): the files are copied, the cursor over the file manager shows that a drop is accepted. The window of the manager is inside the frame of the window manager, and the target is looked for deeper by `XdndAware`; between windows of nui on Xvfb this is checked.
- **KDE at 200 %** (System Settings → Display → Scale, `Xft.dpi` 192): the windows of nui are of the size of other programs', the text is sharp.
- **GNOME on X11 at 200 %:** the same.
- **Fractional scales 125 % and 150 %:** the frames of the widgets do not drift by half a pixel, menus and tooltips open at the cursor.
- **Two monitors of different scales:** X11 has one scale for the whole screen, so on a monitor of 100 % the windows are larger. That is expected; check that nothing breaks.

## Input methods

**Windows** (Microsoft Pinyin or Microsoft IME). Type in a single-line and a multiline TextBox; the multiline one scrolled down, the form with a menu bar.
- **The composition window** (the underlined text) is at the caret, not at the top left corner of the screen.
- **The candidate window** is under the line with the caret and does not cover it.
- **At 150 %** the windows stay in the same places, not 1.5 times farther from the corner of the window.

**macOS** (`go run . notepad`, Edit → Find for a single-line box):
- **Chinese (Pinyin):** the candidate window at the caret; the characters chosen are inserted; Esc cancels the composition without inserting; Backspace during the composition deletes in the composition, not in the text.
- **Japanese (Kotoeri), Korean:** the same.

## Accessibility: Linux, a real screen reader

- **Orca** (Super+Alt+S in GNOME, or `orca` in a terminal), notepad of the examples: the title of the window is read at the start; Tab over the fields reads the name and the role; a checkbox is read with its state; in a list and a tree the arrows read the items; the items of menus are read.
- **Accerciser** shows the tree of the application, the actions of the objects work (Interface Viewer → Action).
- **Without a screen reader** the bridge does not start: no objects of nui on the accessibility bus, no delay at the start.
