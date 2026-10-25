//go:build !linux

package ui

// The accessible objects reach the platform on Linux only (AT-SPI); on
// Windows (UI Automation) and macOS (NSAccessibility) not yet

func startAccessibility() {}

func (c *Form) accessibilityFocusChanged() {}
