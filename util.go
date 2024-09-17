package main

import (
	eb "github.com/hajimehoshi/ebiten/v2"
	"image"
)

func CursorFPt() FPoint {
	mx, my := eb.CursorPosition()
	return FPt(f64(mx), f64(my))
}

func TransformToCenter(
	width, height float64,
	scaleX, scaleY float64,
	rotation float64,
) eb.GeoM {
	geom := eb.GeoM{}
	geom.Translate(-width*0.5, -height*0.5)
	geom.Scale(scaleX, scaleY)
	geom.Rotate(rotation)

	return geom
}

func ImageSize(img image.Image) (int, int) {
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func ImageSizeF(img image.Image) (float64, float64) {
	return f64(img.Bounds().Dx()), f64(img.Bounds().Dy())
}

func ImageSizePt(img image.Image) image.Point {
	return img.Bounds().Size()
}

func ImageSizeFPt(img image.Image) FPoint {
	bound := img.Bounds()
	return FPoint{f64(bound.Dx()), f64(bound.Dy())}
}
