# TabWidget

Tabbed container: a row of tabs and the page of the current one. It is as large as its largest page, so the window doesn't change its size when another tab is chosen.

## Create

```go
general := ui.NewPanel()
general.AddLabel(0, 0, "Name")
general.AddWidget(0, 1, nameBox)

tabs := ui.NewTabWidget()
tabs.AddPage("General", general)
tabs.AddPage("Advanced", ui.NewPanel())
tabs.SetPageNameFunc(1, func() string { return tr("Advanced") }) // follows the language
tabs.OnCurrentChanged.Connect(func(index int) { status.SetText(fmt.Sprint("page ", index)) })
```

- `CurrentIndex()` (-1 if no pages), `SetCurrentIndex(i)`, `CurrentPage()`.
- `PageCount()`, `Page(i)`, `PageIndex(w)` (-1 if `w` isn't a page).
- `InsertPage(i, name, w)`, `RemovePage(i)`: the current page stays current; when it is removed, the next one becomes current.
- `SetPageName(i, name)`, `SetPageNameFunc(i, f)`.
- Signals: `OnCurrentChanged(int)` - another page became current, by the user or by `SetCurrentIndex`; `OnActivated(int)` - the user clicked a tab, also the current one.
