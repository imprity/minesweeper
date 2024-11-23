package main

import (
	"image/color"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
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

	// If ActOnRelease is true,
	// insted of OnClick being called on press
	// it's called when button is released.
	//
	// Does nothing if RepeateOnHold is true
	ActOnRelease        bool
	ReadyToActOnRelease bool

	State ButtonState
}

func (b *BaseButton) Update() {
	// TODO : This whole shit is a god damn mess!!!
	if b.Disabled {
		b.State = ButtonStateNormal
		b.ReadyToActOnRelease = false
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
				if b.ActOnRelease {
					b.ReadyToActOnRelease = true
				} else {
					if b.OnClick != nil {
						b.OnClick()
					}
				}
			}
		}

		if b.ActOnRelease && b.ReadyToActOnRelease && IsMouseButtonJustReleased(eb.MouseButtonLeft) {
			if b.OnClick != nil {
				b.OnClick()
			}
			b.ReadyToActOnRelease = false
		}

		// handle state
		if IsMouseButtonPressed(eb.MouseButtonLeft) {
			b.ReadyToActOnRelease = true
			b.State = ButtonStateDown
		} else {
			b.State = ButtonStateHover
		}
	} else {
		b.State = ButtonStateNormal
		b.ReadyToActOnRelease = false
	}

	if !b.ActOnRelease {
		b.ReadyToActOnRelease = false
	}
}

type ImageButton struct {
	BaseButton

	Image        SubView
	ImageOnHover SubView
	ImageOnDown  SubView

	ImageColor        color.Color
	ImageColorOnHover color.Color
	ImageColorOnDown  color.Color
}

func NewImageButton() *ImageButton {
	b := new(ImageButton)

	b.ImageColor = color.NRGBA{255, 255, 255, 255}
	b.ImageColorOnHover = color.NRGBA{255, 255, 255, 255}
	b.ImageColorOnDown = color.NRGBA{255, 255, 255, 255}

	return b
}

func (b *ImageButton) Draw(dst *eb.Image) {
	var img SubView

	switch b.BaseButton.State {
	case ButtonStateNormal:
		img = b.Image
	case ButtonStateHover:
		img = b.ImageOnHover
	case ButtonStateDown:
		img = b.ImageOnDown
	}

	if img.Image != nil {
		op := &DrawSubViewOptions{}

		imageSize := ImageSizeFPt(img)
		scale := float64(1)

		scale = min(b.Rect.Dx()/imageSize.X, b.Rect.Dy()/imageSize.Y)

		op.GeoM.Concat(TransformToCenter(imageSize.X, imageSize.Y, scale, scale, 0))
		rectCenter := FRectangleCenter(b.Rect)
		op.GeoM.Translate(rectCenter.X, rectCenter.Y)

		var imageColor color.Color

		switch b.BaseButton.State {
		case ButtonStateNormal:
			imageColor = b.ImageColor
		case ButtonStateHover:
			imageColor = b.ImageColorOnHover
		case ButtonStateDown:
			imageColor = b.ImageColorOnDown
		}

		op.ColorScale.ScaleWithColor(imageColor)

		DrawSubView(dst, img, op)
	}
}

type TextButton struct {
	BaseButton

	Text string

	BgColor        color.Color
	BgColorOnHover color.Color
	BgColorOnDown  color.Color

	TextColor        color.Color
	TextColorOnHover color.Color
	TextColorOnDown  color.Color
}

var DefaultTextButton = TextButton{
	Text: "Button",

	BgColor:        color.NRGBA{0x68, 0x84, 0x97, 255},
	BgColorOnHover: color.NRGBA{0x51, 0x99, 0xCC, 255},
	BgColorOnDown:  color.NRGBA{0x8D, 0xBC, 0xDE, 255},

	TextColor:        color.NRGBA{255, 255, 255, 255},
	TextColorOnHover: color.NRGBA{255, 255, 255, 255},
	TextColorOnDown:  color.NRGBA{255, 255, 255, 255},
}

func NewTextButton() *TextButton {
	copy := DefaultTextButton
	return &copy
}

func (b *TextButton) Draw(dst *eb.Image) {
	// determine color
	var bgColor color.Color = color.NRGBA{}
	var textColor color.Color = color.NRGBA{}

	switch b.BaseButton.State {
	case ButtonStateNormal:
		bgColor = b.BgColor
		textColor = b.TextColor
	case ButtonStateHover:
		bgColor = b.BgColorOnHover
		textColor = b.TextColorOnHover
	case ButtonStateDown:
		bgColor = b.BgColorOnDown
		textColor = b.TextColorOnDown
	}

	// draw background color
	FillRect(dst, b.Rect, bgColor)

	// draw text color
	if len(b.Text) > 0 {
		textW, textH := ebt.Measure(b.Text, ClearFace, FontLineSpacing(ClearFace))

		scale := min(b.Rect.Dx()*0.9/textW, b.Rect.Dy()*0.9/textH)

		op := &DrawTextOptions{}
		op.ColorScale.ScaleWithColor(textColor)

		op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
		center := FRectangleCenter(b.Rect)
		op.GeoM.Translate(center.X, center.Y)

		DrawText(dst, b.Text, ClearFace, op)
	}
}
