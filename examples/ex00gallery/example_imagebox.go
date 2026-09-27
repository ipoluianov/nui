package ex00gallery

import (
	"fmt"
	"image"
	"image/color"

	"github.com/ipoluianov/nui/ui"
)

type ExamplePageImageBox struct {
	ui.Widget

	lblStatus *ui.Label

	imgTry    *ui.ImageBox
	cbScaling *ui.ComboBox
}

func NewExamplePageImageBox() *ExamplePageImageBox {
	var c ExamplePageImageBox
	c.InitWidget()

	sample := newSampleImage(96, 64)

	row := 0

	c.lblStatus = c.AddLabel(row, 0, "ImageBox draws an image.Image using one of several scaling modes")
	c.lblStatus.SetUnderline(true)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "ImageBoxScaleNoScaleAdjustBox - the box shrinks/grows to fit the image")
	imgAdjust := ui.NewImageBox()
	imgAdjust.SetScaling(ui.ImageBoxScaleNoScaleAdjustBox)
	imgAdjust.SetImage(sample)
	c.AddWidget(row, 0, imgAdjust)
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "ImageBoxScaleNoScaleImageInLeftTop vs ImageBoxScaleNoScaleImageInCenter")
	imgTopLeft := newFramedImageBox(sample, ui.ImageBoxScaleNoScaleImageInLeftTop, 220, 140)
	c.AddWidget(row, 0, imgTopLeft)
	imgCenter := newFramedImageBox(sample, ui.ImageBoxScaleNoScaleImageInCenter, 220, 140)
	c.AddWidget(row, 1, imgCenter)
	row++
	c.AddLabel(row, 0, "Left/top corner")
	c.AddLabel(row, 1, "Centered")
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "ImageBoxScaleStretchImage vs ImageBoxScaleAdjustImageKeepAspectRatio")
	imgStretch := newFramedImageBox(sample, ui.ImageBoxScaleStretchImage, 220, 140)
	c.AddWidget(row, 0, imgStretch)
	imgAspect := newFramedImageBox(sample, ui.ImageBoxScaleAdjustImageKeepAspectRatio, 220, 140)
	c.AddWidget(row, 1, imgAspect)
	row++
	c.AddLabel(row, 0, "Stretched to fill (distorted)")
	c.AddLabel(row, 1, "Fit, aspect ratio kept (letterboxed)")
	row = addSectionGap(&c.Widget, row+1)

	row = addSectionHeader(&c.Widget, row, "Pick a scaling mode")
	c.imgTry = newFramedImageBox(sample, ui.ImageBoxScaleNoScaleAdjustBox, 220, 140)
	c.AddWidget(row, 0, c.imgTry)

	c.cbScaling = ui.NewComboBox()
	c.cbScaling.AddItem("No scale, adjust box", ui.ImageBoxScaleNoScaleAdjustBox)
	c.cbScaling.AddItem("No scale, top-left", ui.ImageBoxScaleNoScaleImageInLeftTop)
	c.cbScaling.AddItem("No scale, centered", ui.ImageBoxScaleNoScaleImageInCenter)
	c.cbScaling.AddItem("Stretch to fill", ui.ImageBoxScaleStretchImage)
	c.cbScaling.AddItem("Keep aspect ratio", ui.ImageBoxScaleAdjustImageKeepAspectRatio)
	c.cbScaling.SetSelectedIndex(0)
	c.AddWidget(row, 1, c.cbScaling)

	c.AddButton(row, 2, "Apply", func() {
		scaling := c.cbScaling.SelectedItemData().(ui.ImageBoxScale)
		c.imgTry.SetScaling(scaling)
		// SetScaling resets min/max size to fit the image (for AdjustBox) or
		// to no constraint at all (every other mode) - reassert the fixed
		// frame so the demo box doesn't jump around as the mode changes.
		c.imgTry.SetMinSize(220, 140)
		c.imgTry.SetMaxSize(220, 140)
		c.setStatus(fmt.Sprintf("Scaling mode: %s", c.cbScaling.SelectedItemText()))
	})
	row = addSectionGap(&c.Widget, row+1)

	c.AddVSpacer(row, 0)

	return &c
}

func (c *ExamplePageImageBox) setStatus(text string) {
	c.lblStatus.SetText(text)
}

// newFramedImageBox wraps sample in an ImageBox that is deliberately larger
// than the image itself (width x height), with a visible background, so the
// effect of scaling mode is obvious even where the image doesn't fill the
// whole box (e.g. NoScaleImageInLeftTop/InCenter).
func newFramedImageBox(sample image.Image, scaling ui.ImageBoxScale, width, height int) *ui.ImageBox {
	imgBox := ui.NewImageBox()
	imgBox.SetScaling(scaling)
	imgBox.SetImage(sample)
	imgBox.SetMinSize(width, height)
	imgBox.SetMaxSize(width, height)
	imgBox.SetAutoFillBackground(true)
	imgBox.SetBackgroundColor(imgBox.BackgroundColorWithAddElevation(-2))
	return imgBox
}

// newSampleImage builds a small placeholder picture (a color gradient plus a
// border and a diagonal line) so scaling/cropping/stretching effects are
// easy to see without shipping an actual image asset with the example.
func newSampleImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(255 * x / width),
				G: uint8(255 * y / height),
				B: 160,
				A: 255,
			})
		}
	}

	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for x := 0; x < width; x++ {
		img.Set(x, 0, white)
		img.Set(x, height-1, white)
		img.Set(x, x*height/width, white)
	}
	for y := 0; y < height; y++ {
		img.Set(0, y, white)
		img.Set(width-1, y, white)
	}

	return img
}
