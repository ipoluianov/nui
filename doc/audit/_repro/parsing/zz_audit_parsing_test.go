package ui

import (
	"fmt"
	"image"
	"encoding/json"
	"math"
	"sync"
	"net"
	"strings"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func auditTry(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			st := debug.Stack()
			t.Errorf("%s: PANIC %v\n%s", name, r, st[:min(3000, len(st))])
		}
	}()
	f()
}

func auditPaintAll(form *Form) {
	form.UpdateLayout()
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
	form.processResize(10, 10)
	form.UpdateLayout()
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 10, 10)))
	form.processResize(0, 0)
	form.UpdateLayout()
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 1, 1)))
}

func TestAuditPDF(t *testing.T) {
	dir := "/tmp/claude-1000/-home-rb-src-nui/694d11ff-9f9c-4633-9145-2da40169701c/scratchpad/pdf"
	os.MkdirAll(dir, 0o755)
	titles := map[string]string{
		"plain":   "Report (1) \\ x",
		"cr":      "a\rb\r\nc",
		"unicode": "Отчёт 😀 ‮",
		"badutf":  "a\xffb",
		"nul":     "a\x00b",
		"del":     "a\x7fb",
		"unbal":   "((((",
	}
	for name, title := range titles {
		job := &PrintJob{Title: title, Pages: 1, DPI: 10}
		path := filepath.Join(dir, name+".pdf")
		if err := job.SavePDF(path); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, job := range map[string]*PrintJob{
		"zero":    {Pages: 0, DPI: 10},
		"neg":     {Pages: -3, DPI: 10},
		"infdpi":  {Pages: 1, DPI: math.Inf(1)},
		"infpap":  {Pages: 1, DPI: 10, Paper: PaperSize{"x", math.Inf(1), 10}},
		"tinypap": {Pages: 1, DPI: 10, Paper: PaperSize{"x", 0.0001, 0.0001}},
	} {
		auditTry(t, name, func() {
			path := filepath.Join(dir, name+".pdf")
			err := job.SavePDF(path)
			fmt.Println(name, err)
		})
	}
}

// A second start with an argument holding a newline (a legal file name on
// Unix): the running copy gets wrong arguments, or none and no "show"
func TestAuditInstanceNewlineArg(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	dir := t.TempDir()
	first, ok := acquireSingleInstance(dir, nil)
	if !ok {
		t.Fatal("not first")
	}
	defer first.Close()
	got := make(chan []string, 4)
	first.OnActivate(func(args []string) { got <- args })
	for _, args := range [][]string{
		{"/tmp/a\nshow", "/tmp/b"},
		{"/tmp/x\ny.txt"},
		{strings.Repeat("a", 70000)},
	} {
		if _, ok := acquireSingleInstance(dir, args); ok {
			t.Fatal("second start became first")
		}
		var a []string
		delivered := false
		for deadline := time.Now().Add(1500 * time.Millisecond); time.Now().Before(deadline) && !delivered; {
			platforms.RunHeadlessPosted()
			select {
			case a = <-got:
				delivered = true
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
		if delivered {
			t.Logf("sent %q -> got %q", trunc(args), trunc(a))
		} else {
			t.Logf("sent %q -> NOTHING delivered (no activation)", trunc(args))
		}
	}
}

func trunc(a []string) []string {
	var out []string
	for _, s := range a {
		if len(s) > 20 {
			s = s[:20] + fmt.Sprintf("...(%d bytes)", len(s))
		}
		out = append(out, s)
	}
	return out
}

// Close while the accept goroutine runs: a data race on c.listener and,
// when the goroutine loops after Close set it to nil, a nil interface call
func TestAuditInstanceCloseRace(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	for i := 0; i < 50; i++ {
		dir := t.TempDir()
		first, _ := acquireSingleInstance(dir, nil)
		first.OnActivate(func([]string) {})
		sock := instanceSocketPath(dir)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for j := 0; j < 20; j++ {
				if c, err := net.Dial("unix", sock); err == nil {
					c.Close()
				}
			}
		}()
		time.Sleep(time.Millisecond)
		first.Close()
		<-done
	}
}

// A settings file holding "null" (a crash while writing by another tool,
// or a hand edit): OpenSettings succeeds, then Set panics on a nil map
func TestAuditSettingsNull(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("null"), 0o644)
	s, err := OpenSettingsFile(path)
	t.Logf("open err=%v", err)
	auditTry(t, "Set after null", func() { s.Set("theme", "dark") })
}

// A settings file of another JSON type: the error is returned but the
// half-decoded map? (an object with a wrong value type)
func TestAuditSettingsPartial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"a": 1, "b": [}`), 0o644)
	s, err := OpenSettingsFile(path)
	t.Logf("open err=%v a=%d", err, s.Int("a", -1))
}

// "Safe for concurrent use": two goroutines saving at once share one
// temporary file name; one of the renames fails
func TestAuditSettingsConcurrentSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s, _ := OpenSettingsFile(path)
	for i := 0; i < 200; i++ {
		s.Set(fmt.Sprint("k", i), strings.Repeat("v", 1000))
	}
	errs := make(chan error, 1000)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if err := s.Save(); err != nil {
					errs <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	n := 0
	var first error
	for err := range errs {
		if first == nil {
			first = err
		}
		n++
	}
	data, _ := os.ReadFile(path)
	var m map[string]any
	jerr := json.Unmarshal(data, &m)
	t.Logf("failed saves: %d of 400, first: %v; final file valid JSON: %v (%d keys)", n, first, jerr == nil, len(m))
	if n > 0 || jerr != nil {
		t.Errorf("concurrent Save is not safe")
	}
}

func TestAuditBuiltinStrings(t *testing.T) {
	for lang := range builtinUIStrings() {
		s := uiStringsCatalog.Get(lang)
		for _, f := range []struct{ name, v string }{{"FileNotFound", s.FileNotFound}, {"FileExistsReplace", s.FileExistsReplace}} {
			if out := fmt.Sprintf(f.v, "X"); strings.Contains(out, "%!") || !strings.Contains(out, "X") {
				t.Errorf("%s %s: %q", lang, f.name, out)
			}
		}
		if out := monthTitle(s, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)); strings.Contains(out, "%!") || !strings.Contains(out, "2026") {
			t.Errorf("%s MonthYear: %q", lang, out)
		}
		for i, m := range s.MonthNames {
			if m == "" {
				t.Errorf("%s month %d empty", lang, i)
			}
		}
		for i, m := range s.WeekdaysShort {
			if m == "" {
				t.Errorf("%s weekday %d empty", lang, i)
			}
		}
	}
}

func TestAuditCalendarDST(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skip(err)
	}
	old := time.Local
	time.Local = loc
	defer func() { time.Local = old }()
	c := NewCalendar()
	c.SetFirstDayOfWeek(time.Sunday)
	// Brazil 2018: DST began at 00:00 on Nov 4 (midnight doesn't exist)
	c.SetDate(time.Date(2018, 11, 20, 12, 0, 0, 0, loc))
	// The draw loop: day = gridStart, day = day.AddDate(0,0,1) per cell
	day := c.gridStart()
	found := false
	var drift []string
	for i := 0; i < 42; i++ {
		if day.Equal(c.Date()) {
			found = true
		}
		if !day.Equal(DateOnly(day)) {
			drift = append(drift, day.Format("Jan 2 15:04"))
		}
		day = day.AddDate(0, 0, 1)
	}
	t.Logf("selected %v; cells not at midnight: %d (first %v)", c.Date(), len(drift), drift[:min(2, len(drift))])
	if !found {
		t.Errorf("the selected date %v never matches a grid cell: it is not highlighted", c.Date())
	}
	// The hit test of the same cell
	_, hit := c.partAt(c.gridLeft()+1+3*c.cellSize(), c.gridTop()+1+3*c.cellSize())
	t.Logf("partAt cell (3,3) -> %v", hit)
}

func TestAuditCalendarDSTSantiago(t *testing.T) {
	loc, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Skip(err)
	}
	old := time.Local
	time.Local = loc
	defer func() { time.Local = old }()
	c := NewCalendar()
	c.SetFirstDayOfWeek(time.Monday)
	c.SetDate(time.Date(2026, 9, 1, 12, 0, 0, 0, loc))
	day := c.gridStart()
	var labels []int
	for i := 0; i < 14; i++ {
		labels = append(labels, day.Day())
		day = day.AddDate(0, 0, 1)
	}
	t.Logf("labels of the first two rows of Sep 2026: %v", labels)
	t.Logf("DateOnly(Sep 6 2026 12:00) = %v", DateOnly(time.Date(2026, 9, 6, 12, 0, 0, 0, loc)))
	// Keyboard: Right from Sep 4 a few times
	c.SetDate(time.Date(2026, 9, 4, 0, 0, 0, 0, loc))
	var seen []string
	for i := 0; i < 4; i++ {
		c.ProcessKeyDown(KeyArrowRight, KeyModifiers{})
		seen = append(seen, c.Date().Format("Jan 2 15:04"))
	}
	t.Logf("Right x4 from Sep 4: %v", seen)
	if c.Date().Day() != 8 {
		t.Errorf("after 4 x Right from Sep 4 the date is %v, want Sep 8", c.Date())
	}
}

func TestAuditMarkdownQuadratic(t *testing.T) {
	for _, n := range []int{20000, 40000, 80000} {
		for name, text := range map[string]string{
			"brackets": strings.Repeat("[", n),
			"backtick": strings.Repeat("`a", n/2)[1:] + "x",
			"openlink": strings.Repeat("[a](", n/4),
			"plain":    strings.Repeat("a ", n/2),
		} {
			start := time.Now()
			parseMarkdown(text)
			t.Logf("%s n=%d: %v", name, n, time.Since(start))
		}
	}
}

func TestAuditXMLDeep(t *testing.T) {
	for _, n := range []int{1000, 10000, 50000} {
		x := strings.Repeat("<column>", n) + strings.Repeat("</column>", n)
		p := NewPanel()
		form := newTestForm(t, p)
		start := time.Now()
		err := p.SetLayout(x, nil, nil)
		t1 := time.Since(start)
		form.UpdateLayout()
		t2 := time.Since(start)
		form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
		t.Logf("depth %d: parse+build %v, +layout %v, +paint %v, err %v", n, t1, t2, time.Since(start), err)
	}
}

func TestAuditXMLCharData(t *testing.T) {
	for _, n := range []int{10000, 20000, 40000} {
		x := "<column>" + strings.Repeat("aaaaaaaaaa<!---->", n) + "</column>"
		start := time.Now()
		parseLayoutXML(x)
		t.Logf("chardata pieces %d (%d KB): %v", n, len(x)/1024, time.Since(start))
	}
}

func TestAuditMarkdownFontChange(t *testing.T) {
	v := NewMarkdownView()
	form := newTestForm(t, v)
	v.SetMarkdown("Hello **world** and some more words to wrap")
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
	before := v.lines[0].parts[0].size
	v.SetFontSize(30)
	form.UpdateLayout()
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
	after := v.lines[0].parts[0].size
	t.Logf("part font size before %v, after SetFontSize(30): %v; widget FontSize %v", before, after, v.FontSize())
	if after != 30 {
		t.Errorf("the layout keeps the old font size %v", after)
	}
}

func TestAuditMarkdownInline(t *testing.T) {
	for _, s := range []string{"[a](b)", "[a]()", "[](x)", "[a](b", "a ` b", "**", "_x_", "x_y_z", "[a](b) [c](d)", "[x](a)b)", "`", "[", "]("} {
		t.Logf("%q -> %+v", s, parseInline(s))
	}
}

func FuzzAuditMarkdown(f *testing.F) {
	f.Add("# h\n- a **b** [c](d) `e`\n```\nx\n```\n> q\n1. n")
	f.Fuzz(func(t *testing.T, s string) {
		if strings.ContainsAny(s, "\x1c\x1d\x1e\u0085\u2029\r\v\f") { // the known shaping panic
			return
		}
		blocks := parseMarkdown(s)
		v := &MarkdownView{}
		for _, b := range blocks {
			v.wrap(b, FontFamilySans, 14, 22, 50)
		}
	})
}

func FuzzAuditShortcut(f *testing.F) {
	f.Add("Ctrl+Shift++")
	f.Fuzz(func(t *testing.T, s string) {
		ParseShortcut(s)
		parseMnemonic(s)
		parsePageRange(s, 10)
		ParseHexColor(s)
		parseFileNames(s)
		pdfString(s)
	})
}

func FuzzAuditLayoutXML(f *testing.F) {
	f.Add(`<column><row><label text="a"/></row><table><columns><column/></columns><rows><row><cell text="x"/></row></rows></table></column>`)
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 4000 {
			return
		}
		p := NewPanel()
		p.SetLayout(s, map[string]func(){}, nil)
	})
}

func TestAuditShapingPanic(t *testing.T) {
	for _, s := range []string{"00־\x1e0", "0־\x1e0", "־\x1e0", "a־\x1eb", "א\x1e0", "0־\x1e", "0־\n0", "0־\t0", "0א\x1f0", "0ا\x1e0", "x־\x1cy", "א\r0", "א\rb", "aא\rb", "א b", "א\u0085b", "א b\r", "abc\rdef", "א\r\n0", "א\n0"} {
		auditTry(t, fmt.Sprintf("MeasureText(%q)", s), func() { MeasureText(FontFamilySans, 14, s) })
	}
	auditTry(t, "Label paint", func() {
		l := NewLabel("00־\x1e0")
		form := newTestForm(t, l)
		form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
	})
}

func TestAuditTimePickerHideSeconds(t *testing.T) {
	tp := NewTimePicker()
	newTestForm(t, tp)
	tp.SetShowSeconds(true)
	tp.SetTime(10, 20, 30)
	events := 0
	tp.OnTimeChanged.Connect(func(time.Duration) { events++ })
	tp.SetShowSeconds(false)
	t.Logf("duration now %v, OnTimeChanged fired %d times", tp.Duration(), events)
	if tp.Duration() != 10*time.Hour+20*time.Minute+30*time.Second && events == 0 {
		t.Errorf("the value changed without OnTimeChanged")
	}
}

func TestAuditHex(t *testing.T) {
	for _, s := range []string{"#ff0000", "#+f0000", "#-10000", "# f0000", "#0x1234", "#f f", "#+ff", "#1 2", "#ff00 0", "#é1", "#FFFFFF\n", "ff0000ff", "#fff0", "#ffff", "#\tf0000"} {
		c, ok := ParseHexColor(s)
		fmt.Printf("%q -> %v %v\n", s, c, ok)
	}
}

func TestAuditXMLProbes(t *testing.T) {
	cases := map[string]string{
		"fontsizeNaN":   `<column><label text="x" fontsize="NaN"/></column>`,
		"fontsizeInf":   `<column><label text="x" fontsize="Inf"/></column>`,
		"fontsizeNeg":   `<column><label text="x" fontsize="-5"/></column>`,
		"fontsize0":     `<column><label text="x" fontsize="0"/></column>`,
		"fontsizeHuge":  `<column><label text="x" fontsize="1e9"/></column>`,
		"minwidthHuge":  `<column><label text="x" minwidth="9223372036854775807"/><label text="y" minwidth="9223372036854775807"/></column>`,
		"minwidthNeg":   `<column><label text="x" minwidth="-100" width="-5"/></column>`,
		"paddingNeg":    `<column padding="-1000" spacing="-1000"><label text="x"/><label text="y"/></column>`,
		"paddingHuge":   `<column padding="2000000000" spacing="2000000000"><label text="x"/><label text="y"/></column>`,
		"comboSel":      `<column><combobox items="a|b" selected="99"/><combobox items="a|b" selected="-5"/><combobox selected="0"/></column>`,
		"numboxNaN":     `<column><numbox min="NaN" max="NaN" step="NaN" value="NaN"/></column>`,
		"numboxInf":     `<column><numbox min="-Inf" max="Inf" step="Inf" value="Inf"/></column>`,
		"numboxDec":     `<column><numbox decimals="-3" value="1.5"/></column>`,
		"numboxMinMax":  `<column><numbox min="10" max="0" value="5" step="-1"/></column>`,
		"sliderNaN":     `<column><slider min="NaN" max="NaN" step="NaN" value="NaN"/></column>`,
		"sliderInf":     `<column><slider min="-Inf" max="Inf" step="0" value="1"/></column>`,
		"sliderRev":     `<column><slider min="10" max="0" step="-1" value="5"/></column>`,
		"sliderTiny":    `<column><slider min="0" max="1e300" step="1e-300" value="5"/></column>`,
		"progNaN":       `<column><progressbar value="NaN"/><progressbar value="Inf"/><progressbar value="-Inf"/></column>`,
		"splitter1":     `<hsplitter><label text="a"/></hsplitter>`,
		"splitter3":     `<hsplitter><label text="a"/><label text="b"/><label text="c"/></hsplitter>`,
		"splitter0":     `<vsplitter/>`,
		"tabsEmpty":     `<tabwidget><tab/></tabwidget>`,
		"tabsBad":       `<tabwidget><label/></tabwidget>`,
		"tableBadW":     `<table><columns><column text="a" width="-100"/><column width="99999999999"/></columns><rows><row><cell text="1"/><cell text="2"/><cell text="3"/><cell text="4"/></row></rows></table>`,
		"tableNoCols":   `<table><rows><row><cell text="1"/><cell text="2"/></row><row/></rows></table>`,
		"tableOnlyCols": `<table><columns><column/><column/></columns></table>`,
		"linechart":     `<linechart values="NaN,Inf,-Inf,1e308,-1e308,,x,,," title="t"/>`,
		"linechart1":    `<linechart values="5"/>`,
		"linechartE":    `<linechart values=""/>`,
		"barchart":      `<barchart items="a:NaN|b:Inf|:|c|d:-1e308|e:1e308" stacked="true"/>`,
		"barchartE":     `<barchart items="" horizontal="true"/>`,
		"piechart":      `<piechart items="a:NaN|b:-5|c:Inf" donut="NaN"/>`,
		"piechart2":     `<piechart items="a:0|b:0" donut="5"/>`,
		"piechart3":     `<piechart items="a:-1|b:-1" donut="-5"/>`,
		"view3dshape":   `<view3d shape="" autorotate="NaN"/>`,
		"view3dcolor":   `<view3d color="#" />`,
		"view3dcolor2":  `<view3d color="#12345" />`,
		"emptyroot":     `<column/>`,
		"unknownNs":     `<x:column xmlns:x="urn:x"><x:label text="a"/></x:column>`,
		"widgetNoMap":   `<column><widget id="w"/></column>`,
		"onWidget":      `<column><button onclick=""/></column>`,
		"textboxML":     `<column><textbox multiline="true" text="a&#xD;b"/></column>`,
		"utf8bad":       "<column><label text=\"\xff\xfe\"/></column>",
		"bidi":          `<column><label text="&#x202E;abc&#x0301;&#x0301;&#x0301;"/><button text="&#x1F600;&#x200D;&#x1F600;"/></column>`,
		"scrollarea":    `<scrollarea><label text="a"/><label text="b"/></scrollarea>`,
		"groupbox":      `<groupbox title="&#x0301;" padding="-50"><label/></groupbox>`,
		"listview":      `<listview items="|||" multiselect="true"/>`,
		"datetime":      `<column><datepicker/><timepicker/><colorpicker/><imagebox/></column>`,
		"link":          `<link text="" url="javascript:alert(1)"/>`,
	}
	for name, x := range cases {
		auditTry(t, name, func() {
			p := NewPanel()
			form := newTestForm(t, p)
			err := p.SetLayout(x, map[string]func(){}, nil)
			auditPaintAll(form)
			fmt.Println("ok", name, err)
		})
	}
}
