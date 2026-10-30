package examples

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// TestRepaintMonkey moves the mouse over the applications, hovers, scrolls,
// drags, clicks and types at random, and after each action compares the
// frame - painted only where the widgets asked for it - with the whole
// window painted again (NUI_CHECK_REPAINT): a widget that changed without
// asking for its repaint leaves the old picture on the screen, and that is
// what it finds.
func TestRepaintMonkey(t *testing.T) {
	t.Setenv("NUI_CHECK_REPAINT", "panic")
	defer uitest.SetScreen(1920, 1080, 1)
	// The fractional scale is the one where the rounding of the pixels is
	// the hardest; NUI_MONKEY=long takes more actions on more scales
	steps, scales := 60, []float64{1, 1.5}
	if testing.Short() {
		steps, scales = 30, []float64{1.5}
	}
	if os.Getenv("NUI_MONKEY") == "long" {
		steps, scales = 400, []float64{1, 1.5, 2}
	}
	type entry struct {
		name    string
		newForm func() *ui.Form
	}
	entries := []entry{{"gallery", snapshotGallery}}
	for _, app := range Apps {
		if app.Name == "tasks" {
			continue // a tray icon needs the UI thread
		}
		entries = append(entries, entry{app.Name, app.NewForm})
	}
	partial := 0
	for _, scale := range scales {
		for _, e := range entries {
			t.Run(fmt.Sprintf("%s-%v", e.name, scale), func(t *testing.T) {
				uitest.SetScreen(4000, 3000, scale)
				form := e.newForm()
				form.SetSize(1100, 760)
				w := uitest.Show(t, form)
				h := fnv.New64a()
				h.Write([]byte(t.Name()))
				monkey(t, w, rand.New(rand.NewSource(int64(h.Sum64()))), steps)
				_, p := form.RepaintStats()
				partial += p
				form.Close()
				uitest.RunPosted()
			})
		}
	}
	if partial == 0 {
		t.Error("no frame was painted in part")
	}
	t.Logf("%d frames painted in part", partial)
}

// monkey does the random actions on the window; a frame that differs from
// the whole repaint fails the test, with the last actions
func monkey(t *testing.T, w *uitest.Window, rnd *rand.Rand, steps int) {
	var log []string
	defer func() {
		if r := recover(); r != nil {
			if len(log) > 12 {
				log = log[len(log)-12:]
			}
			msg := fmt.Sprint(r)
			if !strings.HasPrefix(msg, "nui: repaint check") {
				msg += "\n" + string(debug.Stack())
			}
			t.Fatalf("%s\nthe last actions:\n  %s", msg, strings.Join(log, "\n  "))
		}
	}()
	width, height := w.Size()
	point := func() uitest.Point { return uitest.Point{X: rnd.Intn(width), Y: rnd.Intn(height)} }
	for i := 0; i < steps && !w.Closed(); i++ {
		// A dialog or a message box the action opened is closed
		if top := w.Top(); top.Form != w.Form {
			log = append(log, "close the dialog "+top.Title())
			top.Key(ui.KeyEsc, ui.KeyModifiers{})
			if top = w.Top(); top.Form != w.Form {
				top.Form.Close()
				uitest.RunPosted()
			}
			continue
		}
		widgets := w.FindAll(func(x ui.Widgeter) bool { return x.IsVisible() && x.Width() > 0 && x.Height() > 0 })
		widget := func() ui.Widgeter { return widgets[rnd.Intn(len(widgets))] }
		switch n := rnd.Intn(100); {
		case n < 45:
			p := point()
			log = append(log, fmt.Sprintf("move to %v", p))
			w.Move(p)
		case n < 60:
			x := widget()
			log = append(log, fmt.Sprintf("hover %s", describe(x)))
			w.Move(x)
		case n < 70:
			p, dy := point(), rnd.Intn(7)-3
			log = append(log, fmt.Sprintf("wheel %d at %v", dy, p))
			w.Wheel(p, 0, dy)
		case n < 77:
			x, p := widget(), point()
			log = append(log, fmt.Sprintf("drag %s to %v", describe(x), p))
			w.Drag(x, p)
		case n < 85:
			x := widget()
			if risky(x) {
				continue
			}
			log = append(log, fmt.Sprintf("click %s", describe(x)))
			w.Click(x)
		case n < 92:
			log = append(log, "wait")
			w.Wait(time.Duration(rnd.Intn(4)) * 50 * time.Millisecond)
		default:
			keys := []ui.Key{ui.KeyTab, ui.KeyArrowDown, ui.KeyArrowUp, ui.KeyArrowRight, ui.KeyEsc}
			if rnd.Intn(2) == 0 {
				log = append(log, "type")
				w.Type("ab")
			} else {
				k := keys[rnd.Intn(len(keys))]
				log = append(log, fmt.Sprintf("key %v", k))
				w.Key(k, ui.KeyModifiers{})
			}
		}
	}
	if !w.Closed() {
		w.Screenshot()
	}
}

func describe(x ui.Widgeter) string {
	s := x.TypeName()
	if x.Name() != "" {
		s += " " + x.Name()
	}
	if t, ok := x.(interface{ Text() string }); ok && t.Text() != "" {
		s += fmt.Sprintf(" %q", t.Text())
	}
	return s
}

// risky tells the widgets not to click: those that quit, or open the
// dialogs of the system
func risky(x ui.Widgeter) bool {
	t, ok := x.(interface{ Text() string })
	if !ok {
		return false
	}
	s := strings.ToLower(t.Text())
	for _, word := range []string{"exit", "quit", "close", "open", "save", "print", "export", "browse"} {
		if strings.Contains(s, word) {
			return true
		}
	}
	return false
}
