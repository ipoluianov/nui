# MessageBox

Shows a message with an OK button, modal over the form of a widget.

## Usage

```go
ui.ShowMessageBox(form.Panel(), "Error", "Something went wrong: "+err.Error())
```

- The first argument is any widget of the form the box is shown over; with `nil` the box is a window of its own, not modal.
- The function returns at once; Enter, Esc and OK close the box.
- The text is wrapped to the width of the box (250 to 600 pixels); `\n` starts a new line; more than 20 lines are cut with "...".
- The button's text follows the language (`ui.UIText().OK`).

To ask a question (Yes/No, OK/Cancel, Yes/No/Cancel), see [QuestionMessageBox](question_messagebox.md); for a dialog of your own, [Dialog](dialog.md).
