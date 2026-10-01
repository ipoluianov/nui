# Printing and PDF

The application draws the pages on a [Canvas](widget.md), the same way widgets draw themselves;
nui sends them to a printer or saves them as a PDF file.

## Printing text

```go
job := ui.NewTextPrintJob("notes.txt", text, nil)
form.Print(job, func(err error) {
	if err != nil && err != ui.ErrPrintCanceled {
		form.ShowToast(err.Error(), ui.ToastError)
	}
})
```

`NewTextPrintJob` wraps the lines to the page width (keeping spaces and indentation, so code
and columns print as they are), splits them into pages and numbers the pages.
`TextPrintOptions`: `FontFamily` (e.g. `ui.FontFamilyMono`), `FontSize` in points (10),
`Margin` in millimeters (20), `NoPageNumbers`.

## Drawing pages

```go
job := &ui.PrintJob{
	Title: "Report",
	Pages: 3, // or Paginate: func(width, height int, dpi float64) int
	DrawPage: func(p *ui.PrintPage) {
		c := p.Canvas
		m := p.MM(20)                    // millimeters to pixels
		c.SetFontSize(p.FontSize(16))    // points to pixels
		c.DrawText(m, m, p.Width-2*m, p.MM(10), fmt.Sprintf("Page %d of %d", p.Index+1, p.Count))
	},
}
```

- `PrintPage`: `Canvas`, `Index` (from 0), `Count`, `Width`/`Height` in pixels, `DPI`.
- The page is white and the text black by default, whatever the theme.
- `Paginate` is for documents whose page count depends on the page size; it is called once the
  paper is known (on Windows, the printer's).
- `Paper` (`ui.PaperA4` by default, `PaperA5`, `PaperA3`, `PaperLetter`, `PaperLegal`),
  `Landscape`, `DPI` (300 by default).

## PDF

```go
err := job.SavePDF("report.pdf")
err = job.WritePDF(w, []int{0, 2}) // pages 1 and 3, to any io.Writer
```

The pages are images in the PDF: the text in it can't be selected or searched.

## The print dialog

`form.Print(job, onDone)` shows the print dialog and prints; `onDone` gets `nil`,
`ui.ErrPrintCanceled` or the error, on the UI thread.

- **Windows**: the system print dialog; the printer's paper, orientation and copies apply, and
  the page is the printer's printable area.
- **Linux, macOS**: printing goes through CUPS (`lpstat`, `lp`). nui shows its own dialog: the
  printer (or *Save as PDF*), copies, pages (`1-3, 5`), paper and orientation.

The pages are drawn on the UI thread, so `DrawPage` may read the widgets.
