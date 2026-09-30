# Expander and Accordion

`Expander` is a section with a header that shows or hides its content. The content is a Panel:
add the widgets to `Content()`, on its grid as usual. A click on the header, Space or Enter
toggles it; Left collapses and Right expands it.

```go
adv := ui.NewExpander("Advanced")
adv.Content().AddLabel(0, 0, "Timeout")
adv.Content().AddWidget(0, 1, timeoutBox)
adv.SetOnExpandedChanged(func() { fmt.Println(adv.Expanded()) })
```

- `Expanded()` / `SetExpanded(bool)` - `SetExpanded` doesn't call the callback.
- `Title()` / `SetTitle(text)` / `SetTitleFunc(f)` (follows the language)
- `Content() *ui.Panel`; `AddWidget(row, col, w)` also adds to the content.
- `SetOnExpandedChanged(func())` - called when the user expands or collapses it.
- Height: a collapsed expander is as tall as its header, an expanded one as its content -
  unless the content stretches (has a Y-expandable widget), then it takes the free space.

## Accordion

A column of expanders (sections). By default only one section is expanded at a time.

```go
acc := ui.NewAccordion()
general := acc.AddSection("General")
general.Content().AddLabel(0, 0, "Name")
network := acc.AddSection("Network")
acc.SetOnSectionChanged(func(index int) { fmt.Println("section", index) })
```

- `AddSection(title) *ui.Expander` - the first section of an exclusive accordion is expanded.
- `Sections()`, `Expand(index)` (-1 collapses all), `ExpandedIndex()`
- `SetExclusive(bool)` - false lets several sections be open.
- `SetOnSectionChanged(func(index int))` - the section the user toggled.
- To let an expanded section with a stretching widget fill the free space, make the accordion
  Y-expandable: `acc.SetYExpandable(true)`.
