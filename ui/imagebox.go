package ui

import (
	"image"
)

type ImageBox struct {
	Widget
	image   image.Image
	mip     *MipImage // the image drawn scaled: smooth when much smaller
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

// SetImage sets the image; after its pixels change, set it again
func (c *ImageBox) SetImage(img image.Image) {
	c.image = img
	c.mip = nil
	if img != nil {
		c.mip = NewMipImage(img)
	}
	c.updateInnerSize()
	c.form.UpdateLayout()
	c.Update()
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
	c.Update()
}

func (c *ImageBox) draw(cnv *Canvas) {
	if c.image == nil {
		return
	}
	b := c.image.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	switch c.scaling {
	case ImageBoxScaleNoScaleImageInLeftTop, ImageBoxScaleNoScaleAdjustBox:
		cnv.DrawImage(0, 0, c.image)
	case ImageBoxScaleNoScaleImageInCenter:
		cnv.DrawImage((c.Width()-w)/2, (c.Height()-h)/2, c.image)
	case ImageBoxScaleStretchImage:
		cnv.DrawImageRect(0, 0, c.Width(), c.Height(), c.mip, b, true)
	case ImageBoxScaleAdjustImageKeepAspectRatio:
		// The largest size of the image's proportions that fits, centered
		dw, dh := c.Width(), c.Width()*h/w
		if dh > c.Height() {
			dw, dh = c.Height()*w/h, c.Height()
		}
		cnv.DrawImageRect((c.Width()-dw)/2, (c.Height()-dh)/2, dw, dh, c.mip, b, true)
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
		return c.image.Bounds().Dx()
	}
	return c.Widget.MinWidth()
}

func (c *ImageBox) MinHeight() int {
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image == nil {
			return 0
		}
		return c.image.Bounds().Dy()
	}
	return c.Widget.MinHeight()
}

func (c *ImageBox) MaxWidth() int {
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image == nil {
			return 0
		}
		return c.image.Bounds().Dx()
	}
	return c.Widget.MaxWidth()
}

func (c *ImageBox) MaxHeight() int {
	if c.scaling == ImageBoxScaleNoScaleAdjustBox {
		if c.image == nil {
			return 0
		}
		return c.image.Bounds().Dy()
	}
	return c.Widget.MaxHeight()
}
