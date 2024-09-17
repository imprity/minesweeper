package main

import (
	"image/color"
	"log"
	"math"
	"os"

	eb "github.com/hajimehoshi/ebiten/v2"
	//ebu "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebi "github.com/hajimehoshi/ebiten/v2/inpututil"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenWidth  = 600
	ScreenHeight = 600
)

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR : ", log.Lshortfile)

type App struct {
	Board Board

	BoardRect FRectangle
}

func NewApp() *App {
	a := new(App)

	const boardWidth = 10
	const boardHeight = 10
	const boardMineCount = 10

	const rectSize = 400

	a.BoardRect = FRect(
		0, 0, rectSize, rectSize,
	)

	a.BoardRect = CenterFRectangle(
		a.BoardRect, ScreenWidth*0.5, ScreenHeight*0.5)

	a.Board = NewBoard(boardWidth, boardHeight, boardMineCount)

	return a
}

func (a *App) MousePosToBoardPos(mousePos FPoint) (int, int) {
	// if mouse is outside the board return -1
	if !mousePos.In(a.BoardRect) {
		return -1, -1
	}

	mousePos.X -= a.BoardRect.Min.X
	mousePos.Y -= a.BoardRect.Min.Y

	boardX := int(math.Floor(mousePos.X / (a.BoardRect.Dx() / float64(a.Board.Width))))
	boardY := int(math.Floor(mousePos.Y / (a.BoardRect.Dy() / float64(a.Board.Height))))

	boardX = min(boardX, a.Board.Width-1)
	boardY = min(boardY, a.Board.Height-1)

	return boardX, boardY
}

func (a *App) Update() error {
	cursor := CursorFPt()

	boardX, boardY := a.MousePosToBoardPos(cursor)

	if boardX >= 0 && boardY >= 0 {
		if !a.Board.Revealed[boardX][boardY] {
			if ebi.IsMouseButtonJustPressed(eb.MouseButtonLeft) {
				a.Board.InteractAt(boardX, boardY, InteractionTypeStep)
			} else if ebi.IsMouseButtonJustPressed(eb.MouseButtonRight) {
				a.Board.InteractAt(boardX, boardY, InteractionTypeFlag)
			}
		} else {
			pressedL := eb.IsMouseButtonPressed(eb.MouseButtonLeft)
			justPressedL := ebi.IsMouseButtonJustPressed(eb.MouseButtonLeft)
			pressedR := eb.IsMouseButtonPressed(eb.MouseButtonRight)
			justPressedR := ebi.IsMouseButtonJustPressed(eb.MouseButtonRight)

			if (justPressedL && pressedR) || (justPressedR && pressedL) {
				a.Board.InteractAt(boardX, boardY, InteractionTypeCheck)
			}
		}
	}

	return nil
}

func GetNumberTile(number int) *eb.Image {
	if !(1 <= number && number <= 9) {
		ErrorLogger.Fatalf("%d is not a valid number", number)
	}

	return SpriteSubImage(TileSprite, number-1)
}

func GetMineTile() *eb.Image {
	return SpriteSubImage(TileSprite, 9)
}

func GetFlagTile() *eb.Image {
	return SpriteSubImage(TileSprite, 10)
}

func (a *App) GetTileRect(boardX, boardY int) FRectangle {
	tileWidth := a.BoardRect.Dx() / f64(a.Board.Width)
	tileHeight := a.BoardRect.Dy() / f64(a.Board.Height)

	return FRectangle{
		Min: FPt(f64(boardX)*tileWidth, f64(boardY)*tileHeight).Add(a.BoardRect.Min),
		Max: FPt(f64(boardX+1)*tileWidth, f64(boardY+1)*tileHeight).Add(a.BoardRect.Min),
	}
}

func (a *App) DrawTile(dst *eb.Image, boardX, boardY int, tile *eb.Image) {
	tileRect := a.GetTileRect(boardX, boardY)

	imgSize := ImageSizeFPt(tile.Bounds())
	rectSize := tileRect.Size()

	scale := min(rectSize.X, rectSize.Y) / max(imgSize.X, imgSize.Y)

	op := &eb.DrawImageOptions{}
	op.GeoM.Concat(TransformToCenter(imgSize.X, imgSize.Y, scale, scale, 0))
	rectCenter := FRectangleCenter(tileRect)
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)

	dst.DrawImage(tile, op)
}

func (a *App) Draw(screen *eb.Image) {
	tileWidth := a.BoardRect.Dx() / f64(a.Board.Width)
	tileHeight := a.BoardRect.Dy() / f64(a.Board.Height)

	regularBg := color.NRGBA{100, 100, 100, 255}
	revealedBg := color.NRGBA{200, 200, 200, 255}

	for y := 0; y < a.Board.Height; y++ {
		for x := 0; x < a.Board.Width; x++ {
			tileX := f64(x)*tileWidth + a.BoardRect.Min.X
			tileY := f64(y)*tileHeight + a.BoardRect.Min.Y

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

			// draw flag
			if a.Board.Flags[x][y] {
				a.DrawTile(screen, x, y, GetFlagTile())
			}

			if a.Board.Mines[x][y] {
				//a.DrawTile(screen, x, y, GetMineTile())
			}

			if a.Board.Revealed[x][y] {
				if count := a.Board.GetNeighborMineCount(x, y); count > 0 {
					a.DrawTile(screen, x, y, GetNumberTile(count))
				}
			}

			// draw border
			ebv.StrokeRect(
				screen,
				f32(tileX), f32(tileY), f32(tileWidth), f32(tileHeight),
				1,
				color.NRGBA{0, 0, 0, 255},
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
