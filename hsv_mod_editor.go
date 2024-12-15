package main

import (
	"fmt"
	"image/color"

	eb "github.com/hajimehoshi/ebiten/v2"
)

type HSVmodEditor struct {
	Rect FRectangle

	// 0 : not focused
	// 1 : Hue focused
	// 2 : Saturation focused
	// 3 : Value focused
	Focused int

	Hue        float64 // -Pi to Pi
	Saturation float64 // -1 to 1
	Value      float64 // -1 to 1
}

func NewHSVmodEditor() *HSVmodEditor {
	return new(HSVmodEditor)
}

func (hm *HSVmodEditor) SetToHSVmod(mod HSVmod) {
	hm.Hue = Clamp(mod.Hue, -Pi, Pi)
	hm.Saturation = Clamp(mod.Saturation, -1, 1)
	hm.Value = Clamp(mod.Value, -1, 1)
}

func (hm *HSVmodEditor) GetHSVmod() HSVmod {
	return HSVmod{
		Hue:        Clamp(hm.Hue, -Pi, Pi),
		Saturation: Clamp(hm.Saturation, -1, 1),
		Value:      Clamp(hm.Value, -1, 1),
	}
}

func (hm *HSVmodEditor) getHSVrect() (FRectangle, FRectangle, FRectangle) {
	rectH := hm.Rect.Dy() / 3.0

	hRect := FRectXYWH(
		hm.Rect.Min.X, hm.Rect.Min.Y+rectH*0,
		hm.Rect.Dx(), rectH,
	)
	sRect := FRectXYWH(
		hm.Rect.Min.X, hm.Rect.Min.Y+rectH*1,
		hm.Rect.Dx(), rectH,
	)
	vRect := FRectXYWH(
		hm.Rect.Min.X, hm.Rect.Min.Y+rectH*2,
		hm.Rect.Dx(), rectH,
	)

	return hRect, sRect, vRect
}

func (hm *HSVmodEditor) getTextRect(rect FRectangle) FRectangle {
	return FRectXYWH(
		rect.Min.X, rect.Min.Y,
		rect.Dx()*0.3, rect.Dy(),
	).Inset(3)
}

func (hm *HSVmodEditor) getSliderRect(rect FRectangle) FRectangle {
	sliderRect := FRectXYWH(
		rect.Min.X+rect.Dx()*0.3, rect.Min.Y,
		rect.Dx()*0.7, rect.Dy(),
	)

	return FRectScaleCentered(sliderRect.Inset(3), 1, 0.6)
}

func (hm *HSVmodEditor) Update() {
	hRect, sRect, vRect := hm.getHSVrect()

	if IsMouseButtonPressed(eb.MouseButtonLeft) {
		hSlider := hm.getSliderRect(hRect)
		sSlider := hm.getSliderRect(sRect)
		vSlider := hm.getSliderRect(vRect)

		cursor := CursorFPt()

		if hm.Focused == 0 {
			if cursor.In(hSlider) {
				hm.Focused = 1
			} else if cursor.In(sSlider) {
				hm.Focused = 2
			} else if cursor.In(vSlider) {
				hm.Focused = 3
			}
		}

		switch hm.Focused {
		case 0:
			// pass
		case 1:
			t := (cursor.X - hSlider.Min.X) / hSlider.Dx()
			t = Clamp(t, 0, 1)
			hm.Hue = Lerp(-Pi, Pi, t)
		case 2:
			t := (cursor.X - sSlider.Min.X) / sSlider.Dx()
			t = Clamp(t, 0, 1)
			hm.Saturation = Lerp(-1, 1, t)
		case 3:
			t := (cursor.X - vSlider.Min.X) / vSlider.Dx()
			t = Clamp(t, 0, 1)
			hm.Value = Lerp(-1, 1, t)
		default:
			panic("UNREACHABLE")
		}
	} else {
		hm.Focused = 0
	}
}

func (hm *HSVmodEditor) Draw(dst *eb.Image) {
	// draw background
	FillRect(dst, hm.Rect, color.NRGBA{0, 0, 0, 130})

	hRect, sRect, vRect := hm.getHSVrect()

	hSlider := hm.getSliderRect(hRect)
	sSlider := hm.getSliderRect(sRect)
	vSlider := hm.getSliderRect(vRect)

	// draw sliders
	FillRect(dst, hSlider, color.NRGBA{255, 255, 255, 255})
	FillRect(dst, sSlider, color.NRGBA{255, 255, 255, 255})
	FillRect(dst, vSlider, color.NRGBA{255, 255, 255, 255})

	drawCursor := func(
		rect FRectangle,
		val, vMin, vMax float64,
	) {
		t := (val - vMin) / (vMax - vMin)
		cursorX := rect.Min.X + rect.Dx()*t
		cursorY := rect.Min.Y + rect.Dy()*0.5

		cursorRect := FRectWH(10, rect.Dy()+5)
		cursorRect = CenterFRectangle(cursorRect, cursorX, cursorY)

		FillRect(dst, cursorRect, color.NRGBA{0, 0, 0, 255})
	}

	drawCursor(hSlider, hm.Hue, -Pi, Pi)
	drawCursor(sSlider, hm.Saturation, -1, 1)
	drawCursor(vSlider, hm.Value, -1, 1)

	drawText := func(rect FRectangle, prefix string, val float64) {
		op := &DrawTextOptions{}
		op.GeoM.Concat(
			FitTextInRect(prefix+" -0.00", ClearFace, rect),
		)
		DrawText(
			dst,
			fmt.Sprintf(prefix+" % 0.2f", val),
			ClearFace,
			op,
		)
	}

	hText := hm.getTextRect(hRect)
	sText := hm.getTextRect(sRect)
	vText := hm.getTextRect(vRect)

	drawText(hText, "h", hm.Hue)
	drawText(sText, "s", hm.Saturation)
	drawText(vText, "v", hm.Value)
}
