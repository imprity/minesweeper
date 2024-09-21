package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"

	eb "github.com/hajimehoshi/ebiten/v2"
)

type Sprite struct {
	*eb.Image

	Width, Height int

	Margin int

	Count int
}

func SpriteRect(sprite Sprite, spriteN int) image.Rectangle {
	if spriteN < 0 || spriteN >= sprite.Count {
		panicMsg := fmt.Sprintf("index out of range [%d] with length %d", spriteN, sprite.Count)
		panic(panicMsg)
	}

	w := sprite.Width + sprite.Margin
	h := sprite.Height + sprite.Margin

	spriteW, spriteH := ImageSize(sprite)

	colCount := spriteW / w
	rowCount := spriteH / h

	_ = rowCount // might use this later

	// prevent dvidision by zero
	// (it also makes no sense for col and row count to be zero)
	colCount = max(colCount, 1)
	rowCount = max(rowCount, 1)

	col := spriteN % colCount
	row := spriteN / colCount

	imageMin := sprite.Bounds().Min

	return image.Rectangle{
		Min: image.Pt(col*w, row*h).Add(imageMin),
		Max: image.Pt(col*w+sprite.Width, row*h+sprite.Height).Add(imageMin),
	}
}

func SpriteFRect(sprite Sprite, spriteN int) FRectangle {
	iRect := SpriteRect(sprite, spriteN)
	return FRect(
		f64(iRect.Min.X), f64(iRect.Min.Y),
		f64(iRect.Max.X), f64(iRect.Max.Y),
	)
}

func DrawSprite(dst *eb.Image, sprite Sprite, spriteN int, geom eb.GeoM, tint color.Color) {
	rect := SpriteFRect(sprite, spriteN)
	rect0 := FRectMoveTo(rect, FPoint{})

	var vs [4]FPoint

	vs[0] = FPt(rect0.Min.X, rect0.Min.Y)
	vs[1] = FPt(rect0.Max.X, rect0.Min.Y)
	vs[2] = FPt(rect0.Max.X, rect0.Max.Y)
	vs[3] = FPt(rect0.Min.X, rect0.Max.Y)

	var xformed [4]FPoint

	xformed[0] = FPointTransform(vs[0], geom)
	xformed[1] = FPointTransform(vs[1], geom)
	xformed[2] = FPointTransform(vs[2], geom)
	xformed[3] = FPointTransform(vs[3], geom)

	var verts [4]eb.Vertex
	var indices [6]uint16

	ri, gi, bi, ai := tint.RGBA()
	r, g, b, a := f64(ri)/0xffff, f64(gi)/0xffff, f64(bi)/0xffff, f64(ai)/0xffff

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
		verts[i].ColorR = f32(r)
		verts[i].ColorG = f32(g)
		verts[i].ColorB = f32(b)
		verts[i].ColorA = f32(a)
	}

	indices = [6]uint16{
		0, 1, 2, 0, 2, 3,
	}

	op := &eb.DrawTrianglesOptions{}

	op.Filter = eb.FilterLinear
	op.AntiAlias = true
	op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha

	dst.DrawTriangles(verts[:], indices[:], sprite.Image, op)
}

func SpriteSubImage(sprite Sprite, spriteN int) *eb.Image {
	return sprite.SubImage(SpriteRect(sprite, spriteN)).(*eb.Image)
}

type spriteJsonMetadata struct {
	SpriteWidth  int
	SpriteHeight int

	SpriteCount int

	SpriteMargin int
}

// Parse sprite json metadata.
// Parsed sprite doen't contain image.
func ParseSpriteJsonMetadata(jsonReader io.Reader) (Sprite, error) {
	sprite := Sprite{}
	metadata := spriteJsonMetadata{}

	decoder := json.NewDecoder(jsonReader)

	if err := decoder.Decode(&metadata); err != nil {
		return sprite, err
	}

	sprite.Width = metadata.SpriteWidth
	sprite.Height = metadata.SpriteHeight
	sprite.Margin = metadata.SpriteMargin
	sprite.Count = metadata.SpriteCount

	return sprite, nil
}
