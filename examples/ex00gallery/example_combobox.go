package ex00gallery

import (
	"fmt"

	"github.com/ipoluianov/nui/ui"
)

type ExamplePageComboBox struct {
	ui.Widget

	lblStatus *ui.Label

	cbSize  *ui.ComboBox
	cbColor *ui.ComboBox
}

func NewExamplePageComboBox() *ExamplePageComboBox {
	var c ExamplePageComboBox
	c.InitWidget()

	row := 0

	c.lblStatus = c.AddLabel(row, 0, "Open a combo box below and pick an item")
	c.lblStatus.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Basic combo box")
	cbFruit := ui.NewComboBox()
	cbFruit.AddItem("Apple", nil)
	cbFruit.AddItem("Banana", nil)
	cbFruit.AddItem("Cherry", nil)
	cbFruit.AddItem("Dragon fruit", nil)
	c.AddWidget(row, 0, cbFruit)
	row++
	lblFruitHint := c.AddLabel(row, 0, "The selected item shows directly in the box above - no button needed.")
	lblFruitHint.SetXExpandable(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Combo box with a data payload")
	cbCountry := ui.NewComboBox()
	cbCountry.AddItem("United States", "US")
	cbCountry.AddItem("Germany", "DE")
	cbCountry.AddItem("Japan", "JP")
	cbCountry.AddItem("Brazil", "BR")
	c.AddWidget(row, 0, cbCountry)
	c.AddButton(row, 1, "Show selected data", func() {
		c.setStatus(fmt.Sprintf("Country: %s (code: %v)", cbCountry.SelectedItemText(), cbCountry.SelectedItemData()))
	})
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Multiple independent combo boxes")
	c.cbSize = ui.NewComboBox()
	c.cbSize.AddItem("Small", nil)
	c.cbSize.AddItem("Medium", nil)
	c.cbSize.AddItem("Large", nil)
	c.AddWidget(row, 0, c.cbSize)

	c.cbColor = ui.NewComboBox()
	c.cbColor.AddItem("Red", nil)
	c.cbColor.AddItem("Green", nil)
	c.cbColor.AddItem("Blue", nil)
	c.cbColor.AddItem("Ultramarine Blue with a Hint of Violet", nil)
	c.AddWidget(row, 1, c.cbColor)

	c.AddButton(row, 2, "Build order summary", func() {
		c.setStatus(fmt.Sprintf("Order: %s, %s", c.cbSize.SelectedItemText(), c.cbColor.SelectedItemText()))
	})
	row = addSectionGap(&c.Widget, row+1)

	c.AddVSpacer(row, 0)

	return &c
}

func (c *ExamplePageComboBox) setStatus(text string) {
	c.lblStatus.SetText(text)
}
