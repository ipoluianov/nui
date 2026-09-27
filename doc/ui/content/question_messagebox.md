# QuestionMessageBox

Convenience function to show an OK/Cancel dialog.

## Usage

```go
ui.ShowQuestionMessageBox(
  "Confirm",
  "Delete file?",
  func() { fmt.Println("ok") },
  func() { fmt.Println("cancel") },
)
```

