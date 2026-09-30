# PropertyGrid

Named values edited in place, grouped in collapsible categories. Each property gets an editor
for its type; the name column is the same width in all the categories.

```go
grid := ui.NewPropertyGrid()
grid.AddString("General", "Name", "Server 1")
grid.AddChoice("General", "Mode", []string{"Dev", "Prod"}, 0)
grid.AddInt("Network", "Port", 8080, 1, 65535)
grid.AddBool("Network", "Use TLS", true)
grid.SetOnChanged(func(p *ui.Property) { fmt.Println(p.Name(), "=", p.Value()) })
```

Adding properties (`category` "" puts the property at the top, without a header):

| Method | Editor | `Value()` |
|---|---|---|
| `AddString(category, name, value)` | TextBox | `string` |
| `AddInt(category, name, value, min, max)` | NumBox | `int` |
| `AddFloat(category, name, value, decimals)` | NumBox | `float64` |
| `AddBool(category, name, value)` | Checkbox | `bool` |
| `AddChoice(category, name, options, index)` | ComboBox | `int` (the index) |
| `AddColor(category, name, value)` | ColorPicker | `color.RGBA` |
| `AddDate(category, name, value)` | DatePicker | `time.Time` |
| `AddCustom(category, name, widget)` | your widget | `nil` |

`Property`: `Name()`, `Category()`, `Value()`, `SetValue(v)` (no callback), `SetReadOnly(bool)`,
`SetDescription(text)` (tooltip), `SetOnChanged(func())`, `Editor()`, `NotifyChanged()` (for
custom editors).

Grid: `Property(name)`, `Properties()`, `Category(name) *ui.Expander`, `Clear()`, `SetOnChanged`.

## Editing a struct

`SetObject` fills the grid from the exported fields of a struct and writes the user's edits
back into it. `Refresh()` shows the struct's values after the code changed them.

```go
type Settings struct {
	Name    string     `category:"General" desc:"Shown in the title"`
	Mode    string     `category:"General" options:"Dev, Staging, Prod"`
	Port    int        `category:"Network" min:"1" max:"65535"`
	TLS     bool       `category:"Network" prop:"Use TLS"`
	Timeout float64    `category:"Network" decimals:"1"`
	Color   color.RGBA `category:"Look"`
	Secret  string     `prop:"-"`
}
s := &Settings{Port: 8080}
grid.SetObject(s)
```

Field types: `string`, `bool`, integers, floats, `color.RGBA`, `time.Time`. Tags: `prop` (the
name, `-` skips the field), `category`, `desc`, `min`, `max`, `decimals`, `options` (a string or
int index chosen from a list), `readonly:"true"`.
