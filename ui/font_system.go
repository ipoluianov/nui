package ui

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/image/font/sfnt"

	"github.com/ipoluianov/nui/ui/i18n"
)

// Chinese text is drawn with a font of the system: CJK fonts are too large
// to embed. The font is looked up the first time text has CJK characters,
// so applications without such text never touch the system fonts.

// SystemFallbackFonts enables the system CJK font. Set it to false before
// any CJK text is shown to use only the fonts registered by the application.
var SystemFallbackFonts = true

// FontFamilyCJK is the family the system CJK font is registered as, when found.
const FontFamilyCJK = "system-cjk"

// systemFontCandidate is a font file to try; families are the preferred
// fonts of a collection (.ttc), in order.
type systemFontCandidate struct {
	pattern  string // a path, may have filepath.Glob wildcards
	families []string
}

// The CJK variant the language wants and the one loaded: "sc" (Simplified
// Chinese), "tc" (Traditional, Taiwan), "hk" (Hong Kong), "jp", "kr". The
// same ideograph is drawn differently in each (see cjkVariantFor).
var (
	cjkMu            sync.Mutex
	cjkVariantWanted = "sc"
	cjkVariantTried  string // the variant loaded, or looked up in vain
)

// ensureFallbackFonts loads the system CJK font the first time text has
// CJK characters, and again after the language wants another variant. Call
// it before taking a face: loading resets the faces.
func ensureFallbackFonts(text string) {
	ensureScriptFonts(text)
	if !hasCJK(text) {
		return
	}
	cjkMu.Lock()
	defer cjkMu.Unlock()
	if cjkVariantTried == cjkVariantWanted {
		return
	}
	cjkVariantTried = cjkVariantWanted
	loadSystemCJKFont(cjkVariantWanted)
}

// setCJKVariant sets the variant for the language; the font is (re)loaded
// with the next CJK text.
func setCJKVariant(variant string) {
	cjkMu.Lock()
	cjkVariantWanted = variant
	cjkMu.Unlock()
	// Texts drawn with the previous variant
	clearAllRenderedTexts()
}

// cjkVariantFor returns the CJK font variant for the language: Unicode gives
// the Chinese, Japanese and Korean forms of an ideograph one code, so the
// font decides the form.
func cjkVariantFor(lang string) string {
	lang = i18n.Normalize(lang)
	switch i18n.Base(lang) {
	case "ja":
		return "jp"
	case "ko":
		return "kr"
	case "zh":
		switch lang {
		case "zh-TW":
			return "tc"
		case "zh-HK", "zh-MO":
			return "hk"
		}
	}
	return "sc"
}

// hasCJK reports whether text has characters from the CJK blocks: radicals,
// punctuation, kana, ideographs, Hangul, full-width forms.
func hasCJK(text string) bool {
	for _, r := range text {
		if r >= 0x2E80 {
			return true
		}
	}
	return false
}

func loadSystemCJKFont(variant string) {
	if !SystemFallbackFonts {
		return
	}
	for _, candidate := range systemCJKFontCandidates(variant) {
		paths, _ := filepath.Glob(candidate.pattern)
		for _, path := range paths {
			if f, err := openSystemFont(path, candidate.families); err == nil {
				setFont(FontFamilyCJK, f)
				AddFallbackFont(FontFamilyCJK)
				return
			}
		}
	}
	// Any other Unix: ask fontconfig
	lang := map[string]string{"sc": "zh-cn", "tc": "zh-tw", "hk": "zh-hk", "jp": "ja", "kr": "ko"}[variant]
	if path := fontconfigMatch(":lang=" + lang); path != "" {
		if f, err := openSystemFont(path, nil); err == nil {
			setFont(FontFamilyCJK, f)
			AddFallbackFont(FontFamilyCJK)
		}
	}
}

// systemCJKFontCandidates returns the fonts to try for the variant: its own
// fonts first, then the Simplified Chinese ones, which also have kana and
// the traditional characters (drawn in the mainland forms).
func systemCJKFontCandidates(variant string) []systemFontCandidate {
	switch runtime.GOOS {
	case "windows":
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		if os.Getenv("WINDIR") == "" {
			dir = `C:\Windows\Fonts`
		}
		own := map[string][]systemFontCandidate{
			"tc": {{filepath.Join(dir, "msjh.ttc"), []string{"Microsoft JhengHei UI", "Microsoft JhengHei"}}},
			"hk": {
				{filepath.Join(dir, "msjh.ttc"), []string{"Microsoft JhengHei UI", "Microsoft JhengHei"}},
				{filepath.Join(dir, "mingliub.ttc"), []string{"MingLiU_HKSCS"}},
			},
			"jp": {
				{filepath.Join(dir, "YuGothR.ttc"), []string{"Yu Gothic UI", "Yu Gothic"}},
				{filepath.Join(dir, "meiryo.ttc"), []string{"Meiryo UI", "Meiryo"}},
				{filepath.Join(dir, "msgothic.ttc"), []string{"MS UI Gothic", "MS Gothic"}},
			},
			"kr": {
				{filepath.Join(dir, "malgun.ttf"), []string{"Malgun Gothic"}},
				{filepath.Join(dir, "gulim.ttc"), []string{"Gulim"}},
			},
		}[variant]
		return append(own,
			systemFontCandidate{filepath.Join(dir, "msyh.ttc"), []string{"Microsoft YaHei UI", "Microsoft YaHei"}},
			systemFontCandidate{filepath.Join(dir, "msyh.ttf"), []string{"Microsoft YaHei"}},
			systemFontCandidate{filepath.Join(dir, "Deng.ttf"), []string{"DengXian"}},
			systemFontCandidate{filepath.Join(dir, "simhei.ttf"), []string{"SimHei"}},
			systemFontCandidate{filepath.Join(dir, "simsun.ttc"), []string{"SimSun"}},
		)
	case "darwin":
		pingFang := map[string]string{"tc": "PingFang TC", "hk": "PingFang HK"}[variant]
		if pingFang == "" {
			pingFang = "PingFang SC"
		}
		own := map[string][]systemFontCandidate{
			"jp": {{"/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc", []string{"Hiragino Sans"}}},
			"kr": {{"/System/Library/Fonts/AppleSDGothicNeo.ttc", []string{"Apple SD Gothic Neo"}}},
		}[variant]
		return append(own,
			systemFontCandidate{"/System/Library/Fonts/PingFang.ttc", []string{pingFang, "PingFang SC"}},
			// Since macOS 13 PingFang is a downloadable asset
			systemFontCandidate{"/System/Library/AssetsV2/com_apple_MobileAsset_Font*/*/AssetData/PingFang.ttc", []string{pingFang, "PingFang SC"}},
			systemFontCandidate{"/System/Library/Fonts/Hiragino Sans GB.ttc", []string{"Hiragino Sans GB"}},
			systemFontCandidate{"/System/Library/Fonts/STHeiti Light.ttc", []string{"Heiti SC", "STHeiti"}},
			systemFontCandidate{"/System/Library/Fonts/STHeiti Medium.ttc", []string{"Heiti SC", "STHeiti"}},
			systemFontCandidate{"/System/Library/Fonts/Supplemental/Arial Unicode.ttf", nil},
			systemFontCandidate{"/Library/Fonts/Arial Unicode.ttf", nil},
		)
	}

	// Noto Sans CJK and Source Han Sans have every variant in one family
	suffix := map[string]string{"sc": "SC", "tc": "TC", "hk": "HK", "jp": "JP", "kr": "KR"}[variant]
	notoCJK := []string{"Noto Sans CJK " + suffix, "Noto Sans CJK SC"}
	sourceHan := []string{"Source Han Sans " + suffix, "Source Han Sans SC", "Source Han Sans CN"}
	return []systemFontCandidate{
		{"/usr/share/fonts/*/NotoSansCJK-Regular.ttc", notoCJK},
		{"/usr/share/fonts/*/*/NotoSansCJK-Regular.ttc", notoCJK},
		{"/usr/share/fonts/*/NotoSansCJK" + strings.ToLower(suffix) + "-Regular.otf", notoCJK},
		{"/usr/share/fonts/*/*/NotoSansCJK" + strings.ToLower(suffix) + "-Regular.otf", notoCJK},
		{"/usr/share/fonts/*/NotoSansCJKsc-Regular.otf", notoCJK},
		{"/usr/share/fonts/*/*/NotoSansCJKsc-Regular.otf", notoCJK},
		{"/usr/share/fonts/*/SourceHanSans*-Regular.tt?", sourceHan},
		{"/usr/share/fonts/*/*/SourceHanSans*-Regular.?t?", sourceHan},
		{"/usr/share/fonts/*/*/wqy-microhei.ttc", []string{"WenQuanYi Micro Hei"}},
		{"/usr/share/fonts/*/wqy-microhei.ttc", []string{"WenQuanYi Micro Hei"}},
		{"/usr/share/fonts/*/*/wqy-zenhei.ttc", []string{"WenQuanYi Zen Hei"}},
		{"/usr/share/fonts/*/*/DroidSansFallbackFull.ttf", nil},
	}
}

// fontconfigMatch returns the file of the font fontconfig picks for the
// pattern, "" if fontconfig isn't there.
func fontconfigMatch(pattern string) string {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return ""
	}
	out, err := exec.Command("fc-match", "-f", "%{file}", pattern).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// openSystemFont opens the font file, reading it on demand rather than
// loading it: CJK fonts are tens of megabytes. From a collection it picks
// the first of the preferred families, the Regular one if there are several.
// The font must have Chinese characters.
func openSystemFont(path string, families []string) (*sfnt.Font, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	collection, err := sfnt.ParseCollectionReaderAt(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	var buf sfnt.Buffer
	fonts := make([]*sfnt.Font, 0, collection.NumFonts())
	for i := 0; i < collection.NumFonts(); i++ {
		if f, err := collection.Font(i); err == nil && hasGlyph(f, &buf, '中') {
			setFontSource(f, fontSource{path: path, index: i})
			fonts = append(fonts, f)
		}
	}

	// The file stays open while the font is in use: the application's lifetime
	for _, family := range families {
		var match *sfnt.Font
		for _, f := range fonts {
			if !strings.EqualFold(fontName(f, &buf, sfnt.NameIDTypographicFamily, sfnt.NameIDFamily), family) {
				continue
			}
			if match == nil || strings.EqualFold(fontName(f, &buf, sfnt.NameIDTypographicSubfamily, sfnt.NameIDSubfamily), "Regular") {
				match = f
			}
		}
		if match != nil {
			return match, nil
		}
	}
	if len(fonts) > 0 {
		return fonts[0], nil
	}
	file.Close()
	return nil, errors.New("no Chinese characters in " + path)
}

// openFontFile opens the font file, reading it on demand as openSystemFont,
// and picks the font of the PostScript name from a collection. exact is
// false if no font of the file has the name: the first one is returned.
func openFontFile(path, postScriptName string) (f *sfnt.Font, exact bool, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	var src io.ReaderAt = file
	if patches := cmapEncodingPatches(file); len(patches) > 0 {
		src = patchedReaderAt{file, patches}
	}
	collection, err := sfnt.ParseCollectionReaderAt(src)
	if err != nil || collection.NumFonts() == 0 {
		file.Close()
		return nil, false, fmt.Errorf("font file %s: %v", path, err)
	}
	var buf sfnt.Buffer
	for i := 0; i < collection.NumFonts(); i++ {
		if f, err := collection.Font(i); err == nil && fontName(f, &buf, sfnt.NameIDPostScript) == postScriptName {
			setFontSource(f, fontSource{path: path, index: i})
			return f, true, nil
		}
	}
	// The file stays open while the font is in use: the application's lifetime
	f, err = collection.Font(0)
	setFontSource(f, fontSource{path: path})
	return f, false, err
}

// Older fonts of Apple (Helvetica Neue) map the characters only in a cmap
// subtable of Unicode 1.0 or 1.1 (platform 0, encoding 0 or 1), which sfnt
// rejects: "unsupported cmap encodings". Its formats are the ones of the
// Unicode BMP encoding (3), so the file is read with those encodings given
// as 3 - a few bytes changed as they are read, the file isn't loaded whole.

// patchedReaderAt reads r with the bytes at the offsets of patches replaced
type patchedReaderAt struct {
	r       io.ReaderAt
	patches map[int64]byte
}

func (p patchedReaderAt) ReadAt(b []byte, off int64) (int, error) {
	n, err := p.r.ReadAt(b, off)
	for at, v := range p.patches {
		if at >= off && at < off+int64(n) {
			b[at-off] = v
		}
	}
	return n, err
}

// cmapEncodingPatches returns the offsets of the low bytes of the old
// Unicode encodings in the cmap tables of the font file (or collection),
// with 3 for each
func cmapEncodingPatches(r io.ReaderAt) map[int64]byte {
	u16 := func(off int64) (uint16, bool) {
		var b [2]byte
		_, err := r.ReadAt(b[:], off)
		return binary.BigEndian.Uint16(b[:]), err == nil
	}
	u32 := func(off int64) (uint32, bool) {
		var b [4]byte
		_, err := r.ReadAt(b[:], off)
		return binary.BigEndian.Uint32(b[:]), err == nil
	}
	fonts := []int64{0}
	if tag, ok := u32(0); ok && tag == 0x74746366 { // "ttcf"
		n, _ := u32(8)
		fonts = nil
		for i := int64(0); i < int64(min(n, 256)); i++ {
			if off, ok := u32(12 + 4*i); ok {
				fonts = append(fonts, int64(off))
			}
		}
	}
	patches := map[int64]byte{}
	for _, font := range fonts {
		tables, _ := u16(font + 4)
		for i := int64(0); i < int64(tables); i++ {
			record := font + 12 + 16*i
			if tag, _ := u32(record); tag != 0x636d6170 { // "cmap"
				continue
			}
			cmap, _ := u32(record + 8)
			subtables, _ := u16(int64(cmap) + 2)
			for j := int64(0); j < int64(subtables); j++ {
				enc := int64(cmap) + 4 + 8*j
				pid, _ := u16(enc)
				psid, _ := u16(enc + 2)
				if pid == 0 && psid <= 2 {
					patches[enc+3] = 3
				}
			}
		}
	}
	return patches
}

func hasGlyph(f *sfnt.Font, buf *sfnt.Buffer, r rune) bool {
	index, err := f.GlyphIndex(buf, r)
	return err == nil && index != 0
}

// fontName returns the first of the names the font has.
func fontName(f *sfnt.Font, buf *sfnt.Buffer, ids ...sfnt.NameID) string {
	for _, id := range ids {
		if name, err := f.Name(buf, id); err == nil && name != "" {
			return name
		}
	}
	return ""
}
