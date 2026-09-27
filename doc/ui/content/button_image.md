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

