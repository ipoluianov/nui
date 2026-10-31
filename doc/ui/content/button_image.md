# ButtonImage

Button that displays an `image.Image`.

## Create

```go
img := /* load image.Image */
btn := ui.NewButtonImage(img)
```

## Click

```go
btn.SetOnButtonClick(func(b *ui.ButtonImage) {
  // ...
})
```

`Press()` calls the click handler from the code.

## Image

- `SetImage(img)` / `Image()` change the picture.
- `SetImageHAlign(ui.HAlignLeft)`, `SetImageVAlign(ui.VAlignTop)`: where the image sits in the button (centered by default).
