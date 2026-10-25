package regression

import (
	"strings"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func TestAccessibleTree(t *testing.T) {
	form := newForm(420, 320)
	form.SetTitle("Settings")
	bar := ui.NewMenuBar()
	bar.AddMenu("&File")
	form.SetMenuBar(bar)

	fields := ui.NewFormLayout()
	name := ui.NewTextBox()
	name.SetText("Ann")
	fields.AddRow("Name:", name)
	password := ui.NewTextBox()
	password.SetIsPassword(true)
	password.SetText("secret")
	fields.AddRow("Password:", password)
	speed := ui.NewSlider()
	speed.SetRange(0, 10)
	speed.SetValue(3)
	fields.AddRow("Speed:", speed)
	box := ui.NewVBox()
	box.Add(fields)
	remember := ui.NewCheckbox("&Remember me")
	box.Add(remember)
	list := ui.NewListView()
	list.SetItems([]string{"red", "green"})
	box.Add(list)
	icon := ui.NewButton("")
	icon.SetAccessibleName("Refresh")
	box.Add(icon)
	ok := ui.NewButton("OK")
	clicks := 0
	ok.OnClick.Connect(func(ui.Widgeter) { clicks++ })
	box.Add(ok)
	form.Panel().AddWidget(0, 0, box)
	w := uitest.Show(t, form)
	w.Click(name)

	root := w.Accessible()
	got := root.String()
	for _, want := range []string{
		`window "Settings"`,
		`  menu bar`,
		`    menu "File"`,
		`label "Name:"`,
		`text "Name" value="Ann" focused`,
		`password text "Password" value="••••••"`,
		`slider "Speed" value="3"`,
		`check box "Remember me"`,
		`list`,
		`  list item "red"`,
		`button "Refresh"`,
		`button "OK"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in\n%s", want, got)
		}
	}
	// The boxes are not objects: the widgets are right below the window
	if n := root.Find(ui.RoleButton, "OK"); n == nil || n.Parent != root {
		t.Errorf("OK not right below the window")
	}

	// The actions do what the user would
	root.Find(ui.RoleButton, "OK").DoAction("press")
	root.Find(ui.RoleCheckBox, "Remember me").DoAction("toggle")
	root.Find(ui.RoleListItem, "green").DoAction("select")
	root.Find(ui.RoleTextField, "Name").SetText("Bob")
	root.Find(ui.RoleSlider, "Speed").SetCurrent(20)
	if clicks != 1 || !remember.Checked() || list.CurrentRow() != 1 || name.Text() != "Bob" || speed.Value() != 10 {
		t.Errorf("after the actions: clicks %d, checked %v, row %d, name %q, speed %v",
			clicks, remember.Checked(), list.CurrentRow(), name.Text(), speed.Value())
	}
	// Disabled: no action
	ok.SetEnabled(false)
	if w.Accessible().Find(ui.RoleButton, "OK").DoAction("press") || clicks != 1 {
		t.Error("a disabled button pressed")
	}
}

func TestAccessibleTabsAndTree(t *testing.T) {
	form := newForm(400, 300)
	tabs := ui.NewTabWidget()
	tree := ui.NewTreeView()
	top := tree.AddNode(nil, "Fruits")
	tree.AddNode(top, "Apple")
	tree.AddNode(nil, "Vegetables")
	tabs.AddPage("Tree", tree)
	tabs.AddPage("Other", ui.NewLabel("other page"))
	form.Panel().AddWidget(0, 0, tabs)
	w := uitest.Show(t, form)

	root := w.Accessible()
	if tab := root.Find(ui.RoleTab, "Tree"); tab == nil || !tab.Selected {
		t.Fatalf("no selected tab Tree in\n%s", root)
	}
	if n := root.Find(ui.RoleTreeItem, "Apple"); n != nil {
		t.Error("a child of a collapsed node is shown")
	}
	root.Find(ui.RoleTreeItem, "Fruits").DoAction("expand or collapse")
	root = w.Accessible()
	if n := root.Find(ui.RoleTreeItem, "Apple"); n == nil || n.Level != 1 {
		t.Errorf("Apple after expanding: %v\n%s", n, root)
	}
	root.Find(ui.RoleTab, "Other").DoAction("select")
	if tabs.CurrentIndex() != 1 || w.Accessible().Find(ui.RoleLabel, "other page") == nil {
		t.Errorf("tab %d after selecting Other", tabs.CurrentIndex())
	}
}
