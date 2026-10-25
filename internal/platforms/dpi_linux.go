//go:build linux

package platforms

import (
	"math"
	"os"
	"strconv"
	"strings"
)

// HiDPI on Linux (X11): the scale of the desktop, the same for all the
// windows and monitors (X11 has no per-monitor scale). Taken from, in order:
//
//   - NUI_SCALE, set for nui only (e.g. NUI_SCALE=1.5);
//   - Xft.dpi / 96: the setting of the desktop's scale, GNOME and KDE write
//     it to the X resources (192 at 200%);
//   - GDK_SCALE, the integer scale of GTK applications.
//
// The scale is kept between 1 and 4; 1 changes nothing.

var displayScale float64

// desktopScale returns the scale, read once the display is open
func desktopScale(display uintptr) float64 {
	if displayScale == 0 {
		resources := ""
		if display != 0 {
			resources = xResourceManagerString(display)
		}
		displayScale = parseDesktopScale(os.Getenv("NUI_SCALE"), resources, os.Getenv("GDK_SCALE"))
	}
	return displayScale
}

// parseDesktopScale chooses the scale from the variables and the X resources
func parseDesktopScale(nuiScale, resources, gdkScale string) float64 {
	clamp := func(s float64) float64 {
		// Steps of 1/8: Xft.dpi 120 is 1.25 exactly, a rounding error isn't
		return math.Round(min(max(s, 1), 4)*8) / 8
	}
	if s, err := strconv.ParseFloat(strings.TrimSpace(nuiScale), 64); err == nil && s > 0 {
		return clamp(s)
	}
	for _, line := range strings.Split(resources, "\n") {
		name, value, ok := strings.Cut(line, ":")
		if ok && strings.TrimSpace(name) == "Xft.dpi" {
			if dpi, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && dpi > 0 {
				if s := dpi / 96; s > 1 {
					return clamp(s)
				}
			}
		}
	}
	if s, err := strconv.Atoi(strings.TrimSpace(gdkScale)); err == nil && s > 1 {
		return clamp(float64(s))
	}
	return 1
}
