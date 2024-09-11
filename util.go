package main

import (
	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

func CursorPtf() PointF {
	mx, my := eb.CursorPosition()
	return Ptf(f64(mx), f64(my))
}

func DrawTextCentered(
	img *eb.Image, face *ebt.GoTextFace,
	x, y float64, size float64, rotation float64,
	str string,
) {
	w, h := ebt.Measure(str, face, face.Size)
	op := &ebt.DrawOptions{}
	scale := size / face.Size
	op.GeoM.Scale(scale, scale)
	op.GeoM.Concat(RotateAround(w*0.5*scale, h*0.5*scale, rotation))
	op.GeoM.Translate(x-w*0.5*scale, y-h*0.5*scale)
	op.Filter = eb.FilterLinear
	ebt.Draw(img, str, face, op)
}
