# XML layout

`SetLayout` builds the widgets of an XML description inside a widget. Each element is a widget, its attributes are the widget's properties, and its child elements are the widgets inside it. `ui.Bind` then puts the widgets into the fields of a struct.

```go
err := panel.SetLayout(`
	<column padding="8">
		<row>
			<label text="Name:"/>
			<textbox id="name" hint="Your name"/>
		</row>
		<checkbox id="remember" text="Remember me" checked="true"/>
		<vspacer/>
		<row><hspacer/><button id="btnOK" text="OK" onclick="OK"/></row>
	</column>`, handlers, nil)

var w struct {
	Name     *ui.TextBox
	Remember *ui.Checkbox
	OK       *ui.Button `ui:"btnOK"`
}
err = ui.Bind(panel, &w)
```

- **`handlers`** holds the functions of the `on…` attributes. It is either a `map[string]func()` or a value whose methods have those names.
- **`widgets`** are the application's own widgets, placed with `<widget id="chart"/>`.
- **Mistakes don't stop the layout.** An unknown element or attribute, a value that isn't a number, a handler or a widget not found, or the same id twice: the rest is built anyway. `SetLayout` returns all the mistakes in a `*ui.LayoutError`, each with its line (`Problems`). Broken XML is an error, and then nothing is built.
- **`ui.Bind`** fills the fields of widget types. A field is matched to the widget whose id is in its tag (`ui:"btnOK"`), or else to its own name with a lower-case first letter (`Name` → `name`). `ui:"id,optional"` may be missing and `ui:"-"` is skipped. The error lists the widgets not found and those of another type.

## Elements

Containers:

| Element | Places its children | Attributes |
|---|---|---|
| `<column>`, `<vbox>`, `<panel>` | one under another | `padding`, `spacing` |
| `<row>`, `<hbox>` | one after another | `padding`, `spacing` |
| `<frame>` | one under another, in a frame | `padding`, `spacing` |
| `<groupbox>` | one under another, under a title | `title`, `padding`, `spacing` |
| `<scrollarea>` | one under another, scrolled | |
| `<hsplitter>`, `<vsplitter>` | exactly two, side by side or one above the other | |
| `<tabwidget>` | `<tab text="…">` elements, each a page whose children go one under another | |
| `<table>` | `<columns>` of `<column text width>`, `<rows>` of `<row>` of `<cell text>` | |

Widgets (no child elements):

| Element | Attributes | Events |
|---|---|---|
| `<label>` | `text` | |
| `<button>` | `text` | `onclick` |
| `<link>` | `text`, `url` | |
| `<textbox>` | `text`, `hint`, `multiline`, `readonly`, `ispassword` | `ontextchanged`, `onkeydown` |
| `<checkbox>` | `text`, `checked`, `tristate` | `onstatechanged` |
| `<radiobutton>` | `text`, `checked` | |
| `<toggleswitch>` | `text`, `checked` | |
| `<combobox>` | `items` (`"One|Two|Three"`), `selected` (index) | |
| `<listview>` | `items`, `multiselect` | |
| `<treeview>` | | |
| `<numbox>` | `min`, `max`, `step`, `decimals`, `value` | `onvaluechanged`, `onchanged` |
| `<slider>` | `min`, `max`, `step`, `value` | |
| `<progressbar>` | `value` (0–100) | |
| `<datepicker>`, `<timepicker>`, `<colorpicker>`, `<imagebox>` | | |
| `<linechart>` | `values` (`"3,5,4,7"`), `title`, `xtitle`, `ytitle` | |
| `<barchart>` | `items` (`"Jan:120|Feb:95"`), `horizontal`, `stacked`, `title`, `xtitle`, `ytitle` | |
| `<piechart>` | `items` (`"Linux:60|Windows:30"`), `donut` (the hole, `0.5`), `title` | |
| `<view3d>` | `model` (an `.obj` or `.stl` file), `shape` (`box`, `sphere`, `cylinder`, `cone`, `torus`, `plane`), `color` (`"#4080e0"`), `wireframe`, `axes`, `grid`, `ortho`, `autorotate` (radians a second) — see [3d.md](3d.md) | |
| `<hspacer>`, `<vspacer>`, `<space>` | | |
| `<widget>` | `id`: an application widget from `widgets` | |

Every element also takes:

- `id`, the name used by `FindWidgetByName` and `Bind`;
- `enabled`, `visible`, `tooltip`;
- `minwidth`, `minheight`, `maxwidth`, `maxheight`, and `width`, `height` (fixed);
- `xexpandable`, `yexpandable`;
- `fontsize`, `fontbold`, `fontitalic`, `fontfamily`;
- `accessiblename` ([accessibility.md](accessibility.md)).

Booleans are `true` and `false`.
