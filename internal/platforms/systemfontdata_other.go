//go:build !windows

package platforms

// The system's text engine gets font files on Windows only: elsewhere nui's
// rasterizer draws them

func registerFontData(data []byte) (FontData, error) {
	return FontData{}, ErrSystemTextNotSupported
}

func openFontData(fd FontData, pixelSize float64) (SystemFont, error) {
	return nil, ErrSystemTextNotSupported
}
