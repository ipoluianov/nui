package regression

import (
	"errors"
	"strings"
	"testing"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

type loginHandlers struct{ clicks int }

func (h *loginHandlers) OK() { h.clicks++ }

func TestLayoutXML(t *testing.T) {
	form := newForm(400, 300)
	h := &loginHandlers{}
	err := form.Panel().SetLayout(`
		<column padding="8">
			<row>
				<label text="Name:"/>
				<textbox id="name" hint="Your name" text="Ann"/>
			</row>
			<checkbox id="remember" text="Remember me" checked="true"/>
			<numbox id="age" min="0" max="120" value="42"/>
			<combobox id="color" items="Red|Green|Blue" selected="1"/>
			<groupbox title="More">
				<slider id="speed" min="0" max="10" value="3"/>
			</groupbox>
			<tabwidget>
				<tab text="First"><label text="page one"/></tab>
				<tab text="Second"><label text="page two"/></tab>
			</tabwidget>
			<vspacer/>
			<row><hspacer/><button id="btnOK" text="OK" onclick="OK"/></row>
		</column>`, h, nil)
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Name     *ui.TextBox
		Remember *ui.Checkbox
		Age      *ui.NumBox
		Color    *ui.ComboBox
		Speed    *ui.Slider
		OK       *ui.Button `ui:"btnOK"`
		Missing  *ui.Label  `ui:"missing,optional"`
		Other    int
	}
	if err := ui.Bind(form.Panel(), &w); err != nil {
		t.Fatal(err)
	}
	win := uitest.Show(t, form)
	if w.Name.Text() != "Ann" || w.Name.Hint() != "Your name" || !w.Remember.Checked() || w.Age.Value() != 42 ||
		w.Color.SelectedItemText() != "Green" || w.Speed.Value() != 3 {
		t.Errorf("values: %q %q %v %v %q %v", w.Name.Text(), w.Name.Hint(), w.Remember.Checked(), w.Age.Value(),
			w.Color.SelectedItemText(), w.Speed.Value())
	}
	win.Click(w.OK)
	if h.clicks != 1 {
		t.Errorf("OK ran %d times", h.clicks)
	}
	if win.Find("page one") == nil || win.Accessible().Find(ui.RoleGroup, "More") == nil {
		t.Error("no tab page or group box")
	}

	// A wrong type and a missing widget
	var bad struct {
		Name  *ui.Button
		Gone  *ui.Label
	}
	err = ui.Bind(form.Panel(), &bad)
	if err == nil || !strings.Contains(err.Error(), `"name" is a *ui.TextBox`) || !strings.Contains(err.Error(), `no widget "gone"`) {
		t.Errorf("Bind error: %v", err)
	}
}

func TestLayoutXMLMistakes(t *testing.T) {
	form := newForm(300, 200)
	err := form.Panel().SetLayout(`<column>
		<buton text="typo"/>
		<label text="ok" colour="red"/>
		<numbox value="many"/>
		<button text="Go" onclick="Missing"/>
		<label id="a"/>
		<label id="a"/>
		<widget id="chart"/>
		<label text="still built"><label/></label>
	</column>`, map[string]func(){}, nil)
	var le *ui.LayoutError
	if !errors.As(err, &le) {
		t.Fatalf("error %v", err)
	}
	for _, want := range []string{
		"line 2: <buton>: unknown element",
		"line 3: <label>: unknown attribute colour",
		`line 4: <numbox>: value: "many" is not a number`,
		`line 5: <button>: no handler "Missing"`,
		`line 7: <label>: the id "a" is used at line 6 too`,
		`line 8: <widget>: no widget "chart" in the widgets given`,
		"line 9: <label>: <label> has no child elements",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("no %q in %v", want, err)
		}
	}
	// What is right is built
	w := uitest.Show(t, form)
	if w.Find("still built") == nil || w.Find("Go") == nil {
		t.Error("the right elements are not built")
	}
	if err := form.Panel().SetLayout(`<column><label>`, nil, nil); err == nil {
		t.Error("broken XML accepted")
	}
}
