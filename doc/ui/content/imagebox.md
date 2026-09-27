# ImageBox

Displays an `image.Image` with configurable scaling.

## Create

```go
ib := ui.NewImageBox()
ib.SetImage(img)
```

## Scaling modes

```go
ib.SetScaling(ui.ImageBoxScaleAdjustImageKeepAspectRatio)
// other modes: StretchImage, NoScaleImageInCenter, etc.
```

