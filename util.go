package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

func UpdateDelta() time.Duration {
	return time.Duration(f64(time.Second) / f64(eb.TPS()))
}

func CursorFPt() FPoint {
	mx, my := eb.CursorPosition()
	return FPt(f64(mx), f64(my))
}

// make rectangle with left corner at 0, 0
// centered at 0, 0
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

// calculate scale to fit srcRect inside dstRect
func GetScaleToFitRectInRect(
	srcRectW, srcRectH float64,
	dstRectW, dstRectH float64,
) float64 {
	return min(dstRectW/srcRectW, dstRectH/srcRectH)
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

func New2DArray[T any](width, height int) [][]T {
	var arr = make([][]T, width)
	for i := 0; i < width; i++ {
		arr[i] = make([]T, height)
	}
	return arr
}

// returns recommended line height for fonts when writing horizontally
func FontLineSpacing(face ebt.Face) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent + m.HLineGap
}

// returns font's size when written horizontally
// not really a 'font size' but it's fine...
func FontSize(face ebt.Face) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent
}

func ExecutablePath() (string, error) {
	path, err := os.Executable()

	if err != nil {
		return "", err
	}

	evaled, err := filepath.EvalSymlinks(path)

	if err != nil {
		return "", err
	}

	return evaled, nil
}

func RelativePath(path string) (string, error) {
	exePath, err := ExecutablePath()

	if err != nil {
		return "", err
	}

	joined := filepath.Join(filepath.Dir(exePath), path)

	return joined, nil
}

func DrawSubViewInRect(
	dst *eb.Image,
	rect FRectangle,
	scale float64,
	offsetX, offsetY float64,
	clr color.Color,
	view SubView,
) {
	imgSize := ImageSizeFPt(view)
	rectSize := rect.Size()

	drawScale := GetScaleToFitRectInRect(imgSize.X, imgSize.Y, rectSize.X, rectSize.Y)
	drawScale *= scale

	op := &DrawSubViewOptions{}
	op.GeoM.Concat(TransformToCenter(imgSize.X, imgSize.Y, drawScale, drawScale, 0))
	rectCenter := FRectangleCenter(rect)
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)
	op.GeoM.Translate(offsetX, offsetY)
	op.ColorScale.ScaleWithColor(clr)

	DrawSubView(dst, view, op)
}
