package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui/i18n"
)

// Texts set with SetTextFunc follow the language, also in the places the
// widget tree doesn't reach: tab pages not shown, submenus
func TestApplyLanguageTree(t *testing.T) {
	word := "Save"
	tr := func() string { return word }

	btn := NewButton("")
	btn.SetTextFunc(tr)

	hidden := NewLabel("")
	hidden.SetTextFunc(tr)
	tabs := NewTabWidget()
	tabs.AddPage("first", NewPanel())
	tabs.AddPage("second", hidden)
	tabs.SetPageNameFunc(1, tr)

	panel := NewPanel()
	menu := NewContextMenu(panel)
	sub := NewContextMenu(panel)
	subItem := sub.AddItem("", nil)
	subItem.SetTextFunc(tr)
	menu.AddItemWithSubmenu("more", sub)
	panel.SetContextMenu(menu)

	word = "Сохранить"
	for _, w := range []Widgeter{btn, tabs, panel} {
		applyLanguageTree(w)
	}

	if btn.Text() != word {
		t.Errorf("button: %q", btn.Text())
	}
	if hidden.Text() != word {
		t.Errorf("label on a hidden tab page: %q", hidden.Text())
	}
	if tabs.pages[1].name != word {
		t.Errorf("tab name: %q", tabs.pages[1].name)
	}
	if subItem.text != word {
		t.Errorf("submenu item: %q", subItem.text)
	}
}

func TestUIText(t *testing.T) {
	t.Cleanup(func() {
		languageMu.Lock()
		languageSet = false
		languageMu.Unlock()
		setCJKVariant("sc")
	})
	for lang, cancel := range map[string]string{"ru-RU": "Отмена", "zh_CN": "取消", "zh-TW": "取消", "de": "Abbrechen",
		"de_AT.UTF-8": "Abbrechen", "no_NO": "Avbryt", "en": "Cancel", "eo": "Cancel"} {
		SetLanguage(lang)
		if got := UIText().Cancel; got != cancel {
			t.Errorf("%s: %q, want %q", lang, got, cancel)
		}
	}
	SetLanguage("pt-BR")
	if got := UIText(); got.Save != "Salvar" || got.Open != "Abrir" {
		t.Errorf("pt-BR: Save %q, Open %q", got.Save, got.Open)
	}

	t.Cleanup(func() {
		uiStringsMu.Lock()
		delete(uiStringsByLang, "eo")
		uiStringsCatalog = i18n.NewCatalog(uiStringsEnglish, uiStringsByLang)
		uiStringsMu.Unlock()
	})
	RegisterUIStrings("eo", UIStrings{Cancel: "Nuligi"})
	SetLanguage("eo")
	if got := UIText(); got.Cancel != "Nuligi" || got.OK != "OK" {
		t.Errorf("registered Esperanto: %+v", got)
	}
}

// Every built-in language has all the texts, and those of a file name
// keep its one %s
func TestBuiltinUIStringsComplete(t *testing.T) {
	builtin := builtinUIStrings()
	catalog := i18n.NewCatalog(uiStringsEnglish, builtin)
	for lang, fields := range catalog.Missing() {
		// MonthYear only where the title is not "month year"
		fields = slices.DeleteFunc(fields, func(f string) bool { return f == "MonthYear" })
		if len(fields) > 0 {
			t.Errorf("%s: missing %v", lang, fields)
		}
	}
	for lang := range builtin {
		s := catalog.Get(lang)
		for i, m := range s.MonthNames {
			if m == "" {
				t.Errorf("%s: month %d empty", lang, i+1)
			}
		}
		for i, d := range s.WeekdaysShort {
			if d == "" {
				t.Errorf("%s: weekday %d empty", lang, i)
			}
		}
		if title := monthTitle(s, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)); strings.Contains(title, "%!") || !strings.Contains(title, "2026") {
			t.Errorf("%s: title %q", lang, title)
		}
		for name, text := range map[string]string{"FileNotFound": s.FileNotFound, "FileExistsReplace": s.FileExistsReplace} {
			if strings.Count(text, "%") != 1 || !strings.Contains(text, "%s") {
				t.Errorf("%s: %s %q has not one %%s", lang, name, text)
			}
		}
	}
}

func TestCJKVariantFor(t *testing.T) {
	for lang, variant := range map[string]string{"zh": "sc", "zh-CN": "sc", "zh-Hant": "tc", "zh-TW": "tc", "zh-HK": "hk", "ja-JP": "jp", "ko": "kr", "ru": "sc", "": "sc"} {
		if got := cjkVariantFor(lang); got != variant {
			t.Errorf("%q: %q, want %q", lang, got, variant)
		}
	}
}

// The title of a calendar's month in the order of the language
func TestMonthTitle(t *testing.T) {
	oct := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	catalog := i18n.NewCatalog(uiStringsEnglish, builtinUIStrings())
	for lang, want := range map[string]string{"en": "October 2026", "de": "Oktober 2026", "ru": "Октябрь 2026",
		"ja": "2026年10月", "zh": "2026年十月", "ko": "2026년 10월", "hu": "2026. október", "vi": "Tháng 10 năm 2026"} {
		if got := monthTitle(catalog.Get(lang), oct); got != want {
			t.Errorf("%s: %q, want %q", lang, got, want)
		}
	}
}
