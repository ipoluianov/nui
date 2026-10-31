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
```

| Mode | What it does |
|---|---|
| `ui.ImageBoxScaleNoScaleAdjustBox` | the default: the box takes the size of the image |
| `ui.ImageBoxScaleNoScaleImageInLeftTop` | the image as is, at the top left |
| `ui.ImageBoxScaleNoScaleImageInCenter` | the image as is, centered |
| `ui.ImageBoxScaleStretchImage` | stretched to the box |
| `ui.ImageBoxScaleAdjustImageKeepAspectRatio` | fit to the box, keeping its proportions |

For an image the user zooms and moves, see [ZoomView](zoomview.md).
