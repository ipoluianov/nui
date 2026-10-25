package regression

import (
	"fmt"
	"image/color"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

// A code editor: line numbers and highlighting
func TestCodeEditor(t *testing.T) {
	form := newForm(420, 220)
	code := ui.NewTextBox()
	code.SetMultiline(true)
	code.SetFontFamily(ui.FontFamilyMono)
	code.SetLineNumbers(true)
	keyword := color.RGBA{0, 0, 200, 255}
	comment := color.RGBA{0, 128, 0, 255}
	code.SetHighlighter(func(line int, text string) []ui.TextSpan {
		var spans []ui.TextSpan
		for _, kw := range []string{"func", "return"} {
			if i := strings.Index(text, kw); i >= 0 {
				n := utf8.RuneCountInString(text[:i])
				spans = append(spans, ui.TextSpan{Start: n, End: n + len(kw), Color: keyword, Bold: true})
			}
		}
		if i := strings.Index(text, "//"); i >= 0 {
			n := utf8.RuneCountInString(text[:i])
			spans = append(spans, ui.TextSpan{Start: n, End: utf8.RuneCountInString(text), Color: comment, Italic: true})
		}
		return spans
	})
	form.Panel().AddWidget(0, 0, code)
	w := uitest.Show(t, form)
	var lines []string
	for i := range 12 {
		lines = append(lines, fmt.Sprintf("func f%d() int { return %d } // the %dth", i, i, i))
	}
	code.SetText(strings.Join(lines, "\n"))
	w.Click(code)
	code.SetCursorPosition(ui.TextPos{Line: 1, Col: 5})
	w.MatchGolden("code-editor")

	// A click right of the line numbers puts the cursor at the start of the
	// line, not under the numbers
	_, h := 0, code.OneLineHeight()
	w.Click(uitest.At(code, 2, 2+h*2+h/2))
	if got := code.CursorPosition(); got.Line != 2 || got.Col != 0 {
		t.Errorf("click in the numbers: %+v", got)
	}
}

func TestLogView(t *testing.T) {
	form := newForm(400, 200)
	logs := ui.NewLogView()
	logs.SetMaxLines(1000)
	form.Panel().AddWidget(0, 0, logs)
	w := uitest.Show(t, form)
	for i := range 50 {
		logs.AppendLine(fmt.Sprint("line ", i))
	}
	if logs.LineCount() != 50 || logs.Line(0) != "line 0" {
		t.Fatalf("%d lines, first %q", logs.LineCount(), logs.Line(0))
	}
	// It follows the end
	_, vh := logs.ViewportWidth(), logs.ViewportHeight()
	if logs.ScrollY()+vh < 49*logs.OneLineHeight() {
		t.Errorf("not at the end: scroll %d", logs.ScrollY())
	}
	// Scrolled up, it stays
	w.Wheel(logs, 0, 5)
	y := logs.ScrollY()
	logs.AppendLine("more")
	if logs.ScrollY() != y {
		t.Errorf("scrolled up: %d, then %d", y, logs.ScrollY())
	}

	// The oldest go past the limit
	for i := range 2000 {
		logs.AppendLine(fmt.Sprint("x", i))
	}
	if n := logs.LineCount(); n > 1000 || n < 900 || logs.Line(n-1) != "x1999" {
		t.Errorf("after 2000 more: %d lines, last %q", n, logs.Line(logs.LineCount()-1))
	}

	// From goroutines, as an io.Writer
	logs.Clear()
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 25 {
				fmt.Fprintf(logs, "g%d %d\n", g, i)
			}
		}()
	}
	wg.Wait()
	w.Settle()
	if n := logs.LineCount(); n != 100 {
		t.Errorf("100 lines written, %d shown", n)
	}
}

const sampleMarkdown = "# Help\n\n" +
	"Press **Ctrl+S** to save, *Ctrl+Z* to undo. The settings are in `~/.config/app`. " +
	"See [the site](https://example.com) for more.\n\n" +
	"## Lists\n\n" +
	"- one item, long enough to wrap onto the next line of the narrow view\n" +
	"- two\n\n" +
	"1. first\n2. second\n\n" +
	"> A quote.\n\n" +
	"```\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\n\n" +
	"---\n\nThe end."

func TestMarkdownView(t *testing.T) {
	form := newForm(360, 520)
	view := ui.NewMarkdownView()
	view.SetMarkdown(sampleMarkdown)
	form.Panel().AddWidget(0, 0, view)
	w := uitest.Show(t, form)
	w.MatchGolden("markdown-view")

	// Narrower: the text wraps into more lines
	h := view.InnerHeight()
	w.Resize(240, 520)
	if view.InnerHeight() <= h {
		t.Errorf("narrower, height %d, wider %d", view.InnerHeight(), h)
	}

	// A click on the link
	clicked := ""
	view.OnLinkClicked.Connect(func(url string) { clicked = url })
	w.Resize(360, 520)
	var at *uitest.Point
	for x := 0; x < view.Width() && at == nil; x += 4 {
		for y := 0; y < 200 && at == nil; y += 3 {
			if view.LinkAt(x, y) != "" {
				p := uitest.At(view, x, y)
				at = &p
			}
		}
	}
	if at == nil {
		t.Fatal("no link in the view")
	}
	w.Click(*at)
	if clicked != "https://example.com" {
		t.Errorf("link clicked: %q", clicked)
	}
	if !strings.Contains(view.Text(), "Press Ctrl+S to save") {
		t.Errorf("Text: %q", view.Text())
	}
}
