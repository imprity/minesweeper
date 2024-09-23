package main

import (
	"image"
	"image/color"

	eb "github.com/hajimehoshi/ebiten/v2"
)

type SubView struct {
	Image *eb.Image
	Rect  FRectangle
}

// image interface functions

func (s SubView) ColorModel() color.Model {
	return s.Image.ColorModel()
}

func (s SubView) Bounds() image.Rectangle {
	return FRectToRect(s.Rect)
}

func (s SubView) At(x, y int) color.Color {
	return s.Image.At(x, y)
}

type DrawSubViewOptions struct {
	// GeoM is a geometry matrix to draw.
	// The default (zero) value is identity, which draws the image at (0, 0).
	GeoM eb.GeoM

	// ColorScale is a scale of color.
	//
	// ColorScale is slightly different from colorm.ColorM's Scale in terms of alphas.
	// ColorScale is applied to premultiplied-alpha colors, while colorm.ColorM is applied to straight-alpha colors.
	// Thus, ColorM.Scale(r, g, b, a) equals to ColorScale.Scale(r*a, g*a, b*a, a).
	//
	// The default (zero) value is identity, which is (1, 1, 1, 1).
	ColorScale eb.ColorScale

	// Blend is a blending way of the source color and the destination color.
	// Blend is used only when CompositeMode is CompositeModeCustom.
	// The default (zero) value is the regular alpha blending.
	Blend eb.Blend

	// Filter is a type of texture filter.
	// The default (zero) value is FilterNearest.
	Filter eb.Filter
}

func DrawSubView(dst *eb.Image, sv SubView, options *DrawSubViewOptions) {
	rect := sv.Rect
	rect0 := FRectMoveTo(rect, 0, 0)

	var vs [4]FPoint

	vs[0] = FPt(rect0.Min.X, rect0.Min.Y)
	vs[1] = FPt(rect0.Max.X, rect0.Min.Y)
	vs[2] = FPt(rect0.Max.X, rect0.Max.Y)
	vs[3] = FPt(rect0.Min.X, rect0.Max.Y)

	var xformed [4]FPoint

	xformed[0] = FPointTransform(vs[0], options.GeoM)
	xformed[1] = FPointTransform(vs[1], options.GeoM)
	xformed[2] = FPointTransform(vs[2], options.GeoM)
	xformed[3] = FPointTransform(vs[3], options.GeoM)

	var verts [4]eb.Vertex
	var indices [6]uint16

	verts[0] = eb.Vertex{
		DstX: f32(xformed[0].X), DstY: f32(xformed[0].Y),
		SrcX: f32(rect.Min.X), SrcY: f32(rect.Min.Y),
	}
	verts[1] = eb.Vertex{
		DstX: f32(xformed[1].X), DstY: f32(xformed[1].Y),
		SrcX: f32(rect.Max.X), SrcY: f32(rect.Min.Y),
	}
	verts[2] = eb.Vertex{
		DstX: f32(xformed[2].X), DstY: f32(xformed[2].Y),
		SrcX: f32(rect.Max.X), SrcY: f32(rect.Max.Y),
	}
	verts[3] = eb.Vertex{
		DstX: f32(xformed[3].X), DstY: f32(xformed[3].Y),
		SrcX: f32(rect.Min.X), SrcY: f32(rect.Max.Y),
	}

	for i := range 4 {
		verts[i].ColorR = options.ColorScale.R()
		verts[i].ColorG = options.ColorScale.G()
		verts[i].ColorB = options.ColorScale.B()
		verts[i].ColorA = options.ColorScale.A()
	}

	indices = [6]uint16{
		0, 1, 2, 0, 2, 3,
	}

	op := &eb.DrawTrianglesOptions{}

	op.Filter = options.Filter
	op.AntiAlias = true // why not
	op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
	op.Blend = options.Blend

	dst.DrawTriangles(verts[:], indices[:], sv.Image, op)
}
