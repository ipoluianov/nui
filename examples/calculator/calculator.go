package calculator

import (
	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

const (
	compactWidth = 360
	historyWidth = 260
	formHeight   = 540
)

// calculator is the window: the display, the keypad and the history panel
// around an Engine.
type calculator struct {
	form   *ui.Form
	engine *Engine

	expr    *ui.Label
	display *ui.Label
	memory  *ui.Label

	digits    []*ui.Button
	memRecall *ui.Button
	memClear  *ui.Button

	historySwitch *ui.ToggleSwitch
	historyPanel  *ui.Panel
	history       *ui.Table
}

// NewForm is a desktop calculator in the style of the Windows one: a large
// right-aligned display under the pending expression, a keypad of
// expanding buttons with the "=" as the accent (primary role), memory keys,
// full keyboard input (digits, operators, Enter, Backspace, Esc, Delete),
// Ctrl+C / Ctrl+V through the clipboard, a main menu, and a history Table
// that is shown with a ToggleSwitch or View > History; a click on a row
// recalls its result, a right click opens a context menu. The logic is in
// Engine, a plain Go type with unit tests.
func NewForm() *ui.Form {
	c := &calculator{form: ui.NewForm(), engine: NewEngine()}
	c.form.SetTitle("Calculator")
	c.form.SetSize(compactWidth, formHeight)
	c.form.SetAllowMaximize(false)
	c.form.SetIcon(icons.App(32, "#2E7D32", func(dc *gg.Context) {
		// "+" over "="
		dc.DrawLine(8, 3, 8, 8)
		dc.DrawLine(5.5, 5.5, 10.5, 5.5)
		dc.DrawLine(5, 10.5, 11, 10.5)
		dc.DrawLine(5, 12.8, 11, 12.8)
		dc.Stroke()
	}))

	c.form.SetMenuBar(c.buildMenu())
	root := c.form.Panel()
	root.AddWidget(0, 0, c.buildKeypadPanel())
	c.historyPanel = c.buildHistoryPanel()
	root.AddWidget(0, 1, c.historyPanel)
	c.historyPanel.SetVisible(false)

	c.setupKeyboard()
	c.refresh()
	return c.form
}

func (c *calculator) buildMenu() *ui.MenuBar {
	bar := ui.NewMenuBar()

	edit := bar.AddMenu("&Edit")
	edit.AddItem("&Copy", c.copy).SetShortcut("Mod+C")
	edit.AddItem("&Paste", c.paste).SetShortcut("Mod+V")
	edit.AddSeparator()
	edit.AddItem("Clear &History", c.clearHistory).SetImage(icons.Cross(icons.Size))

	view := bar.AddMenu("&View")
	view.AddItem("&History", func() { c.showHistory(!c.historySwitch.Checked()) }).SetShortcut("Mod+H")
	view.AddSeparator()
	view.AddItem("&Light Theme", func() { ui.ApplyLightTheme(); c.refresh() }).SetImage(icons.Sun(icons.Size))
	view.AddItem("&Dark Theme", func() { ui.ApplyDarkTheme(); c.refresh() }).SetImage(icons.Moon(icons.Size))
	return bar
}

func (c *calculator) buildKeypadPanel() *ui.Panel {
	panel := ui.NewPanel()

	top := panel.AddPanel(0, 0)
	top.SetPanelPadding(0)
	c.historySwitch = ui.NewToggleSwitch("History")
	c.historySwitch.SetOnStateChanged(func() { c.showHistory(c.historySwitch.Checked()) })
	top.AddWidget(0, 0, c.historySwitch)
	top.AddHSpacer(0, 1)
	c.memory = top.AddLabel(0, 2, "")
	c.memory.SetTooltip("A number is stored in the memory")

	c.expr = ui.NewLabel("")
	c.expr.SetXExpandable(true)
	c.expr.SetTextAlign(ui.HAlignRight)
	panel.AddWidget(1, 0, c.expr)

	c.display = ui.NewLabel("")
	c.display.SetXExpandable(true)
	c.display.SetTextAlign(ui.HAlignRight)
	c.display.SetMinHeight(64)
	panel.AddWidget(2, 0, c.display)

	keys := panel.AddPanel(3, 0)
	keys.SetPanelPadding(0)
	keys.SetCellPadding(3)
	keys.SetYExpandable(true)

	e := c.engine
	key := func(row, col int, text string, onPush func()) *ui.Button {
		b := keys.AddButton(row, col, text, func() {
			onPush()
			c.refresh()
		})
		b.SetXExpandable(true)
		b.SetYExpandable(true)
		b.SetFontSize(17)
		b.SetMinWidth(56)
		// The keypad keys don't take the focus, so Enter and Space typed
		// after a click don't press the clicked key again
		b.SetCanBeFocused(false)
		return b
	}

	// The memory keys are smaller and keep their height
	memKey := func(col int, text, tooltip string, onPush func()) *ui.Button {
		b := key(0, col, text, onPush)
		b.SetFontSize(13)
		b.SetYExpandable(false)
		b.SetTooltip(tooltip)
		return b
	}
	c.memClear = memKey(0, "MC", "Memory clear", e.MemoryClear)
	c.memRecall = memKey(1, "MR", "Memory recall", e.MemoryRecall)
	memKey(2, "M+", "Add to memory", e.MemoryAdd)
	memKey(3, "M−", "Subtract from memory", e.MemorySubtract)

	key(1, 0, "%", e.Percent)
	setGlyph(key(1, 1, "", e.Sqrt), glyphSqrt)
	key(1, 2, "x²", e.Square)
	key(1, 3, "1/x", e.Reciprocal)

	key(2, 0, "CE", e.ClearEntry).SetTooltip("Clear entry (Delete)")
	key(2, 1, "C", e.Clear).SetTooltip("Clear (Esc)")
	bs := key(2, 2, "", e.Backspace)
	bs.SetTooltip("Backspace")
	setGlyph(bs, glyphBackspace)
	key(2, 3, "÷", func() { e.Operator(OpDiv) })
	key(3, 3, "×", func() { e.Operator(OpMul) })
	key(4, 3, "−", func() { e.Operator(OpSub) })
	key(5, 3, "+", func() { e.Operator(OpAdd) })

	c.digits = make([]*ui.Button, 10)
	for d := 1; d <= 9; d++ {
		c.digits[d] = key(5-(d-1)/3, (d-1)%3, string(rune('0'+d)), func() { e.Digit(d) })
	}
	key(6, 0, "±", e.Negate)
	c.digits[0] = key(6, 1, "0", func() { e.Digit(0) })
	key(6, 2, ".", e.Point)
	key(6, 3, "=", e.Equals).SetRole("primary")
	return panel
}

func (c *calculator) buildHistoryPanel() *ui.Panel {
	panel := ui.NewPanel()
	panel.SetMinWidth(historyWidth)
	panel.SetMaxWidth(historyWidth)
	panel.AddLabel(0, 0, "History")

	c.history = ui.NewTable()
	c.history.SetColumnCount(2)
	c.history.SetColumnName(0, "Expression")
	c.history.SetColumnName(1, "Result")
	c.history.SetColumnWidth(0, 145)
	c.history.SetColumnWidth(1, 95)
	c.history.SetColumnHAlign(1, ui.HAlignRight)
	c.history.SetYExpandable(true)
	c.history.SetOnCellMouseDown(func(button ui.MouseButton, row, col, x, y int, mods ui.KeyModifiers) {
		if button == ui.MouseButtonLeft {
			c.recall(row)
		}
	})
	panel.AddWidget(1, 0, c.history)

	menu := ui.NewContextMenu(c.history)
	menu.AddItem("Copy", func() {
		if row := c.history.CurrentRow(); row >= 0 {
			ui.ClipboardSetText(c.history.GetCellText2(row, 1))
		}
	})
	menu.AddItem("Use Result", func() { c.recall(c.history.CurrentRow()) })
	menu.AddSeparator()
	menu.AddItem("Clear History", c.clearHistory)
	c.history.SetContextMenu(menu)

	panel.AddButton(2, 0, "Clear History", c.clearHistory)
	return panel
}

// refresh shows the state of the engine.
func (c *calculator) refresh() {
	e := c.engine
	text := e.Display()
	c.display.SetText(text)
	// Long numbers get a smaller font so they fit
	size := 40.0
	if n := len([]rune(text)); n > 13 {
		size = max(18, 40*13/float64(n))
	}
	c.display.SetFontSize(size)

	c.expr.SetText(e.Expression())
	c.expr.SetForegroundColor(ui.CurrentPalette().PlaceholderText)

	if e.HasMemory() {
		c.memory.SetText("M")
	} else {
		c.memory.SetText("")
	}
	c.memClear.SetEnabled(e.HasMemory())
	c.memRecall.SetEnabled(e.HasMemory())

	// The digits stand out from the function keys, as on a real calculator
	digitColor := ui.MixColors(ui.CurrentPalette().Button, ui.CurrentPalette().ButtonText, 0.08)
	for _, b := range c.digits {
		b.SetBackgroundColor(digitColor)
	}

	c.refreshHistory()
}

// refreshHistory fills the table with the history, the latest on top.
func (c *calculator) refreshHistory() {
	items := c.engine.History()
	if c.history.RowCount() == len(items) {
		return
	}
	c.history.ClearRows()
	for i, h := range items {
		row := len(items) - 1 - i
		c.history.SetCellText2(row, 0, h.Expression)
		c.history.SetCellText2(row, 1, h.Result)
		c.history.SetCellHAlign(row, 1, ui.HAlignRight)
	}
	c.history.SetRowCount(len(items))
	c.history.ClearSelection()
}

func (c *calculator) recall(row int) {
	if row < 0 || row >= c.history.RowCount() {
		return
	}
	c.engine.SetText(c.history.GetCellText2(row, 1))
	c.refresh()
}

func (c *calculator) clearHistory() {
	c.engine.ClearHistory()
	c.refresh()
}

func (c *calculator) showHistory(show bool) {
	c.historySwitch.SetChecked(show)
	c.historyPanel.SetVisible(show)
	_, h := c.form.Size()
	if show {
		c.form.SetSize(compactWidth+historyWidth, h)
	} else {
		c.form.SetSize(compactWidth, h)
	}
}

func (c *calculator) copy() {
	if text := c.engine.Text(); text != "" {
		ui.ClipboardSetText(text)
	}
}

func (c *calculator) paste() {
	text, err := ui.ClipboardGetText()
	if err != nil || !c.engine.SetText(text) {
		c.form.ShowToast("The clipboard does not contain a number", ui.ToastWarning)
		return
	}
	c.refresh()
}
