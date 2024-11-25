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

// ================
// misc
// ================

type TriangleFan struct {
	Vertices    []eb.Vertex
	Indices     []uint16
	IndexStart  uint16
	VertexCount int
}

func NewTriangleFan(vs []eb.Vertex, is []uint16) TriangleFan {
	return TriangleFan{
		Vertices:   vs,
		Indices:    is,
		IndexStart: uint16(len(vs)),
	}
}

func (tf *TriangleFan) ExtendFan(dst FPoint, src FPoint, clr color.Color) ([]eb.Vertex, []uint16) {
	r, g, b, a := clr.RGBA()
	vert := eb.Vertex{
		SrcX: f32(src.X), SrcY: f32(src.Y),
		DstX: f32(dst.X), DstY: f32(dst.Y),
		ColorR: float32(r) / 0xffff,
		ColorG: float32(g) / 0xffff,
		ColorB: float32(b) / 0xffff,
		ColorA: float32(a) / 0xffff,
	}

	if len(tf.Vertices) <= 2 {
		tf.Vertices = append(tf.Vertices, vert)

		if len(tf.Vertices) >= 3 {
			tf.Indices = append(tf.Indices, tf.IndexStart+0, tf.IndexStart+1, tf.IndexStart+2)
		}
		return tf.Vertices, tf.Indices
	}

	i0 := uint16(tf.IndexStart)
	i1 := uint16(tf.Indices[len(tf.Indices)-1])
	i2 := uint16(len(tf.Vertices))

	tf.Indices = append(tf.Indices, i0, i1, i2)
	tf.Vertices = append(tf.Vertices, vert)

	tf.VertexCount++

	return tf.Vertices, tf.Indices
}

func (tf *TriangleFan) CloseFan() ([]eb.Vertex, []uint16) {
	tf.Indices = append(
		tf.Indices,
		tf.IndexStart+0,
		tf.Indices[len(tf.Indices)-1],
		tf.IndexStart+1,
	)

	return tf.Vertices, tf.Indices
}

func addRoundRectVertsImpl(
	vs []eb.Vertex, is []uint16,
	rect FRectangle,
	radiuses [4]float64,
	segments [4]int,
	clr color.Color,
) ([]eb.Vertex, []uint16) {
	radiusMax := min(rect.Dx()*0.5, rect.Dy()*0.5)

	triangleFan := NewTriangleFan(vs, is)

	//clamp the radius to the size of rect
	for i, v := range radiuses {
		radiuses[i] = min(v, radiusMax)
	}

	inLeftTop := FPt(rect.Min.X+radiuses[0], rect.Min.Y+radiuses[0])
	inRightTop := FPt(rect.Max.X-radiuses[1], rect.Min.Y+radiuses[1])
	inRightBottom := FPt(rect.Max.X-radiuses[2], rect.Max.Y-radiuses[2])
	inLeftBottom := FPt(rect.Min.X+radiuses[3], rect.Max.Y-radiuses[3])

	center := FRectangleCenter(rect)

	triangleFan.ExtendFan(center, FPt(1, 1), clr)

	// left top
	if segments[0] <= 1 || radiuses[0] < 1.5 {
		vs, is = triangleFan.ExtendFan(rect.Min, FPt(1, 1), clr)
	} else {
		compass := FPt(-radiuses[0], 0)
		ad := (Pi * 0.5) / f64(segments[0]-1)

		for i := range segments[0] {
			rotated := compass.Rotate(ad * f64(i))
			rotated = rotated.Add(inLeftTop)

			triangleFan.ExtendFan(rotated, FPt(1, 1), clr)
		}
	}

	// right top
	if segments[1] <= 1 || radiuses[1] < 1.5 {
		triangleFan.ExtendFan(FPt(rect.Max.X, rect.Min.Y), FPt(1, 1), clr)
	} else {
		compass := FPt(0, -radiuses[1])
		ad := (Pi * 0.5) / f64(segments[1]-1)

		for i := range segments[1] {
			rotated := compass.Rotate(ad * f64(i))
			rotated = rotated.Add(inRightTop)

			triangleFan.ExtendFan(rotated, FPt(1, 1), clr)
		}
	}
	// right bottom
	if segments[2] <= 1 || radiuses[2] < 1.5 {
		triangleFan.ExtendFan(rect.Max, FPt(1, 1), clr)
	} else {
		compass := FPt(radiuses[2], 0)
		ad := (Pi * 0.5) / f64(segments[2]-1)

		for i := range segments[2] {
			rotated := compass.Rotate(ad * f64(i))
			rotated = rotated.Add(inRightBottom)

			triangleFan.ExtendFan(rotated, FPt(1, 1), clr)
		}
	}
	// left bottom
	if segments[3] <= 1 || radiuses[3] < 1.5 {
		triangleFan.ExtendFan(FPt(rect.Min.X, rect.Max.Y), FPt(1, 1), clr)
	} else {
		compass := FPt(0, radiuses[3])
		ad := (Pi * 0.5) / f64(segments[3]-1)

		for i := range segments[3] {
			rotated := compass.Rotate(ad * f64(i))
			rotated = rotated.Add(inLeftBottom)

			triangleFan.ExtendFan(rotated, FPt(1, 1), clr)
		}
	}

	// close fan
	return triangleFan.CloseFan()
}

func AddRoundRectVerts(
	vs []eb.Vertex,
	is []uint16,
	rect FRectangle,
	radiuses [4]float64,
	radiusInPixels bool,
	segments [4]int,
	clr color.Color,
) ([]eb.Vertex, []uint16) {
	if !radiusInPixels {
		toPx := min(rect.Dx(), rect.Dy()) * 0.5
		for i := range 4 {
			radiuses[i] = radiuses[i] * toPx
		}
	}
	return addRoundRectVertsImpl(
		vs, is,
		rect,
		radiuses,
		segments,
		clr,
	)
}
