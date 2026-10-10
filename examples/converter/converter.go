package converter

import (
	"math"
	"slices"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/i18n"
)

// NewForm is a unit converter in English, Russian and Chinese, switched
// live. The category is chosen with RadioButtons, the units with ComboBoxes;
// the two NumBoxes convert into each other, a Slider sets the precision and
// a Checkbox the thousands separators; a Table shows the value in every unit
// of the category (with a ContextMenu to copy it), and an EditableComboBox
// finds a unit in any category. All the texts come from the i18n catalog in
// strings.go: SetTextFunc, SetTitleFunc, SetTooltipFunc and SetHintFunc for
// the widgets' own texts, SetOnLanguageChanged for the rest.
func NewForm() *ui.Form {
	c := &converter{form: ui.NewForm()}
	form := c.form
	form.SetSize(860, 640)
	form.SetTitleFunc(func() string { return T().Title })
	form.SetIcon(icons.App(32, "#00897B", swapMark))

	panel := form.Panel()
	panel.AddWidget(0, 0, c.newTopBar())

	columns := panel.AddPanel(1, 0)
	columns.AddWidget(0, 0, c.newCategories())
	right := columns.AddPanel(0, 1)
	right.SetPanelPadding(0)
	right.AddWidget(0, 0, c.newConversion())
	common := ui.NewGroupBox("")
	common.SetTitleFunc(func() string { return T().CommonTitle })
	common.AddWidget(0, 0, c.newTable())
	right.AddWidget(1, 0, common)

	c.updateTexts()
	form.SetOnLanguageChanged(c.updateTexts)
	c.radios[0].SetChecked(true)
	return form
}

type converter struct {
	form *ui.Form

	radios []*ui.RadioButton
	cat    int // index in categories

	// A pair of unit lists per category: a ComboBox can't drop its items,
	// so the lists of the chosen category are put into the holders
	fromUnits, toUnits   []*ui.ComboBox
	fromHolder, toHolder *ui.Panel
	fromValue, toValue   *ui.NumBox
	result               *ui.Label
	precision            *ui.Slider
	thousands            *ui.Checkbox
	table                *ui.Table
	search               *ui.EditableComboBox
	searchIndex          [][2]int // category and unit of each search item
	updating             bool     // set while a NumBox is set from code
	showLanguage         func()   // selects the language of the application in its list
}

func (c *converter) newTopBar() *ui.Panel {
	bar := ui.NewPanel()

	bar.AddLabel(0, 0, "").SetTextFunc(func() string { return T().Search })
	c.search = ui.NewEditableComboBox()
	c.search.SetMinWidth(340)
	c.search.TextBox().SetHintFunc(func() string { return T().SearchHint })
	c.search.SetTooltipFunc(func() string { return T().SearchTT })
	c.search.SetOnItemSelected(func(index int, text string) {
		found := c.searchIndex[index]
		c.radios[found[0]].SetChecked(true)
		c.fromUnits[found[0]].SetSelectedIndex(found[1])
		c.convertForward()
		c.search.SetText("")
	})
	bar.AddWidget(0, 1, c.search)
	bar.AddHSpacer(0, 2)

	// The language names stay in their own language
	bar.AddLabel(0, 3, "").SetTextFunc(func() string { return T().Language })
	tags := []string{"en", "ru", "zh"}
	languages := ui.NewComboBox()
	languages.AddItem("English", nil)
	languages.AddItem("Русский", nil)
	languages.AddItem("中文", nil)
	languages.SetOnSelectedIndexChanged(func() { ui.SetLanguage(tags[languages.SelectedIndex()]) })
	bar.AddWidget(0, 4, languages)
	c.showLanguage = func() {
		languages.SetSelectedIndex(max(0, slices.Index(tags, i18n.Base(ui.Language()))))
	}
	return bar
}

func (c *converter) newCategories() *ui.GroupBox {
	group := ui.NewGroupBox("")
	group.SetTitleFunc(func() string { return T().Category })
	group.SetXExpandable(false)
	for i, cat := range categories {
		radio := ui.NewRadioButton("")
		radio.SetTextFunc(func() string { return cat.name(&T().Categories) })
		radio.SetOnStateChanged(func(_ *ui.RadioButton, checked bool) {
			if checked {
				c.setCategory(i)
			}
		})
		group.AddWidget(i, 0, radio)
		c.radios = append(c.radios, radio)
	}
	group.AddVSpacer(len(categories), 0)
	return group
}

func (c *converter) newConversion() *ui.GroupBox {
	group := ui.NewGroupBox("")
	group.SetTitleFunc(func() string { return T().Conversion })

	for _, cat := range categories {
		from, to := ui.NewComboBox(), ui.NewComboBox()
		for range cat.units {
			from.AddItem("", nil)
			to.AddItem("", nil)
		}
		from.SetSelectedIndex(cat.from)
		to.SetSelectedIndex(cat.to)
		from.SetOnSelectedIndexChanged(c.convertForward)
		to.SetOnSelectedIndexChanged(c.convertForward)
		c.fromUnits = append(c.fromUnits, from)
		c.toUnits = append(c.toUnits, to)
	}

	rows := group.AddPanel(0, 0)
	c.fromValue = ui.NewNumBox()
	c.fromValue.SetValue(1)
	c.fromValue.SetOnValueChanged(func() {
		if !c.updating {
			c.convertForward()
		}
	})
	rows.AddWidget(0, 0, c.fromValue)
	c.fromHolder = rows.AddPanel(0, 1)
	c.fromHolder.SetPanelPadding(0)

	swap := ui.NewToolButton(icons.Draw(icons.Size, func(dc *gg.Context) {
		dc.SetHexColor("#1E88E5")
		dc.SetLineWidth(1.5)
		dc.SetLineCapRound()
		swapMark(dc)
	}), "", c.swap)
	swap.SetButtonSize(32, 28)
	swap.SetTooltipFunc(func() string { return T().Swap })
	rows.AddWidget(0, 2, swap)

	c.toValue = ui.NewNumBox()
	c.toValue.SetOnValueChanged(func() {
		if !c.updating {
			c.convertBackward()
		}
	})
	rows.AddWidget(1, 0, c.toValue)
	c.toHolder = rows.AddPanel(1, 1)
	c.toHolder.SetPanelPadding(0)

	c.result = ui.NewLabel("")
	c.result.SetFontSize(20)
	group.AddWidget(1, 0, c.result)

	options := group.AddPanel(2, 0)
	options.AddLabel(0, 0, "").SetTextFunc(func() string { return T().Precision })
	c.precision = ui.NewSlider()
	c.precision.SetRange(0, 10)
	c.precision.SetStep(1)
	c.precision.SetTickInterval(1)
	c.precision.SetValue(3)
	c.precision.SetMinWidth(180)
	c.precision.SetXExpandable(false)
	c.precision.SetTooltipFunc(func() string { return T().PrecisionTT })
	options.AddWidget(0, 1, c.precision)
	digits := options.AddLabel(0, 2, "3")
	digits.SetMinWidth(24)
	c.precision.SetOnValueChanged(func() {
		digits.SetText(formatNumber(c.precision.Value(), 0, false, T()))
		c.updateOutputs()
	})
	c.thousands = ui.NewCheckbox("")
	c.thousands.SetTextFunc(func() string { return T().Thousands })
	c.thousands.SetTooltipFunc(func() string { return T().ThousandsTT })
	c.thousands.SetChecked(true)
	c.thousands.SetOnStateChanged(c.updateOutputs)
	options.AddHSpacer(0, 3)
	group.AddWidget(3, 0, c.thousands)
	return group
}

func (c *converter) newTable() *ui.Table {
	c.table = ui.NewTable()
	c.table.SetColumnCount(2)
	c.table.SetColumnWidth(0, 320)
	c.table.SetColumnWidth(1, 200)
	c.table.SetColumnHAlign(1, ui.HAlignRight)
	c.table.SetStretchLastColumn(true)
	c.table.SetSelectingRows(true)
	c.table.SetTooltipFunc(func() string { return T().CommonTT })

	menu := ui.NewContextMenu(c.table)
	menu.AddItem("", func() {
		if row := c.table.CurrentRow(); row >= 0 {
			ui.ClipboardSetText(c.table.GetCellText2(row, 1))
		}
	}).SetTextFunc(func() string { return T().CopyValue })
	menu.AddItem("", func() {
		if row := c.table.CurrentRow(); row >= 0 {
			c.toUnits[c.cat].SetSelectedIndex(row)
			c.convertForward()
		}
	}).SetTextFunc(func() string { return T().ConvertTo })
	c.table.SetContextMenu(menu)
	return c.table
}

// setCategory shows the unit lists of the category and converts.
func (c *converter) setCategory(i int) {
	c.cat = i
	c.fromHolder.RemoveAllWidgets()
	c.fromHolder.AddWidget(0, 0, c.fromUnits[i])
	c.toHolder.RemoveAllWidgets()
	c.toHolder.AddWidget(0, 0, c.toUnits[i])
	c.table.SetRowCount(len(categories[i].units))
	c.updateUnitNames()
	c.convertForward()
}

func (c *converter) units() (from, to unit) {
	cat := categories[c.cat]
	return cat.units[c.fromUnits[c.cat].SelectedIndex()], cat.units[c.toUnits[c.cat].SelectedIndex()]
}

// setValue sets a NumBox from code without converting back.
func (c *converter) setValue(box *ui.NumBox, v float64) {
	c.updating = true
	box.SetValue(v)
	c.updating = false
}

func (c *converter) convertForward() {
	from, to := c.units()
	c.setValue(c.toValue, convert(c.fromValue.Value(), from, to))
	c.updateOutputs()
}

func (c *converter) convertBackward() {
	from, to := c.units()
	c.setValue(c.fromValue, convert(c.toValue.Value(), to, from))
	c.updateOutputs()
}

// swap exchanges the units and the values.
func (c *converter) swap() {
	from, to := c.fromUnits[c.cat], c.toUnits[c.cat]
	i := from.SelectedIndex()
	from.SetSelectedIndex(to.SelectedIndex())
	to.SetSelectedIndex(i)
	c.setValue(c.fromValue, c.toValue.Value())
	c.convertForward()
}

// updateOutputs formats the result and the table.
func (c *converter) updateOutputs() {
	s := T()
	decimals := int(math.Round(c.precision.Value()))
	group := c.thousands.Checked()
	c.fromValue.SetDecimals(decimals)
	c.toValue.SetDecimals(decimals)

	from, to := c.units()
	v := c.fromValue.Value()
	c.result.SetText(formatNumber(v, decimals, group, s) + " " + from.name(&s.Units).Symbol + " = " +
		formatNumber(c.toValue.Value(), decimals, group, s) + " " + to.name(&s.Units).Symbol)

	for row, u := range categories[c.cat].units {
		c.table.SetCellText2(row, 1, formatNumber(convert(v, from, u), decimals, group, s))
	}
}

// updateUnitNames puts the names of the units of the category into the
// lists and the table.
func (c *converter) updateUnitNames() {
	s := T()
	for row, u := range categories[c.cat].units {
		c.table.SetCellText2(row, 0, unitText(s, u))
		c.table.SetCellHAlign(row, 1, ui.HAlignRight)
	}
}

// updateTexts sets the texts that don't follow the language by themselves.
func (c *converter) updateTexts() {
	s := T()
	c.showLanguage()
	var items []string
	c.searchIndex = c.searchIndex[:0]
	for i, cat := range categories {
		for j, u := range cat.units {
			c.fromUnits[i].SetItemText(j, unitText(s, u))
			c.toUnits[i].SetItemText(j, unitText(s, u))
			items = append(items, unitText(s, u)+" - "+cat.name(&s.Categories))
			c.searchIndex = append(c.searchIndex, [2]int{i, j})
		}
	}
	c.search.SetItems(items)
	c.table.SetColumnName(0, s.ColumnUnit)
	c.table.SetColumnName(1, s.ColumnValue)
	if c.table.RowCount() > 0 { // a category is chosen
		c.updateUnitNames()
		c.updateOutputs()
	}
}

// swapMark draws two opposite arrows, the mark of the application icon.
func swapMark(dc *gg.Context) {
	dc.DrawLine(3.5, 6, 12.5, 6)
	dc.DrawLine(10, 3.5, 12.5, 6)
	dc.DrawLine(3.5, 10, 12.5, 10)
	dc.DrawLine(3.5, 10, 6, 12.5)
	dc.Stroke()
}
