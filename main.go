package main

import (
	"image/color"
	"log"
	"math"
	"os"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	//ebu "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenWidth  = 600
	ScreenHeight = 600
)

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR : ", log.Lshortfile)

type TileHighLight struct {
	Brightness float64
}

type GameState int

const (
	GameStatePlaying GameState = iota
	GameStateWon
	GameStateLost
)

type App struct {
	Board     Board
	BoardRect FRectangle

	MineCount   int
	PlacedMines bool

	TileHighLights [][]TileHighLight

	GameState GameState

	//constants
	HighlightDuraiton time.Duration
}

func NewApp() *App {
	a := new(App)

	a.HighlightDuraiton = time.Millisecond * 100

	const boardWidth = 10
	const boardHeight = 10
	const boardMineCount = 10

	const rectSize = 400

	a.MineCount = boardMineCount

	a.BoardRect = FRect(
		0, 0, rectSize, rectSize,
	)

	a.BoardRect = CenterFRectangle(
		a.BoardRect, ScreenWidth*0.5, ScreenHeight*0.5)

	a.Board = NewBoard(boardWidth, boardHeight)

	a.TileHighLights = New2DArray[TileHighLight](boardWidth, boardHeight)

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

func (a *App) SetTileHightlight(boardX, boardY int) {
	a.TileHighLights[boardX][boardY].Brightness = 1
}

func (a *App) Update() error {
	cursor := CursorFPt()

	boardX, boardY := a.MousePosToBoardPos(cursor)

	prevState := a.GameState

	// =================================
	// handle board interaction
	// =================================
	if a.GameState == GameStatePlaying && boardX >= 0 && boardY >= 0 {
		if a.Board.Revealed[boardX][boardY] {
			pressedL := IsMouseButtonPressed(eb.MouseButtonLeft)
			justPressedL := IsMouseButtonJustPressed(eb.MouseButtonLeft)

			pressedR := IsMouseButtonPressed(eb.MouseButtonRight)
			justPressedR := IsMouseButtonJustPressed(eb.MouseButtonRight)

			if (justPressedL && pressedR) || (justPressedR && pressedL) {
				flagCount := a.Board.GetNeighborFlagCount(boardX, boardY)
				mineCount := a.Board.GetNeighborMineCount(boardX, boardY)

				if flagCount == mineCount { // check if flagged correctly
					flaggedCorrectly := true
					missedMine := false

					iter := NewBoardIterator(
						boardX-1, boardY-1,
						boardX+1, boardY+1,
					)

					for iter.HasNext() {
						x, y := iter.GetNext()

						if a.Board.IsPosInBoard(x, y) {
							if a.Board.Mines[x][y] != a.Board.Flags[x][y] {
								flaggedCorrectly = false
							}

							if a.Board.Mines[x][y] && !a.Board.Flags[x][y] {
								missedMine = true
							}

							if a.Board.Flags[x][y] {
								flagCount += 1
							}
						}
					}

					if flaggedCorrectly {
						iter.Reset()

						for iter.HasNext() {
							x, y := iter.GetNext()
							if a.Board.IsPosInBoard(x, y) {
								a.Board.SpreadSafeArea(x, y)
							}
						}

						// check if user has won the game
						if a.Board.IsAllSafeTileRevealed() {
							a.GameState = GameStateWon
						}
					} else {
						if missedMine {
							a.GameState = GameStateLost
						}
					}
				} else { // just highlight the area
					iter := NewBoardIterator(
						boardX-1, boardY-1,
						boardX+1, boardY+1,
					)

					for iter.HasNext() {
						x, y := iter.GetNext()

						if a.Board.IsPosInBoard(x, y) && !a.Board.Revealed[x][y] && !a.Board.Flags[x][y] {
							a.SetTileHightlight(x, y)
						}
					}
				}
			}
		} else { // not revealed
			if IsMouseButtonJustPressed(eb.MouseButtonLeft) {
				if !a.PlacedMines { // mine is not placed
					a.Board.PlaceMines(a.MineCount, boardX, boardY)
					a.Board.SpreadSafeArea(boardX, boardY)
					a.PlacedMines = true
				} else { // mine has been placed
					if !a.Board.Mines[boardX][boardY] {
						a.Board.SpreadSafeArea(boardX, boardY)
					} else {
						a.GameState = GameStateLost
					}
				}
			} else if IsMouseButtonJustPressed(eb.MouseButtonRight) {
				a.Board.Flags[boardX][boardY] = !a.Board.Flags[boardX][boardY]
			}
		}
	}

	if prevState != a.GameState {
		switch a.GameState {
		case GameStateWon:
			println("!!!! YOU WON !!!!")
		case GameStateLost:
			println("!!!! YOU LOST !!!!")
		}
	}

	// update highlights
	if !(IsMouseButtonPressed(eb.MouseButtonLeft) || IsMouseButtonPressed(eb.MouseButtonRight)) {
		for y := 0; y < a.Board.Height; y++ {
			for x := 0; x < a.Board.Width; x++ {
				a.TileHighLights[x][y].Brightness -= f64(UpdateDelta()) / f64(a.HighlightDuraiton)
				a.TileHighLights[x][y].Brightness = max(a.TileHighLights[x][y].Brightness, 0)
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

			// draw flags
			if a.Board.Flags[x][y] {
				a.DrawTile(screen, x, y, GetFlagTile())
			}

			// draw mines
			if a.GameState == GameStateLost && a.Board.Mines[x][y] && !a.Board.Flags[x][y] {
				a.DrawTile(screen, x, y, GetMineTile())
			}

			if a.Board.Revealed[x][y] {
				if count := a.Board.GetNeighborMineCount(x, y); count > 0 {
					a.DrawTile(screen, x, y, GetNumberTile(count))
				}
			}

			// draw highlight
			if a.TileHighLights[x][y].Brightness > 0 {
				t := a.TileHighLights[x][y].Brightness
				ebv.DrawFilledRect(
					screen,
					f32(tileX), f32(tileY), f32(tileWidth), f32(tileHeight),
					color.NRGBA{255, 255, 255, uint8(t * 255)},
					true,
				)
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
