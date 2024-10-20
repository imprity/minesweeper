package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

var _ = fmt.Printf

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

type RetryButton struct {
	BaseButton

	ButtonHoverOffset float64
}

func NewRetryButton() *RetryButton {
	rb := new(RetryButton)

	return rb
}

func (rb *RetryButton) Update() {
	rb.BaseButton.Update()

	if rb.State == ButtonStateHover {
		rb.ButtonHoverOffset = Lerp(rb.ButtonHoverOffset, 1, 0.3)
	} else if rb.State == ButtonStateDown {
		rb.ButtonHoverOffset = 0
	} else {
		rb.ButtonHoverOffset = Lerp(rb.ButtonHoverOffset, 0, 0.3)
	}

	if rb.Disabled {
		rb.ButtonHoverOffset = 0
	}
}

func (rb *RetryButton) Draw(dst *eb.Image) {
	bottomRect := FRectWH(rb.Rect.Dx(), rb.Rect.Dy()*0.95)
	topRect := bottomRect

	topRect = FRectMoveTo(topRect, rb.Rect.Min.X, rb.Rect.Min.Y)
	bottomRect = FRectMoveTo(bottomRect, rb.Rect.Min.X, rb.Rect.Max.Y-bottomRect.Dy())

	if rb.State == ButtonStateDown {
		topRect = FRectMoveTo(topRect, bottomRect.Min.X, bottomRect.Min.Y)
	} else if rb.State == ButtonStateHover {
		topRect = topRect.Add(FPt(0, -topRect.Dy()*0.025*rb.ButtonHoverOffset))
	}

	const segments = 6
	const radius = 0.4

	radiusPx := min(bottomRect.Dx(), bottomRect.Dy()) * 0.5 * radius

	DrawFilledRoundRectFast(
		dst,
		bottomRect,
		radiusPx,
		segments,
		color.NRGBA{0, 0, 0, 255},
	)

	DrawFilledRoundRectFast(
		dst,
		topRect,
		radiusPx,
		segments,
		color.NRGBA{105, 223, 145, 255},
	)

	imgRect := RectToFRect(RetryButtonImage.Bounds())
	scale := min(topRect.Dx(), topRect.Dy()) / max(imgRect.Dx(), imgRect.Dy())
	scale *= 0.6

	center := FRectangleCenter(topRect)

	op := &DrawImageOptions{}
	op.GeoM.Concat(TransformToCenter(imgRect.Dx(), imgRect.Dy(), scale, scale, 0))
	op.GeoM.Translate(center.X, center.Y-topRect.Dy()*0.02)
	op.ColorScale.ScaleWithColor(color.NRGBA{0, 0, 0, 255})

	DrawImage(dst, RetryButtonImage, op)

	op.GeoM.Translate(0, topRect.Dy()*0.02*2)
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})

	DrawImage(dst, RetryButtonImage, op)
}

type ColorTablePicker struct {
	DoShow bool

	ColorPicker *ColorPicker

	TableIndex ColorTableIndex

	wasShowing bool
}

func NewColorTablePicker() *ColorTablePicker {
	ct := new(ColorTablePicker)
	ct.ColorPicker = NewColorPicker()

	return ct
}

func (ct *ColorTablePicker) Update() {
	if !ct.DoShow {
		return
	}

	if !ct.wasShowing && ct.DoShow {
		ct.ColorPicker.SetColor(TheColorTable[ct.TableIndex])
	}
	ct.wasShowing = ct.DoShow

	ct.ColorPicker.Rect = FRectWH(200, 400)
	ct.ColorPicker.Rect = FRectMoveTo(ct.ColorPicker.Rect, ScreenWidth-210, 10)
	ct.ColorPicker.Update()

	const firstRate = 200 * time.Millisecond
	const repeatRate = 50 * time.Millisecond
	changed := false

	if HandleKeyRepeat(firstRate, repeatRate, ColorPickerUpKey) {
		ct.TableIndex--
		changed = true
	}
	if HandleKeyRepeat(firstRate, repeatRate, ColorPickerDownKey) {
		ct.TableIndex++
		changed = true
	}
	ct.TableIndex = Clamp(ct.TableIndex, 0, ColorTableSize-1)

	if changed {
		ct.ColorPicker.SetColor(TheColorTable[ct.TableIndex])
	}

	TheColorTable[ct.TableIndex] = ct.ColorPicker.Color()
}

func (ct *ColorTablePicker) Draw(dst *eb.Image) {
	if !ct.DoShow {
		return
	}

	ct.ColorPicker.Draw(dst)

	// draw list of table entries
	{
		const textScale = 0.3

		lineSpacing := FontLineSpacing(ClearFace)

		// get bg width
		bgWidth := float64(0)
		for i := ColorTableIndex(0); i < ColorTableSize; i++ {
			text := i.String()
			w, _ := ebt.Measure(text, ClearFace, lineSpacing)
			bgWidth = max(bgWidth, w*textScale)
		}
		bgHeight := lineSpacing * textScale * f64(ColorTableSize)

		bgWidth += 20
		bgHeight += 20

		// draw bg
		DrawFilledRect(
			dst, FRectWH(bgWidth, bgHeight), color.NRGBA{0, 0, 0, 150},
		)

		// draw list texts
		offsetY := float64(0)

		for i := ColorTableIndex(0); i < ColorTableSize; i++ {
			text := i.String()
			op := &DrawTextOptions{}

			op.GeoM.Scale(textScale, textScale)
			op.GeoM.Translate(0, offsetY)
			if i == ct.TableIndex {
				op.ColorScale.ScaleWithColor(color.NRGBA{255, 0, 0, 255})
			} else {
				op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
			}

			DrawText(dst, text, ClearFace, op)

			offsetY += lineSpacing * textScale
		}
	}
}

type DifficultySelectUI struct {
	DoShow bool

	DifficultyButtonLeft  *ImageButton
	DifficultyButtonRight *ImageButton

	TopMenuShowAnimTimer Timer

	TopUIMarginHorizontal float64 // constant
	TopUIMarginTop        float64 // constant
	TopUIMarginBottom     float64 // constant

	TopUIButtonButtonRatio float64 // constant
	TopUIButtonTextRatio   float64 // constant

	Difficulty         Difficulty
	OnDifficultyChange func(difficulty Difficulty)
}

func NewDifficultySelectUI(boardRect FRectangle) *DifficultySelectUI {
	ds := new(DifficultySelectUI)

	ds.TopMenuShowAnimTimer = Timer{
		Duration: time.Millisecond * 200,
	}
	ds.TopMenuShowAnimTimer.Current = ds.TopMenuShowAnimTimer.Duration

	ds.TopUIMarginHorizontal = 5
	ds.TopUIMarginTop = 5
	ds.TopUIMarginBottom = 5

	ds.TopUIButtonButtonRatio = 0.2
	ds.TopUIButtonTextRatio = 0.5

	// ==============================
	// create difficulty buttons
	// ==============================
	{
		leftRect := ds.GetDifficultyButtonRect(boardRect, false)
		rightRect := ds.GetDifficultyButtonRect(boardRect, true)

		// DifficultyButtonLeft
		ds.DifficultyButtonLeft = NewImageButton()

		ds.DifficultyButtonLeft.Rect = leftRect
		ds.DifficultyButtonLeft.OnClick = func() {
			prevDifficulty := ds.Difficulty
			ds.Difficulty = max(ds.Difficulty-1, 0)
			if ds.OnDifficultyChange != nil && prevDifficulty != ds.Difficulty {
				ds.OnDifficultyChange(ds.Difficulty)
			}
		}

		ds.DifficultyButtonLeft.Image = SpriteSubView(TileSprite, 11)
		ds.DifficultyButtonLeft.ImageOnHover = SpriteSubView(TileSprite, 11)
		ds.DifficultyButtonLeft.ImageOnDown = SpriteSubView(TileSprite, 13)

		ds.DifficultyButtonLeft.ImageColor = ColorTopUIButton
		ds.DifficultyButtonLeft.ImageColorOnHover = ColorTopUIButtonOnHover
		ds.DifficultyButtonLeft.ImageColorOnDown = ColorTopUIButtonOnDown

		// DifficultyButtonRight
		ds.DifficultyButtonRight = NewImageButton()

		ds.DifficultyButtonRight.Rect = rightRect
		ds.DifficultyButtonRight.OnClick = func() {
			prevDifficulty := ds.Difficulty
			ds.Difficulty = min(ds.Difficulty+1, DifficultySize-1)
			if ds.OnDifficultyChange != nil && prevDifficulty != ds.Difficulty {
				ds.OnDifficultyChange(ds.Difficulty)
			}
		}

		ds.DifficultyButtonRight.Image = SpriteSubView(TileSprite, 12)
		ds.DifficultyButtonRight.ImageOnHover = SpriteSubView(TileSprite, 12)
		ds.DifficultyButtonRight.ImageOnDown = SpriteSubView(TileSprite, 14)

		ds.DifficultyButtonRight.ImageColor = ColorTopUIButton
		ds.DifficultyButtonRight.ImageColorOnHover = ColorTopUIButtonOnHover
		ds.DifficultyButtonRight.ImageColorOnDown = ColorTopUIButtonOnDown
	}

	return ds
}

func (ds *DifficultySelectUI) GetTopUIRect(boardRect FRectangle) FRectangle {
	return FRect(
		boardRect.Min.X+ds.TopUIMarginHorizontal,
		ds.TopUIMarginTop,
		boardRect.Max.X-ds.TopUIMarginHorizontal,
		boardRect.Min.Y-ds.TopUIMarginBottom,
	)
}

func (ds *DifficultySelectUI) GetDifficultyButtonRect(boardRect FRectangle, forRight bool) FRectangle {
	parentRect := ds.GetTopUIRect(boardRect)
	width := parentRect.Dx() * ds.TopUIButtonButtonRatio

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

func (ds *DifficultySelectUI) GetDifficultyTextRect(boardRect FRectangle) FRectangle {
	parentRect := ds.GetTopUIRect(boardRect)
	width := parentRect.Dx() * ds.TopUIButtonTextRatio

	rect := FRectWH(width, parentRect.Dy())

	pCenter := FRectangleCenter(parentRect)
	rect = CenterFRectangle(rect, pCenter.X, pCenter.Y)

	return rect
}

func (ds *DifficultySelectUI) Update(boardRect FRectangle) {
	if ds.DoShow {
		ds.TopMenuShowAnimTimer.TickUp()
	} else {
		ds.TopMenuShowAnimTimer.TickDown()
	}
	ds.TopMenuShowAnimTimer.ClampCurrent()

	ds.DifficultyButtonLeft.Disabled = !ds.DoShow
	ds.DifficultyButtonRight.Disabled = !ds.DoShow

	// update button rect
	{
		lRect := ds.GetDifficultyButtonRect(boardRect, false)
		rRect := ds.GetDifficultyButtonRect(boardRect, true)

		t := ds.TopMenuShowAnimTimer.Normalize()

		lRectY := Lerp(-lRect.Dy()-10, lRect.Min.Y, t)
		rRectY := Lerp(-rRect.Dy()-10, rRect.Min.Y, t)

		lRect = FRectMoveTo(lRect, lRect.Min.X, lRectY)
		rRect = FRectMoveTo(rRect, rRect.Min.X, rRectY)

		ds.DifficultyButtonLeft.Rect = lRect
		ds.DifficultyButtonRight.Rect = rRect
	}

	ds.DifficultyButtonLeft.Update()
	ds.DifficultyButtonRight.Update()
	// ==========================
}

func (ds *DifficultySelectUI) DrawDifficultyText(dst *eb.Image, boardRect FRectangle) {
	var maxW, maxH float64
	var textW, textH float64

	// TODO : cache this if you can
	for d := Difficulty(0); d < DifficultySize; d++ {
		str := DifficultyStrs[d]
		w, h := ebt.Measure(str, DecoFace, FontLineSpacing(DecoFace))
		maxW = max(w, maxW)
		maxH = max(h, maxH)

		if d == ds.Difficulty {
			textW, textH = w, h
		}
	}

	rect := ds.GetDifficultyTextRect(boardRect)

	t := ds.TopMenuShowAnimTimer.Normalize()

	rectY := Lerp(-rect.Dy()-10, rect.Min.Y, t)
	rect = FRectMoveTo(rect, rect.Min.X, rectY)

	scale := min(rect.Dx()/maxW, rect.Dy()/maxH)

	rectCenter := FRectangleCenter(rect)

	op := &DrawTextOptions{}
	op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)

	op.ColorScale.ScaleWithColor(TheColorTable[ColorTopUITitle])

	DrawText(dst, DifficultyStrs[ds.Difficulty], DecoFace, op)
}

func (ds *DifficultySelectUI) Draw(dst *eb.Image, boardRect FRectangle) {
	ds.DifficultyButtonLeft.Draw(dst)
	ds.DifficultyButtonRight.Draw(dst)

	ds.DrawDifficultyText(dst, boardRect)
}

type TileFgType int

const (
	TileFgTypeNone TileFgType = iota
	TileFgTypeNumber
	TileFgTypeFlag
)

type TileStyle struct {
	DrawBg bool

	BgScale   float64
	BgOffsetX float64
	BgOffsetY float64

	BgFillColor      color.Color
	BgTileHightlight float64

	BgBombAnim float64

	DrawTile bool // does not affect fg

	TileScale   float64
	TileOffsetX float64
	TileOffsetY float64

	TileFillColor   color.Color
	TileStrokeColor color.Color

	DrawFg bool

	FgScale   float64 // relative to tile's scale
	FgOffsetX float64 // relative to tile's center
	FgOffsetY float64 // relative to tile's center

	FgColor color.Color

	FgType TileFgType

	Alpha float64
}

func NewTileStyle() TileStyle {
	return TileStyle{
		BgScale:         1,
		BgFillColor:     color.NRGBA{0, 0, 0, 0},
		TileScale:       1,
		TileFillColor:   color.NRGBA{0, 0, 0, 0},
		TileStrokeColor: color.NRGBA{0, 0, 0, 0},
		FgScale:         1,
		FgColor:         color.NRGBA{0, 0, 0, 0},
		Alpha:           1,
	}
}

type TileAnimation interface {
	Update(x, y int, style TileStyle) TileStyle
	Skip(x, y int, style TileStyle) TileStyle
	Done(x, y int, style TileStyle) bool
}

type CallbackTileAnimation struct {
	OnUpdate func(x, y int, style TileStyle) TileStyle
	OnSkip   func(x, y int, style TileStyle) TileStyle
	OnDone   func(x, y int, style TileStyle) bool
}

func (a CallbackTileAnimation) Update(x, y int, style TileStyle) TileStyle {
	return a.OnUpdate(x, y, style)
}

func (a CallbackTileAnimation) Skip(x, y int, style TileStyle) TileStyle {
	return a.OnSkip(x, y, style)
}

func (a CallbackTileAnimation) Done(x, y int, style TileStyle) bool {
	return a.OnDone(x, y, style)
}

type Game struct {
	Board     Board
	PrevBoard Board

	MineCount      [DifficultySize]int // constant
	BoardTileCount [DifficultySize]int // constant

	BoardTouched bool

	BaseTileStyles [][]TileStyle
	TileAnimations [][]CircularQueue[TileAnimation]

	RenderTileStyles [][]TileStyle

	TileHighLightTimer Timer
	TileHighLightX     int
	TileHighLightY     int

	NumberClickTimer Timer
	NumberClickX     int
	NumberClickY     int

	DefeatAnimTimer Timer

	WinAnimTimer      Timer
	WinTilesAnimTimer [][]Timer

	RetryButton          *RetryButton
	RetryButtonAnimTimer Timer
	ShowRetryButton      bool
	RetryButtonShowPoint float64 // constant

	ResetAnimTimer        Timer
	ResetAnimTippingPoint float64 // constant
	ResetAnimTimerStarted bool

	GameState GameState

	Difficulty Difficulty

	DifficultySelectUI *DifficultySelectUI

	BoardSizeRatio float64 // constant, relative to min(ScreenWidth, ScreenHeight)

	RevealMines     bool
	ColorPickerMode bool

	ColorTablePicker *ColorTablePicker

	MaskImage *eb.Image
}

func NewGame() *Game {
	g := new(Game)

	g.MineCount = [DifficultySize]int{
		10, 20, 30,
	}
	g.BoardTileCount = [DifficultySize]int{
		10, 15, 20,
	}

	g.MaskImage = eb.NewImage(int(ScreenWidth), int(ScreenHeight))

	g.TileHighLightTimer.Duration = time.Millisecond * 100
	g.NumberClickTimer.Duration = time.Millisecond * 30

	g.BoardSizeRatio = 0.8

	g.RetryButtonAnimTimer.Duration = time.Millisecond * 600

	g.RetryButtonShowPoint = 0.4

	g.RetryButton = NewRetryButton()
	g.RetryButton.ActOnRelease = true

	g.RetryButton.OnClick = func() {
		g.ResetAnimTimerStarted = true
	}

	g.ResetAnimTimer.Duration = time.Millisecond * 400
	g.ResetAnimTippingPoint = 0.3

	g.ResetBoard(g.BoardTileCount[g.Difficulty], g.BoardTileCount[g.Difficulty])

	g.DifficultySelectUI = NewDifficultySelectUI(g.BoardRect())
	g.DifficultySelectUI.OnDifficultyChange = func(d Difficulty) {
		g.Difficulty = d
		g.ResetBoard(g.BoardTileCount[d], g.BoardTileCount[d])
	}

	g.ColorTablePicker = NewColorTablePicker()

	return g
}

func (g *Game) ResetBoard(width, height int) {
	g.BoardTouched = false

	g.Board = NewBoard(width, height)
	g.PrevBoard = NewBoard(width, height)

	g.WinTilesAnimTimer = New2DArray[Timer](width, height)

	g.WinAnimTimer.Current = 0

	g.BaseTileStyles = New2DArray[TileStyle](width, height)
	for x := range width {
		for y := range height {
			g.BaseTileStyles[x][y] = NewTileStyle()
		}
	}

	// TODO: This is just a temporary code to set style's background
	// I think we should make a board background reveal animation
	for x := range width {
		for y := range height {
			g.BaseTileStyles[x][y].DrawBg = true

			g.BaseTileStyles[x][y].BgFillColor = ColorTileNormal1
			if IsOddTile(width, height, x, y) {
				g.BaseTileStyles[x][y].BgFillColor = ColorTileNormal2
			}
		}
	}

	g.RenderTileStyles = New2DArray[TileStyle](width, height)

	g.TileAnimations = New2DArray[CircularQueue[TileAnimation]](width, height)
	for x := range width {
		for y := range height {
			// TODO : do we need this much queued animation?
			g.TileAnimations[x][y] = NewCircularQueue[TileAnimation](5)
		}
	}

	g.RetryButtonAnimTimer.Current = 0

	g.ResetAnimTimer.Current = 0

	g.ResetAnimTimerStarted = false
}

func (g *Game) Update() error {
	justPressedL := IsMouseButtonJustPressed(eb.MouseButtonLeft)
	justPressedR := IsMouseButtonJustPressed(eb.MouseButtonRight)
	pressedL := IsMouseButtonPressed(eb.MouseButtonLeft)
	pressedR := IsMouseButtonPressed(eb.MouseButtonRight)

	justPressedM := IsMouseButtonJustPressed(eb.MouseButtonMiddle)
	pressedM := IsMouseButtonPressed(eb.MouseButtonMiddle)

	justPressedAny := justPressedL || justPressedR || justPressedM

	cursor := CursorFPt()

	boardX, boardY := g.MousePosToBoardPos(cursor)

	// =======================================
	prevState := g.GameState
	// =======================================
	_ = prevState // might be handy later

	// =================================
	// handle board interaction
	// =================================

	// =======================================
	stateChanged := false
	// =======================================

	if g.GameState == GameStatePlaying && boardX >= 0 && boardY >= 0 && justPressedAny {

		g.Board.SaveTo(g.PrevBoard)

		if g.Board.Revealed[boardX][boardY] { // interaction on revealed tile
			if (justPressedL && pressedR) || (justPressedR && pressedL) || (justPressedM) { // handle step interaction
				// set number click
				g.NumberClickTimer.Current = g.NumberClickTimer.Duration
				g.NumberClickX = boardX
				g.NumberClickY = boardY

				flagCount := g.Board.GetNeighborFlagCount(boardX, boardY)
				mineCount := g.Board.GetNeighborMineCount(boardX, boardY)

				if flagCount == mineCount {
					// check if flagged correctly
					flaggedCorrectly := true
					missedMine := false

					iter := NewBoardIterator(boardX-1, boardY-1, boardX+1, boardY+1)
					for iter.HasNext() {
						x, y := iter.GetNext()

						if g.Board.IsPosInBoard(x, y) {
							if g.Board.Mines[x][y] != g.Board.Flags[x][y] {
								flaggedCorrectly = false
							}
							if g.Board.Mines[x][y] && !g.Board.Flags[x][y] {
								missedMine = true
							}
							if g.Board.Flags[x][y] {
								flagCount += 1
							}
						}
					}

					if flaggedCorrectly { // if flagged correctly, spread safe area
						iter.Reset()
						for iter.HasNext() {
							x, y := iter.GetNext()
							if g.Board.IsPosInBoard(x, y) {
								g.Board.SpreadSafeArea(x, y)
							}
						}
					} else { // if not, you lost!
						if missedMine {
							g.GameState = GameStateLost
						}
					}
				} else { // if neighbor mine count and flag count is different just highlight the area
					// set tile highlight
					g.TileHighLightTimer.Current = g.TileHighLightTimer.Duration
					g.TileHighLightX = boardX
					g.TileHighLightY = boardY
				}
			}
		} else { // interaction on not revealed tile
			if justPressedL { // one tile stepping
				if !g.Board.Flags[boardX][boardY] {
					if !g.BoardTouched { // first time interaction
						g.BoardTouched = true

						// remove flags that might have been placed
						for x := range g.Board.Width {
							for y := range g.Board.Height {
								g.Board.Flags[x][y] = false
							}
						}

						g.Board.PlaceMines(g.MineCount[g.Difficulty], boardX, boardY)
						g.Board.SpreadSafeArea(boardX, boardY)
					} else { // mine has been placed
						if !g.Board.Mines[boardX][boardY] {
							g.Board.SpreadSafeArea(boardX, boardY)
						} else {
							g.GameState = GameStateLost
						}
					}
				}
			} else if justPressedR { // flagging
				g.Board.Flags[boardX][boardY] = !g.Board.Flags[boardX][boardY]
			}

			// TEST TEST TEST TEST TEST TEST
			if justPressedM {
				g.Board.Revealed[boardX][boardY] = true
			}
			// TEST TEST TEST TEST TEST TEST
		}

		// ==============================
		// check if state has changed
		// ==============================

		// first check game state
		stateChanged = prevState != g.GameState

		// then check board state
		if !stateChanged {
		DIFF_CHECK:
			for x := range g.Board.Width {
				for y := range g.Board.Height {
					if g.Board.Mines[x][y] != g.PrevBoard.Mines[x][y] {
						stateChanged = true
						break DIFF_CHECK
					}

					if g.Board.Flags[x][y] != g.PrevBoard.Flags[x][y] {
						stateChanged = true
						break DIFF_CHECK
					}

					if g.Board.Revealed[x][y] != g.PrevBoard.Revealed[x][y] {
						stateChanged = true
						break DIFF_CHECK
					}
				}
			}
		}

		// ==============================
		// on state changes
		// ==============================
		if stateChanged {
			// remove flags from the revealed tiles
			for x := range g.Board.Width {
				for y := range g.Board.Height {
					if g.Board.Revealed[x][y] {
						g.Board.Flags[x][y] = false
					}
				}
			}

			// check if user has won the game
			if g.Board.IsAllSafeTileRevealed() {
				g.GameState = GameStateWon
			}

			// check if we need to start board reveal animation
		REVEAL_CHECK:
			for x := range g.Board.Width {
				for y := range g.Board.Height {
					if g.Board.Revealed[x][y] && !g.PrevBoard.Revealed[x][y] {
						g.StartRevealAnimation(
							g.PrevBoard.Revealed, g.Board.Revealed, boardX, boardY)

						break REVEAL_CHECK
					}
				}
			}

			if prevState != g.GameState {
				if g.GameState == GameStateLost {
					g.StartDefeatAnimation(boardX, boardY)
				} else if g.GameState == GameStateWon {
					g.StartWinAnimation(boardX, boardY)
				}
			}
		}
	}

	// ============================
	// update animations
	// ============================

	// update BaseTileStyles
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if !g.TileAnimations[x][y].IsEmpty() {
				anim := g.TileAnimations[x][y].At(0)
				g.BaseTileStyles[x][y] = anim.Update(x, y, g.BaseTileStyles[x][y])

				if anim.Done(x, y, g.BaseTileStyles[x][y]) {
					g.TileAnimations[x][y].Dequeue()
				}
			}
		}
	}

	// copy it over to RenderTileStyles
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			g.RenderTileStyles[x][y] = g.BaseTileStyles[x][y]
		}
	}

	// ============================
	// on none user interaction
	// ============================
	if !pressedL && !pressedR && !pressedM {
		g.TileHighLightTimer.TickDown()
		g.NumberClickTimer.TickDown()
	}

	// ===================================
	// update tile highlight
	// ===================================
	if g.TileHighLightTimer.Current >= 0 {
		t := g.TileHighLightTimer.Normalize()
		iter := NewBoardIterator(g.TileHighLightX-1, g.TileHighLightY-1, g.TileHighLightX+1, g.TileHighLightY+1)
		for iter.HasNext() {
			x, y := iter.GetNext()
			if g.Board.IsPosInBoard(x, y) {
				if !g.Board.Revealed[x][y] && !g.Board.Flags[x][y] {
					g.RenderTileStyles[x][y].BgTileHightlight += t
				}
			}
		}
	}

	// ===================================
	// update number click
	// ===================================
	if g.NumberClickTimer.Current >= 0 {
		t := g.NumberClickTimer.Normalize()
		g.RenderTileStyles[g.NumberClickX][g.NumberClickY].FgScale *= (1 + t*0.05)
	}

	// ============================
	// update flag drawing
	// ============================
	if stateChanged {
		for x := range g.Board.Width {
			for y := range g.Board.Height {
				if g.PrevBoard.Flags[x][y] != g.Board.Flags[x][y] {
					if g.Board.Flags[x][y] {
						g.BaseTileStyles[x][y].FgType = TileFgTypeFlag
						g.BaseTileStyles[x][y].FgColor = ColorFlag
						g.BaseTileStyles[x][y].DrawFg = true
					} else {
						g.BaseTileStyles[x][y].FgType = TileFgTypeNone
					}
				}
			}
		}
	}

	// ===================================
	// update defeat animation
	// ===================================
	if g.GameState == GameStateLost {
		g.DefeatAnimTimer.TickUp()
	}

	// skipping defeat animation
	if prevState == GameStateLost {
		if g.DefeatAnimTimer.Current < g.DefeatAnimTimer.Duration && justPressedAny {
			// TODO :
			// nasty hack to stop skipping animation from triggering retry button
			// by delaying RetryPopup from showing up by a tiny bit
			// find some better way to handle it
			g.DefeatAnimTimer.Current = g.DefeatAnimTimer.Duration - time.Millisecond*5

			g.SkipAllTileAnimations()
		}

	}

	// ============================
	// update win animation
	// ============================
	if prevState == GameStateWon {
		g.WinAnimTimer.TickUp()

		for x := range g.Board.Width {
			for y := range g.Board.Height {
				if g.WinTilesAnimTimer[x][y].Duration > 0 {
					g.WinTilesAnimTimer[x][y].TickUp()
				}
			}
		}
	}

	// ===================================
	// update RetryButton
	// ===================================
	{

		if g.GameState == GameStateWon && g.WinAnimTimer.Current >= g.WinAnimTimer.Duration {
			g.ShowRetryButton = true
			g.RetryButtonAnimTimer.TickUp()
		} else if g.GameState == GameStateLost && g.DefeatAnimTimer.Current >= g.DefeatAnimTimer.Duration {
			g.ShowRetryButton = true
			g.RetryButtonAnimTimer.TickUp()
		}

		rect := g.RetryButtonRect()
		t := (g.RetryButtonAnimTimer.Normalize() - g.RetryButtonShowPoint) / (1 - g.RetryButtonShowPoint)
		t = Clamp(t, 0, 1)
		scale := EaseOutElastic(t)

		if g.ResetAnimTimerStarted {
			scale *= Clamp(1-g.ResetAnimTimer.Normalize()/g.ResetAnimTippingPoint, 0, 1)
		}

		rect = FRectScaleCentered(rect, scale)
		g.RetryButton.Rect = rect

		g.RetryButton.Disabled = !(g.ShowRetryButton && scale > 0.5) && !g.ResetAnimTimerStarted
		g.RetryButton.Update()
	}

	// ===================================
	// update restart animation
	// ===================================
	if g.ResetAnimTimerStarted {
		g.ResetAnimTimer.TickUp()
	}
	if g.ResetAnimTimer.Current >= g.ResetAnimTimer.Duration {
		g.ResetBoard(g.BoardTileCount[g.Difficulty], g.BoardTileCount[g.Difficulty])
		g.GameState = GameStatePlaying
	}

	// ===================================
	// update DifficultySelectUI
	// ===================================
	g.DifficultySelectUI.DoShow = !g.BoardTouched
	g.DifficultySelectUI.Update(g.BoardRect())

	// ==========================
	// debug mode
	// ==========================
	if IsKeyJustPressed(ShowMinesKey) {
		g.RevealMines = !g.RevealMines
	}
	if IsKeyJustPressed(SetToDecoBoardKey) {
		g.SetDebugBoardForDecoration()
		g.StartRevealAnimation(
			g.PrevBoard.Revealed, g.Board.Revealed, 0, 0)
	}
	if IsKeyJustPressed(InstantWinKey) {
		g.SetBoardForInstantWin()
		g.StartRevealAnimation(
			g.PrevBoard.Revealed, g.Board.Revealed, 0, 0)
	}

	// ==========================
	// color table picker
	// ==========================
	if IsKeyJustPressed(ShowColorPickerKey) {
		g.ColorTablePicker.DoShow = !g.ColorTablePicker.DoShow
	}
	g.ColorTablePicker.Update()

	return nil
}

func (g *Game) Draw(dst *eb.Image) {
	// clear mask image
	g.MaskImage.Clear()

	// background
	dst.Fill(TheColorTable[ColorBg])

	retryButtonRect := g.RetryButtonRect()
	retryButtonRect = retryButtonRect.Inset(-5)

	getRetryButtonOffset := func(tileRect FRectangle) (float64, bool) {
		overlaps := tileRect.Overlaps(retryButtonRect)
		if !overlaps {
			return 1, false
		}

		if !g.ShowRetryButton {
			return 1, overlaps
		}

		t := g.RetryButtonAnimTimer.Normalize()
		t = t / g.RetryButtonShowPoint
		t = Clamp(t, 0, 1)
		t = EaseInQuint(t)
		return 1 - t, true
	}
	_ = getRetryButtonOffset

	iter := NewBoardIterator(0, 0, g.Board.Width-1, g.Board.Height-1)
	_ = iter

	/*
		// ==============================
		// draw tile background
		// ==============================
		iter.Reset()
		for iter.HasNext() {
			x, y := iter.GetNext()
			tileRect := GetBoardTileRect(g.BoardRect(), g.Board.Width, g.Board.Height, x, y)

			var bgColor color.Color
			var strokeColor color.Color

			bgColor = ColorTable[ColorTileNormal1]
			if IsOddTile(g.Board.Width, g.Board.Height, x, y) {
				bgColor = ColorTable[ColorTileNormal2]
			}

			strokeColor = ColorTable[ColorTileNormalStroke]

			// lerp color if player has won
			if g.GameState == GameStateWon {
				if g.ResetAnimTimer.Normalize() < g.ResetAnimTippingPoint {
					t := EaseOutCirc(g.WinAnimTimer.Normalize())

					bgColor = LerpColorRGBA(bgColor, ColorBg, t)
					strokeColor = LerpColorRGBA(strokeColor, ColorBg, t)
				}
			}

			var scale float64 = 1

			// apply retry button scale offset
			retryScale, inRetry := getRetryButtonOffset(tileRect)

			var resetScale float64 = 1

			if g.ResetAnimTimerStarted {
				limit := g.ResetAnimTippingPoint
				t := g.ResetAnimTimer.Normalize()

				if t < limit {
					resetScale = 1 - Clamp(t/limit, 0, 1)
				} else {
					resetScale = Clamp((t-limit)/(1-limit), 0, 1)
				}
			}

			if inRetry {
				if g.ResetAnimTimer.Normalize() < g.ResetAnimTippingPoint {
					scale = retryScale
				} else {
					scale = resetScale
				}
			} else {
				scale = resetScale
			}

			tileRect = FRectScaleCentered(tileRect, scale)

			DrawFilledRect(dst, tileRect, bgColor, true)
			StrokeRect(dst, tileRect, 0, strokeColor, true)
		}

		// =================
		// draw highlight
		// =================
		iter.Reset()
		for iter.HasNext() {
			x, y := iter.GetNext()
			tileRect := GetBoardTileRect(g.BoardRect(), g.Board.Width, g.Board.Height, x, y)

			if g.TileHighLightTimer.Current > 0 {
				if Abs(x-g.TileHighLightX) <= 1 && Abs(y-g.TileHighLightY) <= 1 && !g.Board.Flags[x][y] && !g.Board.Revealed[x][y] {
					t := g.TileHighLightTimer.Normalize()
					c := ColorTable[ColorTileHighLight]
					DrawFilledRect(
						dst,
						tileRect,
						color.NRGBA{c.R, c.G, c.B, uint8(f64(c.A) * t)},
						true,
					)
				}
			}
		}

		// ==============================
		// draw defeat animation
		// ==============================
		if g.GameState == GameStateLost {
			iter.Reset()
			for iter.HasNext() {
				x, y := iter.GetNext()
				tileRect := GetBoardTileRect(g.BoardRect(), g.Board.Width, g.Board.Height, x, y)

				var scale float64 = 1

				retryScale, _ := getRetryButtonOffset(tileRect)
				scale *= retryScale

				if g.ResetAnimTimerStarted {
					scale *= 1 - Clamp(g.ResetAnimTimer.Normalize()/g.ResetAnimTippingPoint, 0, 1)
				}

				tileRect = FRectScaleCentered(tileRect, scale)

				// draw defeat animation
				if g.DefeatMineRevealTimers[x][y].Duration > 0 && g.DefeatMineRevealTimers[x][y].Current >= 0 {
					timer := g.DefeatMineRevealTimers[x][y]
					t := f64(timer.Current) / f64(timer.Duration)
					t = Clamp(t, 0, 1)

					const outerMargin = 1
					const innerMargin = 2

					outerRect := tileRect.Inset(outerMargin)
					outerRect = outerRect.Inset(min(outerRect.Dx(), outerRect.Dy()) * 0.5 * (1 - t))
					innerRect := outerRect.Inset(innerMargin)

					innerRect = innerRect.Add(FPt(0, innerMargin))

					radius := Lerp(1, 0.1, t*t*t)

					innerRadius := radius * min(innerRect.Dx(), innerRect.Dy()) * 0.5
					outerRadius := radius * min(outerRect.Dx(), outerRect.Dy()) * 0.5

					DrawFilledRoundRectFast(dst, outerRect, outerRadius, 5, ColorMineBg1, true)
					DrawFilledRoundRectFast(dst, innerRect, innerRadius, 5, ColorMineBg2, true)

					DrawSubViewInRect(dst, innerRect, 1, 0, 0, ColorMine, GetMineTile())
				}
			}
		}

		// ==============================
		// draw revealed tiles
		// ==============================
		{
			// TODO: make this cached
			revealedTiles := New2DArray[bool](g.Board.Width, g.Board.Height)
			radiuses := New2DArray[float64](g.Board.Width, g.Board.Height)
			scales := New2DArray[float64](g.Board.Width, g.Board.Height)

			// calculate radiuses and scales
			iter.Reset()
			for iter.HasNext() {
				x, y := iter.GetNext()
				timer := g.RevealAnimTimers[x][y]
				if timer.Duration > 0 {
					t := timer.Normalize()
					t = Clamp(t, 0, 1) // just in case
					t = t * t
					limit := g.RevealTippingPoint
					if t > limit {
						revealedTiles[x][y] = true

						// caculate radiuses
						t2 := ((t - limit) / (1 - limit))
						radiuses[x][y] = max(1-t2, 0.2)

						// calculate scales
						scales[x][y] = Lerp(1.2, 1.0, t2)
					}
				}
			}

			// =============================================
			// if won, use different radius and scales
			// =============================================
			if g.GameState == GameStateWon {
				iter.Reset()
				for iter.HasNext() {
					x, y := iter.GetNext()
					winT := EaseOutElastic(g.WinTilesAnimTimer[x][y].Normalize())
					radiuses[x][y] = max(Clamp(1-winT, 0, 1), 0.2)
					scales[x][y] = winT
				}
			}

			// =================================
			// apply retry button scale offset
			// =================================
			iter.Reset()
			for iter.HasNext() {
				x, y := iter.GetNext()
				tileRect := GetBoardTileRect(g.BoardRect(), g.Board.Width, g.Board.Height, x, y)
				scale, _ := getRetryButtonOffset(tileRect)
				if scale == 0 {
					revealedTiles[x][y] = false
				} else {
					scales[x][y] *= scale
				}
			}

			// =================================
			// apply ResetAnimTimer
			// =================================
			if g.ResetAnimTimerStarted {
				iter.Reset()
				for iter.HasNext() {
					x, y := iter.GetNext()
					scale := 1 - Clamp(g.ResetAnimTimer.Normalize()/g.ResetAnimTippingPoint, 0, 1)
					if scale == 0 {
						revealedTiles[x][y] = false
					} else {
						scales[x][y] *= scale
					}
				}
			}

			var c1 color.Color = ColorTileRevealed1
			var c2 color.Color = ColorTileRevealed2
			var stroke color.Color = ColorTileRevealedStroke

			// draw it on mask image
			boardRect := g.BoardRect()
			DrawRoundBoardTile(
				g.MaskImage,
				revealedTiles,
				boardRect,
				radiuses,
				scales,
				-2,
				stroke, stroke,
			)

			// move board slightly up
			boardRect = boardRect.Add(FPt(0, -1.5))

			DrawRoundBoardTile(
				g.MaskImage,
				revealedTiles,
				boardRect,
				radiuses,
				scales,
				0,
				c1, c2,
			)

			dst.DrawImage(g.MaskImage, nil)
		}

		if g.GameState == GameStateWon {
			rect := g.BoardRect()
			rect = rect.Inset(-3)

			colors := [4]color.Color{
				ColorWater1,
				ColorWater2,
				ColorWater3,
				ColorWater4,
			}

			t := EaseOutQuint(g.WinAnimTimer.Normalize())

			for i, c := range colors {
				nrgba := ColorToNRGBA(c)
				colors[i] = color.NRGBA{nrgba.R, nrgba.G, nrgba.B, uint8(f64(nrgba.A) * t)}
			}

			waterT := EaseOutQuint(t)

			DrawWaterRect(
				g.MaskImage,
				rect,
				GlobalTimerNow()+time.Duration(waterT*f64(time.Second)*10),
				colors,
				FPt(0, 0),
				eb.BlendSourceIn,
			)

			dst.DrawImage(g.MaskImage, nil)
		}

		// ==============================
		// draw foreground elements
		// ==============================
		iter.Reset()
		for iter.HasNext() {
			x, y := iter.GetNext()
			tileRect := GetBoardTileRect(g.BoardRect(), g.Board.Width, g.Board.Height, x, y)

			colorT := EaseOutQuint(g.WinTilesAnimTimer[x][y].Normalize())
			colorT = Clamp(colorT, 0, 1)

			// ==================
			// calculate scales
			// ==================

			// reveal scale
			var revealScale float64 = 1
			{
				t := g.RevealAnimTimers[x][y].Normalize()
				t = Clamp(t, 0, 1)
				t = t * t
				limit := g.RevealTippingPoint
				if t > limit {
					revealScale = (t - limit) / (1 - limit)
				} else {
					revealScale = 0
				}
			}

			// win scale
			var winScale float64 = 1
			if g.GameState == GameStateWon {
				scaleT := g.WinTilesAnimTimer[x][y].NormalizeUnclamped() * 0.5
				scaleT = Clamp(scaleT, 0, 1)
				winScale = EaseOutElastic(scaleT)

				// force scale to be 1 at the end of win animation
				winScale = max(winScale, EaseInQuint(g.WinAnimTimer.Normalize()))
			}

			// retry button scale
			retryScale, _ := getRetryButtonOffset(tileRect)

			var resetScale float64 = 1
			if g.ResetAnimTimerStarted {
				resetScale = 1 - Clamp(g.ResetAnimTimer.Normalize()/g.ResetAnimTippingPoint, 0, 1)
			}

			// draw flags
			if g.Board.Flags[x][y] {
				var c color.Color = ColorFlag
				if g.GameState == GameStateWon {
					c = LerpColorRGBA(c, ColorElementWon, colorT)
				}
				g.DrawTile(dst, x, y, winScale*retryScale*resetScale, 0, 0, c, GetFlagTile())
			}

			// draw number
			if g.Board.Revealed[x][y] {
				if count := g.Board.GetNeighborMineCount(x, y); count > 0 {
					// give scale offset with tile highlight
					var scaleOffset float64
					if x == g.NumberClickX && y == g.NumberClickY {
						scaleOffset = g.NumberClickTimer.Normalize() * -0.06
					}
					nScale := Clamp(revealScale*winScale*retryScale*resetScale, 0, 1) + scaleOffset

					var c color.Color = ColorTableGetNumber(count)
					if g.GameState == GameStateWon {
						c = LerpColorRGBA(c, ColorElementWon, colorT)
					}
					g.DrawTile(dst, x, y, nScale, 0, 0, c, GetNumberTile(count))
				}
			}

			// draw debug mines
			if g.RevealMines && g.Board.Mines[x][y] {
				g.DrawTile(dst, x, y, 1, 0, 0, color.NRGBA{255, 0, 0, 255}, GetMineTile())
			}
		}
	*/
	g.DrawBoard(dst)

	if g.ShowRetryButton {
		g.RetryButton.Draw(dst)
	}

	g.DifficultySelectUI.Draw(dst, g.BoardRect())

	g.ColorTablePicker.Draw(dst)
}

func (g *Game) DrawBoard(dst *eb.Image) {
	// TODO: support alpha
	iter := NewBoardIterator(0, 0, g.Board.Width-1, g.Board.Height-1)

	for iter.HasNext() {
		x, y := iter.GetNext()

		style := g.RenderTileStyles[x][y]

		if !style.DrawTile && !style.DrawFg && !style.DrawBg {
			continue
		}

		ogTileRect := GetBoardTileRect(g.BoardRect(), g.Board.Width, g.Board.Height, x, y)

		// draw background tile
		if style.DrawBg && style.BgScale > 0.001 {
			bgTileRect := ogTileRect
			bgTileRect = bgTileRect.Add(FPt(style.BgOffsetX, style.BgOffsetY))
			bgTileRect = FRectScaleCentered(bgTileRect, style.BgScale)

			DrawFilledRect(dst, bgTileRect, style.BgFillColor)

			// draw highlight
			if style.BgTileHightlight > 0 {
				t := style.BgTileHightlight
				c := TheColorTable[ColorTileHighLight]
				DrawFilledRect(
					dst,
					bgTileRect,
					color.NRGBA{c.R, c.G, c.B, uint8(f64(c.A) * t)},
				)
			}

			// draw bomb animation
			if style.BgBombAnim > 0 {
				t := Clamp(style.BgBombAnim, 0, 1)

				const outerMargin = 1
				const innerMargin = 2

				outerRect := bgTileRect.Inset(outerMargin)
				outerRect = outerRect.Inset(min(outerRect.Dx(), outerRect.Dy()) * 0.5 * (1 - t))
				innerRect := outerRect.Inset(innerMargin)

				innerRect = innerRect.Add(FPt(0, innerMargin))

				radius := Lerp(1, 0.1, t*t*t)

				innerRadius := radius * min(innerRect.Dx(), innerRect.Dy()) * 0.5
				outerRadius := radius * min(outerRect.Dx(), outerRect.Dy()) * 0.5

				DrawFilledRoundRectFast(dst, outerRect, outerRadius, 5, ColorMineBg1)
				DrawFilledRoundRectFast(dst, innerRect, innerRadius, 5, ColorMineBg2)

				DrawSubViewInRect(dst, innerRect, 1, 0, 0, ColorMine, GetMineTile())
			}
		}

		// draw foreground tile
		if style.TileScale > 0.001 {
			tileRect := ogTileRect

			tileRect = tileRect.Add(FPt(style.TileOffsetX, style.TileOffsetY))
			tileRect = FRectScaleCentered(tileRect, style.TileScale)

			//strokeRect := tileRect.Inset(-2)
			strokeRect := tileRect
			fillRect := tileRect.Add(FPt(0, -2))

			radiusPx := float64(0)

			if !CloseTo(style.TileScale, 1) {
				if style.TileScale < 1 {
					radius := 1 - style.TileScale
					radius = Clamp(radius, 0, 1)
					radius = EaseOutQuint(radius)
					radiusPx = min(fillRect.Dx(), fillRect.Dy()) * 0.5 * radius
				} else { // style.TileStyle > 1
					dx := fillRect.Dx() - ogTileRect.Dx()
					dy := fillRect.Dy() - ogTileRect.Dy()

					t := min(dx, dy)
					t = max(t, 0)

					radiusPx = t * 0.7
				}
			}

			const segments = 5

			// TODO: each corner should have it's own radiuses
			if style.DrawTile {
				fillColor := style.TileFillColor
				strokeColor := style.TileStrokeColor

				DrawFilledRoundRectFast(dst, strokeRect, radiusPx, segments, strokeColor)
				DrawFilledRoundRectFast(dst, fillRect, radiusPx, segments, fillColor)
			}

			if style.DrawFg && style.FgType != TileFgTypeNone {
				fgRect := fillRect

				fgColor := style.FgColor

				if style.FgType == TileFgTypeNumber {
					count := g.Board.GetNeighborMineCount(x, y)
					if 1 <= count && count <= 8 {
						DrawSubViewInRect(
							dst,
							fgRect,
							style.FgScale,
							style.FgOffsetX, style.FgOffsetY,
							fgColor,
							GetNumberTile(count),
						)
					}
				} else if style.FgType == TileFgTypeFlag {
					DrawSubViewInRect(
						dst,
						fgRect,
						style.FgScale,
						style.FgOffsetX, style.FgOffsetY,
						fgColor,
						GetFlagTile(),
					)
				}
			}
		}
	}
}

func (g *Game) BoardRect() FRectangle {
	size := min(ScreenWidth, ScreenHeight) * g.BoardSizeRatio
	halfSize := size * 0.5
	halfWidth := ScreenWidth * 0.5
	halfHeight := ScreenHeight * 0.5
	return FRect(
		halfWidth-halfSize, halfHeight-halfSize,
		halfWidth+halfSize, halfHeight+halfSize,
	)
}

func (g *Game) MousePosToBoardPos(mousePos FPoint) (int, int) {
	boardRect := g.BoardRect()

	// if mouse is outside the board return -1
	if !mousePos.In(boardRect) {
		return -1, -1
	}

	mousePos.X -= boardRect.Min.X
	mousePos.Y -= boardRect.Min.Y

	boardX := int(math.Floor(mousePos.X / (boardRect.Dx() / float64(g.Board.Width))))
	boardY := int(math.Floor(mousePos.Y / (boardRect.Dy() / float64(g.Board.Height))))

	boardX = min(boardX, g.Board.Width-1)
	boardY = min(boardY, g.Board.Height-1)

	return boardX, boardY
}

func (g *Game) RetryButtonRect() FRectangle {
	boardRect := g.BoardRect()
	whMin := min(boardRect.Dx(), boardRect.Dy())
	rect := FRectWH(whMin*0.25, whMin*0.25)
	center := FRectangleCenter(boardRect)
	return CenterFRectangle(rect, center.X, center.Y)
}

func (g *Game) StartRevealAnimation(revealsBefore, revealsAfter [][]bool, originX, originy int) {
	g.SkipAllTileAnimations()

	fw, fh := f64(g.Board.Width), f64(g.Board.Height)

	originF := FPt(f64(originX), f64(originy))

	maxDist := math.Sqrt(fw*fw + fh*fh)

	const maxDuration = time.Millisecond * 900
	const minDuration = time.Millisecond * 20

	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if !revealsBefore[x][y] && revealsAfter[x][y] {
				pos := FPt(f64(x), f64(y))
				dist := pos.Sub(originF).Length()
				d := time.Duration(f64(maxDuration) * (dist / maxDist))

				var timer Timer

				timer.Duration = max(d, minDuration)
				timer.Current = 0

				var anim CallbackTileAnimation

				var tileFillColor color.Color = ColorTileRevealed1
				if IsOddTile(g.Board.Width, g.Board.Height, x, y) {
					tileFillColor = ColorTileRevealed2
				}
				var fgColor color.Color = color.NRGBA{0, 0, 0, 0}
				count := g.Board.GetNeighborMineCount(x, y)
				if 1 <= count && count <= 8 {
					fgColor = ColorTableGetNumber(count)
				}

				anim.OnUpdate = func(x, y int, style TileStyle) TileStyle {
					timer.TickUp()

					style.TileFillColor = tileFillColor
					style.TileStrokeColor = ColorTileRevealedStroke
					style.FgColor = fgColor

					t := timer.Normalize()
					t = t * t

					const limit = 0.4

					if t > limit {
						style.DrawTile = true
						style.DrawFg = true
						style.FgScale = 1.0
						style.FgType = TileFgTypeNumber

						t = ((t - limit) / (1 - limit))
						t = Clamp(t, 0, 1)

						style.TileScale = Lerp(1.2, 1.0, t)
					} else {
						style.DrawTile = false
						style.DrawFg = false
					}

					return style
				}

				anim.OnSkip = func(x, y int, style TileStyle) TileStyle {
					timer.Current = timer.Duration
					return anim.OnUpdate(x, y, style)
				}

				anim.OnDone = func(x, y int, style TileStyle) bool {
					return timer.Current >= timer.Duration
				}

				g.TileAnimations[x][y].Enqueue(anim)
			}
		}
	}
}

func (g *Game) StartDefeatAnimation(originX, originY int) {
	g.SkipAllTileAnimations()

	var minePoses []image.Point

	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if g.Board.Mines[x][y] && !g.Board.Flags[x][y] {
				minePoses = append(minePoses, image.Point{X: x, Y: y})
			}
		}
	}

	if len(minePoses) <= 0 {
		return
	}

	distFromOriginSquared := func(pos image.Point) int {
		return (pos.X-originX)*(pos.X-originX) + (pos.Y-originY)*(pos.Y-originY)
	}

	slices.SortFunc(minePoses, func(a, b image.Point) int {
		distA := distFromOriginSquared(a)
		distB := distFromOriginSquared(b)

		return distA - distB
	})

	var defeatDuration time.Duration
	var offset time.Duration

	firstPos := minePoses[0]
	recordDist := distFromOriginSquared(firstPos)

	for _, p := range minePoses {
		var timer Timer

		timer.Duration = time.Millisecond * 150

		dist := distFromOriginSquared(p)
		if dist > recordDist {
			recordDist = dist
			offset -= time.Millisecond * 100
		}
		timer.Current = offset

		// calculate defeatDuration
		defeatDuration = max(defeatDuration, timer.Duration-timer.Current)

		// add new animation
		var anim CallbackTileAnimation

		anim.OnUpdate = func(x, y int, style TileStyle) TileStyle {
			timer.TickUp()
			style.BgBombAnim = timer.Normalize()
			return style
		}

		anim.OnSkip = func(x, y int, style TileStyle) TileStyle {
			timer.Current = timer.Duration
			return anim.OnUpdate(x, y, style)
		}

		anim.OnDone = func(x, y int, style TileStyle) bool {
			return timer.Current >= timer.Duration
		}

		g.TileAnimations[p.X][p.Y].Enqueue(anim)
	}

	defeatDuration += time.Millisecond * 10

	g.DefeatAnimTimer.Duration = defeatDuration
	g.DefeatAnimTimer.Current = 0
}

func (g *Game) StartWinAnimation(originX, originY int) {
	fw, fh := f64(g.Board.Width), f64(g.Board.Height)

	originF := FPt(f64(originX), f64(originY))

	maxDist := math.Sqrt(fw*fw + fh*fh)

	const maxDuration = time.Millisecond * 3000
	const minDuration = time.Millisecond * 100
	const distStartOffset = time.Millisecond * 3

	var winDuration time.Duration

	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if g.Board.Revealed[x][y] {
				pos := FPt(f64(x), f64(y))
				dist := pos.Sub(originF).Length()
				d := time.Duration(f64(maxDuration) * (dist / maxDist))

				var timer Timer

				timer.Duration = max(d, minDuration)
				timer.Current = -time.Duration(dist * f64(distStartOffset))

				winDuration = max(winDuration, timer.Duration-timer.Current)

				g.WinTilesAnimTimer[x][y] = timer
			}
		}
	}

	g.WinAnimTimer.Duration = winDuration + time.Millisecond*400
	g.WinAnimTimer.Current = 0
}

func (g *Game) SkipTileAnimationAt(x, y int) {
	if !g.Board.IsPosInBoard(x, y) {
		return
	}

	for !g.TileAnimations[x][y].IsEmpty() {
		anim := g.TileAnimations[x][y].Dequeue()
		g.BaseTileStyles[x][y] = anim.Skip(x, y, g.BaseTileStyles[x][y])
	}
}

func (g *Game) SkipAllTileAnimations() {
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			g.SkipTileAnimationAt(x, y)
		}
	}
}

func (g *Game) SetDebugBoardForDecoration() {
	g.ResetBoard(15, 15)

	g.BoardTouched = true

	// . : unrevealed
	// @ : revealed
	// * : mine
	// + : flagged
	newBoard := [][]rune{
		[]rune("....@*@*@*"),
		[]rune("......@.@*"),
		[]rune("***+****.."),
		[]rune("*@+@*@*..."),
		[]rune("++**.*...."),
	}

	newBoardHeight := len(newBoard)
	newBoardWidth := len(newBoard[0])

	iter := NewBoardIterator(0, 0, newBoardWidth-1, newBoardHeight-1)
	for iter.HasNext() {
		x, y := iter.GetNext()
		if g.Board.IsPosInBoard(x, y) {
			char := newBoard[y][x] //yeah y and x is reversed

			switch char {
			case '@':
				g.Board.Revealed[x][y] = true
			case '*':
				g.Board.Mines[x][y] = true
			case '+':
				g.Board.Mines[x][y] = true
				g.Board.Flags[x][y] = true
			}
		}
	}

	iter = NewBoardIterator(0, 0, g.Board.Width-1, g.Board.Height-1)
	for iter.HasNext() {
		x, y := iter.GetNext()
		if x < newBoardWidth+1 && y < newBoardHeight+1 {
			continue
		}

		if rand.Int64N(100) < 30 {
			g.Board.Mines[x][y] = true
		}
	}

	iter.Reset()

	for iter.HasNext() {
		x, y := iter.GetNext()

		if !g.Board.Mines[x][y] {
			if rand.Int64N(100) < 30 {
				// flag the surrounding
				innerIter := NewBoardIterator(x-1, y-1, x+1, y+1)
				for innerIter.HasNext() {
					inX, inY := innerIter.GetNext()
					if g.Board.IsPosInBoard(inX, inY) && g.Board.Mines[inX][inY] {
						g.Board.Flags[inX][inY] = true
					}
				}

				g.Board.SpreadSafeArea(x, y)
			}
		}
	}
}

func (g *Game) SetBoardForInstantWin() {
	if !g.BoardTouched {
		g.Board.PlaceMines(g.MineCount[g.Difficulty], g.Board.Width-1, g.Board.Height-1)
	}
	g.BoardTouched = true

	// count how many tiles we have to reveal
	tilesToReveal := 0
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if !g.Board.Mines[x][y] && !g.Board.Revealed[x][y] {
				tilesToReveal++
			}
		}
	}

	// reveal that many tiles EXCEPT ONE
REVEAL_LOOP:
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if tilesToReveal <= 1 {
				break REVEAL_LOOP
			}
			if !g.Board.Mines[x][y] && !g.Board.Revealed[x][y] {
				g.Board.Revealed[x][y] = true
				tilesToReveal--
			}
		}
	}
}

func GetNumberTile(number int) SubView {
	if !(1 <= number && number <= 8) {
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

func GetBoardTileRect(
	boardRect FRectangle,
	boardWidth, boardHeight int,
	boardX, boardY int,
) FRectangle {
	tileWidth := boardRect.Dx() / f64(boardWidth)
	tileHeight := boardRect.Dy() / f64(boardHeight)

	return FRectangle{
		Min: FPt(f64(boardX)*tileWidth, f64(boardY)*tileHeight).Add(boardRect.Min),
		Max: FPt(f64(boardX+1)*tileWidth, f64(boardY+1)*tileHeight).Add(boardRect.Min),
	}
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

	drawScale := min(rectSize.X, rectSize.Y) / max(imgSize.X, imgSize.Y) * scale

	op := &DrawSubViewOptions{}
	op.GeoM.Concat(TransformToCenter(imgSize.X, imgSize.Y, drawScale, drawScale, 0))
	rectCenter := FRectangleCenter(rect)
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)
	op.GeoM.Translate(offsetX, offsetY)
	op.ColorScale.ScaleWithColor(clr)

	DrawSubView(dst, view, op)
}

func (g *Game) DrawTile(
	dst *eb.Image,
	boardX, boardY int,
	scale float64,
	offsetX, offsetY float64,
	clr color.Color,
	tile SubView,
) {
	tileRect := GetBoardTileRect(g.BoardRect(), g.Board.Width, g.Board.Height, boardX, boardY)
	DrawSubViewInRect(dst, tileRect, scale, offsetY, offsetY, clr, tile)
}

func IsOddTile(boardWidth, boardHeight, x, y int) bool {
	index := x + boardHeight*y
	if boardWidth%2 == 0 {
		if y%2 == 0 {
			return index%2 != 0
		} else {
			return index%2 == 0
		}
	} else {
		return index%2 != 0
	}
}

func DrawRoundBoardTileOld(
	dst *eb.Image,
	board [][]bool,
	radiuses [][]float64,
	boardRect FRectangle,
	fillColor1, fillColor2 color.Color,
	strokeColor color.Color, strokeWidth float64,
	doFill bool,
) {
	const segmentCount = 6
	boardWidth := len(board)
	boardHeight := len(board[0])

	isRevealed := func(x, y int) bool {
		if !(0 <= x && x < boardWidth) {
			return false
		}
		if !(0 <= y && y < boardHeight) {
			return false
		}
		return board[x][y]
	}

	iter := NewBoardIterator(0, 0, boardWidth, boardHeight)

	for iter.HasNext() {
		x, y := iter.GetNext()

		boardTileRect := GetBoardTileRect(boardRect, boardWidth, boardHeight, x, y)
		rect := boardTileRect.Add(FPt(-boardTileRect.Dx()*0.5, -boardTileRect.Dy()*0.5))

		//	0 --- 1
		//	|     |
		//	|     |
		//	3 --- 2
		revealed := [4]bool{}

		revealed[0] = isRevealed(x-1, y-1)
		revealed[1] = isRevealed(x, y-1)
		revealed[2] = isRevealed(x, y)
		revealed[3] = isRevealed(x-1, y)

		revealCount := 0
		for _, v := range revealed {
			if v {
				revealCount++
			}
		}

		rectCornerRect := func(r FRectangle, corner int) FRectangle {
			hx := r.Dx() * 0.5
			hy := r.Dy() * 0.5

			switch corner {
			case 0:
				return FRectXYWH(r.Min.X, r.Min.Y, hx, hy)
			case 1:
				return FRectXYWH(r.Min.X+hx, r.Min.Y, hx, hy)
			case 2:
				return FRectXYWH(r.Min.X+hx, r.Min.Y+hx, hx, hy)
			case 3:
				return FRectXYWH(r.Min.X, r.Min.Y+hx, hx, hy)
			}
			return FRectangle{}
		}

		cornerDirs := [4]FPoint{
			{-1, -1},
			{1, -1},
			{1, 1},
			{-1, 1},
		}

		isOddCorner := func(corner int) bool {
			if IsOddTile(boardWidth, boardHeight, x, y) {
				return corner%2 == 0
			} else {
				return corner%2 != 0
			}
		}

		cornerColor := func(corner int) color.Color {
			if !isOddCorner(corner) {
				return fillColor1
			} else {
				return fillColor2
			}
		}

		// in pixels
		cornerRadius := func(corner int) float64 {
			rx, ry := x, y
			switch corner {
			case 0:
				rx -= 1
				ry -= 1
			case 1:
				ry -= 1
			case 2:
				// pass
			case 3:
				rx -= 1
			}

			rx = Clamp(rx, 0, boardWidth-1)
			ry = Clamp(ry, 0, boardHeight-1)

			radius := radiuses[rx][ry]
			radius = Clamp(radius, 0, 1)

			return min(rect.Dx()*0.5, rect.Dy()*0.5) * radius
		}

		cornerOpposite := func(corner int) int {
			return (corner + 2) % 4
		}

		rectCornerVert := func(r FRectangle, corner int) FPoint {
			switch corner {
			case 0:
				return r.Min
			case 1:
				return FPoint{r.Max.X, r.Min.Y}
			case 2:
				return r.Max
			case 3:
				return FPoint{r.Min.X, r.Max.Y}
			}
			return FPoint{}
		}

		getConcaveOrSharpCornersRectPath := func(
			isConcave [4]bool,
			concaveRadius [4]float64,
		) [4]*ebv.Path {
			var paths [4]*ebv.Path

			for i := range 4 {

				if !isConcave[i] {
					cornerRect := rectCornerRect(rect, i)
					paths[i] = GetRectPath(cornerRect)
				} else {
					p := &ebv.Path{}

					cornerRect := rectCornerRect(rect, i)

					edgeCorner := cornerOpposite(i)
					edgeCornerVert := rectCornerVert(cornerRect, edgeCorner)

					arcCenter := FPt(
						edgeCornerVert.X+cornerDirs[i].X*concaveRadius[i],
						edgeCornerVert.Y+cornerDirs[i].Y*concaveRadius[i],
					)

					p.MoveTo(f32(edgeCornerVert.X), f32(edgeCornerVert.Y))

					startAngle := Pi*0.5 + Pi*0.5*f64(i)
					endAngle := startAngle - Pi*0.5
					ArcFast(
						p,
						(arcCenter.X), (arcCenter.Y),
						(concaveRadius[i]),
						startAngle, endAngle,
						ebv.CounterClockwise,
						segmentCount,
					)

					p.Close()

					paths[i] = p
				}
			}

			return paths
		}

		drawCorner := func(corner int, roundCorner int) {
			p := &ebv.Path{}

			roundOpposite := cornerOpposite(roundCorner)

			cornerRect := rectCornerRect(rect, corner)
			roundVert := rectCornerVert(cornerRect, roundCorner)

			radius := cornerRadius(corner)

			arcCenter := FPoint{
				X: roundVert.X + cornerDirs[roundOpposite].X*radius,
				Y: roundVert.Y + cornerDirs[roundOpposite].Y*radius,
			}

			startAngle := Pi + Pi*0.5*f64(roundCorner)
			endAngle := startAngle + Pi*0.5

			ArcFast(
				p,
				(arcCenter.X), (arcCenter.Y),
				(radius), (startAngle), (endAngle), ebv.Clockwise,
				segmentCount,
			)

			c := roundCorner
			c = (c + 1) % 4
			for range 3 {
				v := rectCornerVert(cornerRect, c)
				p.LineTo(f32(v.X), f32(v.Y))
				c = (c + 1) % 4
			}

			p.Close()

			if doFill {
				DrawFilledPath(dst, p, cornerColor(corner))
			} else {
				op := &ebv.StrokeOptions{}
				op.Width = f32(strokeWidth)
				op.MiterLimit = 4
				StrokePath(dst, p, op, strokeColor)
			}
		}

		switch revealCount {
		case 0:
			// pass
		case 1:
			for i, v := range revealed {
				if v {
					drawCorner(i, cornerOpposite(i))
				}
			}
		case 2:
			for i, v := range revealed {
				if v {
					drawCorner(i, cornerOpposite(i))
				}
			}
		case 3:
			var unRevealed int
			for i, v := range revealed {
				if !v {
					unRevealed = i
					break
				}
			}
			var isConcave [4]bool
			var concaveRadius [4]float64

			radius := cornerRadius(unRevealed)

			isConcave[unRevealed] = true
			concaveRadius[unRevealed] = radius

			paths := getConcaveOrSharpCornersRectPath(
				isConcave, concaveRadius,
			)

			if doFill {
				for _, p := range paths {
					DrawFilledPath(dst, p, cornerColor((unRevealed+1)%4))
				}
				drawCorner(cornerOpposite(unRevealed), unRevealed)
			} else {
				for _, p := range paths {
					op := &ebv.StrokeOptions{}
					op.Width = f32(strokeWidth)
					op.MiterLimit = 4
					StrokePath(dst, p, op, strokeColor)
				}
			}
		case 4:
			if doFill {
				for i, v := range revealed {
					if v {
						DrawFilledRect(dst, rectCornerRect(rect, i), cornerColor(i))
					}
				}
			}
		}
	}
}

func DrawRoundBoardTile(
	dst *eb.Image,
	board [][]bool,
	boardRect FRectangle,
	radiuses [][]float64,
	scales [][]float64,
	inset float64,
	fillColor1, fillColor2 color.Color,
) {
	const segments = 5
	boardWidth := len(board)
	boardHeight := len(board[0])

	for x := range boardWidth {
		for y := range boardWidth {
			if !board[x][y] {
				continue
			}
			tileRect := GetBoardTileRect(boardRect, boardWidth, boardHeight, x, y)
			center := FRectangleCenter(tileRect)
			tileRect = CenterFRectangle(
				FRectWH(tileRect.Dx()*scales[x][y], tileRect.Dy()*scales[x][y]),
				center.X, center.Y,
			)
			tileRect = tileRect.Inset(inset)
			radiusPx := min(tileRect.Dx(), tileRect.Dy()) * 0.5 * radiuses[x][y]

			tileRadiuses := [4]float64{
				radiusPx,
				radiusPx,
				radiusPx,
				radiusPx,
			}

			if inset <= 0 && scales[x][y] == 1 {
				for i := range 4 {
					rx := x
					ry := y
					//   0
					//   |
					// 3- -1
					//   |
					//   2

					switch i {
					case 0:
						ry -= 1
					case 1:
						rx += 1
					case 2:
						ry += 1
					case 3:
						rx -= 1
					}

					if 0 <= rx && rx < boardWidth && 0 <= ry && ry < boardHeight {
						if board[rx][ry] {
							tileRadiuses[i] = 0
							tileRadiuses[(i+1)%4] = 0
						}
					}
				}
			}

			clr := fillColor1
			if IsOddTile(boardWidth, boardHeight, x, y) {
				clr = fillColor2
			}
			DrawFilledRoundRectFastEx(
				dst,
				tileRect,
				tileRadiuses,
				[4]int{segments, segments, segments, segments},
				clr,
			)
		}
	}
}

func DrawWaterRect(
	dst *eb.Image,
	rect FRectangle,
	timeOffset time.Duration,
	colors [4]color.Color,
	offset FPoint,
) {
	op := &DrawRectShaderOptions{}

	op.Images[0] = WaterShaderImage1
	op.Images[1] = WaterShaderImage2

	c1 := ColorNormalized(colors[0], true)
	c2 := ColorNormalized(colors[1], true)
	c3 := ColorNormalized(colors[2], true)
	c4 := ColorNormalized(colors[3], true)

	op.Uniforms = make(map[string]any)
	op.Uniforms["Time"] = f64(timeOffset) / f64(time.Second)
	op.Uniforms["Offset"] = [2]float64{offset.X, offset.Y}
	op.Uniforms["Colors"] = [16]float64{
		c1[0], c1[1], c1[2], c1[3],
		c2[0], c2[1], c2[2], c2[3],
		c3[0], c3[1], c3[2], c3[3],
		c4[0], c4[1], c4[2], c4[3],
	}

	imgRect := WaterShaderImage1.Bounds()
	imgFRect := RectToFRect(imgRect)

	op.GeoM.Scale(rect.Dx()/imgFRect.Dx(), rect.Dy()/imgFRect.Dy())
	op.GeoM.Translate(rect.Min.X, rect.Min.Y)

	DrawRectShader(dst, imgRect.Dx(), imgRect.Dy(), WaterShader, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if g.MaskImage.Bounds().Dx() != outsideWidth || g.MaskImage.Bounds().Dy() != outsideHeight {
		g.MaskImage = eb.NewImage(outsideWidth, outsideHeight)
	}
	return outsideWidth, outsideHeight
}
