package main

import (
	eb "github.com/hajimehoshi/ebiten/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
	//"math"
	"image/color"
)

func DrawFilledRect(
	dst *eb.Image,
	rect FRectangle,
	clr color.Color,
	antialias bool,
) {
	ebv.DrawFilledRect(
		dst,
		f32(rect.Min.X), f32(rect.Min.Y), f32(rect.Dx()), f32(rect.Dy()),
		clr,
		antialias,
	)
}

func StrokeRect(
	dst *eb.Image,
	rect FRectangle,
	strokeWidth float64,
	clr color.Color,
	antialias bool,
) {
	ebv.StrokeRect(
		dst,
		f32(rect.Min.X), f32(rect.Min.Y), f32(rect.Dx()), f32(rect.Dy()),
		f32(strokeWidth),
		clr,
		antialias,
	)
}

func DrawFilledCircle(
	dst *eb.Image,
	x, y, r float64,
	clr color.Color,
	antialias bool,
) {
	ebv.DrawFilledCircle(
		dst, f32(x), f32(y), f32(r), clr, antialias)
}

func StrokeCircle(
	dst *eb.Image,
	x, y, r float64,
	strokeWidth float64,
	clr color.Color,
	antialias bool,
) {
	ebv.StrokeCircle(
		dst, f32(x), f32(y), f32(r), f32(strokeWidth), clr, antialias)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func getRoundRectPath(rect FRectangle, radiuses [4]float64) ebv.Path {
	radiusMax := min(rect.Dx()*0.5, rect.Dy()*0.5)

	//clamp the radius to the size of rect
	for i, v := range radiuses {
		radiuses[i] = min(v, radiusMax)
	}

	inLeftTop := FPt(rect.Min.X+radiuses[0], rect.Min.Y+radiuses[0])
	inRightTop := FPt(rect.Max.X-radiuses[1], rect.Min.Y+radiuses[1])
	inRightBottom := FPt(rect.Max.X-radiuses[2], rect.Max.Y-radiuses[2])
	inLeftBottom := FPt(rect.Min.X+radiuses[3], rect.Max.Y-radiuses[3])

	const (
		d0   float32 = Pi * 0.0
		d90  float32 = Pi * 0.5
		d180 float32 = Pi * 1.0
		d270 float32 = Pi * 1.5
		d360 float32 = Pi * 2.0
	)

	var path ebv.Path

	if radiuses[0] != 0 {
		path.Arc(f32(inLeftTop.X), f32(inLeftTop.Y), f32(radiuses[0]), d180, d270, ebv.Clockwise)
	} else {
		path.MoveTo(f32(inLeftTop.X), f32(inLeftTop.Y))
	}
	path.LineTo(f32(inRightTop.X), f32(inRightTop.Y-radiuses[1]))

	if radiuses[1] != 0 {
		path.Arc(f32(inRightTop.X), f32(inRightTop.Y), f32(radiuses[1]), d270, d0, ebv.Clockwise)
	}
	path.LineTo(f32(inRightBottom.X+radiuses[2]), f32(inRightBottom.Y))

	if radiuses[2] != 0 {
		path.Arc(f32(inRightBottom.X), f32(inRightBottom.Y), f32(radiuses[2]), d0, d90, ebv.Clockwise)
	}
	path.LineTo(f32(inLeftBottom.X), f32(inLeftBottom.Y+radiuses[3]))

	if radiuses[3] != 0 {
		path.Arc(f32(inLeftBottom.X), f32(inLeftBottom.Y), f32(radiuses[3]), d90, d180, ebv.Clockwise)
	}
	path.Close()

	return path
}

func DrawFilledRoundRect(
	dst *eb.Image,
	rect FRectangle,
	radius float64,
	clr color.Color,
	antialias bool,
) {
	path := getRoundRectPath(rect, [4]float64{radius, radius, radius, radius})
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	DrawVerticies(dst, vs, is, clr, true)
}

func StrokeRoundRect(
	dst *eb.Image,
	rect FRectangle,
	radius float64,
	stroke float64,
	clr color.Color,
	antialias bool,
) {
	path := getRoundRectPath(rect, [4]float64{radius, radius, radius, radius})
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.Width = float32(stroke)
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)
	DrawVerticies(dst, vs, is, clr, antialias)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func DrawFilledRoundRectEx(
	dst *eb.Image,
	rect FRectangle,
	radiuses [4]float64,
	clr color.Color,
	antialias bool,
) {
	path := getRoundRectPath(rect, radiuses)
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	DrawVerticies(dst, vs, is, clr, true)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func StrokeRoundRectEx(
	dst *eb.Image,
	rect FRectangle,
	radiuses [4]float64,
	stroke float64,
	clr color.Color,
	antialias bool,
) {
	path := getRoundRectPath(rect, radiuses)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.Width = float32(stroke)
	strokeOp.MiterLimit = 4
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)
	DrawVerticies(dst, vs, is, clr, antialias)
}

func DrawVerticies(dst *eb.Image, vs []eb.Vertex, is []uint16, clr color.Color, antialias bool) {
	r, g, b, a := clr.RGBA()
	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}

	op := &eb.DrawTrianglesOptions{}
	op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
	op.AntiAlias = antialias
	dst.DrawTriangles(vs, is, WhiteImage, op)
}
