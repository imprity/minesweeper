package main

import (
	"image/color"
	"math"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	vectorVertexBuffer  []eb.Vertex = make([]eb.Vertex, 0, 256)
	vectorIndicesBuffer []uint16    = make([]uint16, 0, 256)
)

func StrokeLine(
	dst *eb.Image,
	x0, y0, x1, y1 float64,
	strokeWidth float64,
	clr color.Color,
) {
	// TODO : this is slow
	p := &ebv.Path{}
	p.MoveTo(f32(x0), f32(y0))
	p.LineTo(f32(x1), f32(y1))
	StrokePath(dst, p, &ebv.StrokeOptions{Width: f32(strokeWidth)}, clr)
}

func FillRect(
	dst *eb.Image,
	rect FRectangle,
	clr color.Color,
) {
	verts := [4]eb.Vertex{
		{DstX: f32(rect.Min.X), DstY: f32(rect.Min.Y)},
		{DstX: f32(rect.Max.X), DstY: f32(rect.Min.Y)},
		{DstX: f32(rect.Max.X), DstY: f32(rect.Max.Y)},
		{DstX: f32(rect.Min.X), DstY: f32(rect.Max.Y)},
	}
	indices := [6]uint16{
		0, 1, 2, 0, 2, 3,
	}
	DrawVerticies(dst, verts[:], indices[:], clr, eb.FillRuleFillAll)
}

func StrokeRect(
	dst *eb.Image,
	rect FRectangle,
	strokeWidth float64,
	clr color.Color,
) {
	path := GetRectPath(rect)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.MiterLimit = 4
	strokeOp.Width = float32(strokeWidth)
	StrokePath(dst, path, strokeOp, clr)
}

func FillCircle(
	dst *eb.Image,
	x, y, r float64,
	clr color.Color,
) {
	path := &ebv.Path{}
	path.Arc(f32(x), f32(y), f32(r), 0, Pi*2, ebv.Clockwise)
	FillPath(dst, path, clr)
}

func StrokeCircle(
	dst *eb.Image,
	x, y, r float64,
	strokeWidth float64,
	clr color.Color,
) {
	path := &ebv.Path{}
	path.Arc(f32(x), f32(y), f32(r), 0, Pi*2, ebv.Clockwise)

	strokeOp := &ebv.StrokeOptions{}
	strokeOp.Width = f32(strokeWidth)
	strokeOp.MiterLimit = 10

	StrokePath(dst, path, strokeOp, clr)
}

func FillRoundRect(
	dst *eb.Image,
	rect FRectangle,
	radius float64,
	radiusInPixels bool,
	clr color.Color,
) {
	path := GetRoundRectPath(rect, radius, radiusInPixels)
	FillPath(dst, path, clr)
}

func StrokeRoundRect(
	dst *eb.Image,
	rect FRectangle,
	radius float64,
	radiusInPixels bool,
	stroke float64,
	clr color.Color,
) {
	path := GetRoundRectPath(rect, radius, radiusInPixels)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.MiterLimit = 4
	strokeOp.Width = float32(stroke)
	StrokePath(dst, path, strokeOp, clr)
}

func FillRoundRectFast(
	dst *eb.Image,
	rect FRectangle,
	radius float64,
	radiusInPixels bool,
	segments int,
	clr color.Color,
) {
	path := GetRoundRectPathFast(
		rect,
		radius,
		radiusInPixels,
		segments,
	)
	FillPath(dst, path, clr)
}

func StrokeRoundRectFast(
	dst *eb.Image,
	rect FRectangle,
	radius float64,
	radiusInPixels bool,
	segments int,
	stroke float64,
	clr color.Color,
) {
	path := GetRoundRectPathFast(
		rect,
		radius,
		radiusInPixels,
		segments,
	)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.MiterLimit = 4
	strokeOp.Width = float32(stroke)
	StrokePath(dst, path, strokeOp, clr)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func FillRoundRectEx(
	dst *eb.Image,
	rect FRectangle,
	radiuses [4]float64,
	radiusInPixels bool,
	clr color.Color,
) {
	path := GetRoundRectPathEx(rect, radiuses, radiusInPixels)
	FillPath(dst, path, clr)
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
	radiusInPixels bool,
	stroke float64,
	clr color.Color,
) {
	path := GetRoundRectPathEx(rect, radiuses, radiusInPixels)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.Width = float32(stroke)
	strokeOp.MiterLimit = 4
	StrokePath(dst, path, strokeOp, clr)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func FillRoundRectFastEx(
	dst *eb.Image,
	rect FRectangle,
	radiuses [4]float64,
	radiusInPixels bool,
	segments [4]int,
	clr color.Color,
) {
	path := GetRoundRectPathFastEx(rect, radiuses, radiusInPixels, segments)
	FillPath(dst, path, clr)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func StrokeRoundRectFastEx(
	dst *eb.Image,
	rect FRectangle,
	radiuses [4]float64,
	radiusInPixels bool,
	segments [4]int,
	stroke float64,
	clr color.Color,
) {
	path := GetRoundRectPathFastEx(rect, radiuses, radiusInPixels, segments)
	strokeOp := &ebv.StrokeOptions{}
	strokeOp.Width = float32(stroke)
	strokeOp.MiterLimit = 4
	StrokePath(dst, path, strokeOp, clr)
}

func DrawVerticies(
	dst *eb.Image,
	vs []eb.Vertex,
	is []uint16,
	clr color.Color,
	fillRule eb.FillRule,
) {
	r, g, b, a := clr.RGBA()

	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff
	af := float32(a) / 0xffff

	for i := range vs {
		vs[i].SrcX = 1
		vs[i].SrcY = 1
		vs[i].ColorR = rf
		vs[i].ColorG = gf
		vs[i].ColorB = bf
		vs[i].ColorA = af
	}

	op := &DrawTrianglesOptions{}
	op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
	op.FillRule = fillRule
	DrawTriangles(dst, vs, is, WhiteImage, op)
}

func FillPath(
	dst *eb.Image,
	path *ebv.Path,
	clr color.Color,
) {
	FillPathEx(
		dst,
		path,
		clr,
		eb.FillAll,
	)
}

func StrokePath(
	dst *eb.Image,
	path *ebv.Path,
	strokeOp *ebv.StrokeOptions,
	clr color.Color,
) {
	StrokePathEx(
		dst,
		path,
		strokeOp,
		clr,
		eb.FillAll,
	)
}

func FillPathEx(
	dst *eb.Image,
	path *ebv.Path,
	clr color.Color,
	fillRule eb.FillRule,
) {
	vectorVertexBuffer = vectorVertexBuffer[:0]
	vectorIndicesBuffer = vectorIndicesBuffer[:0]
	vectorVertexBuffer, vectorIndicesBuffer = path.AppendVerticesAndIndicesForFilling(vectorVertexBuffer, vectorIndicesBuffer)
	DrawVerticies(dst, vectorVertexBuffer, vectorIndicesBuffer, clr, fillRule)
}

func StrokePathEx(
	dst *eb.Image,
	path *ebv.Path,
	strokeOp *ebv.StrokeOptions,
	clr color.Color,
	fillRule eb.FillRule,
) {
	vectorVertexBuffer = vectorVertexBuffer[:0]
	vectorIndicesBuffer = vectorIndicesBuffer[:0]
	vectorVertexBuffer, vectorIndicesBuffer = path.AppendVerticesAndIndicesForStroke(vectorVertexBuffer, vectorIndicesBuffer, strokeOp)
	DrawVerticies(dst, vectorVertexBuffer, vectorIndicesBuffer, clr, fillRule)
}

// =================
// path functions
// =================

// raidus and segments array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func getRoundRectPathImpl(
	rect FRectangle, radiuses [4]float64, segments [4]int, useSegments bool,
) *ebv.Path {
	radiusMax := min(rect.Dx()*0.5, rect.Dy()*0.5)

	//clamp the radius to the size of rect
	for i, v := range radiuses {
		radiuses[i] = min(v, radiusMax)
		if radiuses[i] < 1.5 {
			radiuses[i] = 0
		}
	}

	inLeftTop := FPt(rect.Min.X+radiuses[0], rect.Min.Y+radiuses[0])
	inRightTop := FPt(rect.Max.X-radiuses[1], rect.Min.Y+radiuses[1])
	inRightBottom := FPt(rect.Max.X-radiuses[2], rect.Max.Y-radiuses[2])
	inLeftBottom := FPt(rect.Min.X+radiuses[3], rect.Max.Y-radiuses[3])

	const (
		d0   = Pi * 0.0
		d90  = Pi * 0.5
		d180 = Pi * 1.0
		d270 = Pi * 1.5
		d360 = Pi * 2.0
	)

	path := &ebv.Path{}

	if segments[0] > 1 && radiuses[0] > 1.5 {
		if useSegments {
			ArcFast(
				path,
				(inLeftTop.X), (inLeftTop.Y),
				(radiuses[0]),
				d180, d270,
				ebv.Clockwise,
				segments[0],
			)
		} else {
			path.Arc(
				f32(inLeftTop.X), f32(inLeftTop.Y),
				f32(radiuses[0]),
				d180, d270,
				ebv.Clockwise,
			)
		}
	} else {
		path.LineTo(f32(rect.Min.X), f32(rect.Min.Y))
	}
	path.LineTo(f32(inRightTop.X), f32(inRightTop.Y-radiuses[1]))

	if segments[1] > 1 && radiuses[1] > 1.5 {
		if useSegments {
			ArcFast(
				path,
				(inRightTop.X), (inRightTop.Y),
				(radiuses[1]),
				d270, d0,
				ebv.Clockwise,
				segments[1],
			)
		} else {
			path.Arc(
				f32(inRightTop.X), f32(inRightTop.Y),
				f32(radiuses[1]),
				d270, d0,
				ebv.Clockwise,
			)
		}
	} else {
		path.LineTo(f32(rect.Max.X), f32(rect.Min.Y))
	}
	path.LineTo(f32(inRightBottom.X+radiuses[2]), f32(inRightBottom.Y))

	if segments[2] > 1 && radiuses[2] > 1.5 {
		if useSegments {
			ArcFast(
				path,
				(inRightBottom.X), (inRightBottom.Y),
				(radiuses[2]),
				d0, d90,
				ebv.Clockwise,
				segments[2],
			)
		} else {
			path.Arc(
				f32(inRightBottom.X), f32(inRightBottom.Y),
				f32(radiuses[2]),
				d0, d90,
				ebv.Clockwise,
			)
		}
	} else {
		path.LineTo(f32(rect.Max.X), f32(rect.Max.Y))
	}
	path.LineTo(f32(inLeftBottom.X), f32(inLeftBottom.Y+radiuses[3]))

	if segments[3] > 1 && radiuses[3] > 1.5 {
		if useSegments {
			ArcFast(
				path,
				(inLeftBottom.X), (inLeftBottom.Y),
				(radiuses[3]),
				d90, d180,
				ebv.Clockwise,
				segments[3],
			)
		} else {
			path.Arc(
				f32(inLeftBottom.X), f32(inLeftBottom.Y),
				f32(radiuses[3]),
				d90, d180,
				ebv.Clockwise,
			)
		}
	} else {
		path.LineTo(f32(rect.Min.X), f32(rect.Max.Y))
	}
	path.Close()

	return path
}

func GetRectPath(rect FRectangle) *ebv.Path {
	path := &ebv.Path{}
	path.MoveTo(f32(rect.Min.X), f32(rect.Min.Y))
	path.LineTo(f32(rect.Max.X), f32(rect.Min.Y))
	path.LineTo(f32(rect.Max.X), f32(rect.Max.Y))
	path.LineTo(f32(rect.Min.X), f32(rect.Max.Y))
	path.Close()
	return path
}

func GetRoundRectPath(
	rect FRectangle,
	radius float64,
	radiusInPixels bool,
) *ebv.Path {
	return GetRoundRectPathEx(
		rect,
		[4]float64{radius, radius, radius, radius},
		radiusInPixels,
	)
}

func GetRoundRectPathFast(
	rect FRectangle,
	radius float64,
	radiusInPixels bool,
	segments int,
) *ebv.Path {
	return GetRoundRectPathFastEx(
		rect,
		[4]float64{radius, radius, radius, radius},
		radiusInPixels,
		[4]int{segments, segments, segments, segments},
	)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func GetRoundRectPathEx(
	rect FRectangle,
	radiuses [4]float64,
	radiusInPixels bool,
) *ebv.Path {
	if !radiusInPixels {
		toPx := min(rect.Dx(), rect.Dy()) * 0.5
		for i := range 4 {
			radiuses[i] = radiuses[i] * toPx
		}
	}
	return getRoundRectPathImpl(rect, radiuses, [4]int{}, false)
}

// raidus array maps like this
//
//	0 --- 1
//	|     |
//	|     |
//	3 --- 2
func GetRoundRectPathFastEx(
	rect FRectangle,
	radiuses [4]float64,
	radiusInPixels bool,
	segments [4]int,
) *ebv.Path {
	if !radiusInPixels {
		toPx := min(rect.Dx(), rect.Dy()) * 0.5
		for i := range 4 {
			radiuses[i] = radiuses[i] * toPx
		}
	}
	return getRoundRectPathImpl(rect, radiuses, segments, true)
}

func ArcFast(p *ebv.Path, x, y, radius, startAngle, endAngle float64, dir ebv.Direction, segments int) {
	if segments <= 1 {
		compass := FPt(radius, 0)
		compass = compass.Rotate((startAngle + endAngle) * 0.5).Add(FPt(x, y))

		p.LineTo(f32(compass.X), f32(compass.Y))

		return
	} else if segments <= 2 {
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
