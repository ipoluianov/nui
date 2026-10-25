# LogView

A growing log: lines are appended at the end, the oldest dropped past a limit (100 000 by default, `SetMaxLines`), and the view follows the end while the user is at it — scrolled up, it stays where the user is. It is a read-only `TextBox` in the monospaced font: the text can be selected, copied and searched.

```go
logs := ui.NewLogView()
logs.AppendLine("started")        // on the UI thread
log.SetOutput(logs)               // an io.Writer: from any goroutine
fmt.Fprintf(logs, "%d files\n", n) // a line shows once its "\n" is written
logs.Clear()
```

Many writes before a frame make one update: a log of thousands of lines a second doesn't slow the window down.
