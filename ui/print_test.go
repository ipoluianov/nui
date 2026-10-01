package ui

import (
	"bytes"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParsePageRange(t *testing.T) {
	cases := map[string][]int{
		"1":      {0},
		"1-3, 5": {0, 1, 2, 4},
		"2-":     {1, 2, 3, 4, 5},
		"5-9":    {4, 5},
		"3,3,1":  {2, 0},
	}
	for s, want := range cases {
		got, err := parsePageRange(s, 6)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %v %v, want %v", s, got, err, want)
		}
	}
	for _, bad := range []string{"0", "a", "3-1", "7-9", ","} {
		if _, err := parsePageRange(bad, 6); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestTextPrintJobPDF(t *testing.T) {
	var text strings.Builder
	for i := 1; i <= 150; i++ {
		text.WriteString("Строка номер ")
		text.WriteString(strings.Repeat("длинный текст ", i%7))
		text.WriteString("\n")
	}
	text.WriteString("    indented line\n")
	job := NewTextPrintJob("Тест", text.String(), nil)
	job.DPI = 100 // fast

	w, h := job.pageSize()
	if w != 827 || h != 1169 {
		t.Errorf("A4 at 100 dpi: %dx%d", w, h)
	}
	pages := job.pageCount(w, h, job.dpi())
	if pages < 3 {
		t.Fatalf("pages: %d", pages)
	}
	var buf bytes.Buffer
	if err := job.WritePDF(&buf, []int{0, pages - 1}); err != nil {
		t.Fatal(err)
	}
	pdf := buf.Bytes()
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) || !bytes.HasSuffix(pdf, []byte("%%EOF\n")) {
		t.Fatal("not a PDF")
	}
	if n := bytes.Count(pdf, []byte("/Type /Page ")); n != 2 {
		t.Errorf("pages in the PDF: %d", n)
	}

	// Checked by poppler when it is installed
	if _, err := exec.LookPath("pdfinfo"); err != nil {
		return
	}
	path := filepath.Join(t.TempDir(), "t.pdf")
	os.WriteFile(path, pdf, 0o644)
	out, err := exec.Command("pdfinfo", path).CombinedOutput()
	if err != nil {
		t.Fatalf("pdfinfo: %v\n%s", err, out)
	}
	info := string(out)
	if !strings.Contains(info, "Pages:           2") || !strings.Contains(info, "595.") || !strings.Contains(info, "Тест") {
		t.Errorf("pdfinfo:\n%s", info)
	}
}

func TestPrintPageDrawing(t *testing.T) {
	job := &PrintJob{Pages: 1, DPI: 72, Paper: PaperSize{"x", 100, 50}, Landscape: true,
		DrawPage: func(p *PrintPage) {
			if p.MM(25.4) != 72 || p.FontSize(12) != 12 {
				t.Errorf("units: %d %v", p.MM(25.4), p.FontSize(12))
			}
			p.Canvas.FillRect(0, 0, 10, 10, color.RGBA{255, 0, 0, 255})
		}}
	w, h := job.pageSize()
	img := job.renderPage(0, 1, w, h, 72)
	if w != 142 || h != 283 {
		t.Errorf("landscape 100x50 mm at 72 dpi: %dx%d", w, h)
	}
	if c := img.RGBAAt(5, 5); c.R != 255 || c.G != 0 {
		t.Errorf("drawn pixel %v", c)
	}
	if c := img.RGBAAt(50, 50); c != (color.RGBA{255, 255, 255, 255}) {
		t.Errorf("the page isn't white: %v", c)
	}
}

func TestWrapKeepingSpaces(t *testing.T) {
	short := "x :=   1 //  aligned"
	if got := wrapKeepingSpaces(short, FontFamilyMono, 14, 1000); len(got) != 1 || got[0] != short {
		t.Errorf("a line that fits: %q", got)
	}
	charW, _, _ := MeasureText(FontFamilyMono, 14, "W")
	got := wrapKeepingSpaces("    alpha  beta  gamma  delta", FontFamilyMono, 14, charW*16)
	want := []string{"    alpha  beta", "    gamma  delta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrapped %q, want %q", got, want)
	}
	long := wrapKeepingSpaces(strings.Repeat("x", 40), FontFamilyMono, 14, charW*10)
	if len(long) != 4 {
		t.Errorf("long word: %q", long)
	}
}
