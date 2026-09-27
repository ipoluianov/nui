package ex11i18n

import "testing"

// Every language has all the texts: a new field in Strings without a
// translation fails here instead of silently showing English.
func TestAllTranslated(t *testing.T) {
	for lang, fields := range catalog.Missing() {
		t.Errorf("%s: not translated: %v", lang, fields)
	}
}
