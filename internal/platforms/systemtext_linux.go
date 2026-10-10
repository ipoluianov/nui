//go:build linux

package platforms

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"

	"github.com/ebitengine/purego"
)

// System fonts on Linux are drawn by FreeType, the engine of the desktop
// toolkits, with the user's fontconfig settings: the subpixel order of the
// screen (rgba), the hinting style and the LCD filter. fontconfig also finds
// the font file of a family and the fonts for the characters a font lacks
// (fc-match, the same way the font files of the CJK fallback are found).
//
// libfreetype is loaded at run time with purego (no cgo). The struct fields
// read below are at the offsets of the FreeType public headers on the 64-bit
// Linux targets (amd64, arm64: LP64).

var ft struct {
	once sync.Once
	err  error
	lib  unsafe.Pointer

	initFreeType    func(lib *unsafe.Pointer) int32
	newFace         func(lib unsafe.Pointer, path string, index int64, face *unsafe.Pointer) int32
	setCharSize     func(face unsafe.Pointer, width, height int64, hres, vres uint32) int32
	getCharIndex    func(face unsafe.Pointer, charcode uint64) uint32
	loadGlyph       func(face unsafe.Pointer, index uint32, flags int32) int32
	renderGlyph     func(slot unsafe.Pointer, mode uint32) int32
	getKerning      func(face unsafe.Pointer, left, right, mode uint32, kerning *[2]int64) int32
	libSetLcdFilter func(lib unsafe.Pointer, filter uint32) int32
	doneFace        func(face unsafe.Pointer) int32
}

// FT_FaceRec, FT_SizeRec and FT_GlyphSlotRec field offsets
const (
	ftFaceFlags     = 16
	ftFaceGlyph     = 152
	ftFaceSize      = 160
	ftSizeAscender  = 48
	ftSizeDescender = 56
	ftSlotAdvanceX  = 128
	ftSlotBitmap    = 152 // FT_Bitmap: rows u32, width u32, pitch i32, buffer ptr @16, num_grays u16 @24, pixel_mode u8 @26
	ftSlotLeft      = 192
	ftSlotTop       = 196

	ftFaceFlagKerning = 1 << 6

	ftLoadNoHinting   = 1 << 1
	ftLoadTargetLight = 1 << 16 // FT_LOAD_TARGET_(FT_RENDER_MODE_LIGHT)
	ftLoadTargetMono  = 2 << 16
	ftLoadTargetLCD   = 3 << 16

	ftRenderNormal = 0
	ftRenderMono   = 2
	ftRenderLCD    = 3

	ftPixelMono = 1
	ftPixelGray = 2
	ftPixelLCD  = 5
)

func loadFreeType() error {
	ft.once.Do(func() {
		h, err := dlopenFirst("libfreetype.so.6", "libfreetype.so")
		if err != nil {
			ft.err = fmt.Errorf("%w: %v", ErrSystemTextNotSupported, err)
			return
		}
		purego.RegisterLibFunc(&ft.initFreeType, h, "FT_Init_FreeType")
		purego.RegisterLibFunc(&ft.newFace, h, "FT_New_Face")
		purego.RegisterLibFunc(&ft.setCharSize, h, "FT_Set_Char_Size")
		purego.RegisterLibFunc(&ft.getCharIndex, h, "FT_Get_Char_Index")
		purego.RegisterLibFunc(&ft.loadGlyph, h, "FT_Load_Glyph")
		purego.RegisterLibFunc(&ft.renderGlyph, h, "FT_Render_Glyph")
		purego.RegisterLibFunc(&ft.getKerning, h, "FT_Get_Kerning")
		purego.RegisterLibFunc(&ft.libSetLcdFilter, h, "FT_Library_SetLcdFilter")
		purego.RegisterLibFunc(&ft.doneFace, h, "FT_Done_Face")
		if ft.initFreeType(&ft.lib) != 0 {
			ft.err = fmt.Errorf("%w: FT_Init_FreeType failed", ErrSystemTextNotSupported)
		}
	})
	return ft.err
}

// fcFont is what fontconfig says about a font: its file and the rendering
// settings the user chose for it
type fcFont struct {
	file      string
	index     int64
	families  []string
	rgba      int // 1 rgb, 2 bgr, 3 vrgb, 4 vbgr, 5 none
	hintStyle int // 0 none, 1 slight, 2 medium, 3 full
	antialias bool
	hinting   bool
	lcdFilter int // 0 none, 1 default, 2 light, 3 legacy
}

// fcMatch asks fontconfig for the font of the pattern
func fcMatch(pattern string) (fcFont, bool) {
	out, err := exec.Command("fc-match", "-f",
		"%{file}\t%{index}\t%{family}\t%{rgba}\t%{hintstyle}\t%{antialias}\t%{hinting}\t%{lcdfilter}", pattern).Output()
	if err != nil {
		return fcFont{}, false
	}
	f := strings.Split(string(out), "\t")
	if len(f) < 8 || f[0] == "" {
		return fcFont{}, false
	}
	atoi := func(s string, def int) int {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return n
		}
		return def
	}
	return fcFont{
		file:      f[0],
		index:     int64(atoi(f[1], 0)),
		families:  strings.Split(f[2], ","),
		rgba:      atoi(f[3], 0),
		hintStyle: atoi(f[4], 1),
		antialias: f[5] != "False",
		hinting:   f[6] != "False",
		lcdFilter: atoi(f[7], 1),
	}, true
}

func systemUIFontName() string {
	if f, ok := fcMatch("sans-serif"); ok && len(f.families) > 0 {
		return f.families[0]
	}
	return "Sans"
}

// ftGlyph is a rendered character
type ftGlyph struct {
	advance   int64 // 26.6
	index     uint32
	face      *ftFace
	left, top int
	w, h      int // in pixels
	stride    int
	mask      []byte
	subpixel  bool
}

// ftFace is one FreeType face at a size, with the settings to render it
type ftFace struct {
	face       unsafe.Pointer
	loadFlags  int32
	renderMode uint32
	bgr        bool
}

type linuxFont struct {
	mu      sync.Mutex
	name    string
	size    float64
	primary *ftFace
	ascent  int
	descent int
	glyphs  map[rune]*ftGlyph
	// fallback faces by font file, for the characters the primary font lacks
	fallbacks map[string]*ftFace
}

func openSystemFont(name string, pixelSize float64) (SystemFont, error) {
	if err := loadFreeType(); err != nil {
		return nil, err
	}
	fc, ok := fcMatch(name)
	if !ok {
		return nil, fmt.Errorf("%w: fc-match is not available", ErrSystemTextNotSupported)
	}
	found := false
	for _, family := range fc.families {
		if strings.EqualFold(strings.TrimSpace(family), name) {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("%w: %q", ErrSystemFontNotFound, name)
	}
	primary, err := newFTFace(fc, pixelSize)
	if err != nil {
		return nil, err
	}
	size := *(*unsafe.Pointer)(unsafe.Add(primary.face, ftFaceSize))
	ascender := *(*int64)(unsafe.Add(size, ftSizeAscender))
	descender := *(*int64)(unsafe.Add(size, ftSizeDescender))
	return &linuxFont{
		name:      name,
		size:      pixelSize,
		primary:   primary,
		ascent:    int((ascender + 63) >> 6),
		descent:   int((-descender + 63) >> 6),
		glyphs:    map[rune]*ftGlyph{},
		fallbacks: map[string]*ftFace{},
	}, nil
}

// ftLibMu guards the FreeType library shared by all the fonts: FreeType
// requires a lock around FT_New_Face and FT_Done_Face on a library used by
// several threads, and FT_Library_SetLcdFilter changes the library itself.
// (The fonts are safe for concurrent use; the glyphs of a font are guarded
// by its own mutex.)
var ftLibMu sync.Mutex

func newFTFace(fc fcFont, pixelSize float64) (*ftFace, error) {
	ftLibMu.Lock()
	defer ftLibMu.Unlock()
	var face unsafe.Pointer
	if e := ft.newFace(ft.lib, fc.file, fc.index, &face); e != 0 {
		return nil, fmt.Errorf("nui: FreeType can't open %s (error %d)", fc.file, e)
	}
	if e := ft.setCharSize(face, 0, int64(math.Round(pixelSize*64)), 72, 72); e != 0 {
		ft.doneFace(face)
		return nil, fmt.Errorf("nui: FreeType can't size %s (error %d)", fc.file, e)
	}

	f := &ftFace{face: face}
	subpixel := fc.antialias && (fc.rgba == 1 || fc.rgba == 2)
	switch {
	case !fc.antialias:
		f.loadFlags, f.renderMode = ftLoadTargetMono, ftRenderMono
	case subpixel:
		f.renderMode = ftRenderLCD
		f.bgr = fc.rgba == 2
		filter := map[int]uint32{0: 0, 1: 1, 2: 2, 3: 16}[fc.lcdFilter]
		// An error only means FreeType draws subpixels without a filter
		// (its own Harmony method): fine
		ft.libSetLcdFilter(ft.lib, filter)
	default:
		f.renderMode = ftRenderNormal
	}
	switch {
	case !fc.hinting || fc.hintStyle == 0:
		f.loadFlags |= ftLoadNoHinting
	case fc.hintStyle == 1:
		f.loadFlags |= ftLoadTargetLight
	case subpixel:
		f.loadFlags |= ftLoadTargetLCD
	}
	return f, nil
}

func (f *linuxFont) Metrics() (int, int) {
	return f.ascent, f.descent
}

// glyph renders the character, from the primary font or a fallback
func (f *linuxFont) glyph(r rune) *ftGlyph {
	if g, ok := f.glyphs[r]; ok {
		return g
	}
	face := f.primary
	index := ft.getCharIndex(face.face, uint64(r))
	if index == 0 && r > ' ' {
		if fb := f.fallbackFace(r); fb != nil {
			if i := ft.getCharIndex(fb.face, uint64(r)); i != 0 {
				face, index = fb, i
			}
		}
	}
	g := &ftGlyph{face: face, index: index}
	f.glyphs[r] = g
	if ft.loadGlyph(face.face, index, face.loadFlags) != 0 {
		return g
	}
	slot := *(*unsafe.Pointer)(unsafe.Add(face.face, ftFaceGlyph))
	g.advance = *(*int64)(unsafe.Add(slot, ftSlotAdvanceX))
	if ft.renderGlyph(slot, face.renderMode) != 0 {
		return g
	}
	bm := unsafe.Add(slot, ftSlotBitmap)
	rows := int(*(*uint32)(bm))
	width := int(*(*uint32)(unsafe.Add(bm, 4)))
	pitch := int(*(*int32)(unsafe.Add(bm, 8)))
	buffer := *(*unsafe.Pointer)(unsafe.Add(bm, 16))
	pixelMode := *(*uint8)(unsafe.Add(bm, 26))
	g.left = int(*(*int32)(unsafe.Add(slot, ftSlotLeft)))
	g.top = int(*(*int32)(unsafe.Add(slot, ftSlotTop)))
	if rows == 0 || width == 0 || buffer == nil {
		return g
	}
	if pitch < 0 {
		pitch = -pitch
	}
	src := unsafe.Slice((*byte)(buffer), rows*pitch)

	g.h = rows
	switch pixelMode {
	case ftPixelLCD:
		g.w = width / 3
		g.stride = g.w * 3
		g.subpixel = true
		g.mask = make([]byte, rows*g.stride)
		for y := 0; y < rows; y++ {
			row := g.mask[y*g.stride : (y+1)*g.stride]
			copy(row, src[y*pitch:y*pitch+g.stride])
			if face.bgr {
				for x := 0; x < g.w; x++ {
					row[x*3], row[x*3+2] = row[x*3+2], row[x*3]
				}
			}
		}
	case ftPixelGray:
		g.w, g.stride = width, width
		g.mask = make([]byte, rows*width)
		for y := 0; y < rows; y++ {
			copy(g.mask[y*width:(y+1)*width], src[y*pitch:])
		}
	case ftPixelMono:
		g.w, g.stride = width, width
		g.mask = make([]byte, rows*width)
		for y := 0; y < rows; y++ {
			for x := 0; x < width; x++ {
				if src[y*pitch+x/8]&(0x80>>(x%8)) != 0 {
					g.mask[y*width+x] = 255
				}
			}
		}
	default:
		g.h = 0 // color emoji and the like are not drawn
	}
	return g
}

// fallbackFace asks fontconfig for a font that has the character
func (f *linuxFont) fallbackFace(r rune) *ftFace {
	fc, ok := fcMatch(fmt.Sprintf("%s:charset=%x", f.name, r))
	if !ok {
		return nil
	}
	key := fc.file + "#" + strconv.FormatInt(fc.index, 10)
	if face, ok := f.fallbacks[key]; ok {
		return face
	}
	face, err := newFTFace(fc, f.size)
	if err != nil {
		face = nil
	}
	f.fallbacks[key] = face
	return face
}

// layout returns the glyphs of text and the pen x (26.6) of each one, and
// the width: the advances plus the kerning within a font
func (f *linuxFont) layout(text string) ([]*ftGlyph, []int64, int64) {
	glyphs := make([]*ftGlyph, 0, utf8.RuneCountInString(text))
	pens := make([]int64, 0, cap(glyphs))
	var pen int64
	var prev *ftGlyph
	for _, r := range text {
		g := f.glyph(r)
		if prev != nil && prev.face == g.face {
			if *(*int64)(unsafe.Add(g.face.face, ftFaceFlags))&ftFaceFlagKerning != 0 {
				var k [2]int64
				if ft.getKerning(g.face.face, prev.index, g.index, 0, &k) == 0 {
					pen += k[0]
				}
			}
		}
		glyphs = append(glyphs, g)
		pens = append(pens, pen)
		pen += g.advance
		prev = g
	}
	return glyphs, pens, pen
}

func (f *linuxFont) Positions(text string) []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, pens, width := f.layout(text)
	positions := make([]int, len(pens)+1)
	for i, p := range pens {
		positions[i] = int((p + 32) >> 6)
	}
	positions[len(pens)] = int((width + 32) >> 6)
	return positions
}

func (f *linuxFont) Draw(dst *image.RGBA, text string, col color.RGBA, x, y int, clip image.Rectangle) {
	f.mu.Lock()
	defer f.mu.Unlock()
	glyphs, pens, _ := f.layout(text)
	baseline := y + f.ascent
	for i, g := range glyphs {
		if g.h == 0 {
			continue
		}
		gx := x + int((pens[i]+32)>>6) + g.left
		gy := baseline - g.top
		blendMask(dst, col, g.mask, g.w, g.h, g.stride, g.subpixel, gx, gy, clip)
	}
}
