# Standard dialogs

Each is modal over the form of the widget given, sized by its content, and returns at once; the result comes in a callback after it closes.

## Input

```go
ui.ShowInputDialog(form.Panel(), ui.InputDialog{
	Title: "Rename", Prompt: "New name:", Value: name, // the value is selected
	Validate: func(text string) error {                // optional: an error keeps the dialog open
		if text == "" {
			return errors.New("The name is empty")
		}
		return nil
	},
}, func(text string) { rename(text) }) // OK or Enter; Cancel and Esc call nothing
```

`Password: true` shows stars.

## Progress

```go
dialog := ui.ShowProgressDialog(form.Panel(), "Copying", "Copying the files...", cancelCopy)
dialog.SetProgress(0.5) // 0..1
dialog.SetText("file 3 of 6")
dialog.Close()
```

With a work in the background, `ui.RunWithProgress` does all of it (see [async](async.md)).

## About

```go
ui.ShowAboutDialog(form.Panel(), ui.AboutDialog{
	Name: "Notepad", Version: "Version 1.0", Description: "A primitive text editor.",
	Copyright: "© 2026", Website: "https://example.com", Icon: icon64,
})
```
