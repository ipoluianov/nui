package ui

import "testing"

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
	for lang, cancel := range map[string]string{"ru-RU": "Отмена", "zh_CN": "取消", "zh-TW": "取消", "de": "Cancel", "en": "Cancel"} {
		SetLanguage(lang)
		if got := UIText().Cancel; got != cancel {
			t.Errorf("%s: %q, want %q", lang, got, cancel)
		}
	}

	RegisterUIStrings("de", UIStrings{Cancel: "Abbrechen"})
	SetLanguage("de")
	if got := UIText(); got.Cancel != "Abbrechen" || got.OK != "OK" {
		t.Errorf("registered German: %+v", got)
	}
}

func TestCJKVariantFor(t *testing.T) {
	for lang, variant := range map[string]string{"zh": "sc", "zh-CN": "sc", "zh-Hant": "tc", "zh-TW": "tc", "zh-HK": "hk", "ja-JP": "jp", "ko": "kr", "ru": "sc", "": "sc"} {
		if got := cjkVariantFor(lang); got != variant {
			t.Errorf("%q: %q, want %q", lang, got, variant)
		}
	}
}
