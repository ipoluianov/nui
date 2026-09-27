package ex00gallery

import "github.com/ipoluianov/nui/ui"

type ExamplePageContextMenu struct {
	ui.Widget

	lblStatus *ui.Label
}

func NewExamplePageContextMenu() *ExamplePageContextMenu {
	var c ExamplePageContextMenu
	c.InitWidget()

	row := 0

	c.lblStatus = c.AddLabel(row, 0, "Right-click a label below to open its context menu")
	c.lblStatus.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Basic menu")
	lblBasic := c.AddLabel(row, 0, "Right-click me")
	menuBasic := ui.NewContextMenu(lblBasic)
	menuBasic.AddItem("Copy", func() { c.setStatus("Basic menu: Copy clicked") })
	menuBasic.AddItem("Paste", func() { c.setStatus("Basic menu: Paste clicked") })
	menuBasic.AddItem("Delete", func() { c.setStatus("Basic menu: Delete clicked") })
	lblBasic.SetContextMenu(menuBasic)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Menu that changes the label")
	lblColor := c.AddLabel(row, 0, "Right-click me to change my color")
	menuColor := ui.NewContextMenu(lblColor)
	menuColor.AddItem("Red", func() {
		lblColor.SetForegroundColor(ui.ColorFromHex("#E53935"))
		c.setStatus("Color menu: Red selected")
	})
	menuColor.AddItem("Green", func() {
		lblColor.SetForegroundColor(ui.ColorFromHex("#43A047"))
		c.setStatus("Color menu: Green selected")
	})
	menuColor.AddItem("Blue", func() {
		lblColor.SetForegroundColor(ui.ColorFromHex("#1E88E5"))
		c.setStatus("Color menu: Blue selected")
	})
	lblColor.SetContextMenu(menuColor)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Menu with a submenu")
	lblSubmenu := c.AddLabel(row, 0, "Right-click me for a submenu")
	menuSubmenu := ui.NewContextMenu(lblSubmenu)
	menuSubmenu.AddItem("Rename", func() { c.setStatus("Submenu menu: Rename clicked") })
	moveToSubmenu := ui.NewContextMenu(lblSubmenu)
	moveToSubmenu.AddItem("Archive", func() { c.setStatus("Submenu menu: Move to Archive clicked") })
	moveToSubmenu.AddItem("Trash", func() { c.setStatus("Submenu menu: Move to Trash clicked") })
	menuSubmenu.AddItemWithSubmenu("Move to...", moveToSubmenu)
	lblSubmenu.SetContextMenu(menuSubmenu)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Very short items (menu shrinks to its minimum width)")
	lblShort := c.AddLabel(row, 0, "Right-click me (short items)")
	menuShort := ui.NewContextMenu(lblShort)
	menuShort.AddItem("Go", func() { c.setStatus("Short menu: Go clicked") })
	menuShort.AddItem("OK", func() { c.setStatus("Short menu: OK clicked") })
	menuShort.AddItem("No", func() { c.setStatus("Short menu: No clicked") })
	lblShort.SetContextMenu(menuShort)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Very long items (menu grows up to its maximum width)")
	lblLong := c.AddLabel(row, 0, "Right-click me (long items)")
	menuLong := ui.NewContextMenu(lblLong)
	menuLong.AddItem("Export the current document as a PDF file", func() { c.setStatus("Long menu: Export clicked") })
	menuLong.AddItem("Send a copy to everyone on the review team", func() { c.setStatus("Long menu: Send clicked") })
	menuLong.AddItem("Permanently delete this item and all its history", func() { c.setStatus("Long menu: Delete clicked") })
	lblLong.SetContextMenu(menuLong)
	row = addSectionGap(&c.Widget, row+1)

	c.AddVSpacer(row, 0)

	return &c
}

func (c *ExamplePageContextMenu) setStatus(text string) {
	c.lblStatus.SetText(text)
}
