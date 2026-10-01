package platforms

import (
	"errors"
	"image"
)

// ErrPrintCanceled is returned when the user cancels printing
var ErrPrintCanceled = errors.New("nui: printing canceled")

// ErrNoPrinting is returned where there is no way to print: no print dialog
// (Windows) and no CUPS (lp) elsewhere
var ErrNoPrinting = errors.New("nui: printing is not available")

// PrintRequest is a document to print with the system print dialog. The page
// size is the printable area of the printer the user chose, in pixels at dpi.
type PrintRequest struct {
	Title string
	// PageCount returns the number of pages for the page size
	PageCount func(width, height int, dpi float64) int
	// Render draws the page (0-based)
	Render func(page, width, height int, dpi float64) *image.RGBA
	// MaxDPI caps the resolution the pages are drawn at; the printer scales
	// them up to its own
	MaxDPI float64
}
