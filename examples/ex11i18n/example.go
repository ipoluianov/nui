package ex11i18n

import "github.com/ipoluianov/nui/ui"

// NewExampleForm shows an application in several languages switched live:
// the texts come from the catalog in strings.go.
//
//   - Simple texts are set with SetTextFunc (SetTitleFunc, SetHintFunc,
//     SetTooltipFunc, SetPageNameFunc) and follow the language by themselves.
//   - The rest (combo box items, table columns) is updated in
//     SetOnLanguageChanged.
//   - The library's own texts (the buttons of the message box) are translated
//     by the library.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetSize(720, 520)
	form.SetTitleFunc(func() string { return T().Title })

	panel := form.Panel()

	// The language names stay in their own language
	languages := panel.AddPanel(0, 0)
	for i, l := range []struct{ tag, name string }{{"en", "English"}, {"ru", "Русский"}, {"zh", "中文"}} {
		languages.AddButton(0, i, l.name, func() { ui.SetLanguage(l.tag) })
	}
	languages.AddHSpacer(0, 3)

	greeting := panel.AddLabel(1, 0, "")
	greeting.SetTextFunc(func() string { return T().Greeting })
	greeting.SetContextMenu(newMenu(greeting))

	tabs := ui.NewTabWidget()
	tabs.AddPage("", newGeneralPage(form))
	about := ui.NewLabel("")
	about.SetTextFunc(func() string { return T().About })
	tabs.AddPage("", about)
	tabs.SetPageNameFunc(0, func() string { return T().TabGeneral })
	tabs.SetPageNameFunc(1, func() string { return T().TabAbout })
	panel.AddWidget(2, 0, tabs)

	return form
}

func newGeneralPage(form *ui.Form) *ui.Panel {
	page := ui.NewPanel()

	name := ui.NewTextBox()
	name.SetHintFunc(func() string { return T().NameHint })
	page.AddWidget(0, 0, name)

	remember := ui.NewCheckbox("")
	remember.SetTextFunc(func() string { return T().Remember })
	page.AddWidget(1, 0, remember)

	theme := page.AddPanel(2, 0)
	themeLabel := theme.AddLabel(0, 0, "")
	themeLabel.SetTextFunc(func() string { return T().Theme })
	themes := ui.NewComboBox()
	for range 3 {
		themes.AddItem("", nil)
	}
	theme.AddWidget(0, 1, themes)
	theme.AddHSpacer(0, 2)

	// A text with a number: the func picks the plural form
	files := 1
	counter := page.AddPanel(3, 0)
	filesLabel := counter.AddLabel(0, 0, "")
	filesLabel.SetTextFunc(func() string { return T().FilesSelected(files) })
	counter.AddButton(0, 1, "-", func() {
		files = max(0, files-1)
		filesLabel.SetText(T().FilesSelected(files))
	})
	counter.AddButton(0, 2, "+", func() {
		files++
		filesLabel.SetText(T().FilesSelected(files))
	})
	counter.AddHSpacer(0, 3)

	table := ui.NewTable()
	table.SetColumnCount(2)
	table.SetRowCount(3)
	for row, f := range []struct{ name, size string }{{"readme.txt", "2 KB"}, {"photo.jpg", "3.1 MB"}, {"报告.pdf", "740 KB"}} {
		table.SetCellText2(row, 0, f.name)
		table.SetCellText2(row, 1, f.size)
	}
	page.AddWidget(4, 0, table)

	buttons := page.AddPanel(5, 0)
	buttons.AddHSpacer(0, 0)
	save := buttons.AddButton(0, 1, "", func() {
		// The OK and Cancel buttons are translated by the library
		ui.ShowQuestionMessageBoxOKCancel(form.Panel(), T().ConfirmTitle, T().ConfirmDelete("readme.txt"), nil, nil)
	})
	save.SetTextFunc(func() string { return T().Save })
	save.SetTooltipFunc(func() string { return T().SaveTooltip })

	// Texts that aren't widgets' own
	updateTexts := func() {
		themes.SetItemText(0, T().ThemeLight)
		themes.SetItemText(1, T().ThemeDark)
		themes.SetItemText(2, T().ThemeSystem)
		table.SetColumnName(0, T().ColumnName)
		table.SetColumnName(1, T().ColumnSize)
	}
	updateTexts()
	form.SetOnLanguageChanged(updateTexts)

	return page
}

func newMenu(owner ui.Widgeter) *ui.ContextMenu {
	menu := ui.NewContextMenu(owner)
	menu.AddItem("", nil).SetTextFunc(func() string { return T().Menu.Copy })
	menu.AddItem("", nil).SetTextFunc(func() string { return T().Menu.Paste })
	more := ui.NewContextMenu(owner)
	more.AddItem("", nil).SetTextFunc(func() string { return T().Menu.Help })
	menu.AddItemWithSubmenu("", more).SetTextFunc(func() string { return T().Menu.More })
	return menu
}
