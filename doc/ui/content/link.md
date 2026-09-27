# Link

A clickable text, like a hyperlink: underlined under the mouse and when focused, activated by a
click on the text, Enter or Space. It calls its function, or opens its URL when it has none.

```go
more := ui.NewLink("Show details", func() { details.SetVisible(true) })
docs := ui.NewLink("Documentation", nil)
docs.SetURL("https://example.com/docs")
```

- `SetText` / `SetTextFunc` / `Text()`, `SetOnClick(func())`, `SetURL(url)` / `URL()`
- `Activate()` - does what a click does; `Visited()` - it was activated (drawn in a softer color).
- `ui.OpenURL(url) error` - opens an address or a file with the system's default application
  (`xdg-open` on Linux, `open` on macOS, the URL handler on Windows).
