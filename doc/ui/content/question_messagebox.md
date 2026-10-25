# QuestionMessageBox

Ask a question in a modal message box over the form of a widget. The callbacks run after the box closes; any of them may be nil.

```go
ui.ShowQuestionMessageBoxYesNo(form.Panel(), "Delete", "Delete the file?",
	func() { deleteFile() }, // Yes (Enter)
	func() {},               // No (Esc)
)

ui.ShowQuestionMessageBoxOKCancel(form.Panel(), "Replace", "Replace all the matches?",
	func() { replaceAll() }, // OK (Enter)
	func() {},               // Cancel (Esc)
)

// Before closing a document: Esc and the close button of the box cancel,
// they must not discard the changes
ui.ShowQuestionMessageBoxYesNoCancel(form.Panel(), "Notepad", "Save the changes?",
	func() { save(close) }, // Yes (Enter)
	close,                  // No: discard
	nil,                    // Cancel (Esc): keep editing
)
```
