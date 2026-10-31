package regression

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// What outlives a form - a model, an action - is connected to its widgets
// only while the form is open: a closed form is neither kept in memory
// nor updated by it, and is connected again when shown again

func TestModelLinkedWhileShown(t *testing.T) {
	model := ui.NewListModel("a", "b")
	form := newForm(300, 200)
	table := ui.NewTable()
	table.SetModel(model)
	list := ui.NewListView()
	list.SetModel(model)
	form.Panel().AddWidget(0, 0, table)
	form.Panel().AddWidget(1, 0, list)
	w := uitest.Show(t, form)
	if !model.ModelChanged().Connected() {
		t.Fatal("the shown views don't follow the model")
	}

	w.RequestClose()
	if model.ModelChanged().Connected() {
		t.Error("the closed form still follows the model")
	}

	// Changed while closed, shown again: connected and up to date
	model.SetItems([]string{"a", "b", "c"})
	uitest.Show(t, form)
	if !model.ModelChanged().Connected() || list.RowCount() != 3 {
		t.Errorf("shown again: connected %v, rows %d", model.ModelChanged().Connected(), list.RowCount())
	}

	// A view taken out of the form lets the model go too
	form.Panel().RemoveWidget(table)
	form.Panel().RemoveWidget(list)
	if model.ModelChanged().Connected() {
		t.Error("the removed views still follow the model")
	}
}

func TestActionLinkedWhileShown(t *testing.T) {
	asked := 0
	save := ui.NewAction("&Save", nil).SetEnabledFunc(func() bool { asked++; return true })

	form := newForm(300, 200)
	bar := ui.NewMenuBar()
	bar.AddMenu("&File").AddAction(save)
	form.SetMenuBar(bar)
	toolbar := ui.NewToolBar()
	toolbar.AddAction(save)
	form.Panel().AddWidget(0, 0, toolbar)
	w := uitest.Show(t, form)
	if !save.OnChanged.Connected() {
		t.Fatal("the shown items don't follow the action")
	}
	w.RequestClose()
	if save.OnChanged.Connected() {
		t.Error("the closed form's items still follow the action")
	}

	// The enabled function isn't asked for the closed form's sake
	other := newForm(200, 100)
	o := uitest.Show(t, other)
	asked = 0
	o.Wait(500 * time.Millisecond)
	if asked != 0 {
		t.Errorf("asked %d times after its form closed", asked)
	}
}

// freed reports whether the object of the weak pointer is collected, after
// the posted functions ran (they may refer to a form closed just now)
func freed[T any](t *testing.T, ref weak.Pointer[T]) bool {
	o := uitest.Show(t, newForm(10, 10))
	o.Settle()
	for range 5 {
		runtime.GC()
		if ref.Value() == nil {
			return true
		}
	}
	return false
}

// A closed form is freed though the model, the action and the data source
// it showed live on (uitest.Show keeps its form until the test ends: the
// form is shown and closed by itself here)
func TestClosedFormFreed(t *testing.T) {
	model := ui.NewListModel("a")
	action := ui.NewAction("&Go", nil).SetEnabledFunc(func() bool { return true })
	source := ui.NewChartMemorySource()

	ref := func() weak.Pointer[ui.Form] {
		form := newForm(300, 300)
		table := ui.NewTable()
		table.SetModel(model)
		bar := ui.NewMenuBar()
		bar.AddMenu("&File").AddAction(action)
		form.SetMenuBar(bar)
		toolbar := ui.NewToolBar()
		toolbar.AddAction(action)
		chart := ui.NewLineChart()
		chart.Series("s").SetSource(source)
		form.Panel().AddWidget(0, 0, toolbar)
		form.Panel().AddWidget(1, 0, table)
		form.Panel().AddWidget(2, 0, chart)
		form.Show()
		form.Close()
		return weak.Make(form)
	}()
	if !freed(t, ref) {
		t.Error("the closed form is kept in memory")
	}
	runtime.KeepAlive(model)
	runtime.KeepAlive(action)
	runtime.KeepAlive(source)
}

// A disconnected handler isn't kept by its signal, also when it was the
// last one
func TestDisconnectedHandlerFreed(t *testing.T) {
	var signal ui.Signal[int]
	type data struct{ big [1 << 10]byte }
	ref := func() weak.Pointer[data] {
		d := &data{}
		conn := signal.Connect(func(int) { _ = d.big[0] })
		conn.Disconnect()
		return weak.Make(d)
	}()
	if !freed(t, ref) {
		t.Error("the handler's data is kept")
	}
	runtime.KeepAlive(&signal)
}

// A form connected to a signal that outlives it is called while open,
// not after it closes, again when shown again, and is freed when dropped
func TestConnectWhileShown(t *testing.T) {
	var theme ui.Signal[bool]
	calls := 0
	form := newForm(200, 100)
	label := form.Panel().AddLabel(0, 0, "x")
	theme.ConnectWhileShown(label, func(bool) { calls++ })
	panelCalls := 0
	theme.ConnectWhileShown(form.Panel(), func(bool) { panelCalls++ })
	w := uitest.Show(t, form)
	theme.Emit(true)
	w.RequestClose()
	theme.Emit(true)
	if calls != 1 || panelCalls != 1 {
		t.Errorf("calls %d, of the panel %d, want 1 (not after closing)", calls, panelCalls)
	}
	uitest.Show(t, form)
	theme.Emit(true)
	if calls != 2 {
		t.Errorf("calls %d after showing again", calls)
	}

	ref := func() weak.Pointer[ui.Form] {
		f := newForm(200, 100)
		theme.ConnectWhileShown(f.Panel(), func(bool) { _ = f.Panel() })
		f.Show()
		f.Close()
		return weak.Make(f)
	}()
	if !freed(t, ref) {
		t.Error("the closed form is kept by the signal")
	}
}
