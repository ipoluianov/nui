//go:build !darwin

package platforms

func setNativeMenuBar(appName string, menus []*NativeMenu) bool {
	return false
}
