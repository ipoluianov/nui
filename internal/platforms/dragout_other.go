//go:build !windows

package platforms

// StartFileDrag: dragging files out to other applications is not there yet on
// this platform; the drag stays in the window
func (c *nativeWindow) StartFileDrag(files []string) (effect DragEffect, dropped bool, started bool) {
	_ = files
	return DragEffectNone, false, false
}
