package main

import (
	//"fmt"
	"os"
	"log"
	"image/color"

	eb "github.com/hajimehoshi/ebiten/v2"
	//ebu "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	//ebi "github.com/hajimehoshi/ebiten/v2/inpututil"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenWidth  = 600
	ScreenHeight = 600
)

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR : ", log.Lshortfile)

type App struct {
	Board Board

	BoardRect RectangleF
}

func NewApp() *App {
	a := new(App)

	const boardWidth = 10
	const boardHeight = 10
	const boardMineCount = 10

	const rectSize = 400

	a.BoardRect = Rectf(
		0, 0, rectSize, rectSize,
	)
	
	a.BoardRect = CenterRectangleF(
		a.BoardRect, ScreenWidth*0.5, ScreenHeight*0.5)

	a.Board = NewBoard(boardWidth, boardHeight, boardMineCount)

	return a
}

func (a *App) Update() error {
	
}

func (a *App) Draw(screen *eb.Image) {
	tileWidth := a.BoardRect.Dx() / f64(a.Board.Width)
	tileHeight := a.BoardRect.Dy() / f64(a.Board.Height)

	regularBg := color.NRGBA{100,100,100,255}
	revealedBg := color.NRGBA{200,200,200,255}

	for y:=0; y<a.Board.Height; y++ {
		for x:=0; x<a.Board.Width; x++ {		
			tileX := f64(x) * tileWidth + a.BoardRect.Min.X
			tileY := f64(y) * tileHeight + a.BoardRect.Min.Y

			// draw the tile background
			bgColor := regularBg

			if a.Board.Revealed[x][y] {
				bgColor = revealedBg
			}

			ebv.DrawFilledRect(
				screen, 
				f32(tileX), f32(tileY), f32(tileWidth), f32(tileHeight),
				bgColor,
				true,
			)

			// draw border
			ebv.StrokeRect(
				screen, 
				f32(tileX), f32(tileY), f32(tileWidth), f32(tileHeight),
				1,
				color.NRGBA{0,0,0,255},
				true,
			)
		}
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func main() {
	LoadAssets()

	app := NewApp()

	eb.SetWindowSize(ScreenWidth, ScreenHeight)
	eb.SetWindowTitle("test")
	
	if err := eb.RunGame(app); err != nil {
		panic(err)
	}
}
