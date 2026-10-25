//go:build windows || linux

package platforms

// StartFileDrag runs the drag of the files out to the other applications
// (startFileDragSync of the platform) and gives its result to done before
// it returns
func (c *nativeWindow) StartFileDrag(files []string, done func(effect DragEffect, dropped bool)) bool {
	effect, dropped, started := c.startFileDragSync(files)
	if started && done != nil {
		done(effect, dropped)
	}
	return started
}
