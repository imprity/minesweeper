package main

import (
	"image"
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

func FitRectInRect(
	srcRect FRectangle,
	dstRect FRectangle,
) eb.GeoM {
	scaleX := dstRect.Dx() / srcRect.Dx()
	scaleY := dstRect.Dy() / srcRect.Dy()

	geom := eb.GeoM{}
	geom.Scale(scaleX, scaleY)
	geom.Translate(dstRect.Min.X, dstRect.Min.Y)

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
