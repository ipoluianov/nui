package recipes

import (
	"fmt"
	"image"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/i18n"
)

// Main is the main function of an application: one copy of it, its
// settings, the theme and the language of last time, the main window.
// args are the command line arguments after the program's name.
func Main(args []string) {
	instance, ok := ui.AcquireSingleInstance("recipes-editor", args)
	if !ok {
		return // the running copy got the arguments and came to the front
	}
	defer instance.Close()

	settings, _ := ui.OpenSettings("recipes-editor") // empty on an error
	if settings.Bool("dark", false) {
		ui.ApplyDarkTheme()
	}
	if lang := settings.String("language", ""); lang != "" {
		ui.SetLanguage(lang) // else the system's
	}

	editor := NewEditor(settings)
	instance.OnActivate(func(args []string) { // on the UI thread
		editor.Form.Activate()
	})
	editor.Form.Exec() // returns when the window closes
	settings.Save()
}

// ToggleTheme switches between the light and the dark theme and keeps the
// choice; the open forms repaint themselves.
func ToggleTheme(settings *ui.Settings) {
	dark := !settings.Bool("dark", false)
	if dark {
		ui.ApplyDarkTheme()
	} else {
		ui.ApplyLightTheme()
	}
	settings.Set("dark", dark)
}

// Strings are the texts of the application in one language: a misspelled
// name is a compile error, a missing translation falls back to English.
type Strings struct {
	Title    string
	Greeting string
	Files    func(n int) string
}

var catalog = i18n.NewCatalog(Strings{
	Title:    "Files",
	Greeting: "Hello",
	Files: func(n int) string {
		return fmt.Sprint(n, " ", i18n.Plural("en", n, "file", "files"))
	},
}, map[string]Strings{
	"ru": {
		Title:    "Файлы",
		Greeting: "Привет",
		Files: func(n int) string {
			return fmt.Sprint(n, " ", i18n.Plural("ru", n, "файл", "файла", "файлов"))
		},
	},
})

// T returns the texts in the language of the application.
func T() *Strings { return catalog.Get(ui.Language()) }

// TranslatedForm has texts that follow the language live: SetLanguage
// sets them again; the library's own texts (OK, Cancel, the calendar)
// follow by themselves.
func TranslatedForm(count func() int) *ui.Form {
	form := ui.NewForm()
	form.SetTitleFunc(func() string { return T().Title })
	greeting := ui.NewLabel("")
	greeting.SetTextFunc(func() string { return T().Greeting })
	files := ui.NewLabel("")
	files.SetTextFunc(func() string { return T().Files(count()) })
	form.Panel().AddWidget(0, 0, greeting)
	form.Panel().AddWidget(1, 0, files)
	form.Panel().AddVSpacer(2, 0)
	return form
}

// TrayApp keeps the application in the tray: the close button hides the
// window, the icon shows it, Quit in the icon's menu quits.
func TrayApp(form *ui.Form, icon image.Image) error {
	tray, err := ui.NewTrayIcon()
	if err != nil {
		return err // no tray here (ui.ErrTrayNotSupported): keep the window
	}
	quitting := false
	tray.SetIcon(icon)
	tray.SetTooltip("Recipes")
	tray.SetOnClick(func() { form.Activate() })
	tray.SetMenu(
		ui.TrayMenuItem{Text: "Show", OnClick: func() { form.Activate() }},
		ui.TrayMenuItem{Separator: true},
		ui.TrayMenuItem{Text: "Quit", OnClick: func() {
			quitting = true
			tray.Close()
			form.Close()
		}},
	)
	form.OnClose = func() bool {
		if quitting {
			return true
		}
		form.Hide()
		tray.ShowNotification("Recipes", "Still running in the tray")
		return false
	}
	return nil
}
