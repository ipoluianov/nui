package ui

import "golang.design/x/clipboard"

func init() {
	clipboard.Init()
}

func ClipboardSetText(text string) {
	clipboard.Write(clipboard.FmtText, []byte(text))
}

func ClipboardGetText() (string, error) {
	data := clipboard.Read(clipboard.FmtText)
	return string(data), nil
}
