package ui

import "testing"

func TestKeyEquivalent(t *testing.T) {
	for key, want := range map[Key]string{
		KeyS: "s", Key1: "1", KeyF1: "", KeyF5: "", KeyEqual: "=", KeyDelete: "", KeyArrowLeft: "", KeyCtrl: "",
	} {
		if got := keyEquivalent(key); got != want {
			t.Errorf("keyEquivalent(%v) = %q, want %q", key, got, want)
		}
	}
	for text, want := range map[string]string{"&Open": "Open", "Save &As...": "Save As...", "Q&&A": "Q&A", "Plain": "Plain"} {
		if got := stripMnemonic(text); got != want {
			t.Errorf("stripMnemonic(%q) = %q, want %q", text, got, want)
		}
	}
}
