package ui

import (
	"bytes"
	"image"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Hiding the seconds zeroes them: a change of the value, which used to
// come without OnTimeChanged
func TestTimePickerHideSecondsEmits(t *testing.T) {
	tp := NewTimePicker()
	newTestForm(t, tp)
	tp.SetShowSeconds(true)
	tp.SetTime(10, 20, 30)
	var got []time.Duration
	tp.OnTimeChanged.Connect(func(d time.Duration) { got = append(got, d) })
	tp.SetShowSeconds(false)
	want := 10*time.Hour + 20*time.Minute
	if tp.Duration() != want || len(got) != 1 || got[0] != want {
		t.Errorf("duration %v, OnTimeChanged %v; want %v once", tp.Duration(), got, want)
	}
	// No seconds to drop: no change, no signal
	tp.SetShowSeconds(true)
	tp.SetShowSeconds(false)
	if len(got) != 1 {
		t.Errorf("OnTimeChanged without a change: %v", got)
	}
}

// ParseHexColor used Sscanf on each pair, which took spaces, signs and a
// "0x" for digits
func TestParseHexColorStrict(t *testing.T) {
	for _, s := range []string{"# f0000", "#\tf0000", "#0x1234", "#ff00 0", "#+f0000", "#-10000", "#f f", "#+ff", "#1 2", "#é1", "#ff00g0"} {
		if c, ok := ParseHexColor(s); ok {
			t.Errorf("%q accepted as %v", s, c)
		}
	}
	for s, want := range map[string][4]uint8{
		"#ff0000":   {255, 0, 0, 255},
		"0aBc":      {}, // 4 digits: not a form
		"#abc":      {0xaa, 0xbb, 0xcc, 255},
		"#12345678": {0x12, 0x34, 0x56, 0x78},
		" #FFFFFF ": {255, 255, 255, 255},
	} {
		c, ok := ParseHexColor(s)
		if want == ([4]uint8{}) {
			if ok {
				t.Errorf("%q accepted as %v", s, c)
			}
			continue
		}
		if !ok || [4]uint8{c.R, c.G, c.B, c.A} != want {
			t.Errorf("%q -> %v %v, want %v", s, c, ok, want)
		}
	}
}

// A non-finite DPI or paper size used to panic in image.NewRGBA, a tiny
// paper gave a PDF image of 0 pixels, and no pages a PDF of no pages; the
// file was left behind
func TestPrintJobInvalidInput(t *testing.T) {
	dir := t.TempDir()
	for name, job := range map[string]*PrintJob{
		"infdpi":   {Pages: 1, DPI: math.Inf(1)},
		"nandpi":   {Pages: 1, DPI: math.NaN()},
		"hugedpi":  {Pages: 1, DPI: 1e6},
		"infpaper": {Pages: 1, DPI: 10, Paper: PaperSize{"x", math.Inf(1), 10}},
		"zero":     {Pages: 0, DPI: 10},
		"negative": {Pages: -3, DPI: 10},
	} {
		path := filepath.Join(dir, name+".pdf")
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: panic %v", name, r)
				}
			}()
			if err := job.SavePDF(path); err == nil {
				t.Errorf("%s: no error", name)
			}
		}()
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s: the file is left after the error", name)
		}
	}
	var buf bytes.Buffer
	if err := (&PrintJob{Pages: 1, DPI: 10, Paper: PaperSize{"x", 0.0001, 0.0001}}).WritePDF(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte("/Width 0")) || bytes.Contains(buf.Bytes(), []byte("/Height 0")) {
		t.Error("a PDF image of 0 pixels")
	}
}

// A font change resizes nothing: the view used to keep the layout of the
// old font
func TestMarkdownViewRelayoutOnFontChange(t *testing.T) {
	v := NewMarkdownView()
	form := newTestForm(t, v)
	v.SetMarkdown("Hello **world** and some more words to wrap")
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
	v.SetFontSize(30)
	form.UpdateLayout()
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
	if got := v.lines[0].parts[0].size; got != 30 {
		t.Errorf("the layout keeps the font size %v", got)
	}
}

// Many "[" or "`" without their closing marks: each one searched to the
// end of the paragraph, quadratic time (a few seconds for a megabyte)
func TestMarkdownInlineLinearTime(t *testing.T) {
	for name, text := range map[string]string{
		"brackets": strings.Repeat("[", 400000),
		"backtick": strings.Repeat("a`", 200000),
		"openlink": strings.Repeat("[a](", 100000),
		"farlink":  strings.Repeat("[", 400000) + "](",
	} {
		start := time.Now()
		parseInline(text)
		if d := time.Since(start); d > time.Second {
			t.Errorf("%s: %v for %d bytes", name, d, len(text))
		}
	}
	// The marks still work
	runs := parseInline("[x [a](u) `c` [b](v) `")
	var links, code int
	for _, r := range runs {
		if r.link != "" {
			links++
		}
		if r.code {
			code++
		}
	}
	if links != 2 || code != 1 {
		t.Errorf("runs %+v", runs)
	}
}

// The text between the elements of a layout was gathered with += for each
// piece, though nothing reads it: memory and time quadratic in the pieces
func TestLayoutXMLCharDataLinear(t *testing.T) {
	x := "<column>" + strings.Repeat("aaaaaaaaaa<!---->", 5000) + "</column>"
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	if _, err := parseLayoutXML(x); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	// The += gathering alone allocated 5000*5000*10/2 bytes, 125 MB
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 40<<20 {
		t.Errorf("%d MB allocated for %d KB of XML", alloc>>20, len(x)>>10)
	}
}

// A link of a MarkdownView without a handler went to OpenURL whatever it
// was: a file, a program, an option of the opener
func TestMarkdownViewLinkSchemes(t *testing.T) {
	var opened []string
	saved := markdownOpenURL
	markdownOpenURL = func(url string) error { opened = append(opened, url); return nil }
	defer func() { markdownOpenURL = saved }()

	v := NewMarkdownView()
	form := newTestForm(t, v)
	links := []string{"https://example.com", "file:///etc/passwd", "HTTP://x.org", "/usr/bin/xterm", "mailto:a@b.c", "-a", "javascript:x", "http://y.org"}
	var md []string
	for _, l := range links {
		md = append(md, "[link]("+l+")")
	}
	v.SetMarkdown(strings.Join(md, "\n\n"))
	form.processPaint(image.NewRGBA(image.Rect(0, 0, 500, 400)))
	for _, line := range v.lines {
		for _, part := range line.parts {
			if part.run.link != "" {
				v.mouseDown(MouseButtonLeft, part.x+1, line.y+1, KeyModifiers{})
			}
		}
	}
	want := []string{"https://example.com", "HTTP://x.org", "mailto:a@b.c", "http://y.org"}
	if strings.Join(opened, " ") != strings.Join(want, " ") {
		t.Errorf("opened %q, want %q", opened, want)
	}
}

// OpenURL took an address starting with "-" for an option of the opener,
// and never waited for the opener: a zombie process per link
func TestOpenURLOptionAndReap(t *testing.T) {
	if err := OpenURL("-a Terminal"); err == nil {
		t.Error("an address starting with - is opened")
	}
	if runtime.GOOS != "linux" {
		return // a zombie is seen in /proc
	}
	path, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no true command")
	}
	cmd := exec.Command(path)
	if err := startReaped(cmd); err != nil {
		t.Fatal(err)
	}
	// A zombie keeps its /proc entry until it is waited for
	stat := filepath.Join("/proc", strconv.Itoa(cmd.Process.Pid), "stat")
	for deadline := time.Now().Add(5 * time.Second); ; {
		if _, err := os.Stat(stat); os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			data, _ := os.ReadFile(stat)
			t.Fatalf("the child is not reaped: %s", data)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
