package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"golang.org/x/image/font/sfnt"
)

// Text of the scripts the built-in fonts lack - Arabic, Hebrew, the Indic
// scripts, Thai, Armenian, Georgian, Ethiopic... - is drawn with a font of
// the system that has them, looked up the first time such a character is
// drawn and added to the fallback fonts (CJK has its own, see
// ensureFallbackFonts). With SystemFallbackFonts false, none is looked up.

var (
	scriptFontsMu sync.Mutex
	// scriptBlocks are the blocks of 128 characters looked up already
	scriptBlocks = map[rune]bool{}
	// scriptFiles are the files added already
	scriptFiles = map[string]bool{}
)

// ensureScriptFonts adds a font of the system for each character of the
// text that no font of nui has
func ensureScriptFonts(text string) {
	for _, r := range text {
		// Below Hebrew the built-in fonts have the scripts; CJK is apart
		if r < 0x0590 || isCJKRune(r) || (r >= 0xFE00 && r <= 0xFE0F) || r == 0x200C || r == 0x200D {
			continue
		}
		block := r >> 7
		scriptFontsMu.Lock()
		tried := scriptBlocks[block]
		scriptBlocks[block] = true
		scriptFontsMu.Unlock()
		if tried || !SystemFallbackFonts || anyFontHas(r) {
			continue
		}
		addScriptFont(r)
	}
}

// isCJKRune tells whether the CJK font draws the character
func isCJKRune(r rune) bool {
	return (r >= 0x2E80 && r <= 0x9FFF) || (r >= 0xAC00 && r <= 0xD7AF) || (r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFF00 && r <= 0xFFEF) || (r >= 0x20000 && r <= 0x3FFFF)
}

// anyFontHas tells whether the default font or a fallback has the character
func anyFontHas(r rune) bool {
	fontsMu.RLock()
	defer fontsMu.RUnlock()
	var buf sfnt.Buffer
	if f, _, ok := lookupFont(defaultFontKey, FontRegular); ok && hasGlyph(f, &buf, r) {
		return true
	}
	for _, name := range fallbackFonts {
		if f, _, ok := lookupFont(name, FontRegular); ok && hasGlyph(f, &buf, r) {
			return true
		}
	}
	return false
}

// addScriptFont adds the first font of the system that has the character
func addScriptFont(r rune) {
	for _, path := range scriptFontFiles(r) {
		scriptFontsMu.Lock()
		done := scriptFiles[path]
		scriptFontsMu.Unlock()
		if done {
			continue
		}
		f, _, err := openFontFile(path, "")
		if err != nil {
			continue
		}
		var buf sfnt.Buffer
		if !hasGlyph(f, &buf, r) {
			continue
		}
		scriptFontsMu.Lock()
		scriptFiles[path] = true
		name := "system-script-" + strconv.Itoa(len(scriptFiles))
		scriptFontsMu.Unlock()
		setFont(name, f)
		AddFallbackFont(name)
		return
	}
}

// scriptFontFiles returns the files of the fonts of the system to try for
// the character: fontconfig's choice on Linux, the fonts of Windows and
// macOS that have many scripts
func scriptFontFiles(r rune) []string {
	switch runtime.GOOS {
	case "windows":
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		if os.Getenv("WINDIR") == "" {
			dir = `C:\Windows\Fonts`
		}
		var out []string
		for _, f := range []string{"segoeui.ttf", "Nirmala.ttc", "Nirmala.ttf", "LeelawUI.ttf", "ebrima.ttf",
			"gadugi.ttf", "seguihis.ttf", "arial.ttf", "tahoma.ttf", "seguisym.ttf", "mvboli.ttf", "himalaya.ttf"} {
			out = append(out, filepath.Join(dir, f))
		}
		return out
	case "darwin":
		var out []string
		for _, pattern := range []string{"/System/Library/Fonts/SFArabic.ttf", "/System/Library/Fonts/SFHebrew.ttf",
			"/System/Library/Fonts/GeezaPro.ttc", "/System/Library/Fonts/ArialHB.ttc", "/System/Library/Fonts/Kohinoor*.ttc",
			"/System/Library/Fonts/Thonburi.ttc", "/System/Library/Fonts/Supplemental/*.ttf", "/System/Library/Fonts/Supplemental/*.ttc",
			"/System/Library/Fonts/*.ttc", "/System/Library/Fonts/*.ttf"} {
			paths, _ := filepath.Glob(pattern)
			out = append(out, paths...)
		}
		return out
	}
	if path := fontconfigMatch(fmt.Sprintf(":charset=%x", r)); path != "" {
		return []string{path}
	}
	return nil
}
