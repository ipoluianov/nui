# EnterStringDialog

Convenience function to ask the user for a string.

## Usage

```go
ui.ShowEnterStringDialog(
  "Name",
  "Enter your name",
  "",
  func(value string) {
    fmt.Println("entered:", value)
  },
)
```

