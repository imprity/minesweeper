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

type RetryPopup struct {
	DoShow bool
	DidWin bool

	Button *ImageButton

	PopupHeightRatio      float64 // relative to min(ScreenWidth, ScreenHeight)
	ButtonHeightRatio     float64 // relative to popup's height
	TextHeightRatio       float64 // relative to popup's height
	ButtonTextMarginRatio float64 // margin between retry button and text, relative to popup's height

	AnimTimer Timer
}

func NewRetryPopup() *RetryPopup {
	rp := new(RetryPopup)

	rp.PopupHeightRatio = 0.35
	rp.ButtonHeightRatio = 0.5
	rp.TextHeightRatio = 0.2
	rp.ButtonTextMarginRatio = 0.1

	rp.AnimTimer = Timer{
		Duration: time.Millisecond * 100,
	}

	rp.Button = &ImageButton{
		Image:        SpriteSubView(TileSprite, 17),
		ImageOnHover: SpriteSubView(TileSprite, 17),
		ImageOnDown:  SpriteSubView(TileSprite, 17),

		ImageColor:        color.NRGBA{0, 255, 0, 100},
		ImageColorOnHover: color.NRGBA{0, 255, 0, 255},
		ImageColorOnDown:  color.NRGBA{0, 255, 0, 255},
	}

	return rp
}

func (rp *RetryPopup) RegisterButtonCallback(cb func()) {
	rp.Button.OnClick = cb
}

func (rp *RetryPopup) PopupRect() FRectangle {
	animT := f64(rp.AnimTimer.Current) / f64(rp.AnimTimer.Duration)
	animT = Clamp(animT, 0, 1)

	height := min(ScreenWidth, ScreenHeight) * rp.PopupHeightRatio
	rect := FRectWH(height, height)

	rectCenterX, rectCenterY := ScreenWidth*0.5, ScreenHeight*0.5

	rectCenterY += (1 - animT) * 60 // give a little bit of an offset

	return CenterFRectangle(rect, rectCenterX, rectCenterY)
}

func (rp *RetryPopup) ButtonRect() FRectangle {
	pRect := rp.PopupRect()

	childrenHeight := (rp.TextHeightRatio +
		rp.ButtonTextMarginRatio +
		rp.ButtonHeightRatio) * pRect.Dy()

	childrenMaxY := pRect.Max.Y - pRect.Dy()*0.5 + childrenHeight*0.5

	buttonSize := rp.ButtonHeightRatio * pRect.Dy()
	buttonRect := FRectWH(buttonSize, buttonSize)

	buttonRect = FRectMoveTo(buttonRect, ScreenWidth*0.5-buttonSize*0.5, childrenMaxY-buttonSize)

	return buttonRect
}

func (rp *RetryPopup) TextRect() FRectangle {
	pRect := rp.PopupRect()

	childrenHeight := (rp.TextHeightRatio +
		rp.ButtonTextMarginRatio +
		rp.ButtonHeightRatio) * pRect.Dy()

	childrenMinY := pRect.Min.Y + pRect.Dy()*0.5 - childrenHeight*0.5

	textHeight := rp.TextHeightRatio * pRect.Dy()
	textRect := FRectWH(pRect.Dx(), textHeight)

	textRect = FRectMoveTo(textRect, ScreenWidth*0.5-textRect.Dx()*0.5, childrenMinY)

	return textRect
}

func (rp *RetryPopup) Update() error {
	// update button
	rp.Button.Disabled = !rp.DoShow

	//update timer
	if rp.DoShow {
		rp.AnimTimer.TickUp()
	} else {
		rp.AnimTimer.Current = 0
	}

	rp.Button.Rect = rp.ButtonRect()

	rp.Button.Update()
	return nil
}

func (rp *RetryPopup) Draw(dst *eb.Image) {
	if rp.DoShow {
		// draw popup background
		popupRect := rp.PopupRect()
		DrawFilledRect(dst, popupRect, color.NRGBA{255, 255, 255, 255}, true)

		// draw text
		{
			text := "You Lost!"
			if rp.DidWin {
				text = "You Won!"
			}

			textW, textH := ebt.Measure(text, FontFace, FontLineSpacing(FontFace))
			textRect := rp.TextRect()
			scale := min(textRect.Dx()/textW, textRect.Dy()/textH)

			op := &ebt.DrawOptions{}
			op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
			center := FRectangleCenter(textRect)
			op.GeoM.Translate(center.X, center.Y)
			op.ColorScale.ScaleWithColor(color.NRGBA{0, 0, 0, 255})
			op.Filter = eb.FilterLinear

			ebt.Draw(dst, text, FontFace, op)
		}

		// draw button
		rp.Button.Draw(dst)
	}
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

	RetryPopup *RetryPopup

	DifficultyButtonLeft  *ImageButton
	DifficultyButtonRight *ImageButton

	TopMenuShowAnimTimer Timer

	BoardSizeRatio float64 // relative to min(ScreenWidth, ScreenHeight)

	TopUIMarginHorizontal float64
	TopUIMarginTop        float64
	TopUIMarginBottom     float64

	TopUIButtonButtonRatio float64
	TopUIButtonTextRatio   float64

	DebugMode bool
}

func NewApp() *App {
	a := new(App)

	a.RetryPopup = NewRetryPopup()

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
		a.BoardTouched = false
	}
	initBoard()

	a.RetryPopup.RegisterButtonCallback(func() {
		initBoard()
		a.GameState = GameStatePlaying
	})

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

			Image:        SpriteSubView(TileSprite, 11),
			ImageOnHover: SpriteSubView(TileSprite, 11),
			ImageOnDown:  SpriteSubView(TileSprite, 13),

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
			Image:        SpriteSubView(TileSprite, 12),
			ImageOnHover: SpriteSubView(TileSprite, 12),
			ImageOnDown:  SpriteSubView(TileSprite, 14),

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
	_ = prevState // might be handy later

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

	// update highlights
	if !(IsMouseButtonPressed(eb.MouseButtonLeft) || IsMouseButtonPressed(eb.MouseButtonRight)) {
		for y := 0; y < a.Board.Height; y++ {
			for x := 0; x < a.Board.Width; x++ {
				a.TileHighLights[x][y].Brightness -= f64(UpdateDelta()) / f64(a.HighlightDuraiton)
				a.TileHighLights[x][y].Brightness = max(a.TileHighLights[x][y].Brightness, 0)
			}
		}
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
	// update top menu buttons
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

		lRect = FRectMoveTo(lRect, lRect.Min.X, lRectY)
		rRect = FRectMoveTo(rRect, rRect.Min.X, rRectY)

		a.DifficultyButtonLeft.Rect = lRect
		a.DifficultyButtonRight.Rect = rRect
	}

	a.DifficultyButtonLeft.Update()
	a.DifficultyButtonRight.Update()
	// ==========================

	a.RetryPopup.DoShow = a.GameState == GameStateLost || a.GameState == GameStateWon
	a.RetryPopup.DidWin = a.GameState == GameStateWon

	a.RetryPopup.Update()

	// TEST TEST TEST TEST TEST TEST
	if IsKeyJustPressed(eb.KeyF1) {
		a.DebugMode = !a.DebugMode
	}
	// TEST TEST TEST TEST TEST TEST

	return nil
}

func GetNumberTile(number int) SubView {
	if !(1 <= number && number <= 9) {
		ErrorLogger.Fatalf("%d is not a valid number", number)
	}

	return SpriteSubView(TileSprite, number-1)
}

func GetMineTile() SubView {
	return SpriteSubView(TileSprite, 9)
}

func GetFlagTile() SubView {
	return SpriteSubView(TileSprite, 10)
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

func (a *App) DrawTile(dst *eb.Image, boardX, boardY int, tile SubView, clr color.Color) {
	tileRect := a.GetTileRect(boardX, boardY)

	imgSize := ImageSizeFPt(tile)
	rectSize := tileRect.Size()

	scale := min(rectSize.X, rectSize.Y) / max(imgSize.X, imgSize.Y)

	op := &DrawSubViewOptions{}
	op.GeoM.Concat(TransformToCenter(imgSize.X, imgSize.Y, scale, scale, 0))
	rectCenter := FRectangleCenter(tileRect)
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)
	op.ColorScale.ScaleWithColor(clr)
	op.Filter = eb.FilterLinear

	DrawSubView(dst, tile, op)
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
	rect = FRectMoveTo(rect, rect.Min.X, rectY)

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
				a.DrawTile(dst, x, y, GetFlagTile(), color.NRGBA{255, 255, 255, 255})
			}

			// draw mines
			if a.GameState == GameStateLost && a.Board.Mines[x][y] && !a.Board.Flags[x][y] {
				a.DrawTile(dst, x, y, GetMineTile(), color.NRGBA{255, 255, 255, 255})
			}

			if a.Board.Revealed[x][y] {
				if count := a.Board.GetNeighborMineCount(x, y); count > 0 {
					a.DrawTile(dst, x, y, GetNumberTile(count), color.NRGBA{255, 255, 255, 255})
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

			// TEST TEST TEST TEST TEST TEST
			if a.DebugMode && a.Board.Mines[x][y] {
				a.DrawTile(dst, x, y, GetMineTile(), color.NRGBA{255, 0, 0, 255})
			}
			// TEST TEST TEST TEST TEST TEST
		}
	}

	a.DifficultyButtonLeft.Draw(dst)
	a.DifficultyButtonRight.Draw(dst)

	a.DrawDifficultyText(dst)

	a.RetryPopup.Draw(dst)
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
