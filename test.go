//go:build ignore

package main

import (
	eb "github.com/hajimehoshi/ebiten/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
	"image"
	"image/color"
	"math"
)

const (
	ScreenWidth  = 600
	ScreenHeight = 600
)

type App struct {
}

func NewApp() *App {
	a := new(App)
	return a
}

func (a *App) Update() error {
	return nil
}

func (a *App) Draw(dst *eb.Image) {
	p := ebv.Path{}

	p.Arc(ScreenWidth*0.5, ScreenHeight*0.5, 50, math.Pi, math.Pi+math.Pi*0.5, ebv.Clockwise)
	p.Close()

	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	drawVerticesForUtil(dst, vs, is, color.NRGBA{255, 255, 255, 255}, true)
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	app := NewApp()

	eb.SetVsyncEnabled(true)
	eb.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	eb.SetWindowTitle("test")

	if err := eb.RunGame(app); err != nil {
		panic(err)
	}
}

var (
	whiteImage    = eb.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*eb.Image)
)

func init() {
	b := whiteImage.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	for i := range pix {
		pix[i] = 0xff
	}
	// This is hacky, but WritePixels is better than Fill in term of automatic texture packing.
	whiteImage.WritePixels(pix)
}

func drawVerticesForUtil(dst *eb.Image, vs []eb.Vertex, is []uint16, clr color.Color, antialias bool) {
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
	dst.DrawTriangles(vs, is, whiteSubImage, op)
}
