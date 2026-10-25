package platforms

import "math"

// Logical and physical pixels. A window on a scaled screen (Windows at 150%,
// Linux with Xft.dpi 192) has more pixels than nui's sizes say: the
// platform converts the sizes, positions and mouse coordinates it gives to
// nui to logical pixels, and the ones nui gives back to physical ones; the
// form paints at the scale (see Scale).

// toPhysical converts logical pixels to the window's physical ones
func (c *nativeWindow) toPhysical(v int) int {
	return scaleToPhysical(v, c.Scale())
}

// toLogical converts a physical position (e.g. of the mouse) to the logical
// pixel it's in
func (c *nativeWindow) toLogical(v int) int {
	return scaleToLogical(v, c.Scale())
}

// toLogicalSize converts a physical size to a logical one that covers it whole
func (c *nativeWindow) toLogicalSize(v int) int {
	s := c.Scale()
	if s == 1 {
		return v
	}
	return int(math.Ceil(float64(v)/s - 1e-9))
}

// toLogicalScreen converts a screen coordinate to the logical one that
// toPhysical turns back into it
func (c *nativeWindow) toLogicalScreen(v int) int {
	s := c.Scale()
	if s == 1 {
		return v
	}
	return int(math.Round(float64(v) / s))
}

func scaleToPhysical(v int, s float64) int {
	if s == 1 {
		return v
	}
	return int(math.Round(float64(v) * s))
}

func scaleToLogical(v int, s float64) int {
	if s == 1 {
		return v
	}
	return int(math.Floor(float64(v) / s))
}
