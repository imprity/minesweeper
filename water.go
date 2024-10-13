//go:build ignore

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	_ "image/png"
	"os"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebu "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebi "github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	ScreenWidth  float64 = 600
	ScreenHeight float64 = 600
)

var (
	//go:embed assets/noise6.png
	image1Bytes []byte
	Image1      *eb.Image

	//go:embed assets/noise8.png
	image2Bytes []byte
	Image2      *eb.Image
)

func init() {
	var err error
	Image1, _, err = ebu.NewImageFromReader(bytes.NewReader(image1Bytes))
	if err != nil {
		panic(err)
	}

	Image2, _, err = ebu.NewImageFromReader(bytes.NewReader(image2Bytes))
	if err != nil {
		panic(err)
	}
}

type App struct {
	Shader          *eb.Shader
	ShaderLoadError error
	DeltaTime       float64
}

func NewApp() *App {
	a := new(App)
	return a
}

func (a *App) LoadShader() (*eb.Shader, error) {
	shaderCode, err := os.ReadFile("water_shader.go")
	if err != nil {
		return nil, err
	}

	shader, err := eb.NewShader(shaderCode)
	if err != nil {
		return nil, err
	}

	return shader, nil
}

func (a *App) Update() error {
	if ebi.IsKeyJustPressed(eb.KeyF5) {
		if shader, err := a.LoadShader(); err == nil {
			a.Shader = shader
			a.ShaderLoadError = nil
		} else {
			a.ShaderLoadError = err
		}
	}

	a.DeltaTime += 0.1

	return nil
}

func (a *App) Draw(dst *eb.Image) {
	cursorX, cursorY := eb.CursorPosition()

	if a.Shader != nil {
		op := &eb.DrawRectShaderOptions{}

		op.Images[0] = Image1
		op.Images[1] = Image2

		op.Uniforms = make(map[string]any)
		op.Uniforms["Time"] = a.DeltaTime
		op.Uniforms["Cursor"] = [2]float64{float64(cursorX), float64(cursorY)}

		imgSizeX := float64(Image1.Bounds().Dx())
		imgSizeY := float64(Image1.Bounds().Dy())

		op.GeoM.Scale(ScreenWidth/imgSizeX, ScreenHeight/imgSizeY)

		dst.DrawRectShader(Image1.Bounds().Dx(), Image1.Bounds().Dy(), a.Shader, op)
	} else {
		if a.ShaderLoadError == nil {
			ebu.DebugPrint(dst, "shader is not loaded")
		}
	}

	if a.ShaderLoadError != nil {
		ebu.DebugPrint(dst, fmt.Sprintf("error :%v", a.ShaderLoadError))
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	ScreenWidth = float64(outsideWidth)
	ScreenHeight = float64(outsideHeight)

	return outsideWidth, outsideHeight
}

func main() {
	app := NewApp()

	eb.SetVsyncEnabled(true)
	eb.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	//eb.SetWindowResizingMode(eb.WindowResizingModeEnabled)
	eb.SetWindowTitle("water")

	if err := eb.RunGame(app); err != nil {
		panic(err)
	}
}
