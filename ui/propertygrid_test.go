package ui

import (
	"image/color"
	"testing"
	"time"
)

type testSettings struct {
	Name    string     `category:"General" desc:"Shown in the title"`
	Level   string     `category:"General" options:"Low, Medium, High"`
	Port    int        `category:"Network" min:"1" max:"65535"`
	TLS     bool       `category:"Network" prop:"Use TLS"`
	Ratio   float64    `decimals:"3"`
	Color   color.RGBA `category:"Look"`
	Since   time.Time  `category:"Look"`
	Secret  string     `prop:"-"`
	private int
}

func TestPropertyGridManual(t *testing.T) {
	g := NewPropertyGrid()
	form := newTestForm(t, g)
	name := g.AddString("General", "Name", "Server")
	port := g.AddInt("Network", "Port", 80, 1, 100)
	tls := g.AddBool("Network", "TLS", false)
	mode := g.AddChoice("General", "Mode", []string{"A", "B", "C"}, 1)
	form.UpdateLayout()

	var changed []string
	g.SetOnChanged(func(p *Property) { changed = append(changed, p.Name()) })

	if len(g.sections) != 2 || g.Category("General") == nil || g.Category("Network") == nil {
		t.Fatalf("sections: %d", len(g.sections))
	}
	if port.Value() != 80 || tls.Value() != false || mode.Value() != 1 || name.Value() != "Server" {
		t.Fatalf("values: %v %v %v %v", port.Value(), tls.Value(), mode.Value(), name.Value())
	}
	// Code changes don't notify, and are clamped by the editor
	port.SetValue(500)
	if port.Value() != 100 || len(changed) != 0 {
		t.Errorf("SetValue: %v, changes %v", port.Value(), changed)
	}
	// A user change notifies
	tls.editor.(*Checkbox).SetChecked(true)
	mode.editor.(*ComboBox).selectByUser(2)
	if len(changed) != 2 || changed[0] != "TLS" || changed[1] != "Mode" {
		t.Errorf("changes: %v", changed)
	}
	// The name column is the same in all the categories
	if name.label.Width() != port.label.Width() || name.label.Width() < propertyGridMinNameWidth {
		t.Errorf("name widths %d and %d", name.label.Width(), port.label.Width())
	}
	// Editors line up: the same X on the window
	nx, _ := name.editor.RectClientAreaOnWindow()
	px, _ := port.editor.RectClientAreaOnWindow()
	if nx != px {
		t.Errorf("editors at x %d and %d", nx, px)
	}
	g.Clear()
	if len(g.Properties()) != 0 || g.Category("General") != nil {
		t.Error("Clear left properties")
	}
}

func TestPropertyGridObject(t *testing.T) {
	s := &testSettings{Name: "srv", Level: "High", Port: 8080, TLS: true, Ratio: 0.5,
		Color: color.RGBA{1, 2, 3, 255}, Since: time.Date(2024, 5, 6, 0, 0, 0, 0, time.Local)}
	// Filled before it's on a form, like an application builds it
	g := NewPropertyGrid()
	if err := g.SetObject(s); err != nil {
		t.Fatal(err)
	}
	newTestForm(t, g)
	// The editors line up in all the categories
	nameX, _ := g.Property("Name").editor.RectClientAreaOnWindow()
	portX, _ := g.Property("Port").editor.RectClientAreaOnWindow()
	colorX, _ := g.Property("Color").editor.RectClientAreaOnWindow()
	if nameX != portX || portX != colorX {
		t.Errorf("editors at x %d, %d, %d", nameX, portX, colorX)
	}
	if err := g.SetObject(*s); err == nil {
		t.Error("a struct value was accepted")
	}
	if len(g.Properties()) != 7 {
		t.Fatalf("properties: %d", len(g.Properties()))
	}
	if g.Property("Secret") != nil || g.Property("Use TLS") == nil {
		t.Error("tags prop:\"-\" / prop:\"Use TLS\" ignored")
	}
	if g.Property("Level").Value() != 2 || g.Property("Port").Value() != 8080 {
		t.Errorf("read: level %v, port %v", g.Property("Level").Value(), g.Property("Port").Value())
	}
	if box := g.Property("Port").editor.(*NumBox); box.max != 65535 {
		t.Errorf("max tag: %v", box.max)
	}

	// User edits go back into the struct
	g.Property("Level").editor.(*ComboBox).selectByUser(0)
	g.Property("Use TLS").editor.(*Checkbox).SetChecked(false)
	port := g.Property("Port").editor.(*NumBox)
	port.SetValue(9090)
	nameBox := g.Property("Name").editor.(*TextBox)
	nameBox.Focus()
	typeKey(nameBox.form, KeyEnd, 0, KeyModifiers{})
	typeKey(nameBox.form, KeyZ, 'z', KeyModifiers{})
	if s.Name != "srvz" {
		t.Errorf("typed name: %q", s.Name)
	}
	if s.Level != "Low" || s.TLS || s.Port != 9090 {
		t.Errorf("write back: %+v", s)
	}

	// Refresh shows the struct's new values
	s.Port = 1234
	s.Level = "Medium"
	g.Refresh()
	if g.Property("Port").Value() != 1234 || g.Property("Level").Value() != 1 {
		t.Errorf("refresh: port %v, level %v", g.Property("Port").Value(), g.Property("Level").Value())
	}
}

func TestToastInsideForm(t *testing.T) {
	form := NewForm()
	form.processResize(600, 400)
	form.ShowToastFor("Saved", ToastSuccess, time.Hour)
	form.ShowToastFor("A long message that surely wraps onto more than one line of the toast because it is long", ToastError, time.Hour)
	if len(form.toasts) != 2 {
		t.Fatalf("toasts: %d", len(form.toasts))
	}
	older, newer := form.toasts[0], form.toasts[1]
	if newer.y+newer.h != 400-toastMargin || older.y+older.h > newer.y {
		t.Errorf("stacking: older %d+%d, newer %d+%d", older.y, older.h, newer.y, newer.h)
	}
	if len(newer.lines) < 2 || newer.w > toastMaxWidth {
		t.Errorf("wrapping: %d lines, width %d", len(newer.lines), newer.w)
	}
	// A click closes the toast under the mouse
	form.processMouseDown(MouseButtonLeft, newer.x+5, newer.y+5)
	if len(form.toasts) != 1 || form.toasts[0] != older {
		t.Fatalf("click: %d toasts", len(form.toasts))
	}
	// Expired toasts go away on the timer; no more than toastMaxVisible stay
	older.expires = time.Now().Add(-time.Second)
	form.toastsProcessTimer()
	for i := 0; i < toastMaxVisible+3; i++ {
		form.ShowToast("x", ToastInfo)
	}
	if len(form.toasts) != toastMaxVisible {
		t.Errorf("toasts after overflow: %d", len(form.toasts))
	}
}
