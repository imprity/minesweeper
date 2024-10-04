package main

import (
	"image/color"
	"math"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

func GetRectPath(rect FRectangle) *ebv.Path {
	path := &ebv.Path{}
	path.MoveTo(f32(rect.Min.X), f32(rect.Min.Y))
	path.LineTo(f32(rect.Max.X), f32(rect.Min.Y))
	path.LineTo(f32(rect.Max.X), f32(rect.Max.Y))
	path.LineTo(f32(rect.Min.X), f32(rect.Max.Y))
	path.Close()
	return path
}

func DrawFilledRect(
	dst *eb.Image,
	rect FRectangle,
	clr color.Color,
	antialias bool,
) {
	path := GetRectPath(rect)
	DrawFilledPath(dst, path, clr, antialias)
}

func StrokeRect(
	dst *eb.Image,
	rect FRectangle,
	strokeWidth float64,
	clr color.Color,
	antialias bool,
) {
	path := GetRectPath(rect)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.MiterLimit = 4
	strokeOp.Width = float32(strokeWidth)
	StrokePath(dst, path, strokeOp, clr, antialias)
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
func GetRoundRectPath(rect FRectangle, radiuses [4]float64) *ebv.Path {
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

	path := &ebv.Path{}

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
	path := GetRoundRectPath(rect, [4]float64{radius, radius, radius, radius})
	DrawFilledPath(dst, path, clr, antialias)
}

func StrokeRoundRect(
	dst *eb.Image,
	rect FRectangle,
	radius float64,
	stroke float64,
	clr color.Color,
	antialias bool,
) {
	path := GetRoundRectPath(rect, [4]float64{radius, radius, radius, radius})
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.MiterLimit = 4
	strokeOp.Width = float32(stroke)
	StrokePath(dst, path, strokeOp, clr, antialias)
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
	path := GetRoundRectPath(rect, radiuses)
	DrawFilledPath(dst, path, clr, antialias)
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
	path := GetRoundRectPath(rect, radiuses)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.Width = float32(stroke)
	strokeOp.MiterLimit = 4
	StrokePath(dst, path, strokeOp, clr, antialias)
}

func FastArc(p *ebv.Path, x, y, radius, startAngle, endAngle float64, dir ebv.Direction, segments int) {
	if segments == 0 {
		compass := FPt(radius, 0)

		start := compass.Rotate(startAngle).Add(FPt(x, y))
		end := compass.Rotate(endAngle).Add(FPt(x, y))

		p.LineTo(f32(start.X), f32(start.Y))
		p.LineTo(f32(end.X), f32(end.Y))

		return
	}

	// copy pasted from ebiten Arc function
	// Adjust the angles.
	var da float64
	if dir == ebv.Clockwise {
		for startAngle > endAngle {
			endAngle += 2 * math.Pi
		}
		da = float64(endAngle - startAngle)
	} else {
		for startAngle < endAngle {
			startAngle += 2 * math.Pi
		}
		da = float64(startAngle - endAngle)
	}

	if da >= 2*math.Pi {
		da = 2 * math.Pi
		if dir == ebv.Clockwise {
			endAngle = startAngle + 2*math.Pi
		} else {
			startAngle = endAngle + 2*math.Pi
		}
	}

	compass := FPt(radius, 0)
	arcCenter := FPt(x, y)

	start := compass.Rotate(startAngle).Add(arcCenter)

	p.LineTo(f32(start.X), f32(start.Y))

	segmentAngle := da / f64(segments+1)
	angle := startAngle

	for range segments {
		angle += segmentAngle
		v := compass.Rotate(angle).Add(arcCenter)
		p.LineTo(f32(v.X), f32(v.Y))
	}

	end := compass.Rotate(endAngle).Add(arcCenter)
	p.LineTo(f32(end.X), f32(end.Y))
}

func DrawVerticies(
	dst *eb.Image,
	vs []eb.Vertex,
	is []uint16,
	clr color.Color,
	antialias bool,
	fillRule eb.FillRule,
) {
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
	op.FillRule = fillRule
	dst.DrawTriangles(vs, is, WhiteImage, op)
}

func DrawFilledPath(
	dst *eb.Image,
	path *ebv.Path,
	clr color.Color,
	antialias bool,
) {
	DrawFilledPathEx(
		dst,
		path,
		clr,
		antialias,
		eb.FillAll,
	)
}

func StrokePath(
	dst *eb.Image,
	path *ebv.Path,
	strokeOp *ebv.StrokeOptions,
	clr color.Color,
	antialias bool,
) {
	StrokePathEx(
		dst,
		path,
		strokeOp,
		clr,
		antialias,
		eb.FillAll,
	)
}

func DrawFilledPathEx(
	dst *eb.Image,
	path *ebv.Path,
	clr color.Color,
	antialias bool,
	fillRule eb.FillRule,
) {
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	DrawVerticies(dst, vs, is, clr, antialias, fillRule)
}

func StrokePathEx(
	dst *eb.Image,
	path *ebv.Path,
	strokeOp *ebv.StrokeOptions,
	clr color.Color,
	antialias bool,
	fillRule eb.FillRule,
) {
	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, strokeOp)
	DrawVerticies(dst, vs, is, clr, antialias, fillRule)
}
