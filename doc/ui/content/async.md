# Work in the background

`ui.RunAsync` runs a function in a goroutine and gives its result to the UI thread, where the widgets may be changed:

```go
task := ui.RunAsync(form.Panel(), func(ctx context.Context, p *ui.Progress) ([]File, error) {
	return scan(ctx, dir, func(done, all int) {
		p.Report(float64(done)/float64(all), "") // from the goroutine; joined if too frequent
	})
}, func(files []File, err error) { // on the UI thread
	if err != nil {
		ui.ShowMessageBox(form.Panel(), "Error", err.Error())
		return
	}
	list.SetModel(filesModel(files))
})
task.OnProgress.Connect(func(r ui.ProgressReport) { bar.SetValue(r.Fraction) }) // bar := ui.NewProgressBar(0, 1, 0)
// task.Cancel() cancels the context
```

- The work is cancelled (its context done) by `task.Cancel()`, or when the form of the widget given closes — then the result is dropped, the widgets are gone.
- The work must not touch the widgets: it reports with `p.Report`, and returns.
- `ProgressReport` is `Fraction` (0..1, negative if unknown) and `Text`. `task.Finished()` tells whether the result was given.

With a progress dialog that has a Cancel button:

```go
ui.RunWithProgress(form.Panel(), "Copying", "Copying the files...",
	func(ctx context.Context, p *ui.Progress) (int, error) { return copyFiles(ctx, p) },
	func(n int, err error) { status.ShowMessage(fmt.Sprint(n, " files copied"), 3*time.Second) })
```

For something to repaint from a goroutine, without a result: `widget.RequestRepaint()`. For anything else on the UI thread: `ui.Invoke(f)`.
