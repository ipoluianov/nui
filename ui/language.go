package ui

import (
	"sync"

	"github.com/ipoluianov/nui/internal/platforms"
	"github.com/ipoluianov/nui/ui/i18n"
)

// The language of the application. Texts follow it live: set them with
// SetTextFunc (and SetTitleFunc, SetTooltipFunc...) and they are updated on
// SetLanguage; other texts are updated in Form.SetOnLanguageChanged.

var (
	languageMu  sync.RWMutex
	language    string
	languageSet bool

	systemLanguageOnce  sync.Once
	systemLanguageValue string
)

// Language returns the language of the application as a tag like "ru" or
// "zh-TW": the one set by SetLanguage, else the system's.
func Language() string {
	languageMu.RLock()
	lang, set := language, languageSet
	languageMu.RUnlock()
	if set {
		return lang
	}
	return SystemLanguage()
}

// SystemLanguage returns the language of the system's user interface, "en"
// if unknown.
func SystemLanguage() string {
	systemLanguageOnce.Do(func() {
		systemLanguageValue = i18n.Normalize(platforms.SystemLanguage())
		if systemLanguageValue == "" {
			systemLanguageValue = "en"
		}
	})
	return systemLanguageValue
}

// SetLanguage switches the application to the language: the open forms
// update their texts (see SetTextFunc and Form.SetOnLanguageChanged) and
// are laid out again. Chinese, Japanese and Korean also pick their variant
// of the system CJK font.
func SetLanguage(lang string) {
	lang = i18n.Normalize(lang)
	languageMu.Lock()
	language, languageSet = lang, true
	languageMu.Unlock()

	setCJKVariant(cjkVariantFor(lang))
	for _, form := range openForms() {
		form.Invoke(form.applyLanguage)
	}
}

// setTextFunc makes the widget's text come from f, now and on every
// language change; set is the widget's own SetText.
func (c *Widget) setTextFunc(f func() string, set func(string)) {
	c.textFunc, c.textSetter = f, set
	if f != nil {
		set(f())
	}
}

// SetTooltipFunc makes the tooltip come from f, now and on every language change.
func (c *Widget) SetTooltipFunc(f func() string) {
	c.tooltipFunc = f
	if f != nil {
		c.SetTooltip(f())
	}
}

// applyLanguage updates the texts that come from funcs. Widgets with more
// such texts override it.
func (c *Widget) applyLanguage() {
	if c.textFunc != nil {
		c.textSetter(c.textFunc())
	}
	if c.tooltipFunc != nil {
		c.SetTooltip(c.tooltipFunc())
	}
}

// applyLanguageTree updates the widget, its context menu and its children.
func applyLanguageTree(w Widgeter) {
	w.applyLanguage()
	if withMenu, ok := w.(interface{ ContextMenu() *ContextMenu }); ok {
		if menu := withMenu.ContextMenu(); menu != nil {
			applyLanguageTree(menu)
		}
	}
	for _, child := range w.Widgets() {
		applyLanguageTree(child)
	}
}

// SetTitleFunc makes the form's title come from f, now and on every
// language change.
func (c *Form) SetTitleFunc(f func() string) {
	c.titleFunc = f
	if f != nil {
		c.SetTitle(f())
	}
}

// SetOnLanguageChanged sets the function that updates the texts SetTextFunc
// doesn't cover (table headers, combo box items, texts put together...)
// when the language changes.
func (c *Form) SetOnLanguageChanged(f func()) {
	c.onLanguageChanged = f
}

// applyLanguage updates the form's texts for a new language.
func (c *Form) applyLanguage() {
	if c.topWidget == nil {
		return
	}
	// Menus and dropdowns get their texts when opened
	c.closePopups()
	if c.titleFunc != nil {
		c.SetTitle(c.titleFunc())
	}

	// Lay out once, after all the texts changed
	c.layoutingBlockStack++
	applyLanguageTree(c.topWidget)
	if c.onLanguageChanged != nil {
		c.onLanguageChanged()
	}
	applyThemeMetricsTree(c.topWidget)
	c.layoutingBlockStack--

	c.UpdateLayout()
	c.forceUpdate()
}

func (c *Button) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

func (c *Label) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

func (c *Checkbox) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

func (c *RadioButton) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

func (c *ContextMenuItem) SetTextFunc(f func() string) {
	c.setTextFunc(f, c.SetText)
}

// applyLanguage also updates the submenu, which isn't a child of the item.
func (c *ContextMenuItem) applyLanguage() {
	c.Widget.applyLanguage()
	if c.innerMenu != nil {
		applyLanguageTree(c.innerMenu)
	}
}

// SetHintFunc makes the hint come from f, now and on every language change.
func (c *TextBox) SetHintFunc(f func() string) {
	c.hintFunc = f
	if f != nil {
		c.SetHint(f())
	}
}

func (c *TextBox) applyLanguage() {
	c.Widget.applyLanguage()
	if c.hintFunc != nil {
		c.SetHint(c.hintFunc())
	}
}

// SetPageNameFunc makes the name of the page come from f, now and on every
// language change.
func (c *TabWidget) SetPageNameFunc(index int, f func() string) {
	if index < 0 || index >= len(c.pages) {
		return
	}
	c.pages[index].nameFunc = f
	c.applyPageNames()
}

func (c *TabWidget) applyPageNames() {
	names := make([]string, len(c.pages))
	for i := range c.pages {
		if c.pages[i].nameFunc != nil {
			c.pages[i].name = c.pages[i].nameFunc()
		}
		names[i] = c.pages[i].name
	}
	c.panelTop.SetItems(names)
}

// applyLanguage also updates the pages that aren't shown, which aren't
// among the tab widget's children.
func (c *TabWidget) applyLanguage() {
	c.Widget.applyLanguage()
	c.applyPageNames()
	for i, page := range c.pages {
		if i != c.currentPage {
			applyLanguageTree(page.widget)
		}
	}
}
