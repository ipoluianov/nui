//go:build !darwin

package platforms

// SystemFontFile is used on macOS only, where the system fonts are drawn by
// nui's rasterizer: elsewhere OpenSystemFont draws them
func SystemFontFile(name string) (path, family string, err error) {
	return "", "", ErrSystemTextNotSupported
}
