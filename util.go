package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
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

func GetHourMinuteSeconds(duration time.Duration) (int, int, int) {
	hours := duration / time.Hour
	minutes := (duration % time.Hour) / time.Minute
	seconds := (duration % time.Minute) / time.Second

	return int(hours), int(minutes), int(seconds)
}

// examples
// CheckFileExt("image.png", ".png") => return true
// CheckFileExt("image.PNG", ".png") => return true
// CheckFileExt("image.jpg", ".png") => return false
func CheckFileExt(filepath string, ext string) bool {
	filepath = strings.ToLower(filepath)
	ext = strings.ToLower(ext)

	return strings.HasSuffix(filepath, ext)
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

// ========================
// text utils
// ========================

// returns font's size when written horizontally
// not really a 'font size' but it's fine...
func FontSize(face ebt.Face) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent
}

// returns recommended line height for fonts when writing horizontally
func FontLineSpacing(face ebt.Face) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent + m.HLineGap
}

// returns recommended line height for fonts when writing horizontally
func FontLineSpacingSized(face ebt.Face, size float64) float64 {
	scale := FontScale(face, size)
	return FontLineSpacing(face) * scale
}

func FontScale(face ebt.Face, size float64) float64 {
	return size / FontSize(face)
}

func MeasureTextSized(text string, face ebt.Face, size float64, lineSpacingInPixels float64) (float64, float64) {
	w, h := ebt.Measure(text, face, lineSpacingInPixels)
	scale := FontScale(face, size)

	return w * scale, h * scale
}

func TextToBaseLine(
	fontFace ebt.Face,
	fontSize float64,
	x, y float64,
) eb.GeoM {
	scale := FontScale(fontFace, fontSize)

	newX := x
	newY := y - fontFace.Metrics().HAscent*scale

	var geom eb.GeoM
	geom.Scale(scale, scale)
	geom.Translate(newX, newY)

	return geom
}

func TextToBaseLineLimitWidth(
	text string,
	fontFace ebt.Face,
	fontSize float64,
	x, y float64,
	maxWidth float64,
) eb.GeoM {
	w, _ := MeasureTextSized(text, fontFace, fontSize, FontLineSpacingSized(fontFace, fontSize))

	if w > maxWidth {
		fontSize *= maxWidth / w
	}

	return TextToBaseLine(fontFace, fontSize, x, y)
}

func TextToYcenter(
	fontFace ebt.Face,
	fontSize float64,
	x, y float64,
) eb.GeoM {
	scale := FontScale(fontFace, fontSize)

	newX := x
	newY := y - fontSize*0.5

	var geom eb.GeoM
	geom.Scale(scale, scale)
	geom.Translate(newX, newY)

	return geom
}

func TextToYcenterLimitWidth(
	text string,
	fontFace ebt.Face,
	fontSize float64,
	x, y float64,
	maxWidth float64,
) eb.GeoM {
	w, _ := MeasureTextSized(text, fontFace, fontSize, FontLineSpacingSized(fontFace, fontSize))

	if w > maxWidth {
		fontSize *= maxWidth / w
	}

	return TextToYcenter(fontFace, fontSize, x, y)
}

func FitTextInRect(
	text string,
	fontFace ebt.Face,
	rect FRectangle,
) eb.GeoM {
	textW, textH := ebt.Measure(text, fontFace, FontLineSpacing(fontFace))
	scale := min(rect.Dx()/textW, rect.Dy()/textH)
	geom := TransformToCenter(textW, textH, scale, scale, 0)
	center := FRectangleCenter(rect)
	geom.Translate(center.X, center.Y)

	return geom
}
