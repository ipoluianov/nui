package regression

import (
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// The rule of the value signals: ...Changed on every change, OnActivated
// only on the user's

// counter counts the emissions of a signal
type counter[T any] struct {
	n    int
	last T
}

func count[T any](s *ui.Signal[T]) *counter[T] {
	c := &counter[T]{}
	s.Connect(func(v T) { c.n++; c.last = v })
	return c
}

func TestComboBoxSignals(t *testing.T) {
	form := newForm(300, 200)
	combo := ui.NewComboBox()
	for _, s := range []string{"One", "Two", "Three"} {
		combo.AddItem(s, nil)
	}
	form.Panel().AddWidget(0, 0, combo)
	w := uitest.Show(t, form)
	changed, activated := count(&combo.OnSelectionChanged), count(&combo.OnActivated)

	combo.SetSelectedIndex(1)
	combo.SetSelectedIndex(1) // no change
	if changed.n != 1 || activated.n != 0 {
		t.Fatalf("by the code: changed %d, activated %d", changed.n, activated.n)
	}
	w.Click(combo)
	w.Key(ui.KeyArrowDown, ui.KeyModifiers{})
	w.Key(ui.KeyEnter, ui.KeyModifiers{})
	if changed.n != 2 || activated.n != 1 || activated.last != 2 {
		t.Errorf("by the user: changed %d, activated %d (%d)", changed.n, activated.n, activated.last)
	}
}

func TestCheckboxSignals(t *testing.T) {
	form := newForm(300, 200)
	box := ui.NewCheckbox("Box")
	sw := ui.NewToggleSwitch("Switch")
	form.Panel().AddWidget(0, 0, box)
	form.Panel().AddWidget(1, 0, sw)
	w := uitest.Show(t, form)
	boxChanged, boxActivated := count(&box.OnCheckedChanged), count(&box.OnActivated)
	swChanged, swActivated := count(&sw.OnCheckedChanged), count(&sw.OnActivated)

	box.SetChecked(true)
	sw.SetChecked(true)
	if boxChanged.n != 1 || boxActivated.n != 0 || swChanged.n != 1 || swActivated.n != 0 {
		t.Fatalf("by the code: box %d/%d, switch %d/%d", boxChanged.n, boxActivated.n, swChanged.n, swActivated.n)
	}
	w.Click(box)
	w.Click(sw)
	if boxChanged.n != 2 || boxActivated.n != 1 || boxActivated.last || swActivated.n != 1 || swActivated.last {
		t.Errorf("by the user: box %d/%d %v, switch %d %v", boxChanged.n, boxActivated.n, boxActivated.last, swActivated.n, swActivated.last)
	}
}

func TestRadioButtonSignals(t *testing.T) {
	form := newForm(300, 200)
	group := form.Panel().AddPanel(0, 0)
	a, b := ui.NewRadioButton("A"), ui.NewRadioButton("B")
	group.AddWidget(0, 0, a)
	group.AddWidget(1, 0, b)
	w := uitest.Show(t, form)
	aChanged, bActivated := count(&a.OnCheckedChanged), count(&b.OnActivated)
	a.SetChecked(true)
	w.Click(b)
	if aChanged.n != 2 || aChanged.last || bActivated.n != 1 {
		t.Errorf("a changed %d (last %v), b activated %d", aChanged.n, aChanged.last, bActivated.n)
	}
}

func TestNumBoxAndSliderSignals(t *testing.T) {
	form := newForm(300, 200)
	num := ui.NewNumBox()
	slider := ui.NewSlider()
	slider.SetRange(0, 10)
	form.Panel().AddWidget(0, 0, num)
	form.Panel().AddWidget(1, 0, slider)
	form.Panel().AddVSpacer(2, 0) // see known bug #16
	w := uitest.Show(t, form)
	numChanged, numActivated := count(&num.OnValueChanged), count(&num.OnActivated)
	slChanged, slActivated := count(&slider.OnValueChanged), count(&slider.OnActivated)

	num.SetValue(5)
	slider.SetValue(3)
	if numChanged.n != 1 || numActivated.n != 0 || slChanged.n != 1 || slActivated.n != 0 {
		t.Fatalf("by the code: num %d/%d, slider %d/%d", numChanged.n, numActivated.n, slChanged.n, slActivated.n)
	}
	w.Click(num)
	w.Key(ui.KeyArrowUp, ui.KeyModifiers{})
	w.Click(slider)
	w.Key(ui.KeyArrowRight, ui.KeyModifiers{})
	if numActivated.n != 1 || numActivated.last != 5.01 || slActivated.n == 0 {
		t.Errorf("by the user: num activated %d (%v), slider activated %d", numActivated.n, numActivated.last, slActivated.n)
	}
}

func TestTextBoxSignals(t *testing.T) {
	form := newForm(300, 200)
	tb, other := ui.NewTextBox(), ui.NewTextBox()
	form.Panel().AddWidget(0, 0, tb)
	form.Panel().AddWidget(1, 0, other)
	w := uitest.Show(t, form)
	changed, edited := count(&tb.OnTextChanged), count(&tb.OnTextEdited)
	submitted, finished := count(&tb.OnSubmit), count(&tb.OnEditingFinished)

	tb.SetText("abc")
	tb.SetText("abc")
	if changed.n != 1 || edited.n != 0 {
		t.Fatalf("SetText: changed %d, edited %d", changed.n, edited.n)
	}
	w.Click(tb)
	w.Type("d")
	if changed.n != 2 || edited.n != 1 || edited.last != "abcd" {
		t.Errorf("typing: changed %d, edited %d %q", changed.n, edited.n, edited.last)
	}
	w.Key(ui.KeyEnter, ui.KeyModifiers{})
	if submitted.n != 1 || submitted.last != "abcd" || finished.n != 1 {
		t.Errorf("Enter: submitted %d %q, finished %d", submitted.n, submitted.last, finished.n)
	}
	// Leaving without a change finishes nothing; after a change it does
	w.Click(other)
	w.Click(tb)
	w.Type("e")
	w.Click(other)
	if finished.n != 2 || finished.last != "abcde" {
		t.Errorf("leaving: finished %d %q", finished.n, finished.last)
	}
}

func TestTableSignals(t *testing.T) {
	form := newForm(400, 300)
	table := ui.NewTable()
	table.SetColumnCount(2)
	table.SetRowCount(3)
	table.SetCellText2(1, 1, "x")
	form.Panel().AddWidget(0, 0, table)
	w := uitest.Show(t, form)
	selection, cells := count(&table.OnSelectionChanged), count(&table.OnCellChanged)
	var dbl *ui.TableCellEvent
	table.OnCellDoubleClick.Connect(func(e *ui.TableCellEvent) { dbl = e; e.Handled = true })

	table.SetCurrentCell2(2, 0)
	table.SetCellText2(0, 0, "a")
	table.SetCellText2(0, 0, "a") // no change
	if selection.n == 0 || selection.last.Row != 2 || cells.n != 1 {
		t.Errorf("selection %d %+v, cells %d", selection.n, selection.last, cells.n)
	}
	w.Key(ui.KeyArrowUp, ui.KeyModifiers{}) // the table isn't focused yet
	w.Click(table)
	before := selection.n
	w.Key(ui.KeyArrowDown, ui.KeyModifiers{})
	if selection.n == before {
		t.Error("no selection signal for the arrow key")
	}

	// A double click on a cell: the one the table makes current
	w.DoubleClick(uitest.At(table, 40, table.Height()/3))
	if dbl == nil || dbl.Row != table.CurrentRow() || dbl.Col != table.CurrentColumn() {
		t.Errorf("double click: %+v, current %d, %d", dbl, table.CurrentRow(), table.CurrentColumn())
	}
}

func TestTreeViewSignals(t *testing.T) {
	form := newForm(400, 300)
	tree := ui.NewTreeView()
	root := tree.AddNode(nil, "Root")
	child := root.AddChild("Child")
	form.Panel().AddWidget(0, 0, tree)
	uitest.Show(t, form)
	selection, expanding, collapsed := count(&tree.OnSelectionChanged), count(&tree.OnExpanding), count(&tree.OnCollapsed)
	root.Expand()
	tree.SetCurrentNode(child)
	if selection.last != child || expanding.n != 1 || expanding.last != root {
		t.Errorf("selection %v, expanding %d", selection.last, expanding.n)
	}
	// Collapsing hides the current node: the current one becomes the root
	root.Collapse()
	if collapsed.n != 1 || selection.last != root {
		t.Errorf("collapsed %d, selection %v", collapsed.n, selection.last)
	}
}

func TestThemeSignal(t *testing.T) {
	dark := count(&ui.OnThemeChanged)
	ui.ApplyDarkTheme()
	ui.ApplyLightTheme()
	if dark.n != 2 || dark.last {
		t.Errorf("theme signals %d, last dark %v", dark.n, dark.last)
	}
}

// Changing the items of a combo box keeps the selected item
func TestComboBoxItems(t *testing.T) {
	form := newForm(300, 200)
	combo := ui.NewComboBox()
	combo.SetItems([]string{"a", "b", "c", "d"})
	form.Panel().AddWidget(0, 0, combo)
	uitest.Show(t, form)
	changed := count(&combo.OnSelectionChanged)

	combo.SetSelectedIndex(2) // c
	combo.InsertItem(0, "z", 26)
	if combo.SelectedItemText() != "c" || combo.SelectedIndex() != 3 {
		t.Errorf("after InsertItem: %q at %d", combo.SelectedItemText(), combo.SelectedIndex())
	}
	combo.RemoveItem(1) // a
	if combo.SelectedItemText() != "c" || combo.SelectedIndex() != 2 {
		t.Errorf("after removing a: %q at %d", combo.SelectedItemText(), combo.SelectedIndex())
	}
	combo.RemoveItem(2) // c, the selected one: the next is selected
	if combo.SelectedItemText() != "d" {
		t.Errorf("after removing c: %q", combo.SelectedItemText())
	}
	if combo.FindText("b") != 1 || combo.FindData(26) != 0 || combo.ItemData(0) != 26 || combo.ItemText(9) != "" {
		t.Errorf("find: %d %d", combo.FindText("b"), combo.FindData(26))
	}
	combo.SetItems([]string{"x", "d", "y"})
	if combo.SelectedItemText() != "d" {
		t.Errorf("SetItems: %q", combo.SelectedItemText())
	}
	if changed.n != 5 {
		t.Errorf("OnSelectionChanged %d times, want 5", changed.n)
	}
	combo.Clear()
	if combo.ItemCount() != 0 || combo.SelectedItemText() != "" {
		t.Errorf("Clear: %d items", combo.ItemCount())
	}
	combo.AddItem("new", nil)
	if combo.SelectedItemText() != "new" {
		t.Errorf("first item after Clear: %q", combo.SelectedItemText())
	}
}

// The pages of a tab widget: the current one, changes keep it
func TestTabWidgetPages(t *testing.T) {
	form := newForm(400, 300)
	tabs := ui.NewTabWidget()
	pages := map[string]*ui.Panel{}
	for _, name := range []string{"A", "B", "C"} {
		pages[name] = ui.NewPanel()
		tabs.AddPage(name, pages[name])
	}
	form.Panel().AddWidget(0, 0, tabs)
	w := uitest.Show(t, form)
	current, activated := count(&tabs.OnCurrentChanged), count(&tabs.OnActivated)

	tabs.SetCurrentIndex(1)
	if tabs.CurrentIndex() != 1 || tabs.CurrentPage() != ui.Widgeter(pages["B"]) || current.n != 1 || activated.n != 0 {
		t.Fatalf("SetCurrentIndex: %d, changed %d, activated %d", tabs.CurrentIndex(), current.n, activated.n)
	}
	tabs.InsertPage(0, "Z", ui.NewPanel())
	if tabs.CurrentPage() != ui.Widgeter(pages["B"]) || tabs.CurrentIndex() != 2 || tabs.PageCount() != 4 {
		t.Errorf("after InsertPage: %d", tabs.CurrentIndex())
	}
	tabs.RemovePage(tabs.PageIndex(pages["B"])) // the current one: the next is current
	if tabs.CurrentPage() != ui.Widgeter(pages["C"]) {
		t.Errorf("after removing the current page: %d", tabs.CurrentIndex())
	}
	tabs.SetPageName(0, "First")

	// A click on the first tab: the user's choice
	w.Click(uitest.At(tabs, 20, 10))
	if tabs.CurrentIndex() != 0 || activated.n != 1 || activated.last != 0 {
		t.Errorf("click: current %d, activated %d", tabs.CurrentIndex(), activated.n)
	}
}
