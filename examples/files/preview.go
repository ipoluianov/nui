package files

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // image formats for image.Decode
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/ipoluianov/nui/ui"
)

const (
	maxImageFile   = 20 << 20 // bigger images aren't previewed
	maxImagePixels = 50e6
	maxTextPreview = 64 << 10 // the first 64 KB of a text file are shown
)

// previewPane shows the details of the selected entry and, for an image or
// a text file, its content.
type previewPane struct {
	panel    *ui.Panel
	name     *ui.Label
	size     *ui.Label
	modified *ui.Label
	mode     *ui.Label
	note     *ui.Label
	image    *ui.ImageBox
	text     *ui.TextBox
	spacer   *ui.VSpacer

	shown string // the path shown
}

func (p *previewPane) init() ui.Widgeter {
	p.panel = ui.NewPanel()
	p.panel.SetMinWidth(200)
	p.name = p.panel.AddLabel(0, 0, "")
	p.name.SetUnderline(true)

	details := p.panel.AddPanel(1, 0)
	details.SetPanelPadding(0)
	row := func(r int, title string) *ui.Label {
		details.AddLabel(r, 0, title)
		return details.AddLabel(r, 1, "")
	}
	p.size = row(0, "Size:")
	p.modified = row(1, "Modified:")
	p.mode = row(2, "Permissions:")
	details.AddHSpacer(0, 2)

	p.note = p.panel.AddLabel(2, 0, "")

	p.image = ui.NewImageBox()
	p.image.SetScaling(ui.ImageBoxScaleAdjustImageKeepAspectRatio)
	p.image.SetXExpandable(true)
	p.image.SetYExpandable(true)
	p.panel.AddWidget(3, 0, p.image)

	p.text = ui.NewTextBox()
	p.text.SetMultiline(true)
	p.text.SetReadOnly(true)
	p.text.SetYExpandable(true)
	p.panel.AddWidget(4, 0, p.text)

	// Keeps the details at the top when there is no content
	p.spacer = p.panel.AddVSpacer(5, 0)
	p.clear()
	return p.panel
}

func (p *previewPane) clear() {
	p.shown = ""
	p.name.SetText("No selection")
	p.size.SetText("")
	p.modified.SetText("")
	p.mode.SetText("")
	p.setContent("Select a file to see its preview", nil, "")
}

// setContent shows a note and either an image, a text or nothing.
func (p *previewPane) setContent(note string, img image.Image, text string) {
	p.note.SetText(note)
	p.note.SetVisible(note != "")
	p.image.SetImage(img)
	p.image.SetVisible(img != nil)
	p.text.SetText(text)
	p.text.ScrollToBegin()
	p.text.SetVisible(text != "")
	p.spacer.SetVisible(img == nil && text == "")
}

func (p *previewPane) focus() {
	if p.text.IsVisible() {
		p.text.Focus()
	}
}

func (p *previewPane) show(en entry) {
	if p.shown == en.path {
		return
	}
	p.shown = en.path
	p.name.SetText(en.name)
	if en.dir {
		p.size.SetText("Folder")
	} else {
		p.size.SetText(formatSize(en.size))
	}
	p.modified.SetText(en.mod.Format("2006-01-02 15:04:05"))
	p.mode.SetText(en.mode.String())

	switch {
	case en.dir:
		p.setContent("", nil, "")
	case !en.mode.IsRegular():
		p.setContent("No preview for this kind of file", nil, "")
	case isImage(en.name):
		img, err := loadImage(en.path, en.size)
		if err != nil {
			p.setContent("No preview: "+err.Error(), nil, "")
			return
		}
		b := img.Bounds()
		p.setContent(fmt.Sprintf("%d x %d pixels", b.Dx(), b.Dy()), img, "")
	default:
		text, truncated, err := loadText(en.path)
		switch {
		case err != nil:
			p.setContent("No preview: "+err.Error(), nil, "")
		case en.size == 0:
			p.setContent("Empty file", nil, "")
		case text == "":
			p.setContent("Binary file", nil, "")
		case truncated:
			p.setContent(fmt.Sprintf("The first %s of the file", formatSize(maxTextPreview)), nil, text)
		default:
			p.setContent("", nil, text)
		}
	}
}

func isImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	}
	return false
}

// loadImage decodes an image file that isn't too big.
func loadImage(path string, size int64) (image.Image, error) {
	if size > maxImageFile {
		return nil, fmt.Errorf("the file is larger than %s", formatSize(maxImageFile))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if cfg.Width*cfg.Height > maxImagePixels {
		return nil, fmt.Errorf("the image is too large (%d x %d)", cfg.Width, cfg.Height)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// loadText reads the beginning of a file; a binary file gives "".
func loadText(path string) (text string, truncated bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxTextPreview+1))
	if err != nil {
		return "", false, err
	}
	if len(data) > maxTextPreview {
		data, truncated = data[:maxTextPreview], true
		// Don't cut a character in the middle
		for i := 0; i < utf8.UTFMax && len(data) > 0 && !utf8.Valid(data); i++ {
			data = data[:len(data)-1]
		}
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return "", false, nil
	}
	text = strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.ReplaceAll(text, "\t", "    "), truncated, nil
}
