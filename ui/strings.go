package ui

import (
	"sync"

	"github.com/ipoluianov/nui/ui/i18n"
)

// UIStrings are the texts of the library's own widgets, e.g. the buttons of
// message boxes. English, Russian and Chinese are built in; an application
// adds other languages with RegisterUIStrings.
type UIStrings struct {
	OK     string
	Cancel string
	Yes    string
	No     string
}

var (
	uiStringsMu     sync.RWMutex
	uiStringsByLang = map[string]UIStrings{
		"ru": {OK: "OK", Cancel: "Отмена", Yes: "Да", No: "Нет"},
		"zh": {OK: "确定", Cancel: "取消", Yes: "是", No: "否"},
	}
	uiStringsEnglish = UIStrings{OK: "OK", Cancel: "Cancel", Yes: "Yes", No: "No"}
	uiStringsCatalog = i18n.NewCatalog(uiStringsEnglish, uiStringsByLang)
)

// UIText returns the library's texts in the language of the application.
func UIText() *UIStrings {
	uiStringsMu.RLock()
	defer uiStringsMu.RUnlock()
	return uiStringsCatalog.Get(Language())
}

// RegisterUIStrings adds or replaces the library's texts for a language.
func RegisterUIStrings(lang string, s UIStrings) {
	uiStringsMu.Lock()
	defer uiStringsMu.Unlock()
	uiStringsByLang[i18n.Normalize(lang)] = s
	uiStringsCatalog = i18n.NewCatalog(uiStringsEnglish, uiStringsByLang)
}
