# ToolButton

Fixed-size image button for toolbars (48x48 by default).

- Disabled state: a grayed-out copy of the image is shown and clicks are ignored.
- Checked state for toggles: a lighter background and a bright bar along the bottom.
- Highlight: a colored bar along the bottom, e.g. for the action to start with.
- A lighter bottom edge, so the buttons stand out from the toolbar.

## Create

```go
img := /* load image.Image, e.g. a 32x32 icon */
btn := ui.NewToolButton(img, "Add host (A)", func() {
  // ...
})
toolbar.AddWidget(0, 0, btn)
```

## Size

```go
// The main action: twice as wide
btn.SetButtonSize(ui.ToolButtonDefaultSize*2, ui.ToolButtonDefaultSize)
```

## State

```go
btn.SetEnabled(false)       // grayed out, clicks ignored
btn.SetChecked(visible)     // the button does not toggle itself on click
btn.SetHighlight(ui.ColorFromHex("#3fb950")) // colored bottom bar to draw attention; nil - usual look
```
