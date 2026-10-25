package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The accessible objects of the widgets of nui: the role, the name, the
// value, the states and the actions of each (see accessible.go)

// labelName is the name a buddy label gives (Label.SetBuddy, FormLayout):
// its text without the colon
func labelName(w *Widget) string {
	if w.accessibleLabel == nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(w.accessibleLabel.Text()), ":"))
}

func formatAccessibleNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (c *Button) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleButton
	n.Name = stripMnemonic(c.Text())
	if n.Name == "" {
		n.Name = c.Tooltip()
	}
	pressAction(n, func() { c.click() })
}

func (c *ButtonImage) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleButton
	n.Name = c.Tooltip()
	pressAction(n, func() {
		if c.onButtonClick != nil {
			c.onButtonClick(c)
		}
	})
}

func (c *ToolButton) accessibleInfo(n *AccessibleNode) {
	c.ButtonImage.accessibleInfo(n)
	n.Checkable, n.Checked = c.checked, c.checked
}

func (c *Checkbox) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleCheckBox
	n.Name = stripMnemonic(c.Text())
	n.Checkable, n.Checked = true, c.Checked()
	toggleAction(n, c.toggle)
}

func (c *RadioButton) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleRadioButton
	n.Name = stripMnemonic(c.Text())
	n.Checkable, n.Checked = true, c.Checked()
	pressAction(n, c.activate)
}

func (c *ToggleSwitch) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleSwitch
	n.Name = c.Text()
	if n.Name == "" {
		n.Name = labelName(&c.Widget)
	}
	n.Checkable, n.Checked = true, c.Checked()
	toggleAction(n, c.toggle)
}

func (c *Label) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleLabel
	n.Name = stripMnemonic(c.Text())
}

func (c *Link) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleLink
	n.Name = c.Text()
	n.Description = c.URL()
	pressAction(n, c.Activate)
}

func (c *TextBox) accessibleInfo(n *AccessibleNode) {
	n.Name = labelName(&c.Widget)
	n.Role = RoleTextField
	n.Value = c.Text()
	switch {
	case c.GetPropBool("ispassword", false):
		n.Role = RolePasswordField
		n.Value = strings.Repeat("•", utf8.RuneCountInString(n.Value))
	case c.Multiline():
		n.Role = RoleTextArea
		n.Multiline = true
	}
	n.Editable = !c.GetPropBool("readonly", false)
	n.Caret = c.OffsetOf(c.CursorPosition())
	n.setText = func(text string) bool {
		c.SelectAllText()
		c.InsertText(text)
		return true
	}
}

func (c *MarkdownView) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleTextArea
	n.Name = labelName(&c.Widget)
	n.Value = c.Text()
	n.Multiline = true
}

func (c *NumBox) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleSpinButton
	n.Name = labelName(&c.Widget)
	n.Value = c.Text()
	n.Min, n.Max, n.Current = c.Min(), c.Max(), c.Value()
	n.Editable = true
	n.setValue = c.SetValue
}

func (c *Slider) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleSlider
	n.Name = labelName(&c.Widget)
	n.Min, n.Max, n.Current = c.Min(), c.Max(), c.Value()
	n.Value = formatAccessibleNumber(c.Value())
	n.setValue = c.SetValue
}

func (c *ProgressBar) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleProgressBar
	n.Name = labelName(&c.Widget)
	n.Min, n.Max, n.Current = c.minValue, c.maxValue, c.value
	if c.maxValue > c.minValue {
		n.Value = fmt.Sprintf("%.0f%%", (c.value-c.minValue)/(c.maxValue-c.minValue)*100)
	}
	n.Focusable = false
}

func (c *ScrollBar) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleScrollBar
	n.Min, n.Max, n.Current = float64(c.min), float64(c.max), float64(c.value)
	n.Value = strconv.Itoa(c.value)
	n.setValue = func(v float64) { c.SetValue(int(v)) }
}

func (c *ComboBox) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleComboBox
	n.Name = labelName(&c.Widget)
	n.Value = c.SelectedItemText()
	pressAction(n, c.OpenPopup)
}

func (c *DatePicker) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleComboBox
	n.Name = labelName(&c.Widget)
	n.Value = c.Text()
	pressAction(n, c.OpenPopup)
}

func (c *TimePicker) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleSpinButton
	n.Name = labelName(&c.Widget)
	n.Value = c.Text()
}

func (c *ColorPicker) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleButton
	n.Name = labelName(&c.Widget)
	col := c.Color()
	n.Value = fmt.Sprintf("#%02X%02X%02X", col.R, col.G, col.B)
	pressAction(n, c.OpenPopup)
}

func (c *GroupBox) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleGroup
	n.Name = c.Title()
}

func (c *Expander) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleGroup
	n.Name = c.Title()
	n.Expandable, n.Expanded = true, c.Expanded()
	toggleAction(n, func() { c.SetExpanded(!c.Expanded()) })
}

func (c *ImageBox) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleImage
	n.Name = labelName(&c.Widget)
}

func (c *Chart) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleImage
	n.Name = labelName(&c.Widget)
}

func (c *TimeChart) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleImage
	n.Name = labelName(&c.Widget)
}

func (c *ZoomView) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleCanvas
	n.Name = labelName(&c.Widget)
}

func (c *Calendar) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleCalendar
	n.Name = labelName(&c.Widget)
	n.Value = c.Date().Format("2006-01-02")
}

func (c *StatusBar) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleStatusBar
}

func (c *ToolBar) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleToolBar
}

func (c *MenuBar) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleMenuBar
}

func (c *MenuBarItem) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleMenu
	n.Name = stripMnemonic(c.Text())
	pressAction(n, func() { c.bar.openMenu(c) })
}

// The tabs of a TabWidget are items of its header

func (c *tabWidgetHeader) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleTabList
}

func (c *tabWidgetHeader) accessibleItemCount() int {
	return len(c.items)
}

func (c *tabWidgetHeader) accessibleItem(n *AccessibleNode, i int) {
	n.Role = RoleTab
	n.Name = stripMnemonic(c.items[i])
	n.Selectable, n.Selected = true, i == c.currentIndex
	x, y := c.RectClientAreaOnWindow()
	for j := 0; j < i && j < len(c.itemsWidths); j++ {
		x += c.itemsWidths[j]
	}
	if i < len(c.itemsWidths) {
		n.Rect = image.Rect(x, y, x+c.itemsWidths[i], y+c.Height())
	}
	selectAction(n, func() { c.setCurrentPage(i) })
}

// The rows of a ListView

func (c *ListView) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleList
	n.Name = labelName(&c.Widget)
}

func (c *ListView) accessibleItemCount() int {
	return c.RowCount()
}

func (c *ListView) accessibleItem(n *AccessibleNode, row int) {
	n.Role = RoleListItem
	if m := c.Model(); m != nil && row < m.RowCount() {
		n.Name = m.Text(row, 0)
	}
	n.Selectable, n.Selected = true, c.IsRowSelected(row)
	n.Focused = c.IsFocused() && row == c.CurrentRow()
	x, y, w, h := c.ItemRect(row)
	wx, wy := c.RectClientAreaOnWindow()
	n.Rect = image.Rect(wx+x, wy+y, wx+x+w, wy+y+h)
	selectAction(n, func() { c.selectRow(row, KeyModifiers{}) })
}

// The rows of a Table: a row is read as its cells

func (c *Table) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleTable
	n.Name = labelName(&c.Widget)
}

func (c *Table) accessibleItemCount() int {
	return c.RowCount()
}

func (c *Table) accessibleItem(n *AccessibleNode, row int) {
	n.Role = RoleTableRow
	var cells []string
	for col := range c.ColumnCount() {
		if text := c.GetCellText2(row, col); text != "" {
			cells = append(cells, text)
		}
	}
	n.Name = strings.Join(cells, ", ")
	n.Selectable = true
	for _, r := range c.SelectedRows() {
		n.Selected = n.Selected || r == row
	}
	n.Focused = c.IsFocused() && row == c.CurrentRow()
	selectAction(n, func() {
		c.SetCurrentCell2(row, 0)
		c.SetSelectedRows([]int{row})
	})
}

// The nodes of a TreeView shown now (their parents expanded), in the order
// shown, with their level

func (c *TreeView) accessibleInfo(n *AccessibleNode) {
	n.Role = RoleTree
	n.Name = labelName(&c.Widget)
}

// accessibleNodes are the nodes shown, in order
func (c *TreeView) accessibleNodes() (nodes []*TreeNode, levels []int) {
	var walk func(list []*TreeNode, level int)
	walk = func(list []*TreeNode, level int) {
		for _, node := range list {
			nodes = append(nodes, node)
			levels = append(levels, level)
			if node.IsExpanded() {
				walk(node.Children(), level+1)
			}
		}
	}
	walk(c.Nodes(), 0)
	return nodes, levels
}

func (c *TreeView) accessibleItemCount() int {
	nodes, _ := c.accessibleNodes()
	return len(nodes)
}

func (c *TreeView) accessibleItem(n *AccessibleNode, i int) {
	nodes, levels := c.accessibleNodes()
	if i >= len(nodes) {
		return
	}
	node := nodes[i]
	n.Role = RoleTreeItem
	n.Name = node.Text(0)
	n.Level = levels[i]
	n.Expandable, n.Expanded = node.HasChildren(), node.IsExpanded()
	n.Selectable = true
	for _, s := range c.SelectedNodes() {
		n.Selected = n.Selected || s == node
	}
	n.Focused = c.IsFocused() && c.CurrentNode() == node
	n.Actions = []string{"select"}
	if n.Expandable {
		n.Actions = append(n.Actions, "expand or collapse")
	}
	n.doIt = func(action string) bool {
		if action == "select" {
			c.SetCurrentNode(node)
		} else {
			node.SetExpanded(!node.IsExpanded())
		}
		return true
	}
}

// selectAction is the action of an item that can be chosen
func selectAction(n *AccessibleNode, choose func()) {
	n.Actions = []string{"select"}
	n.doIt = func(string) bool {
		if !n.Enabled {
			return false
		}
		choose()
		return true
	}
}
