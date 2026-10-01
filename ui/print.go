package ui

import (
	"bufio"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/ipoluianov/nui/internal/platforms"
)

// Printing. The application draws the pages on a Canvas, like widgets (see
// PrintJob.DrawPage); nui prints them, or saves them as a PDF file.
//
//	job := &ui.PrintJob{
//		Title: "Report",
//		Pages: 2,
//		DrawPage: func(p *ui.PrintPage) {
//			p.Canvas.SetFontSize(p.FontSize(12)) // 12 pt
//			p.Canvas.DrawText(p.MM(20), p.MM(20), p.Width-p.MM(40), p.MM(10), "Page "+strconv.Itoa(p.Index+1))
//		},
//	}
//	form.Print(job, func(err error) { ... }) // the print dialog, then the printer
//	job.SavePDF("report.pdf")
//
// The pages are images: the PDF has no selectable text.

// ErrPrintCanceled is given to Form.Print's callback when the user cancels
var ErrPrintCanceled = platforms.ErrPrintCanceled

// ErrNoPrinting: no printer can be used (no CUPS on Linux or macOS)
var ErrNoPrinting = platforms.ErrNoPrinting

// PaperSize is a paper size in millimeters, portrait
type PaperSize struct {
	Name          string
	Width, Height float64
}

var (
	PaperA4     = PaperSize{"A4", 210, 297}
	PaperA5     = PaperSize{"A5", 148, 210}
	PaperA3     = PaperSize{"A3", 297, 420}
	PaperLetter = PaperSize{"Letter", 215.9, 279.4}
	PaperLegal  = PaperSize{"Legal", 215.9, 355.6}
)

// PaperSizes are the sizes offered in nui's print dialog
var PaperSizes = []PaperSize{PaperA4, PaperA5, PaperA3, PaperLetter, PaperLegal}

// DefaultPrintDPI is the resolution the pages are drawn at
const DefaultPrintDPI = 300

// PrintJob is a document to print or to save as PDF
type PrintJob struct {
	Title string
	// Pages is the number of pages; or set Paginate when it depends on the
	// page size
	Pages int
	// Paginate returns the number of pages for the page size (e.g. how many
	// pages a text takes); called before drawing them
	Paginate func(width, height int, dpi float64) int
	// DrawPage draws one page on the white page canvas
	DrawPage func(p *PrintPage)

	// Paper and Landscape are the page for SavePDF and nui's print dialog;
	// with the Windows print dialog the printer's settings apply. A4 if not set.
	Paper     PaperSize
	Landscape bool
	// DPI is the resolution the pages are drawn at, DefaultPrintDPI if 0
	DPI float64
}

// PrintPage is the page being drawn
type PrintPage struct {
	Canvas *Canvas
	// Index is the page's number, from 0; Count is the number of pages
	Index, Count int
	// Width and Height are the page size in pixels at DPI: the whole paper
	// in a PDF, the printable area on a printer
	Width, Height int
	DPI           float64
}

// MM converts millimeters to the page's pixels
func (p *PrintPage) MM(mm float64) int {
	return int(math.Round(mm * p.DPI / 25.4))
}

// FontSize converts a font size in points (1/72 inch) to the pixels the
// canvas takes: Canvas.SetFontSize(p.FontSize(10)) for 10 pt text
func (p *PrintPage) FontSize(points float64) float64 {
	return points * p.DPI / 72
}

func (j *PrintJob) dpi() float64 {
	if j.DPI > 0 {
		return j.DPI
	}
	return DefaultPrintDPI
}

func (j *PrintJob) paper() PaperSize {
	if j.Paper.Width > 0 && j.Paper.Height > 0 {
		return j.Paper
	}
	return PaperA4
}

// pageSize returns the paper size in pixels at the job's DPI
func (j *PrintJob) pageSize() (int, int) {
	paper := j.paper()
	w := int(math.Round(paper.Width * j.dpi() / 25.4))
	h := int(math.Round(paper.Height * j.dpi() / 25.4))
	if j.Landscape {
		w, h = h, w
	}
	return w, h
}

func (j *PrintJob) pageCount(width, height int, dpi float64) int {
	if j.Paginate != nil {
		return j.Paginate(width, height, dpi)
	}
	return j.Pages
}

// renderPage draws the page on a white image
func (j *PrintJob) renderPage(index, count, width, height int, dpi float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	if j.DrawPage != nil {
		cnv := NewCanvas(img)
		cnv.SetDirectTranslateAndClip(0, 0, width, height)
		cnv.SetFontFamily(ThemeFontFamily())
		cnv.SetFontSize(12 * dpi / 72)
		// Paper is white whatever the theme: black text by default
		cnv.SetColor(color.Black)
		j.DrawPage(&PrintPage{Canvas: cnv, Index: index, Count: count, Width: width, Height: height, DPI: dpi})
	}
	return img
}

// SavePDF writes all the pages to a PDF file
func (j *PrintJob) SavePDF(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := j.WritePDF(f, nil); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// WritePDF writes the pages (0-based; nil for all) as a PDF document
func (j *PrintJob) WritePDF(w io.Writer, pages []int) error {
	width, height := j.pageSize()
	dpi := j.dpi()
	count := j.pageCount(width, height, dpi)
	if pages == nil {
		for i := 0; i < count; i++ {
			pages = append(pages, i)
		}
	}
	pdf := newPDFWriter(w)
	return pdf.write(j.Title, len(pages), float64(width)*72/dpi, float64(height)*72/dpi, func(n int) *image.RGBA {
		return j.renderPage(pages[n], count, width, height, dpi)
	})
}

// pdfWriter writes a PDF whose pages are images, one per page
type pdfWriter struct {
	w       *bufio.Writer
	offset  int
	offsets []int // of the objects, from 1
	err     error
}

func newPDFWriter(w io.Writer) *pdfWriter {
	return &pdfWriter{w: bufio.NewWriter(w)}
}

func (p *pdfWriter) printf(format string, args ...any) {
	if p.err != nil {
		return
	}
	n, err := fmt.Fprintf(p.w, format, args...)
	p.offset += n
	p.err = err
}

func (p *pdfWriter) bytes(b []byte) {
	if p.err != nil {
		return
	}
	n, err := p.w.Write(b)
	p.offset += n
	p.err = err
}

// beginObject starts the object of the number (from 1)
func (p *pdfWriter) beginObject(n int) {
	for len(p.offsets) < n {
		p.offsets = append(p.offsets, 0)
	}
	p.offsets[n-1] = p.offset
	p.printf("%d 0 obj\n", n)
}

// pdfString escapes a text for a PDF string literal; non-ASCII as UTF-16
func pdfString(s string) string {
	ascii := true
	for _, r := range s {
		if r > 126 {
			ascii = false
		}
	}
	if ascii {
		r := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
		return "(" + r.Replace(s) + ")"
	}
	var b strings.Builder
	b.WriteString("<FEFF")
	for _, u := range utf16Encode(s) {
		fmt.Fprintf(&b, "%04X", u)
	}
	b.WriteString(">")
	return b.String()
}

func utf16Encode(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r >= 0x10000 {
			r -= 0x10000
			out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
		} else {
			out = append(out, uint16(r))
		}
	}
	return out
}

// write writes the document: objects 1 catalog, 2 pages, 3 info, then
// for each page its page, contents and image objects
func (p *pdfWriter) write(title string, pageCount int, widthPt, heightPt float64, render func(n int) *image.RGBA) error {
	p.printf("%%PDF-1.4\n%%\xE2\xE3\xCF\xD3\n")

	p.beginObject(1)
	p.printf("<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	p.beginObject(2)
	p.printf("<< /Type /Pages /Count %d /Kids [", pageCount)
	for i := 0; i < pageCount; i++ {
		p.printf(" %d 0 R", 4+i*3)
	}
	p.printf(" ] >>\nendobj\n")

	p.beginObject(3)
	p.printf("<< /Title %s /Producer (nui) >>\nendobj\n", pdfString(title))

	for i := 0; i < pageCount; i++ {
		pageObj, contentsObj, imageObj := 4+i*3, 5+i*3, 6+i*3
		img := render(i)
		w, h := img.Rect.Dx(), img.Rect.Dy()

		p.beginObject(pageObj)
		p.printf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /XObject << /Im0 %d 0 R >> >> >>\nendobj\n",
			widthPt, heightPt, contentsObj, imageObj)

		contents := fmt.Sprintf("q %.2f 0 0 %.2f 0 0 cm /Im0 Do Q\n", widthPt, heightPt)
		p.beginObject(contentsObj)
		p.printf("<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(contents), contents)

		data, err := deflateRGB(img)
		if err != nil {
			return err
		}
		p.beginObject(imageObj)
		p.printf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n",
			w, h, len(data))
		p.bytes(data)
		p.printf("\nendstream\nendobj\n")
	}

	xref := p.offset
	p.printf("xref\n0 %d\n0000000000 65535 f \n", len(p.offsets)+1)
	for _, off := range p.offsets {
		p.printf("%010d 00000 n \n", off)
	}
	p.printf("trailer\n<< /Size %d /Root 1 0 R /Info 3 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(p.offsets)+1, xref)
	if p.err != nil {
		return p.err
	}
	return p.w.Flush()
}

// deflateRGB compresses the image's RGB bytes
func deflateRGB(img *image.RGBA) ([]byte, error) {
	var buf strings.Builder
	zw, _ := zlib.NewWriterLevel(&buf, zlib.BestSpeed)
	w, h := img.Rect.Dx(), img.Rect.Dy()
	row := make([]byte, w*3)
	for y := 0; y < h; y++ {
		src := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			row[x*3], row[x*3+1], row[x*3+2] = src[x*4], src[x*4+1], src[x*4+2]
		}
		if _, err := zw.Write(row); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// Print prints the job: it shows the print dialog - the system's on
// Windows, nui's elsewhere, with the printers of CUPS and "Save as PDF" - and
// prints on the printer the user chose. onDone is called on the UI thread
// with nil when the job went to the printer (or the PDF is saved),
// ErrPrintCanceled when the user canceled, or the error.
func (c *Form) Print(job *PrintJob, onDone func(err error)) {
	done := func(err error) {
		if onDone != nil {
			onDone(err)
		}
	}
	if platforms.HasPrintDialog() {
		Invoke(func() {
			err := platforms.PrintWithDialog(c.wnd, platforms.PrintRequest{
				Title: job.Title,
				PageCount: func(width, height int, dpi float64) int {
					return job.pageCount(width, height, dpi)
				},
				Render: func(page, width, height int, dpi float64) *image.RGBA {
					return job.renderPage(page, job.pageCount(width, height, dpi), width, height, dpi)
				},
				MaxDPI: job.dpi(),
			})
			done(err)
		})
		return
	}
	showPrintDialog(c, job, done)
}

// parsePageRange parses "1-3, 5" into 0-based page indices within count
func parsePageRange(s string, count int) ([]int, error) {
	var pages []int
	seen := map[int]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		from, to := part, part
		if a, b, ok := strings.Cut(part, "-"); ok {
			from, to = strings.TrimSpace(a), strings.TrimSpace(b)
		}
		f, err1 := strconv.Atoi(from)
		t, err2 := strconv.Atoi(to)
		if to == "" {
			t, err2 = count, nil
		}
		if err1 != nil || err2 != nil || f < 1 || t < f {
			return nil, fmt.Errorf("wrong page range %q", part)
		}
		for p := f; p <= min(t, count); p++ {
			if !seen[p] {
				seen[p] = true
				pages = append(pages, p-1)
			}
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no pages in %q", s)
	}
	return pages, nil
}
