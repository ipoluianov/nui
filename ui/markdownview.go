package ui

import (
	"image/color"
	"strings"
	"unicode/utf8"
)

// MarkdownView shows Markdown text, wrapped to its width and scrolled:
// headings, paragraphs, bulleted and numbered lists, quotes, code blocks,
// horizontal rules; **bold**, *italic*, `code` and [links](url) in the
// text. A click on a link emits OnLinkClicked; without a handler the link
// opens in the browser.
//
//	help := ui.NewMarkdownView()
//	help.SetMarkdown("# Help\n\nPress **Ctrl+S** to save. See [the site](https://example.com).")
type MarkdownView struct {
	Widget

	// OnLinkClicked: a link was clicked; its URL
	OnLinkClicked Signal[string]

	markdown string
	blocks   []mdBlock
	lines    []mdLine // the layout at layoutWidth
	width    int      // the width the lines are laid out for
	hover    string   // the link under the mouse
}

type mdBlockKind int

const (
	mdParagraph mdBlockKind = iota
	mdHeading
	mdBullet
	mdNumbered
	mdQuote
	mdCode
	mdRule
)

type mdBlock struct {
	kind   mdBlockKind
	level  int    // of a heading: 1..3
	marker string // of a list item: "•", "1."
	runs   []mdRun
	code   []string // the lines of a code block
}

// mdRun is a part of a text in one style
type mdRun struct {
	text   string
	bold   bool
	italic bool
	code   bool
	link   string
}

// mdLine is a laid out line: its parts at their places
type mdLine struct {
	y, height int
	parts     []mdPart
	codeBlock bool // a line of a code block: its background
	quote     bool // a line of a quote: its bar
	rule      bool
}

type mdPart struct {
	x, width int
	run      mdRun
	family   string
	size     float64
}

// NewMarkdownView makes an empty view
func NewMarkdownView() *MarkdownView {
	var c MarkdownView
	c.InitWidget()
	c.SetTypeName("MarkdownView")
	c.SetAllowScroll(false, true)
	c.SetXExpandable(true)
	c.SetYExpandable(true)
	c.SetMinSize(100, 60)
	c.SetOnPaint(c.draw)
	c.SetOnMouseMove(c.mouseMove)
	c.SetOnMouseDown(c.mouseDown)
	c.OnResize.Connect(func(Size) { c.relayout() })
	return &c
}

// SetMarkdown shows the text
func (c *MarkdownView) SetMarkdown(markdown string) {
	c.markdown = markdown
	c.blocks = parseMarkdown(markdown)
	c.width = 0
	c.relayout()
	c.setScrollY(0)
	c.form.Update()
}

// Markdown returns the text shown
func (c *MarkdownView) Markdown() string {
	return c.markdown
}

// Text returns the text shown without the Markdown marks, a line a line of
// the layout's paragraphs
func (c *MarkdownView) Text() string {
	var b strings.Builder
	for i, block := range c.blocks {
		if i > 0 {
			b.WriteByte('\n')
		}
		if block.marker != "" {
			b.WriteString(block.marker + " ")
		}
		for _, run := range block.runs {
			b.WriteString(run.text)
		}
		b.WriteString(strings.Join(block.code, "\n"))
	}
	return b.String()
}

// ---------------------------------------------------------------- Parsing

// parseMarkdown splits the text into blocks
func parseMarkdown(text string) []mdBlock {
	var blocks []mdBlock
	var paragraph []string
	flush := func() {
		if len(paragraph) > 0 {
			blocks = append(blocks, mdBlock{kind: mdParagraph, runs: parseInline(strings.Join(paragraph, " "))})
			paragraph = nil
		}
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```"):
			flush()
			var code []string
			for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```"); i++ {
				code = append(code, strings.ReplaceAll(lines[i], "\t", "    "))
			}
			blocks = append(blocks, mdBlock{kind: mdCode, code: code})
		case trimmed == "":
			flush()
		case trimmed == "---" || trimmed == "***" || trimmed == "___":
			flush()
			blocks = append(blocks, mdBlock{kind: mdRule})
		case strings.HasPrefix(trimmed, "#"):
			flush()
			level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			blocks = append(blocks, mdBlock{kind: mdHeading, level: min(level, 3),
				runs: parseInline(strings.TrimSpace(trimmed[level:]))})
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ "):
			flush()
			blocks = append(blocks, mdBlock{kind: mdBullet, marker: "•", runs: parseInline(trimmed[2:])})
		case numberedItem(trimmed) > 0:
			flush()
			n := numberedItem(trimmed)
			blocks = append(blocks, mdBlock{kind: mdNumbered, marker: trimmed[:n], runs: parseInline(strings.TrimSpace(trimmed[n:]))})
		case strings.HasPrefix(trimmed, ">"):
			flush()
			blocks = append(blocks, mdBlock{kind: mdQuote, runs: parseInline(strings.TrimSpace(trimmed[1:]))})
		default:
			paragraph = append(paragraph, trimmed)
		}
	}
	flush()
	return blocks
}

// numberedItem returns the length of the "12." of a numbered list item, 0
// if the line isn't one
func numberedItem(line string) int {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(line) && line[i] == '.' && line[i+1] == ' ' {
		return i + 1
	}
	return 0
}

// parseInline splits a text into runs of **bold**, *italic*, `code` and
// [links](url)
func parseInline(text string) []mdRun {
	var runs []mdRun
	var cur strings.Builder
	bold, italic := false, false
	emit := func() {
		if cur.Len() > 0 {
			runs = append(runs, mdRun{text: cur.String(), bold: bold, italic: italic})
			cur.Reset()
		}
	}
	for i := 0; i < len(text); {
		switch {
		case strings.HasPrefix(text[i:], "**") || strings.HasPrefix(text[i:], "__"):
			emit()
			bold = !bold
			i += 2
		case text[i] == '*' || text[i] == '_' && (i == 0 || text[i-1] == ' ' || italic):
			emit()
			italic = !italic
			i++
		case text[i] == '`':
			if end := strings.IndexByte(text[i+1:], '`'); end >= 0 {
				emit()
				runs = append(runs, mdRun{text: text[i+1 : i+1+end], code: true})
				i += end + 2
				continue
			}
			cur.WriteByte(text[i])
			i++
		case text[i] == '[':
			closeText := strings.Index(text[i:], "](")
			closeURL := -1
			if closeText > 0 {
				closeURL = strings.IndexByte(text[i+closeText:], ')')
			}
			if closeText > 0 && closeURL > 0 {
				emit()
				url := text[i+closeText+2 : i+closeText+closeURL]
				runs = append(runs, mdRun{text: text[i+1 : i+closeText], bold: bold, italic: italic, link: url})
				i += closeText + closeURL + 1
				continue
			}
			cur.WriteByte(text[i])
			i++
		default:
			_, size := utf8.DecodeRuneInString(text[i:])
			cur.WriteString(text[i : i+size])
			i += size
		}
	}
	emit()
	return runs
}

// ---------------------------------------------------------------- Layout

// mdIndent is the indent of the list items and the quotes
const mdIndent = 22

func (c *MarkdownView) relayout() {
	vw, _ := c.viewportSize()
	width := vw - 2*themeTextInset
	if width <= 0 || width == c.width {
		return
	}
	c.width = width
	c.lines = c.lines[:0]
	base := c.FontSize()
	family := c.FontFamily()
	_, lineHeight, _ := MeasureText(family, base, "Ag")
	y := themeTextInset
	for i, block := range c.blocks {
		if i > 0 {
			y += lineHeight / 2 // between the blocks
		}
		switch block.kind {
		case mdRule:
			c.lines = append(c.lines, mdLine{y: y, height: lineHeight / 2, rule: true})
			y += lineHeight / 2
		case mdCode:
			for _, code := range block.code {
				part := mdPart{x: themeTextInset * 2, run: mdRun{text: code, code: true}, family: FontFamilyMono, size: base}
				c.lines = append(c.lines, mdLine{y: y, height: lineHeight, parts: []mdPart{part}, codeBlock: true})
				y += lineHeight
			}
		default:
			size, indent := base, 0
			switch block.kind {
			case mdHeading:
				size = base * []float64{0, 1.6, 1.35, 1.15}[block.level]
			case mdBullet, mdNumbered, mdQuote:
				indent = mdIndent
			}
			_, h, _ := MeasureText(family, size, "Ag")
			lines := c.wrap(block, family, size, indent, width)
			for k, parts := range lines {
				line := mdLine{y: y, height: h, parts: parts, quote: block.kind == mdQuote}
				if k == 0 && block.marker != "" {
					line.parts = append([]mdPart{{x: themeTextInset + 4, run: mdRun{text: block.marker}, family: family, size: size}}, line.parts...)
				}
				c.lines = append(c.lines, line)
				y += h
			}
		}
	}
	c.innerWidth = vw
	c.innerHeight = y + themeTextInset
	c.checkScrolls()
}

// wrap lays the runs of the block out in lines of the width
func (c *MarkdownView) wrap(block mdBlock, family string, size float64, indent, width int) [][]mdPart {
	var lines [][]mdPart
	var line []mdPart
	left := themeTextInset + indent
	x := left
	for _, run := range block.runs {
		runFamily := family
		if run.code {
			runFamily = FontFamilyMono
		} else if run.bold || run.italic || block.kind == mdHeading {
			runFamily = StyledFont(family, fontStyleOf(run.bold || block.kind == mdHeading, run.italic || block.kind == mdQuote))
		} else if block.kind == mdQuote {
			runFamily = StyledFont(family, FontItalic)
		}
		// Words with the space after each
		for _, word := range splitWords(run.text) {
			w, _, _ := MeasureText(runFamily, size, word)
			if x+w > left+width-indent && x > left && strings.TrimSpace(word) != "" {
				lines = append(lines, line)
				line, x = nil, left
				word = strings.TrimLeft(word, " ")
				w, _, _ = MeasureText(runFamily, size, word)
			}
			wordRun := run
			wordRun.text = word
			line = append(line, mdPart{x: x, width: w, run: wordRun, family: runFamily, size: size})
			x += w
		}
	}
	if len(line) > 0 || len(lines) == 0 {
		lines = append(lines, line)
	}
	return lines
}

// splitWords splits the text after each space, keeping the spaces
func splitWords(text string) []string {
	var words []string
	for len(text) > 0 {
		i := strings.IndexByte(text, ' ')
		if i < 0 {
			words = append(words, text)
			break
		}
		words = append(words, text[:i+1])
		text = text[i+1:]
	}
	return words
}

// ---------------------------------------------------------------- Painting

func (c *MarkdownView) draw(cnv *Canvas) {
	if c.width == 0 {
		c.relayout()
	}
	p := CurrentPalette()
	vw, vh := c.viewportSize()
	codeBack := MixColors(p.Base, p.Text, 0.06)
	for _, line := range c.lines {
		if line.y+line.height < c.scrollY || line.y > c.scrollY+vh {
			continue
		}
		switch {
		case line.rule:
			cnv.FillRect(themeTextInset, line.y+line.height/2, vw-2*themeTextInset, 1, p.Divider)
			continue
		case line.codeBlock:
			cnv.FillRect(themeTextInset, line.y, vw-2*themeTextInset, line.height, codeBack)
		case line.quote:
			cnv.FillRect(themeTextInset+mdIndent/2-2, line.y, 3, line.height, p.Divider)
		}
		for _, part := range line.parts {
			col := color.Color(p.Text)
			if part.run.link != "" {
				col = p.Link
			}
			if part.run.code && !line.codeBlock {
				cnv.FillRect(part.x, line.y+1, part.width, line.height-2, codeBack)
			}
			cnv.SetColor(col)
			cnv.SetFontFamily(part.family)
			cnv.SetFontSize(part.size)
			cnv.SetHAlign(HAlignLeft)
			cnv.SetVAlign(VAlignCenter)
			cnv.SetUnderline(part.run.link != "" && part.run.link == c.hover)
			cnv.DrawText(part.x, line.y, MaxInt/4, line.height, part.run.text)
			cnv.SetUnderline(false)
		}
	}
}

// ---------------------------------------------------------------- Links

// LinkAt returns the URL of the link at the point of the widget, "" if
// none: e.g. for a tooltip
func (c *MarkdownView) LinkAt(x, y int) string {
	return c.linkAt(x+c.scrollX, y+c.scrollY)
}

// linkAt returns the URL of the link at the point of the content, "" if
// none
func (c *MarkdownView) linkAt(x, y int) string {
	for _, line := range c.lines {
		if y < line.y || y >= line.y+line.height {
			continue
		}
		for _, part := range line.parts {
			if part.run.link != "" && x >= part.x && x < part.x+part.width {
				return part.run.link
			}
		}
	}
	return ""
}

func (c *MarkdownView) mouseMove(x, y int, mods KeyModifiers) bool {
	link := c.linkAt(x, y)
	if link != c.hover {
		c.hover = link
		if link != "" {
			c.SetMouseCursor(MouseCursorPointer)
		} else {
			c.SetMouseCursor(MouseCursorArrow)
		}
		c.form.Update()
	}
	return true
}

func (c *MarkdownView) mouseDown(button MouseButton, x, y int, mods KeyModifiers) bool {
	if button != MouseButtonLeft {
		return false
	}
	link := c.linkAt(x, y)
	if link == "" {
		return false
	}
	if c.OnLinkClicked.Connected() {
		c.OnLinkClicked.Emit(link)
	} else {
		OpenURL(link)
	}
	return true
}
