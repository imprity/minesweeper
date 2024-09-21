package main

import (
	"image/color"
	"log"
	"math"
	"os"
	"time"

	_ "github.com/silbinarywolf/preferdiscretegpu"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	ScreenWidth  float64 = 600
	ScreenHeight float64 = 600
)

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR : ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO : ", log.Lshortfile)

type TileHighLight struct {
	Brightness float64
}

type GameState int

const (
	GameStatePlaying GameState = iota
	GameStateWon
	GameStateLost
)

type Difficulty int

const (
	DifficultyEasy Difficulty = iota
	DifficultyMedium
	DifficultyHard
	DifficultySize
)

var DifficultyStrs = [DifficultySize]string{
	"Easy",
	"Medium",
	"Hard",
}

type App struct {
	Board Board

	MineCount      [DifficultySize]int
	BoardTileCount [DifficultySize]int

	BoardTouched bool

	TileHighLights    [][]TileHighLight
	HighlightDuraiton time.Duration

	GameState GameState

	Difficulty Difficulty

	DifficultyButtonLeft  *ImageButton
	DifficultyButtonRight *ImageButton

	GameResultAnimTimer  Timer
	TopMenuShowAnimTimer Timer

	BoardSizeRatio float64 // relative to min(ScreenWidth, ScreenHeight)

	TopUIMarginHorizontal float64
	TopUIMarginTop        float64
	TopUIMarginBottom     float64

	TopUIButtonButtonRatio float64
	TopUIButtonTextRatio   float64
}

func NewApp() *App {
	a := new(App)

	a.GameResultAnimTimer = Timer{
		Duration: time.Millisecond * 300,
	}
	a.TopMenuShowAnimTimer = Timer{
		Duration: time.Millisecond * 200,
	}
	a.TopMenuShowAnimTimer.Current = a.TopMenuShowAnimTimer.Duration

	a.HighlightDuraiton = time.Millisecond * 100

	a.MineCount = [DifficultySize]int{
		10, 20, 30,
	}
	a.BoardTileCount = [DifficultySize]int{
		10, 15, 20,
	}

	a.BoardSizeRatio = 0.8

	a.TopUIMarginHorizontal = 5
	a.TopUIMarginTop = 5
	a.TopUIMarginBottom = 5

	a.TopUIButtonButtonRatio = 0.2
	a.TopUIButtonTextRatio = 0.5

	initBoard := func() {
		a.Board = NewBoard(
			a.BoardTileCount[a.Difficulty],
			a.BoardTileCount[a.Difficulty],
		)

		a.TileHighLights = New2DArray[TileHighLight](a.Board.Width, a.Board.Height)
	}
	initBoard()

	// ==============================
	// create difficulty buttons
	// ==============================
	{
		leftRect := a.GetDifficultyButtonRect(false)
		rightRect := a.GetDifficultyButtonRect(true)

		a.DifficultyButtonLeft = &ImageButton{
			BaseButton: BaseButton{
				Rect: leftRect,
				OnClick: func() {
					a.Difficulty -= 1
					a.Difficulty = max(a.Difficulty, 0)
					initBoard()
				},
			},

			Image:        SpriteSubImage(TileSprite, 11),
			ImageOnHover: SpriteSubImage(TileSprite, 11),
			ImageOnDown:  SpriteSubImage(TileSprite, 13),

			ImageColor:        color.NRGBA{255, 255, 255, 255},
			ImageColorOnHover: color.NRGBA{255, 255, 255, 255},
			ImageColorOnDown:  color.NRGBA{255, 255, 255, 255},
		}

		a.DifficultyButtonRight = &ImageButton{
			BaseButton: BaseButton{
				Rect: rightRect,
				OnClick: func() {
					a.Difficulty += 1
					a.Difficulty = min(a.Difficulty, DifficultySize-1)
					initBoard()
				},
			},
			Image:        SpriteSubImage(TileSprite, 12),
			ImageOnHover: SpriteSubImage(TileSprite, 12),
			ImageOnDown:  SpriteSubImage(TileSprite, 14),

			ImageColor:        color.NRGBA{255, 255, 255, 255},
			ImageColorOnHover: color.NRGBA{255, 255, 255, 255},
			ImageColorOnDown:  color.NRGBA{255, 255, 255, 255},
		}
	}

	return a
}

func (a *App) BoardRect() FRectangle {
	size := min(ScreenWidth, ScreenHeight) * a.BoardSizeRatio
	halfSize := size * 0.5
	halfWidth := ScreenWidth * 0.5
	halfHeight := ScreenHeight * 0.5
	return FRect(
		halfWidth-halfSize, halfHeight-halfSize,
		halfWidth+halfSize, halfHeight+halfSize,
	)
}

func (a *App) GetTopUIRect() FRectangle {
	boardRect := a.BoardRect()
	return FRect(
		boardRect.Min.X+a.TopUIMarginHorizontal,
		a.TopUIMarginTop,
		boardRect.Max.X-a.TopUIMarginHorizontal,
		boardRect.Min.Y-a.TopUIMarginBottom,
	)
}

func (a *App) GetDifficultyButtonRect(forRight bool) FRectangle {
	parentRect := a.GetTopUIRect()
	width := parentRect.Dx() * a.TopUIButtonButtonRatio

	if forRight {
		return FRect(
			parentRect.Max.X-width, parentRect.Min.Y,
			parentRect.Max.X, parentRect.Max.Y,
		)
	} else {
		return FRect(
			parentRect.Min.X, parentRect.Min.Y,
			parentRect.Min.X+width, parentRect.Max.Y,
		)
	}
}

func (a *App) GetDifficultyTextRect() FRectangle {
	parentRect := a.GetTopUIRect()
	width := parentRect.Dx() * a.TopUIButtonTextRatio

	rect := FRectWH(width, parentRect.Dy())

	pCenter := FRectangleCenter(parentRect)
	rect = CenterFRectangle(rect, pCenter.X, pCenter.Y)

	return rect
}

func (a *App) MousePosToBoardPos(mousePos FPoint) (int, int) {
	boardRect := a.BoardRect()

	// if mouse is outside the board return -1
	if !mousePos.In(boardRect) {
		return -1, -1
	}

	mousePos.X -= boardRect.Min.X
	mousePos.Y -= boardRect.Min.Y

	boardX := int(math.Floor(mousePos.X / (boardRect.Dx() / float64(a.Board.Width))))
	boardY := int(math.Floor(mousePos.Y / (boardRect.Dy() / float64(a.Board.Height))))

	boardX = min(boardX, a.Board.Width-1)
	boardY = min(boardY, a.Board.Height-1)

	return boardX, boardY
}

func (a *App) SetTileHightlight(boardX, boardY int) {
	a.TileHighLights[boardX][boardY].Brightness = 1
}

func (a *App) Update() error {

	// ==========================
	// update global timer
	// ==========================
	UpdateGlobalTimer()
	// ==========================

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
				if !a.Board.Flags[boardX][boardY] {
					if !a.BoardTouched { // mine is not placed
						a.Board.PlaceMines(a.MineCount[a.Difficulty], boardX, boardY)
						a.Board.SpreadSafeArea(boardX, boardY)

						iter := NewBoardIterator(
							0, 0, a.Board.Width-1, a.Board.Height-1,
						)
						// remove any flags that might have been placed
						for iter.HasNext() {
							x, y := iter.GetNext()
							a.Board.Flags[x][y] = false
						}
						a.BoardTouched = true
					} else { // mine has been placed
						if !a.Board.Mines[boardX][boardY] {
							a.Board.SpreadSafeArea(boardX, boardY)
						} else {
							a.GameState = GameStateLost
						}
					}
				}
			} else if IsMouseButtonJustPressed(eb.MouseButtonRight) {
				a.Board.Flags[boardX][boardY] = !a.Board.Flags[boardX][boardY]
			}
		}
	}

	if prevState != a.GameState {
		// reset GameResultAnimProgress
		if a.GameState == GameStateWon || a.GameState == GameStateLost {
			a.GameResultAnimTimer.Current = 0
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
	if a.GameState == GameStateWon || a.GameState == GameStateLost {
		a.GameResultAnimTimer.TickUp()
	}

	// ==========================
	// update TopMenuShowAnimT
	// ==========================
	if a.BoardTouched {
		a.TopMenuShowAnimTimer.TickDown()
	} else {
		a.TopMenuShowAnimTimer.TickUp()
	}

	// ==========================
	// update buttons
	// ==========================
	a.DifficultyButtonLeft.Disabled = a.BoardTouched
	a.DifficultyButtonRight.Disabled = a.BoardTouched

	// update button rect
	{
		lRect := a.GetDifficultyButtonRect(false)
		rRect := a.GetDifficultyButtonRect(true)

		t := f64(a.TopMenuShowAnimTimer.Current) / f64(a.TopMenuShowAnimTimer.Duration)
		t = Clamp(t, 0, 1)

		lRectY := Lerp(-lRect.Dy()-10, lRect.Min.Y, t)
		rRectY := Lerp(-rRect.Dy()-10, rRect.Min.Y, t)

		lRect = FRectMoveTo(lRect, FPt(lRect.Min.X, lRectY))
		rRect = FRectMoveTo(rRect, FPt(rRect.Min.X, rRectY))

		a.DifficultyButtonLeft.Rect = lRect
		a.DifficultyButtonRight.Rect = rRect
	}

	a.DifficultyButtonLeft.Update()
	a.DifficultyButtonRight.Update()
	// ==========================

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
	boardRect := a.BoardRect()

	tileWidth := boardRect.Dx() / f64(a.Board.Width)
	tileHeight := boardRect.Dy() / f64(a.Board.Height)

	return FRectangle{
		Min: FPt(f64(boardX)*tileWidth, f64(boardY)*tileHeight).Add(boardRect.Min),
		Max: FPt(f64(boardX+1)*tileWidth, f64(boardY+1)*tileHeight).Add(boardRect.Min),
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
	op.Filter = eb.FilterLinear

	dst.DrawImage(tile, op)
}

func (a *App) DrawGameResult(dst *eb.Image) {
	if !(a.GameState == GameStateLost || a.GameState == GameStateWon) {
		return
	}

	var resultStr string

	if a.GameState == GameStateLost {
		resultStr = "You Lost!"
	} else {
		resultStr = "You Won!"
	}

	textW, textH := ebt.Measure(resultStr, FontFace, FontLineSpacing(FontFace))

	const textSize = 80
	scale := textSize / FontSize(FontFace)

	scaledW, scaledH := textW*scale, textH*scale

	startMinY := 0 - scaledH - 10 // up extra 10 just to be safe
	endMinY := f64(ScreenHeight)*0.5 - scaledH*0.5

	t := f64(a.GameResultAnimTimer.Current) / f64(a.GameResultAnimTimer.Duration)
	t = Clamp(t, 0, 1)

	minY := Lerp(startMinY, endMinY, t)
	minX := f64(ScreenWidth)*0.5 - scaledW*0.5

	op := &ebt.DrawOptions{}
	op.LayoutOptions.LineSpacing = FontLineSpacing(FontFace)

	op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
	op.GeoM.Translate(minX+scaledW*0.5, minY+scaledH*0.5)
	op.Filter = eb.FilterLinear

	if a.GameState == GameStateLost {
		op.ColorScale.ScaleWithColor(
			color.NRGBA{245, 24, 24, 255},
		)
	} else {
		op.ColorScale.ScaleWithColor(
			color.NRGBA{255, 215, 18, 255},
		)
	}

	ebt.Draw(dst, resultStr, FontFace, op)
}

func (a *App) DrawDifficultyText(dst *eb.Image) {
	var maxW, maxH float64
	var textW, textH float64

	// TODO : cache this if you can
	for d := Difficulty(0); d < DifficultySize; d++ {
		str := DifficultyStrs[d]
		w, h := ebt.Measure(str, FontFace, FontLineSpacing(FontFace))
		maxW = max(w, maxW)
		maxH = max(h, maxH)

		if d == a.Difficulty {
			textW, textH = w, h
		}
	}

	rect := a.GetDifficultyTextRect()

	t := f64(a.TopMenuShowAnimTimer.Current) / f64(a.TopMenuShowAnimTimer.Duration)
	t = Clamp(t, 0, 1)

	rectY := Lerp(-rect.Dy()-10, rect.Min.Y, t)
	rect = FRectMoveTo(rect, FPt(rect.Min.X, rectY))

	scale := min(rect.Dx()/maxW, rect.Dy()/maxH)

	rectCenter := FRectangleCenter(rect)

	op := &ebt.DrawOptions{}
	op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)

	op.Filter = eb.FilterLinear

	op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})

	ebt.Draw(dst, DifficultyStrs[a.Difficulty], FontFace, op)
}

func (a *App) Draw(dst *eb.Image) {
	boardRect := a.BoardRect()
	// ===========================
	// draw board
	// ===========================
	tileWidth := boardRect.Dx() / f64(a.Board.Width)
	tileHeight := boardRect.Dy() / f64(a.Board.Height)

	regularBg := color.NRGBA{100, 100, 100, 255}
	revealedBg := color.NRGBA{200, 200, 200, 255}

	for y := 0; y < a.Board.Height; y++ {
		for x := 0; x < a.Board.Width; x++ {
			tileX := f64(x)*tileWidth + boardRect.Min.X
			tileY := f64(y)*tileHeight + boardRect.Min.Y

			// draw the tile background
			bgColor := regularBg

			if a.Board.Revealed[x][y] {
				bgColor = revealedBg
			}

			ebv.DrawFilledRect(
				dst,
				f32(tileX), f32(tileY), f32(tileWidth), f32(tileHeight),
				bgColor,
				true,
			)

			// draw flags
			if a.Board.Flags[x][y] {
				a.DrawTile(dst, x, y, GetFlagTile())
			}

			// draw mines
			if a.GameState == GameStateLost && a.Board.Mines[x][y] && !a.Board.Flags[x][y] {
				a.DrawTile(dst, x, y, GetMineTile())
			}

			if a.Board.Revealed[x][y] {
				if count := a.Board.GetNeighborMineCount(x, y); count > 0 {
					a.DrawTile(dst, x, y, GetNumberTile(count))
				}
			}

			// draw highlight
			if a.TileHighLights[x][y].Brightness > 0 {
				t := a.TileHighLights[x][y].Brightness
				ebv.DrawFilledRect(
					dst,
					f32(tileX), f32(tileY), f32(tileWidth), f32(tileHeight),
					color.NRGBA{255, 255, 255, uint8(t * 255)},
					true,
				)
			}

			// draw border
			ebv.StrokeRect(
				dst,
				f32(tileX), f32(tileY), f32(tileWidth), f32(tileHeight),
				1,
				color.NRGBA{0, 0, 0, 255},
				true,
			)
		}
	}

	a.DrawGameResult(dst)

	a.DifficultyButtonLeft.Draw(dst)
	a.DifficultyButtonRight.Draw(dst)

	a.DrawDifficultyText(dst)
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	ScreenWidth = f64(outsideWidth)
	ScreenHeight = f64(outsideHeight)

	return outsideWidth, outsideHeight
}

func main() {
	LoadAssets()

	app := NewApp()

	eb.SetVsyncEnabled(true)
	eb.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	eb.SetWindowResizingMode(eb.WindowResizingModeEnabled)
	eb.SetWindowTitle("test")

	if err := eb.RunGame(app); err != nil {
		panic(err)
	}
}
