package ui

import (
	"fmt"
	"strings"
)

// TextPrintOptions set up NewTextPrintJob; the zero values give the theme
// font at 10 pt, 20 mm margins and page numbers
type TextPrintOptions struct {
	FontFamily string
	// FontSize in points
	FontSize float64
	// Margin on all the sides, in millimeters
	Margin float64
	// NoPageNumbers leaves out the "page / pages" line at the bottom
	NoPageNumbers bool
}

// NewTextPrintJob makes a job that prints plain text: the lines are wrapped
// to the page width (keeping their indentation) and split into pages, with
// page numbers at the bottom.
//
//	form.Print(ui.NewTextPrintJob("notes.txt", text, nil), nil)
//	ui.NewTextPrintJob("log", logText, &ui.TextPrintOptions{FontFamily: ui.FontFamilyMono}).SavePDF("log.pdf")
func NewTextPrintJob(title, text string, opts *TextPrintOptions) *PrintJob {
	tp := &textPrint{text: text}
	if opts != nil {
		tp.opts = *opts
	}
	if tp.opts.FontFamily == "" {
		tp.opts.FontFamily = ThemeFontFamily()
	}
	if tp.opts.FontSize <= 0 {
		tp.opts.FontSize = 10
	}
	if tp.opts.Margin <= 0 {
		tp.opts.Margin = 20
	}
	return &PrintJob{Title: title, Paginate: tp.paginate, DrawPage: tp.draw}
}

type textPrint struct {
	text string
	opts TextPrintOptions

	// The layout for the page size it was made for
	width, height int
	dpi           float64
	lines         []string
	lineHeight    int
	perPage       int
}

// layout wraps the text for the page size
func (t *textPrint) layout(width, height int, dpi float64) {
	if t.lines != nil && t.width == width && t.height == height && t.dpi == dpi {
		return
	}
	t.width, t.height, t.dpi = width, height, dpi
	page := &PrintPage{DPI: dpi}
	size := page.FontSize(t.opts.FontSize)
	margin := page.MM(t.opts.Margin)
	textWidth := max(width-margin*2, 1)

	_, t.lineHeight, _ = MeasureText(t.opts.FontFamily, size, "Ag")
	t.lineHeight = max(t.lineHeight, 1)
	footer := 0
	if !t.opts.NoPageNumbers {
		footer = t.lineHeight * 2
	}
	t.perPage = max((height-margin*2-footer)/t.lineHeight, 1)

	t.lines = nil
	text := strings.ReplaceAll(strings.ReplaceAll(t.text, "\r\n", "\n"), "\t", "    ")
	for _, paragraph := range strings.Split(text, "\n") {
		t.lines = append(t.lines, wrapKeepingSpaces(paragraph, t.opts.FontFamily, size, textWidth)...)
	}
}

// wrapKeepingSpaces wraps a line at the spaces to fit maxWidth, keeping the
// spaces within it (code, aligned columns); the continuation lines get the
// indentation of the first one. A word wider than a line is broken.
func wrapKeepingSpaces(line, fontFamily string, fontSize float64, maxWidth int) []string {
	fits := func(s string) bool {
		w, _, err := MeasureText(fontFamily, fontSize, s)
		return err != nil || w <= maxWidth
	}
	if fits(line) {
		return []string{line}
	}
	body := strings.TrimLeft(line, " ")
	indent := line[:len(line)-len(body)]
	if !fits(indent + "W") {
		indent = ""
	}

	// Tokens: a word with the spaces before it
	var tokens []string
	for i := 0; i < len(body); {
		j := i
		for j < len(body) && body[j] == ' ' {
			j++
		}
		for j < len(body) && body[j] != ' ' {
			j++
		}
		tokens = append(tokens, body[i:j])
		i = j
	}

	var lines []string
	current := indent
	for _, token := range tokens {
		if fits(current + token) {
			current += token
			continue
		}
		if strings.TrimSpace(current) != "" {
			lines = append(lines, current)
		}
		word := strings.TrimLeft(token, " ")
		if fits(indent + word) {
			current = indent + word
			continue
		}
		// Wider than a line: broken by characters
		var parts []string
		tail := breakLongWord(word, fontFamily, fontSize, maxWidth, &parts)
		lines = append(lines, parts...)
		current = tail
	}
	if strings.TrimSpace(current) != "" || len(lines) == 0 {
		lines = append(lines, current)
	}
	return lines
}

func (t *textPrint) paginate(width, height int, dpi float64) int {
	t.layout(width, height, dpi)
	return max((len(t.lines)+t.perPage-1)/t.perPage, 1)
}

func (t *textPrint) draw(p *PrintPage) {
	t.layout(p.Width, p.Height, p.DPI)
	margin := p.MM(t.opts.Margin)
	cnv := p.Canvas
	cnv.SetFontFamily(t.opts.FontFamily)
	cnv.SetFontSize(p.FontSize(t.opts.FontSize))
	cnv.SetHAlign(HAlignLeft)
	cnv.SetVAlign(VAlignTop)

	first := p.Index * t.perPage
	for i := 0; i < t.perPage && first+i < len(t.lines); i++ {
		cnv.DrawText(margin, margin+i*t.lineHeight, p.Width-margin*2, t.lineHeight, t.lines[first+i])
	}
	if !t.opts.NoPageNumbers {
		cnv.SetHAlign(HAlignCenter)
		cnv.DrawText(margin, p.Height-margin-t.lineHeight, p.Width-margin*2, t.lineHeight, fmt.Sprintf("%d / %d", p.Index+1, p.Count))
	}
}
