// fontcheck shows the faces of the built-in and the system fonts and prints
// their widths; see doc/platform-checks.md
package main

import (
	"flag"
	"fmt"
	"image/color"
	"runtime"
	"unicode/utf8"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/ipoluianov/nui/ui"
)

const sample = "Fox — Съешь ещё, датский тариф 0O1lI 中文"

// System fonts by name: [0] has all four faces, [1] has no bold or italic
// (the system or nui synthesizes them), [2] is the interface font
func systemFonts() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"Times New Roman", "MS Gothic", ui.SystemUIFontName()}
	case "darwin":
		return []string{"Helvetica Neue", "Apple Chancery", ui.SystemUIFontName()}
	}
	return []string{"DejaVu Serif", "Droid Sans Fallback", ui.SystemUIFontName()}
}

var styles = []struct {
	name  string
	style ui.FontStyle
}{{"Regular", ui.FontRegular}, {"Bold", ui.FontBold}, {"Italic", ui.FontItalic}, {"Bold Italic", ui.FontBold | ui.FontItalic}}

func main() {
	system := flag.Bool("system", false, "draw the interface with the system font (UseSystemFont)")
	flag.Parse()
	if *system {
		if err := ui.UseSystemFont(ui.SystemUIFontName()); err != nil {
			fmt.Println("UseSystemFont:", err)
		}
	}
	ui.RegisterFont("goregular", goregular.TTF) // the regular face only

	// "" is the theme font: Noto Sans, or the system font with -system
	families := append([]string{"", ui.FontFamilySans, ui.FontFamilyMono, "goregular"}, systemFonts()...)
	families = append(families, "No Such Font 123")

	// Widths with the system's text engine (GDI on Windows) and with nui's
	// rasterizer: a real face is about as wide in both. A synthesized bold
	// is wider than the regular face by a pixel a character at 15 px.
	fmt.Printf("sample: %d characters, 15 px\n", utf8.RuneCountInString(sample))
	fmt.Printf("%-36s %8s %8s %12s\n", "face", "native", "nui", "native-regular")
	for _, family := range families {
		if family == "" {
			family = ui.ThemeFontFamily()
		}
		regular := 0
		for _, s := range styles {
			f := ui.StyledFont(family, s.style)
			ui.NativeFontRendering = true
			native, _, _ := ui.MeasureText(f, 15, sample)
			ui.NativeFontRendering = false
			own, _, _ := ui.MeasureText(f, 15, sample)
			if s.style == ui.FontRegular {
				regular = native
			}
			fmt.Printf("%-36s %8d %8d %12d\n", f, native, own, native-regular)
		}
	}
	ui.NativeFontRendering = true

	form := ui.NewForm()
	form.SetTitle("nui font check")
	form.SetSize(1200, 860)
	p := form.Panel()
	rows := []int{0, 0} // in the two columns
	for i, family := range families {
		col := i * 2 / len(families)
		name := family
		if family == "" {
			name = "theme font: " + ui.ThemeFontFamily()
		}
		title := ui.NewLabel(name)
		title.SetFontBold(true)
		title.SetFontSize(11)
		title.SetForegroundColor(color.RGBA{0x60, 0xa0, 0xff, 0xff})
		p.AddWidget(rows[col], col, title)
		rows[col]++
		for _, s := range styles {
			l := ui.NewLabel(s.name + ":  " + sample)
			l.SetFontFamily(family)
			l.SetFontBold(s.style&ui.FontBold != 0)
			l.SetFontItalic(s.style&ui.FontItalic != 0)
			l.SetFontSize(15)
			p.AddWidget(rows[col], col, l)
			rows[col]++
		}
	}
	row := max(rows[0], rows[1])
	big := ui.NewLabel("Italic 40 px: lIfI")
	big.SetFontItalic(true)
	big.SetFontSize(40)
	big.SetMinHeight(60) // a Label is as high as a control whatever its font size
	p.AddWidget(row, 0, big)
	row++
	tb := ui.NewTextBox()
	tb.SetFontFamily("jetbrainsmono:bold:italic")
	tb.SetText("Text box in JetBrains Mono Bold Italic: click between letters")
	p.AddWidget(row, 0, tb)
	row++
	p.AddVSpacer(row, 0)
	form.Show()
	form.Exec()
}
