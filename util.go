package minesweeper

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

func TouchFPt(touchId eb.TouchID) FPoint {
	tx, ty := eb.TouchPosition(touchId)
	return FPt(f64(tx), f64(ty))
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

func ImageImageFromEbImage(ebImg *eb.Image) image.Image {
	img := image.NewNRGBA(RectWH(ebImg.Bounds().Dx(), ebImg.Bounds().Dy()))
	ebImg.ReadPixels(img.Pix)
	return img
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

// looks at current screen ratio and guess if it's on mobile or not
func ProbablyOnMobile() bool {
	// NOTE : from https://en.wikipedia.org/wiki/Display_aspect_ratio
	//
	// From 2010 to 2017 most smartphone manufacturers switched to using 16:9 widescreen displays
	//
	// So if height ratio is bigger than that
	// we are going to guess it's on mobile
	//
	// we are being bit more genorous with our check
	const mobileRatio = 14.0 / 9.0
	screenRatio := ScreenHeight / ScreenWidth

	if screenRatio > mobileRatio {
		return true
	}
	return false
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

// returns face's size when written horizontally
// not really a 'face size' but it's fine...
func FaceSize(face ebt.Face) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent
}

// returns recommended line height for faces when writing horizontally
func FaceLineSpacing(face ebt.Face) float64 {
	m := face.Metrics()
	return m.HAscent + m.HDescent + m.HLineGap
}

// limits face's size with width
func WidthLimitFace(
	text string, face *ebt.GoTextFace, width float64,
) {
	w, _ := ebt.Measure(text, face, FaceLineSpacing(face))
	if w > width {
		face.Size *= width / w
	}
}

func FitTextInRect(
	text string,
	face ebt.Face,
	rect FRectangle,
) eb.GeoM {
	textW, textH := ebt.Measure(text, face, FaceLineSpacing(face))
	scale := min(rect.Dx()/textW, rect.Dy()/textH)
	geom := TransformToCenter(textW, textH, scale, scale, 0)
	center := FRectangleCenter(rect)
	geom.Translate(center.X, center.Y)

	return geom
}
