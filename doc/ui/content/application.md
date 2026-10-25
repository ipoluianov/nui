# Application services

## Settings

The settings of the application, kept between its starts in a JSON file of the user's configuration directory (`~/.config/<app>` on Linux, `%AppData%\<app>` on Windows, `~/Library/Application Support/<app>` on macOS).

```go
settings, _ := ui.OpenSettings("myapp") // an error leaves them empty, in memory
theme := settings.String("theme", "light")
settings.Set("theme", "dark")
settings.Set("recent", recentFiles)     // anything JSON encodes
var recent []string
settings.Get("recent", &recent)
settings.Save()
```

`ui.SettingsDir(app)` is the directory, for the application's other files.

## The window where the user left it

```go
form.RememberState(settings, "main") // restores now, saves when the window closes
form.Show()
```

The state is the window's place and size when not maximized, and whether it is maximized; a place off the screens (a monitor gone) comes back onto one. `form.State()` and `form.SetState(state)` do the same by hand, e.g. for several windows.

## One copy of the application

```go
instance, ok := ui.AcquireSingleInstance("myapp", os.Args[1:])
if !ok {
	return // the running copy got the arguments and shows itself
}
defer instance.Close()
instance.OnActivate(func(args []string) { // on the UI thread
	form.Activate()
	openFiles(args)
})
```

## Activate

`form.Activate()` brings the window to the front and gives it the focus: a minimized window is restored, a maximized one stays maximized, a hidden one is shown.
