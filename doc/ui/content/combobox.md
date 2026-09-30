# ComboBox

Dropdown list.

## Create

```go
cb := ui.NewComboBox()
cb.AddItem("One", 1)
cb.AddItem("Two", 2)
cb.SetSelectedIndex(0)
```

## Read selection

```go
index := cb.SelectedIndex()
text := cb.SelectedItemText()
data := cb.SelectedItemData()
```

## Selection changes

```go
cb.SetOnSelectedIndexChanged(func() {
	fmt.Println("picked", cb.SelectedItemText())
})
```

Called when the user picks another item in the popup; `SetSelectedIndex` doesn't call it.
`ItemCount()` returns the number of items.

