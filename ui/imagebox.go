package ui

import (
	"image"

	"github.com/nfnt/resize"
)

type ImageBox struct {
	Widget
	image   image.Image
	scaling ImageBoxScale
}

type ImageBoxScale int

const (
	ImageBoxScaleNoScaleAdjustBox           ImageBoxScale = 0
	ImageBoxScaleNoScaleImageInLeftTop      ImageBoxScale = 1
	ImageBoxScaleNoScaleImageInCenter       ImageBoxScale = 2
	ImageBoxScaleStretchImage               ImageBoxScale = 3
	ImageBoxScaleAdjustImageKeepAspectRatio ImageBoxScale = 4
)

func NewImageBox() *ImageBox {
	var c ImageBox
	c.InitWidget()
	c.SetTypeName("ImageBox")
	c.SetOnPaint(c.draw)
	c.scaling = ImageBoxScaleNoScaleAdjustBox

	return &c
}

func (c *ImageBox) Image() image.Image {
	return c.image
}

func (c *ImageBox) SetImage(img image.Image) {
	c.image = img
	c.updateInnerSize()
	c.form.UpdateLayout()
}

func (c *ImageBox) Scaling() ImageBoxScale {
	return c.scaling
}

func (c *ImageBox) SetScaling(scaling ImageBoxScale) {
	if c.scaling == scaling {
		return
	}
	c.scaling = scaling
	c.updateInnerSize()
	c.form.UpdateLayout()
}

func (c *ImageBox) draw(cnv *Canvas) {
	if c.image == nil {
		return
	}

	if c.scaling == ImageBoxScaleNoScaleImageInLeftTop || c.scaling == ImageBoxScaleNoScaleAdjustBox {
		//b := c.img.Bounds()
		cnv.DrawImage(0, 0, c.image)
	}

	if c.scaling == ImageBoxScaleNoScaleImageInCenter {
		b := c.image.Bounds()
		offsetX := (c.Width() - b.Max.X) / 2
		offsetY := (c.Height() - b.Max.Y) / 2
		cnv.DrawImage(offsetX, offsetY, c.image)
	}

	if c.scaling == ImageBoxScaleStretchImage {
		image := resize.Resize(uint(c.Width()), uint(c.Height()), c.image, resize.Bicubic)
		cnv.DrawImage(0, 0, image)
	}

	if c.scaling == ImageBoxScaleAdjustImageKeepAspectRatio {
		b := c.image.Bounds()
		aspRatioImg := float64(b.Max.X) / float64(b.Max.Y)
		aspRationWidget := float64(c.Width()) / float64(c.Height())
		if aspRatioImg > aspRationWidget {
			image := resize.Resize(uint(c.Width()), 0, c.image, resize.Bicubic)
			b := image.Bounds()
			offsetX := (c.Width() - b.Max.X) / 2
			offsetY := (c.Height() - b.Max.Y) / 2
			cnv.DrawImage(offsetX, offsetY, image)
		} else {
			image := resize.Resize(0, uint(c.Height()), c.image, resize.Bicubic)
			b := image.Bounds()
			offsetX := (c.Width() - b.Max.X) / 2
			offsetY := (c.Height() - b.Max.Y) / 2
			cnv.DrawImage(offsetX, offsetY, image)
		}
	}
}

func (c *ImageBox) updateInnerSize() {
	minWidth := 0
	minHeight := 0
	maxWidth := 10000
	maxHeight := 10000
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image != nil {
			minWidth = c.image.Bounds().Dx()
			minHeight = c.image.Bounds().Dy()
			maxWidth = minWidth
			maxHeight = minHeight
		}
	}
	c.SetMinSize(minWidth, minHeight)
	c.SetMaxSize(maxWidth, maxHeight)
}

func (c *ImageBox) MinWidth() int {
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image == nil {
			return 0
		}
		return c.image.Bounds().Max.X
	}
	return c.Widget.MinWidth()
}

func (c *ImageBox) MinHeight() int {
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image == nil {
			return 0
		}
		return c.image.Bounds().Max.Y
	}
	return c.Widget.MinHeight()
}

func (c *ImageBox) MaxWidth() int {
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image == nil {
			return 0
		}
		return c.image.Bounds().Max.X
	}
	return c.Widget.MaxWidth()
}

func (c *ImageBox) MaxHeight() int {
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image == nil {
			return 0
		}
		return c.image.Bounds().Max.Y
	}
	return c.Widget.MaxHeight()
}
