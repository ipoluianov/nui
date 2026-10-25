# MarkdownView

Shows Markdown text, wrapped to the width of the view and scrolled: help pages, chat messages, release notes.

```go
help := ui.NewMarkdownView()
help.SetMarkdown("# Help\n\nPress **Ctrl+S** to save. See [the site](https://example.com).")
help.OnLinkClicked.Connect(func(url string) { open(url) }) // without a handler: the browser
```

Supported: headings (`#`, `##`, `###`), paragraphs, bulleted (`-`, `*`, `+`) and numbered (`1.`) lists, quotes (`>`), code blocks (```` ``` ````), horizontal rules (`---`); in the text **bold**, *italic*, `code` and [links](url).

`Text()` returns the text without the marks; `LinkAt(x, y)` the link at a point, e.g. for a tooltip.
