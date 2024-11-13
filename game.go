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
)

var _ = fmt.Printf

type GameState int

const (
	GameStatePlaying GameState = iota
	GameStateWon
	GameStateLost
)

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

	DrawFilledRoundRectFast(
		dst,
		bottomRect,
		radius,
		false,
		segments,
		color.NRGBA{0, 0, 0, 255},
	)

	DrawFilledRoundRectFast(
		dst,
		topRect,
		radius,
		false,
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

	BgAlpha float64

	DrawTile bool // does not affect fg

	TileScale   float64
	TileOffsetX float64
	TileOffsetY float64

	TileFillColor   color.Color
	TileStrokeColor color.Color

	TileAlpha float64

	DrawFg bool

	FgScale   float64 // relative to tile's scale
	FgOffsetX float64 // relative to tile's center
	FgOffsetY float64 // relative to tile's center

	FgColor color.Color

	FgType TileFgType

	FgAlpha float64
}

func NewTileStyle() TileStyle {
	return TileStyle{
		BgScale:     1,
		BgFillColor: color.NRGBA{0, 0, 0, 0},
		BgAlpha:     1,

		TileScale:       1,
		TileFillColor:   color.NRGBA{0, 0, 0, 0},
		TileStrokeColor: color.NRGBA{0, 0, 0, 0},
		TileAlpha:       1,

		FgScale: 1,
		FgColor: color.NRGBA{0, 0, 0, 0},
		FgAlpha: 1,
	}
}

type AnimationTag int

const (
	AnimationTagNone AnimationTag = iota
	AnimationTagTileReveal
	AnimationTagWin
	AnimationTagDefeat
	AnimationTagRetryButtonReveal
	AnimationTagHideBoard
	AnimationTagShowBoard
)

type CallbackAnimation struct {
	Update func()
	Skip   func()
	Done   func() bool

	// optional
	AfterDone func()

	Tag AnimationTag
}

type Game struct {
	Rect FRectangle

	Board     Board
	PrevBoard Board

	MineCount int

	PlacedMinesOnBoard bool

	BaseTileStyles   [][]TileStyle
	RenderTileStyles [][]TileStyle

	TileAnimations [][]CircularQueue[CallbackAnimation]

	GameAnimations CircularQueue[CallbackAnimation]

	TileHighLightTimer Timer
	TileHighLightX     int
	TileHighLightY     int

	NumberClickTimer Timer
	NumberClickX     int
	NumberClickY     int

	RetryButton *RetryButton

	RetryButtonSize float64

	DrawRetryButton    bool
	RetryButtonScale   float64
	RetryButtonOffsetX float64
	RetryButtonOffsetY float64

	GameState GameState

	RevealMines bool

	WaterAlpha      float64
	WaterFlowOffset time.Duration

	WaterRenderTarget *eb.Image
}

func NewGame(boardWidth, boardHeight, mineCount int) *Game {
	g := new(Game)

	g.MineCount = mineCount

	g.WaterRenderTarget = eb.NewImage(int(ScreenWidth), int(ScreenHeight))

	g.TileHighLightTimer.Duration = time.Millisecond * 100
	g.NumberClickTimer.Duration = time.Millisecond * 30

	g.RetryButton = NewRetryButton()
	g.RetryButton.Disabled = true
	g.RetryButton.ActOnRelease = true
	g.RetryButtonScale = 1

	g.RetryButton.OnClick = func() {
		g.QueueResetBoardAnimation()
	}

	g.GameAnimations = NewCircularQueue[CallbackAnimation](10)

	g.ResetBoard(boardWidth, boardHeight)

	return g
}

func (g *Game) ResetBoardWithNoStyles(width, height int) {
	g.PlacedMinesOnBoard = false

	g.GameState = GameStatePlaying
	g.GameAnimations.Clear()

	g.Board = NewBoard(width, height)
	g.PrevBoard = NewBoard(width, height)

	g.DrawRetryButton = false
	g.RetryButtonScale = 1
	g.RetryButtonOffsetX = 0
	g.RetryButtonOffsetY = 0

	g.BaseTileStyles = New2DArray[TileStyle](width, height)
	g.RenderTileStyles = New2DArray[TileStyle](width, height)

	g.TileAnimations = New2DArray[CircularQueue[CallbackAnimation]](width, height)
	for x := range width {
		for y := range height {
			// TODO : do we need this much queued animation?
			g.TileAnimations[x][y] = NewCircularQueue[CallbackAnimation](5)
		}
	}

	for x := range width {
		for y := range height {
			g.BaseTileStyles[x][y] = NewTileStyle()
			g.RenderTileStyles[x][y] = NewTileStyle()
		}
	}
}

func (g *Game) ResetBoard(width, height int) {
	g.ResetBoardWithNoStyles(width, height)

	for x := range width {
		for y := range height {
			targetStyle := GetAnimationTargetTileStyle(g.Board, x, y)
			g.BaseTileStyles[x][y] = targetStyle
			g.RenderTileStyles[x][y] = targetStyle
		}
	}
}

func (g *Game) Update() {
	justPressedL := IsMouseButtonJustPressed(eb.MouseButtonLeft)
	justPressedR := IsMouseButtonJustPressed(eb.MouseButtonRight)
	pressedL := IsMouseButtonPressed(eb.MouseButtonLeft)
	pressedR := IsMouseButtonPressed(eb.MouseButtonRight)

	justPressedM := IsMouseButtonJustPressed(eb.MouseButtonMiddle)
	pressedM := IsMouseButtonPressed(eb.MouseButtonMiddle)

	justPressedAny := justPressedL || justPressedR || justPressedM

	cursor := CursorFPt()

	boardX, boardY := g.MousePosToBoardPos(cursor)

	// =================================
	// handle board interaction
	// =================================

	// =======================================
	prevState := g.GameState
	g.Board.SaveTo(g.PrevBoard)
	// =======================================

	// =======================================
	stateChanged := false
	// =======================================

	if g.GameState == GameStatePlaying && boardX >= 0 && boardY >= 0 && justPressedAny {

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
					if !g.PlacedMinesOnBoard { // first time interaction
						g.PlacedMinesOnBoard = true

						// remove flags that might have been placed
						for x := range g.Board.Width {
							for y := range g.Board.Height {
								g.Board.Flags[x][y] = false
							}
						}

						g.Board.PlaceMines(g.MineCount, boardX, boardY)
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
	}

	// ======================================
	// changing board for debugging purpose
	// ======================================
	if IsKeyJustPressed(SetToDecoBoardKey) {
		g.SetDebugBoardForDecoration()
		g.QueueRevealAnimation(
			g.PrevBoard.Revealed, g.Board.Revealed, 0, 0)
		stateChanged = true
	}
	if IsKeyJustPressed(InstantWinKey) {
		g.SetBoardForInstantWin()
		g.QueueRevealAnimation(
			g.PrevBoard.Revealed, g.Board.Revealed, 0, 0)
		stateChanged = true
	}

	// reveal the board if mouse button 4 is pressed
	if g.GameState == GameStatePlaying &&
		boardX >= 0 && boardY >= 0 &&
		IsMouseButtonJustPressed(eb.MouseButton4) {

		g.PlacedMinesOnBoard = true

		g.Board.Revealed[boardX][boardY] = true
		stateChanged = true
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
					g.QueueRevealAnimation(
						g.PrevBoard.Revealed, g.Board.Revealed, boardX, boardY)

					break REVEAL_CHECK
				}
			}
		}

		if prevState != g.GameState {
			if g.GameState == GameStateLost {
				g.QueueDefeatAnimation(boardX, boardY)
			} else if g.GameState == GameStateWon {
				g.QueueWinAnimation(boardX, boardY)
			}
		}
	}

	// ============================
	// update animations
	// ============================

	// update GameAnimations
	if !g.GameAnimations.IsEmpty() {
		anim := g.GameAnimations.At(0)
		anim.Update()

		if anim.Done() {
			g.GameAnimations.Dequeue()

			if anim.AfterDone != nil {
				anim.AfterDone()
			}
		}
	}

	// update BaseTileStyles
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if !g.TileAnimations[x][y].IsEmpty() {
				anim := g.TileAnimations[x][y].At(0)
				anim.Update()

				if anim.Done() {
					g.TileAnimations[x][y].Dequeue()
					if anim.AfterDone != nil {
						anim.AfterDone()
					}
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

	// skipping animations
	if prevState == GameStateLost || prevState == GameStateWon {
		// all animations are skippable except AnimationTagRetryButtonReveal
		if justPressedAny {
			g.SkipAllAnimationsUntilTag(AnimationTagRetryButtonReveal)
		}
	}

	// ===================================
	// update RetryButton
	// ===================================
	g.RetryButton.Rect = g.TransformedRetryButtonRect()
	g.RetryButton.Update()
	if !g.DrawRetryButton {
		g.RetryButton.Disabled = true
	}

	// ==========================
	// debug mode
	// ==========================
	if IsKeyJustPressed(ShowMinesKey) {
		g.RevealMines = !g.RevealMines
	}
}

func (g *Game) Draw(dst *eb.Image) {
	// background
	dst.Fill(TheColorTable[ColorBg])

	DrawBoard(
		dst,

		g.Board, g.Rect,
		g.RenderTileStyles,

		g.GameState == GameStateWon, g.WaterRenderTarget, g.WaterAlpha, g.WaterFlowOffset,
	)

	if g.DrawRetryButton {
		g.RetryButton.Draw(dst)
	}
}

func forEachBgTile(
	board Board,
	boardRect FRectangle,
	tileStyles [][]TileStyle,
	callback func(x, y int, style TileStyle, bgTileRect FRectangle),
) {
	iter := NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	for iter.HasNext() {
		x, y := iter.GetNext()

		style := tileStyles[x][y]

		ogTileRect := GetBoardTileRect(boardRect, board.Width, board.Height, x, y)

		// draw background tile
		bgTileRect := ogTileRect
		bgTileRect = bgTileRect.Add(FPt(style.BgOffsetX, style.BgOffsetY))
		bgTileRect = FRectScaleCentered(bgTileRect, style.BgScale)

		callback(x, y, style, bgTileRect)
	}
}

func isTileFirmlyPlaced(style TileStyle) bool {
	const e = 0.08
	return style.DrawTile &&
		CloseToEx(style.TileScale, 1, e) &&
		CloseToEx(style.TileOffsetX, 0, e) &&
		CloseToEx(style.TileOffsetY, 0, e) &&
		style.TileAlpha > e
}

func forEachTile(
	board Board,
	boardRect FRectangle,
	tileStyles [][]TileStyle,
	callback func(x, y int, style TileStyle, strokeRect, fillRect FRectangle, radiusPx [4]float64),
) {
	iter := NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	for iter.HasNext() {
		x, y := iter.GetNext()

		style := tileStyles[x][y]

		ogTileRect := GetBoardTileRect(boardRect, board.Width, board.Height, x, y)

		tileRect := ogTileRect

		tileRect = tileRect.Add(FPt(style.TileOffsetX, style.TileOffsetY))
		tileRect = FRectScaleCentered(tileRect, style.TileScale)

		strokeRect := tileRect.Inset(-2)
		//strokeRect := tileRect
		fillRect := tileRect.Add(FPt(0, -1.5))

		radiusPx := float64(0)

		if style.TileScale < 1.00 {
			radius := 1 - style.TileScale
			radius = Clamp(radius, 0, 1)
			radius = EaseOutQuint(radius)
			radiusPx = min(fillRect.Dx(), fillRect.Dy()) * 0.5 * radius
		} else {
			dx := fillRect.Dx() - ogTileRect.Dx()
			dy := fillRect.Dy() - ogTileRect.Dy()

			t := min(dx, dy)
			t = max(t, 0)

			radiusPx = t * 0.7
		}

		radiusPx = max(fillRect.Dx(), fillRect.Dy()) * 0.5 * 0.2

		tileRadiuses := [4]float64{
			radiusPx,
			radiusPx,
			radiusPx,
			radiusPx,
		}

		if isTileFirmlyPlaced(style) {
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

				if 0 <= rx && rx < board.Width && 0 <= ry && ry < board.Height {
					if isTileFirmlyPlaced(tileStyles[rx][ry]) {
						tileRadiuses[i] = 0
						tileRadiuses[(i+1)%4] = 0
					}
				}
			}
		}

		callback(x, y, style, strokeRect, fillRect, tileRadiuses)
	}
}

func forEachFgTile(
	board Board,
	boardRect FRectangle,
	tileStyles [][]TileStyle,
	callback func(x, y int, style TileStyle, fgRect FRectangle),
) {
	forEachTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, strokeRect, fillRect FRectangle, radiusPx [4]float64) {
			fgRect := fillRect
			callback(x, y, style, fgRect)
		},
	)
}

func ShouldDrawBgTile(style TileStyle) bool {
	return style.DrawBg && style.BgScale > 0.001 && style.BgAlpha > 0.001
}

func ShouldDrawTile(style TileStyle) bool {
	return style.TileScale > 0.001 && style.DrawTile && style.TileAlpha > 0.001
}

func ShouldDrawFgTile(style TileStyle) bool {
	return style.TileScale > 0.001 &&
		style.FgScale > 0.001 &&
		style.DrawFg &&
		style.FgType != TileFgTypeNone &&
		style.FgAlpha > 0.001
}

func DrawBoard(
	dst *eb.Image,

	board Board,
	boardRect FRectangle,
	tileStyles [][]TileStyle,

	// params for water effect
	doWaterEffect bool,
	waterRenderTarget *eb.Image,
	waterAlpha float64,
	waterFlowOffset time.Duration,
) {
	// ============================
	// draw background tiles
	// ============================
	forEachBgTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, bgTileRect FRectangle) {
			if ShouldDrawBgTile(style) {
				DrawFilledRect(dst, bgTileRect, ColorFade(style.BgFillColor, style.BgAlpha))

				// draw highlight
				if style.BgTileHightlight > 0 {
					t := style.BgTileHightlight
					c := TheColorTable[ColorTileHighLight]
					DrawFilledRect(
						dst,
						bgTileRect,
						ColorFade(color.NRGBA{c.R, c.G, c.B, uint8(f64(c.A) * t)}, style.BgAlpha),
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

					DrawFilledRoundRectFast(dst, outerRect, outerRadius, true, 5, ColorFade(ColorMineBg1, style.BgAlpha))
					DrawFilledRoundRectFast(dst, innerRect, innerRadius, true, 5, ColorFade(ColorMineBg2, style.BgAlpha))

					DrawSubViewInRect(dst, innerRect, 1, 0, 0, ColorFade(ColorMine, style.BgAlpha), GetMineTile())
				}
			}
		},
	)

	// ============================
	// draw tiles
	// ============================
	const segments = 5

	waterRenderTarget.Clear()

	forEachTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, strokeRect, fillRect FRectangle, radiusPx [4]float64) {
			if ShouldDrawTile(style) {
				strokeColor := ColorFade(style.TileStrokeColor, style.TileAlpha)

				DrawFilledRoundRectFastEx(waterRenderTarget, strokeRect, radiusPx, true, [4]int{segments, segments, segments, segments}, strokeColor)
			}
		},
	)

	forEachTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, strokeRect, fillRect FRectangle, radiusPx [4]float64) {
			if ShouldDrawTile(style) {
				fillColor := ColorFade(style.TileFillColor, style.TileAlpha)

				DrawFilledRoundRectFastEx(waterRenderTarget, fillRect, radiusPx, true, [4]int{segments, segments, segments, segments}, fillColor)
			}
		},
	)

	// draw water effect
	if doWaterEffect {
		rect := boardRect
		rect = rect.Inset(-3)

		colors := [4]color.Color{
			ColorWater1,
			ColorWater2,
			ColorWater3,
			ColorWater4,
		}

		for i, c := range colors {
			nrgba := ColorToNRGBA(c)
			colors[i] = color.NRGBA{nrgba.R, nrgba.G, nrgba.B, uint8(f64(nrgba.A) * waterAlpha)}
		}

		BeginBlend(eb.BlendSourceAtop)
		DrawWaterRect(
			waterRenderTarget,
			rect,
			GlobalTimerNow()+waterFlowOffset,
			colors,
			FPt(0, 0),
		)
		EndBlend()
	}

	// draw waterRenderTarget
	DrawImage(dst, waterRenderTarget, nil)

	forEachFgTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, fgRect FRectangle) {
			if ShouldDrawFgTile(style) {
				fgColor := style.FgColor

				if style.FgType == TileFgTypeNumber {
					count := board.GetNeighborMineCount(x, y)
					if 1 <= count && count <= 8 {
						DrawSubViewInRect(
							dst,
							fgRect,
							style.FgScale,
							style.FgOffsetX, style.FgOffsetY,
							ColorFade(fgColor, style.FgAlpha),
							GetNumberTile(count),
						)
					}
				} else if style.FgType == TileFgTypeFlag {
					DrawSubViewInRect(
						dst,
						fgRect,
						style.FgScale,
						style.FgOffsetX, style.FgOffsetY,
						ColorFade(fgColor, style.FgAlpha),
						GetFlagTile(),
					)
				}
			}
		},
	)
}

func DrawDummyBgBoard(
	dst *eb.Image,
	boardWidth, boardHeight int,
	boardRect FRectangle,
	waterRenderTarget *eb.Image,
) {
	// TODO: this is fucking stupid. We are creating a dummy board only to throw it away
	// each time we draw
	dummyBoard := NewBoard(boardWidth, boardHeight)
	dummyStyles := New2DArray[TileStyle](boardWidth, boardHeight)

	for x := range boardWidth {
		for y := range boardWidth {
			dummyStyles[x][y] = GetAnimationTargetTileStyle(dummyBoard, x, y)
		}
	}

	DrawBoard(
		dst,

		dummyBoard, boardRect,
		dummyStyles,

		false, waterRenderTarget, 0, 0,
	)
}

func (g *Game) MousePosToBoardPos(mousePos FPoint) (int, int) {
	boardRect := g.Rect

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
	boardRect := g.Rect
	rect := FRectWH(g.RetryButtonSize, g.RetryButtonSize)
	center := FRectangleCenter(boardRect)
	rect = CenterFRectangle(rect, center.X, center.Y)

	return rect
}

func (g *Game) TransformedRetryButtonRect() FRectangle {
	rect := g.RetryButtonRect()
	rect = FRectScaleCentered(rect, g.RetryButtonScale)
	rect = rect.Add(FPt(g.RetryButtonOffsetX, g.RetryButtonOffsetY))
	return rect
}

func GetAnimationTargetTileStyle(board Board, x, y int) TileStyle {
	style := NewTileStyle()

	style.DrawBg = true
	style.BgFillColor = ColorTileNormal1
	if IsOddTile(board.Width, board.Height, x, y) {
		style.BgFillColor = ColorTileNormal2
	}

	if board.IsPosInBoard(x, y) {
		if board.Revealed[x][y] {
			style.DrawTile = true

			style.TileFillColor = ColorTileRevealed1
			if IsOddTile(board.Width, board.Height, x, y) {
				style.TileFillColor = ColorTileRevealed2
			}
			style.TileStrokeColor = ColorTileRevealedStroke

			count := board.GetNeighborMineCount(x, y)

			if 1 <= count && count <= 8 {
				style.DrawFg = true
				style.FgType = TileFgTypeNumber
				style.FgColor = ColorTableGetNumber(count)
			}
		}

		if board.Flags[x][y] {
			style.FgType = TileFgTypeFlag
			style.FgColor = ColorFlag
		}
	}

	return style
}

func (g *Game) QueueRevealAnimation(revealsBefore, revealsAfter [][]bool, originX, originy int) {
	g.SkipAllAnimations()

	fw, fh := f64(g.Board.Width-1), f64(g.Board.Height-1)

	originP := FPt(f64(originX), f64(originy))

	maxDist := math.Sqrt(fw*fw + fh*fh)

	const maxDuration = time.Millisecond * 900
	const minDuration = time.Millisecond * 20

	for x := range g.Board.Width {
		for y := range g.Board.Height {
			if !revealsBefore[x][y] && revealsAfter[x][y] {
				pos := FPt(f64(x), f64(y))
				dist := pos.Sub(originP).Length()
				d := time.Duration(f64(maxDuration) * (dist / maxDist))

				var timer Timer

				timer.Duration = max(d, minDuration)
				timer.Current = 0

				targetStyle := GetAnimationTargetTileStyle(g.Board, x, y)

				var anim CallbackAnimation
				anim.Tag = AnimationTagTileReveal

				anim.Update = func() {
					style := g.BaseTileStyles[x][y]
					timer.TickUp()

					t := timer.Normalize()
					t = t * t

					const limit = 0.4

					if t > limit {
						style = targetStyle

						t = ((t - limit) / (1 - limit))
						t = Clamp(t, 0, 1)

						style.TileScale = Lerp(1.2, 1.0, t)
					} else {
						style.DrawTile = false
						style.DrawFg = false
					}

					g.BaseTileStyles[x][y] = style
				}

				anim.Skip = func() {
					timer.Current = timer.Duration
					anim.Update()
				}

				anim.Done = func() bool {
					return timer.Current >= timer.Duration
				}

				g.TileAnimations[x][y].Enqueue(anim)
			}
		}
	}
}

func (g *Game) QueueDefeatAnimation(originX, originY int) {
	g.SkipAllAnimations()

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
		var anim CallbackAnimation
		anim.Tag = AnimationTagDefeat

		anim.Update = func() {
			style := g.BaseTileStyles[p.X][p.Y]
			timer.TickUp()
			style.BgBombAnim = timer.Normalize()
			g.BaseTileStyles[p.X][p.Y] = style
		}

		anim.Skip = func() {
			timer.Current = timer.Duration
			anim.Update()
		}

		anim.Done = func() bool {
			return timer.Current >= timer.Duration
		}

		anim.AfterDone = func() {
		}

		g.TileAnimations[p.X][p.Y].Enqueue(anim)
	}

	defeatDuration += time.Millisecond * 10

	var defeatAnimTimer Timer
	defeatAnimTimer.Duration = defeatDuration

	var anim CallbackAnimation
	anim.Tag = AnimationTagDefeat

	anim.Update = func() {
		defeatAnimTimer.TickUp()
	}

	anim.Skip = func() {
		defeatAnimTimer.Current = defeatAnimTimer.Duration
		anim.Update()
	}

	anim.Done = func() bool {
		return defeatAnimTimer.Current >= defeatAnimTimer.Duration
	}

	anim.AfterDone = func() {
		g.QueueRetryButtonAnimation()
	}

	g.GameAnimations.Enqueue(anim)
}

func (g *Game) QueueWinAnimation(originX, originY int) {
	g.SkipAllAnimations()

	fw, fh := f64(g.Board.Width), f64(g.Board.Height)

	originP := FPt(f64(originX), f64(originY))

	maxDist := math.Sqrt(fw*fw + fh*fh)

	const maxDuration = time.Millisecond * 1000
	const minDuration = time.Millisecond * 50
	const distStartOffset = time.Millisecond * 3

	var winDuration time.Duration

	// queue tile animations
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			pos := FPt(f64(x), f64(y))
			dist := pos.Sub(originP).Length()
			d := time.Duration(f64(maxDuration) * (dist / maxDist))

			var timer Timer

			timer.Duration = max(d, minDuration)
			timer.Current = -time.Duration(dist * f64(distStartOffset))

			winDuration = max(winDuration, timer.Duration-timer.Current)

			// add new animation
			var anim CallbackAnimation
			anim.Tag = AnimationTagWin

			var ogFgColor color.Color

			anim.Update = func() {
				style := g.BaseTileStyles[x][y]
				timer.TickUp()

				scaleT := timer.NormalizeUnclamped() * 0.8
				scaleT = Clamp(scaleT, 0, 1)
				scaleT = EaseOutElastic(scaleT)

				// force scale to be 1 at the end of win animation
				scaleT = max(scaleT, EaseInQuint(timer.Normalize()))

				style.TileScale = scaleT
				style.FgScale = Clamp(scaleT, 0, 1)

				colorT := EaseOutQuint(timer.Normalize())
				colorT = Clamp(colorT, 0, 1)

				if ogFgColor == nil {
					ogFgColor = style.FgColor
				}

				if g.Board.Revealed[x][y] {
					style.FgColor = LerpColorRGBA(ogFgColor, ColorElementWon, colorT)
				} else {
					style.FgAlpha = 1 - colorT
				}

				style.BgAlpha = 1 - colorT

				g.BaseTileStyles[x][y] = style
			}

			anim.Skip = func() {
				timer.Current = timer.Duration
				anim.Update()
			}

			anim.Done = func() bool {
				return timer.Current >= timer.Duration
			}

			anim.AfterDone = func() {
				style := g.BaseTileStyles[x][y]
				if !g.Board.Revealed[x][y] {
					style.DrawFg = false
				}
				style.DrawBg = false
				g.BaseTileStyles[x][y] = style
			}

			g.TileAnimations[x][y].Enqueue(anim)
		}
	}

	// queue game animation
	var winAnimTimer Timer

	winAnimTimer.Duration = winDuration + time.Millisecond*100
	winAnimTimer.Current = 0

	var anim CallbackAnimation
	anim.Tag = AnimationTagWin

	anim.Update = func() {
		winAnimTimer.TickUp()

		t := winAnimTimer.Normalize()

		g.WaterAlpha = EaseOutQuint(t)

		waterT := EaseOutQuint(t)

		g.WaterFlowOffset = time.Duration(waterT * f64(time.Second) * 10)
	}

	anim.Skip = func() {
		winAnimTimer.Current = winAnimTimer.Duration
		anim.Update()
	}

	anim.Done = func() bool {
		return winAnimTimer.Current >= winAnimTimer.Duration
	}

	anim.AfterDone = func() {
		g.QueueRetryButtonAnimation()
	}

	g.GameAnimations.Enqueue(anim)
}

func (g *Game) QueueRetryButtonAnimation() {
	g.SkipAllAnimations()

	buttonRect := g.RetryButtonRect()
	buttonRect = buttonRect.Inset(-max(buttonRect.Dx(), buttonRect.Dy()) * 0.03)

	toAnimate := make([]image.Point, 0)

	for x := range g.Board.Width {
		for y := range g.Board.Height {
			tileRect := GetBoardTileRect(g.Rect, g.Board.Width, g.Board.Height, x, y)

			animate := false

			if tileRect.Overlaps(buttonRect) {
				animate = true
			}

			if animate {
				toAnimate = append(toAnimate, image.Pt(x, y))
			}
		}
	}

	var maxDuration time.Duration

	for _, p := range toAnimate {
		var timer Timer

		timer.Duration = time.Millisecond * 300
		timerStart := -time.Duration(rand.Int64N(i64(time.Millisecond * 200)))
		timer.Current = timerStart

		var anim CallbackAnimation
		anim.Tag = AnimationTagRetryButtonReveal

		anim.Update = func() {
			style := g.BaseTileStyles[p.X][p.Y]
			timer.TickUp()

			t := timer.Normalize()
			offsetT := EaseInQuint(t)

			style.TileOffsetY = -offsetT * 20

			const limit = 0.8
			if t > limit {
				colorT := (t - limit) / (1 - limit)
				//colorT = EaseInQuint(colorT)
				colorT = Clamp(colorT, 0, 1)

				style.TileAlpha = (1 - colorT)
				style.FgAlpha = (1 - colorT)
			}

			//bgT := f64(timer.Current - timerStart) / (f64(timer.Duration) * 0.3)
			bgT := t
			bgT = Clamp(bgT, 0, 1)
			bgT = EaseInCubic(bgT)
			style.BgAlpha = min(style.BgAlpha, (1 - bgT))
			style.BgScale = (1 - bgT)
			style.BgOffsetY = bgT * 3

			g.BaseTileStyles[p.X][p.Y] = style
		}

		anim.Skip = func() {
			timer.Current = timer.Duration
			anim.Update()
		}

		anim.Done = func() bool {
			return timer.Current >= timer.Duration
		}

		anim.AfterDone = func() {
			style := g.BaseTileStyles[p.X][p.Y]
			style.DrawBg = false
			style.DrawTile = false
			style.DrawFg = false
			g.BaseTileStyles[p.X][p.Y] = style
		}

		g.TileAnimations[p.X][p.Y].Enqueue(anim)

		maxDuration = max(maxDuration, timer.Duration-timer.Current)
	}

	// queue game animation
	{
		var timer Timer
		timer.Duration = time.Millisecond * 400
		// give a little delay
		timer.Current = -time.Duration(f64(maxDuration) * 0.8)

		var anim CallbackAnimation
		anim.Tag = AnimationTagRetryButtonReveal

		anim.Update = func() {
			g.DrawRetryButton = true
			timer.TickUp()

			t := timer.Normalize()

			g.RetryButtonScale = EaseOutElastic(t)

			g.RetryButton.Disabled = true
			g.RetryButton.Disabled = !(g.RetryButtonScale > 0.5)
		}

		anim.Skip = func() {
			timer.Current = timer.Duration
			anim.Update()
		}

		anim.Done = func() bool {
			return timer.Current >= timer.Duration
		}

		g.GameAnimations.Enqueue(anim)
	}
}

func (g *Game) QueueResetBoardAnimation() {
	g.SkipAllAnimations()

	fw, fh := f64(g.Board.Width), f64(g.Board.Height)

	centerP := FPt(f64(g.Board.Width-1)*0.5, f64(g.Board.Height-1)*0.5)

	maxDist := math.Sqrt(fw*0.5*fw*0.5 + fh*0.5*fh*0.5)

	const minDuration = time.Millisecond * 120
	const maxDuration = time.Millisecond * 400

	var tileAnimationTotal time.Duration

	for x := range g.Board.Width {
		for y := range g.Board.Height {
			pos := FPt(f64(x), f64(y))
			dist := pos.Sub(centerP).Length()
			d := time.Duration(Lerp(f64(minDuration), f64(maxDuration), 1-dist/maxDist))

			var timer Timer
			timer.Duration = d

			var anim CallbackAnimation
			anim.Tag = AnimationTagHideBoard

			anim.Update = func() {
				style := g.BaseTileStyles[x][y]

				timer.TickUp()

				t := timer.Normalize()

				tileCurve := TheBezierTable[BezierBoardHideTile]
				alphaCurve := TheBezierTable[BezierBoardHideTileAlpha]

				tileT := BezierCurveDataAsGraph(tileCurve, t)
				alphaT := BezierCurveDataAsGraph(alphaCurve, t)

				style.BgOffsetY = Lerp(0, 30, tileT)
				style.TileOffsetY = Lerp(0, 30, tileT)

				style.BgAlpha = Clamp(alphaT, 0, 1)
				style.TileAlpha = Clamp(alphaT, 0, 1)
				style.FgAlpha = Clamp(alphaT, 0, 1)

				sizeT := 1 - tileT

				style.TileScale = sizeT
				style.BgScale = Clamp(sizeT, 0, 1)
				style.BgScale *= style.BgScale

				g.BaseTileStyles[x][y] = style
			}

			anim.Skip = func() {
				timer.Current = timer.Duration
				anim.Update()
			}

			anim.Done = func() bool {
				return timer.Current >= timer.Duration
			}

			anim.AfterDone = func() {
				style := g.BaseTileStyles[x][y]
				style.DrawBg = false
				style.DrawTile = false
				style.DrawFg = false
				g.BaseTileStyles[x][y] = style
			}

			g.TileAnimations[x][y].Enqueue(anim)

			tileAnimationTotal = max(tileAnimationTotal, timer.Duration-timer.Current)
		}
	}

	// queue game animation
	{
		var buttonTimer Timer
		buttonTimer.Duration = maxDuration * 6 / 10

		var resetAnimTimer Timer
		resetAnimTimer.Duration = tileAnimationTotal + time.Millisecond*200

		var anim CallbackAnimation
		anim.Tag = AnimationTagHideBoard

		anim.Update = func() {
			g.RetryButton.Disabled = true

			buttonTimer.TickUp()
			resetAnimTimer.TickUp()

			t := buttonTimer.Normalize()

			curve := TheBezierTable[BezierBoardHideButton]
			curveT := BezierCurveDataAsGraph(curve, t)

			g.RetryButtonScale = 1 - curveT
			g.RetryButtonScale = max(g.RetryButtonScale, 0)
		}

		anim.Skip = func() {
			buttonTimer.Current = buttonTimer.Duration
			resetAnimTimer.Current = resetAnimTimer.Duration
			anim.Update()
		}

		anim.Done = func() bool {
			return buttonTimer.Current >= buttonTimer.Duration && resetAnimTimer.Current >= resetAnimTimer.Duration
		}

		anim.AfterDone = func() {
			g.DrawRetryButton = true
			g.ResetBoardWithNoStyles(g.Board.Width, g.Board.Height)
			g.QueueShowBoardAnimation(
				g.Board.Width/2,
				g.Board.Height/2,
			)
		}

		g.GameAnimations.Enqueue(anim)
	}
}

func (g *Game) QueueShowBoardAnimation(originX, originy int) {
	g.SkipAllAnimations()

	originP := FPt(f64(originX), f64(originy))

	var maxDist float64

	maxDist = max(maxDist, originP.Sub(FPt(0, 0)).Length())
	maxDist = max(maxDist, originP.Sub(FPt(f64(g.Board.Width-1), 0)).Length())
	maxDist = max(maxDist, originP.Sub(FPt(0, f64(g.Board.Height-1))).Length())
	maxDist = max(maxDist, originP.Sub(FPt(f64(g.Board.Width-1), f64(g.Board.Height-1))).Length())

	const minDuration = time.Millisecond * 80
	const maxDuration = time.Millisecond * 200

	var tileAnimationTotal time.Duration

	for x := range g.Board.Width {
		for y := range g.Board.Height {
			pos := FPt(f64(x), f64(y))
			dist := pos.Sub(originP).Length()
			d := time.Duration(Lerp(f64(minDuration), f64(maxDuration), dist/maxDist))

			var timer Timer
			timer.Duration = d

			var anim CallbackAnimation
			anim.Tag = AnimationTagShowBoard

			targetStyle := GetAnimationTargetTileStyle(g.Board, x, y)

			anim.Update = func() {
				timer.TickUp()

				t := timer.Normalize()

				yOffsetT := BezierCurveDataAsGraph(TheBezierTable[BezierBoardShowTileYoffset], t)
				alphaT := BezierCurveDataAsGraph(TheBezierTable[BezierBoardShowTileAlpha], t)
				scaleT := BezierCurveDataAsGraph(TheBezierTable[BezierBoardShowTileScale], t)

				targetStyle.BgOffsetY = yOffsetT * 50
				targetStyle.BgAlpha = alphaT
				targetStyle.BgScale = scaleT

				g.BaseTileStyles[x][y] = targetStyle
			}

			anim.Skip = func() {
				timer.Current = timer.Duration
				anim.Update()
			}

			anim.Done = func() bool {
				return timer.Current >= timer.Duration
			}

			g.TileAnimations[x][y].Enqueue(anim)

			tileAnimationTotal = max(tileAnimationTotal, timer.Duration-timer.Current)
		}
	}

	// queue game animation
	{
		var timer Timer
		timer.Duration = tileAnimationTotal

		var anim CallbackAnimation
		anim.Tag = AnimationTagShowBoard

		anim.Update = func() {
			timer.TickUp()
		}

		anim.Skip = func() {
			timer.Current = timer.Duration
			anim.Update()
		}

		anim.Done = func() bool {
			return timer.Current >= timer.Duration
		}

		anim.AfterDone = func() {
			g.GameState = GameStatePlaying
		}

		g.GameAnimations.Enqueue(anim)
	}
}

func (g *Game) SkipAllAnimations() {
	for x := range g.Board.Width {
		for y := range g.Board.Height {
			for !g.TileAnimations[x][y].IsEmpty() {
				anim := g.TileAnimations[x][y].Dequeue()
				anim.Skip()
				if anim.AfterDone != nil {
					anim.AfterDone()
				}
			}
		}
	}

	for !g.GameAnimations.IsEmpty() {
		gameAnim := g.GameAnimations.Dequeue()

		gameAnim.Skip()

		if gameAnim.AfterDone != nil {
			gameAnim.AfterDone()
		}
	}
}

func (g *Game) SkipAllAnimationsUntilTag(tag AnimationTag) {
	for x := range g.Board.Width {
		for y := range g.Board.Height {
		TAG_LOOP:
			for !g.TileAnimations[x][y].IsEmpty() {
				tileAnim := g.TileAnimations[x][y].At(0)
				if tileAnim.Tag != tag {
					g.TileAnimations[x][y].Dequeue()

					tileAnim.Skip()

					if tileAnim.AfterDone != nil {
						tileAnim.AfterDone()
					}
				} else {
					break TAG_LOOP
				}
			}
		}
	}

	for !g.GameAnimations.IsEmpty() {
		gameAnim := g.GameAnimations.At(0)
		if gameAnim.Tag != tag {
			g.GameAnimations.Dequeue()

			gameAnim.Skip()

			if gameAnim.AfterDone != nil {
				gameAnim.AfterDone()
			}
		} else {
			break
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

	drawScale := min(rectSize.X/imgSize.X, rectSize.Y/imgSize.Y)

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
	tileRect := GetBoardTileRect(g.Rect, g.Board.Width, g.Board.Height, boardX, boardY)
	DrawSubViewInRect(dst, tileRect, scale, offsetY, offsetY, clr, tile)
}

func IsOddTile(boardWidth, boardHeight, x, y int) bool {
	if y%2 == 0 {
		return x%2 != 0
	} else {
		return x%2 == 0
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

func (g *Game) Layout(outsideWidth, outsideHeight int) {
	if g.WaterRenderTarget.Bounds().Dx() != outsideWidth || g.WaterRenderTarget.Bounds().Dy() != outsideHeight {
		g.WaterRenderTarget = eb.NewImage(outsideWidth, outsideHeight)
	}
}

// =====================
// debugging functions
// =====================

func (g *Game) SetDebugBoardForDecoration() {

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

	g.ResetBoard(max(g.Board.Width, newBoardWidth), max(g.Board.Height, newBoardHeight))
	g.QueueShowBoardAnimation(
		(g.Board.Width-1)/2,
		(g.Board.Height-1)/2,
	)

	g.PlacedMinesOnBoard = true

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
	if !g.PlacedMinesOnBoard {
		g.Board.PlaceMines(g.MineCount, g.Board.Width-1, g.Board.Height-1)
	}
	g.PlacedMinesOnBoard = true

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
