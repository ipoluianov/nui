# Toasts

A toast is a short message in the bottom-right corner of the form that disappears by itself.
A click closes it at once. Several toasts stack up, the newest at the bottom.

```go
form.ShowToast("Saved", ui.ToastSuccess)
form.ShowToastFor("Connection failed", ui.ToastError, 6*time.Second)
ui.ShowToast(someWidget, "Copied", ui.ToastInfo) // in the widget's form
```

- Kinds (the color of the stripe): `ToastInfo`, `ToastSuccess`, `ToastWarning`, `ToastError`.
- `ShowToast` shows it for `ToastDefaultDuration` (3 s); long texts wrap.
- A toast is a native popup window, so it isn't clipped by a small form; where the platform has
  no popups it's drawn inside the form. Hiding or closing the form closes its toasts.

For a system notification (outside the application), see
[TrayIcon.ShowNotification](trayicon.md).
