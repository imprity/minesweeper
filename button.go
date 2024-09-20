package main

import (
	"image/color"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	//ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type ButtonState int

const (
	ButtonStateNormal ButtonState = iota
	ButtonStateHover
	ButtonStateDown
)

type BaseButton struct {
	Rect FRectangle

	Disabled bool

	OnClick func()

	RepeateOnHold         bool
	FirstRate, RepeatRate time.Duration

	State ButtonState
}

func (b *BaseButton) Update() {
	if b.Disabled {
		b.State = ButtonStateNormal
		return
	}

	pt := CursorFPt()

	if pt.In(b.Rect) {
		// handle callback
		if b.RepeateOnHold {
			if HandleMouseButtonRepeat(
				b.FirstRate, b.RepeatRate, eb.MouseButtonLeft,
			) {
				if b.OnClick != nil {
					b.OnClick()
				}
			}
		} else {
			if IsMouseButtonJustPressed(eb.MouseButtonLeft) {
				if b.OnClick != nil {
					b.OnClick()
				}
			}
		}

		// handle state
		if IsMouseButtonPressed(eb.MouseButtonLeft) {
			b.State = ButtonStateDown
		} else {
			b.State = ButtonStateHover
		}
	} else {
		b.State = ButtonStateNormal
	}
}

type ImageButton struct {
	BaseButton

	Image        *eb.Image
	ImageOnHover *eb.Image
	ImageOnDown  *eb.Image

	ImageColor        color.NRGBA
	ImageColorOnHover color.NRGBA
	ImageColorOnDown  color.NRGBA
}

func (b *ImageButton) Draw(dst *eb.Image) {
	// TEST TEST TEST TEST TEST TEST
	// draw button rect
	StrokeRect(
		dst, b.Rect, 3, color.NRGBA{255, 0, 0, 255}, true,
	)
	// TEST TEST TEST TEST TEST TEST

	var img *eb.Image

	switch b.BaseButton.State {
	case ButtonStateNormal:
		img = b.Image
	case ButtonStateHover:
		img = b.ImageOnHover
	case ButtonStateDown:
		img = b.ImageOnDown
	}

	if img != nil {
		op := &eb.DrawImageOptions{}

		imageSize := ImageSizeFPt(img)
		scale := float64(1)

		scale = min(b.Rect.Dx()/imageSize.X, b.Rect.Dy()/imageSize.Y)

		op.GeoM.Concat(TransformToCenter(imageSize.X, imageSize.Y, scale, scale, 0))
		rectCenter := FRectangleCenter(b.Rect)
		op.GeoM.Translate(rectCenter.X, rectCenter.Y)

		op.Filter = eb.FilterLinear

		var imageColor color.NRGBA

		switch b.BaseButton.State {
		case ButtonStateNormal:
			imageColor = b.ImageColor
		case ButtonStateHover:
			imageColor = b.ImageColorOnHover
		case ButtonStateDown:
			imageColor = b.ImageColorOnDown
		}

		op.ColorScale.ScaleWithColor(imageColor)

		dst.DrawImage(img, op)
	}
}
