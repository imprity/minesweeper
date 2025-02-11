package minesweeper

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

var _ = fmt.Printf

type GameInputType int

const (
	InputTypeNone GameInputType = iota
	InputTypeStep
	InputTypeFlag
	InputTypeCheck
	InputTypeHover
	InputTypeHL
)

type GameInput struct {
	Type GameInputType

	BoardX int
	BoardY int

	ByTouch bool
}

type GameInputHandler struct {
	NoInputZones []FRectangle

	gameInput GameInput

	// pinch
	pinchPos     FPoint
	prevPinchPos FPoint

	pinch     float64
	prevPinch float64

	pinchStarted bool

	// dragging
	dragDelta     FPoint
	dragStartPos  FPoint
	dragStarted   bool
	dragPastLimit bool

	// flggging stuff
	draggingForFlag bool
	flagTouchId     eb.TouchID
	ignoreForFlag   map[eb.TouchID]bool
}

func NewGameInputHandler() *GameInputHandler {
	gi := new(GameInputHandler)

	gi.ignoreForFlag = make(map[eb.TouchID]bool)

	return gi
}

func (gi *GameInputHandler) Update(
	board Board,
	boardRect FRectangle,
	gameState GameState,
) {
	im := &TheInputManager
	// =============================
	// update mouse input
	// =============================
	input := GameInput{}

	pressedL := IsMouseButtonPressed(eb.MouseButtonLeft)
	pressedR := IsMouseButtonPressed(eb.MouseButtonRight)
	pressedM := IsMouseButtonPressed(eb.MouseButtonMiddle)

	justPressedL := IsMouseButtonJustPressed(eb.MouseButtonLeft)
	justPressedR := IsMouseButtonJustPressed(eb.MouseButtonRight)
	justPressedM := IsMouseButtonJustPressed(eb.MouseButtonMiddle)

	cursor := CursorFPt()

	input.BoardX, input.BoardY = MousePosToBoardPos(
		boardRect,
		board.Width, board.Height,
		cursor,
	)

	cursorInRect := cursor.In(boardRect)
	var cursorInNoInputZone bool

	for _, zone := range gi.NoInputZones {
		if cursor.In(zone) {
			cursorInNoInputZone = true
			break
		}
	}

	if cursorInRect && !cursorInNoInputZone {
		input.Type = InputTypeHover

		if (pressedL && pressedR) || pressedM {
			input.Type = InputTypeHL
		}

		if justPressedR {
			input.Type = InputTypeFlag
		}

		if justPressedL {
			input.Type = InputTypeStep
		}

		if (justPressedL && pressedR) || (pressedL && justPressedR) || justPressedM {
			input.Type = InputTypeCheck
		}
	}

	// =============================
	// update touch board input
	// =============================

	for _, touchId := range im.JustTouchedBuf {
		delete(gi.ignoreForFlag, touchId)
	}

	// check touch
	for touchId, info := range im.TouchInfos {
		if info.MaxTouchCount > 1 {
			continue
		}

		// ignore touches that started in NoInputZones
		{
			var startedInNoInputZone bool
			for _, zone := range gi.NoInputZones {
				if info.StartedPos.In(zone) {
					startedInNoInputZone = true
					break
				}
			}
			if startedInNoInputZone {
				continue
			}
		}

		curPos := TouchFPt(touchId)

		curBX, curBY := MousePosToBoardPos(
			boardRect,
			board.Width, board.Height,
			curPos,
		)
		_ = curBX
		_ = curBY

		startedBX, startedBY := MousePosToBoardPos(
			boardRect,
			board.Width, board.Height,
			info.StartedPos,
		)
		_ = startedBX
		_ = startedBY

		endedBX, endedBY := MousePosToBoardPos(
			boardRect,
			board.Width, board.Height,
			info.EndedPos,
		)

		justReleased := info.DidEnd && info.EndedTime == GlobalTimerNow()

		tapped := justReleased && !info.Dragged

		if tapped && board.IsPosInBoard(endedBX, endedBY) {
			input.BoardX, input.BoardY = endedBX, endedBY
			input.ByTouch = true
			if board.Revealed.Get(endedBX, endedBY) {
				input.Type = InputTypeCheck
			} else {
				if board.Flags.Get(endedBX, endedBY) {
					input.Type = InputTypeFlag
				} else {
					input.Type = InputTypeStep
				}
			}
		}

		var startedInNum bool

		// check if touch started in number tile
		startedInNum = board.IsPosInBoard(startedBX, startedBY)
		startedInNum = startedInNum && board.Revealed.Get(startedBX, startedBY)
		startedInNum = startedInNum && board.GetNeighborMineCount(startedBX, startedBY) > 0

		// if it did start in number tile, check it it has any space to left to flag
		if startedInNum {
			iter := NewBoardIterator(startedBX-1, startedBY-1, startedBX+1, startedBY+1)

			tilesCanBeFlagged := 0
			tilesThatAreFlagged := 0

			for iter.HasNext() {
				x, y := iter.GetNext()

				if !board.IsPosInBoard(x, y) {
					continue
				}

				if !board.Revealed.Get(x, y) {
					tilesCanBeFlagged++
				}

				if board.Flags.Get(x, y) {
					tilesThatAreFlagged++
				}
			}

			if tilesCanBeFlagged == tilesThatAreFlagged {
				startedInNum = false
			}
		}

		if !info.DidEnd && startedInNum {
			var safeNeighbors [9]bool

			iter := NewBoardIterator(
				startedBX-1, startedBY-1,
				startedBX+1, startedBY+1,
			)

			for iter.HasNext() {
				x, y := iter.GetNext()

				if board.IsPosInBoard(x, y) && !board.Revealed.Get(x, y) {
					innerIter := NewBoardIterator(
						max(x-1, startedBX-1), max(y-1, startedBY-1),
						min(x+1, startedBX+1), min(y+1, startedBY+1),
					)

					for innerIter.HasNext() {
						x2, y2 := innerIter.GetNext()
						if board.IsPosInBoard(x2, y2) {
							rx, ry := x2-startedBX+1, y2-startedBY+1
							index := ry*3 + rx
							safeNeighbors[index] = true
						}
					}
				}
			}

			inSafeNeighbor := false

			iter = NewBoardIterator(0, 0, 2, 2)
			for iter.HasNext() {
				rx, ry := iter.GetNext()
				index := ry*3 + rx

				if !safeNeighbors[index] {
					continue
				}

				x, y := startedBX+(rx-1), startedBY+(ry-1)

				tileRect := GetBoardTileRect(
					boardRect,
					board.Width, board.Height,
					x, y,
				)

				tileRect = FRectScaleCentered(tileRect, 2, 2)

				if curPos.In(tileRect) {
					inSafeNeighbor = true
					break
				}
			}

			if !inSafeNeighbor {
				gi.ignoreForFlag[touchId] = true
			}
		}

		if !gi.ignoreForFlag[touchId] && startedInNum && !info.DidEnd {
			input.BoardX, input.BoardY = startedBX, startedBY
			input.Type = InputTypeHL
			input.ByTouch = true

			gi.draggingForFlag = true
			gi.flagTouchId = touchId
		} else {
			gi.draggingForFlag = false
		}

		if !gi.ignoreForFlag[touchId] && startedInNum && info.DidEnd {
			var foundNeighboar bool = false
			var neighborX, neighborY int

			if (startedBX != endedBX) || (startedBY != endedBY) {
				iter := NewBoardIterator(startedBX-1, startedBY-1, startedBX+1, startedBY+1)

				var minDist float64 = math.MaxFloat64

				for iter.HasNext() {
					x, y := iter.GetNext()
					if !board.IsPosInBoard(x, y) {
						continue
					}
					if board.Revealed.Get(x, y) {
						continue
					}
					if x == startedBX && y == startedBY {
						continue
					}
					if max(Abs(x-endedBX), Abs(y-endedBY)) > 1 {
						continue
					}

					tile := GetBoardTileRect(
						boardRect,
						board.Width, board.Height,
						x, y,
					)
					tileCenter := FRectangleCenter(tile)
					dist := tileCenter.Sub(info.EndedPos).LengthSquared()

					if dist < minDist {
						minDist = dist
						foundNeighboar = true
						neighborX, neighborY = x, y
					}
				}
			}

			if foundNeighboar {
				gi.ignoreForFlag[touchId] = true
				input.BoardX, input.BoardY = neighborX, neighborY
				input.Type = InputTypeFlag
				input.ByTouch = true
			}
		}

		// ======================
		// handle dragging
		// ======================
		if (!startedInNum || gi.ignoreForFlag[touchId] || gameState != GameStatePlaying) && !info.DidEnd {
			gi.ignoreForFlag[touchId] = true
			if !gi.dragStarted {
				gi.dragDelta = FPt(0, 0)

				gi.dragStartPos = info.StartedPos
				gi.dragPastLimit = false

				gi.dragStarted = true
			} else {
				pos := TouchFPt(touchId)
				prevPos := PrevTouchFPt(touchId)

				gi.dragDelta = pos.Sub(prevPos)
				if gi.dragStartPos.Sub(pos).LengthSquared() > 20*20 {
					gi.dragPastLimit = true
				}
			}
		} else {
			gi.dragPastLimit = false
			gi.dragStarted = false
		}
	}

	// =============================
	// update touch pinch input
	// =============================
	if len(im.TouchingBuf) == 2 {
		pos1 := TouchFPt(im.TouchingBuf[0])
		pos2 := TouchFPt(im.TouchingBuf[1])

		newPinch := pos1.Sub(pos2).Length()
		newPinchPos := pos1.Add(pos2).Mul(FPt(0.5, 0.5))

		if !gi.pinchStarted {
			gi.pinchPos = newPinchPos
			gi.prevPinchPos = newPinchPos

			gi.pinch = newPinch
			gi.prevPinch = newPinch

			gi.pinchStarted = true
		} else {
			gi.prevPinch = gi.pinch
			gi.prevPinchPos = gi.pinchPos

			gi.pinch = newPinch
			gi.pinchPos = newPinchPos
		}
	} else {
		gi.pinchStarted = false
	}

	// =============================
	// update touch drag input
	// =============================
	if len(im.TouchingBuf) != 1 {
		gi.dragPastLimit = false
		gi.dragStarted = false
	}

	gi.gameInput = input
}

func (gi *GameInputHandler) Draw(dst *eb.Image) {
}

func (gi *GameInputHandler) GetGameInput() GameInput {
	return gi.gameInput
}

func (gi *GameInputHandler) GetZoomAndOffset(
	oldZoom float64, oldOffset FPoint,
	boardCenter FPoint,
) (float64, FPoint) {
	if gi.pinchStarted {
		zoom := gi.pinch / gi.prevPinch
		newZoom := oldZoom * zoom

		toBoardCenter := boardCenter.Sub(gi.pinchPos)
		newBoardCenter := gi.pinchPos.Add(toBoardCenter.Mul(FPt(zoom, zoom)))
		toNewBoardCenter := newBoardCenter.Sub(boardCenter)

		newOffset := oldOffset.Add(toNewBoardCenter)

		pinchDiff := gi.pinchPos.Sub(gi.prevPinchPos)
		newOffset = newOffset.Add(pinchDiff)

		if gi.dragStarted {
			newOffset = newOffset.Add(gi.dragDelta)
		}

		oldZoom = newZoom
		oldOffset = newOffset
	}

	if gi.dragPastLimit {
		oldOffset = oldOffset.Add(gi.dragDelta)
	}

	return oldZoom, oldOffset
}

func (gi *GameInputHandler) IsPinching() bool {
	return gi.pinchStarted
}

func (gi *GameInputHandler) IsDragging() bool {
	return gi.dragPastLimit
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

	BgFillColor color.Color

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

	FgFlagAnim float64

	FgNumber int

	Highlight float64
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

type TileParticle struct {
	SubView SubView

	Color1 color.Color
	Color2 color.Color

	// default is linear
	ColorLerpFunc func(t float64) float64

	// origin
	BoardX int
	BoardY int

	// except rotation, everything is in min of
	// one tile width or height
	Width  float64
	Height float64

	OffsetX float64
	OffsetY float64

	VelocityX float64
	VelocityY float64

	GravityX float64
	GravityY float64

	RotVelocity float64
	Rotation    float64

	// ticks for every update
	// doesn't actually kills the particle (for now)
	// it only controls color for now
	Timer Timer

	Dead bool
}

type TileParticleUnitConverter struct {
	BoardWidth  int
	BoardHeight int

	BoardRect FRectangle
}

func (tc *TileParticleUnitConverter) ToPx(v float64) float64 {
	tileW, tileH := GetBoardTileSize(tc.BoardRect, tc.BoardWidth, tc.BoardHeight)
	return min(tileW, tileH) * v
}

func (tc *TileParticleUnitConverter) FromPx(px float64) float64 {
	tileW, tileH := GetBoardTileSize(tc.BoardRect, tc.BoardWidth, tc.BoardHeight)
	return px / min(tileW, tileH)
}

// convert particle offset to position on screen
func (tc *TileParticleUnitConverter) OffsetToScreen(p TileParticle) (float64, float64) {
	offsetX := tc.ToPx(p.OffsetX)
	offsetY := tc.ToPx(p.OffsetY)

	tileRect := GetBoardTileRect(
		tc.BoardRect,
		tc.BoardWidth, tc.BoardHeight,
		p.BoardX, p.BoardY,
	)

	tileCenter := FRectangleCenter(tileRect)

	return tileCenter.X + offsetX, tileCenter.Y + offsetY
}

func AppendTileParticle(particles []TileParticle, p TileParticle) []TileParticle {
	for i := range particles {
		if particles[i].Dead {
			particles[i] = p
			return particles
		}
	}

	return append(particles, p)
}

type AnimationTag int

const (
	AnimationTagNone AnimationTag = iota

	// animations used by Game
	AnimationTagTileReveal
	AnimationTagAddFlag
	AnimationTagRemoveFlag

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

func AnimationQueueUpdate(queue *CircularQueue[CallbackAnimation]) {
	if !queue.IsEmpty() {
		anim := queue.At(0)
		anim.Update()

		if anim.Done() {
			queue.Dequeue()

			if anim.AfterDone != nil {
				anim.AfterDone()
			}
		}
	}
}

func AnimationQueueSkipAll(queue *CircularQueue[CallbackAnimation]) {
	for !queue.IsEmpty() {
		gameAnim := queue.Dequeue()

		gameAnim.Skip()

		if gameAnim.AfterDone != nil {
			gameAnim.AfterDone()
		}
	}
}

func AnimationQueueSkipUntilTag(queue *CircularQueue[CallbackAnimation], tags ...AnimationTag) {
	for !queue.IsEmpty() {
		gameAnim := queue.At(0)
		if !slices.Contains(tags, gameAnim.Tag) {
			queue.Dequeue()

			gameAnim.Skip()

			if gameAnim.AfterDone != nil {
				gameAnim.AfterDone()
			}
		} else {
			break
		}
	}
}

// called every update
type StyleModifier func(
	prevBoard, board Board,
	boardRect FRectangle,
	interaction BoardInteractionType,
	stateChanged bool, // GameState or board has changed
	prevGameState, gameState GameState,
	tileStyles Array2D[TileStyle], // modify these to change style
	gi GameInput,
) bool

type Game struct {
	Rect FRectangle

	// maximum screen space
	// available to game
	MaxRect FRectangle

	DisableZoomAndPanControl bool
	DoingZoomAnimation       bool

	Zoom   float64
	Offset FPoint

	OnAfterBoardReset  func()
	OnBeforeBoardReset func()
	OnGameEnd          func(didWin bool)
	OnFirstInteraction func()

	BaseTileStyles   Array2D[TileStyle]
	RenderTileStyles Array2D[TileStyle]

	TileAnimations Array2D[*CircularQueue[CallbackAnimation]]

	GameAnimations CircularQueue[CallbackAnimation]

	StyleModifiers []StyleModifier

	InputHandler *GameInputHandler

	RetryButton *RetryButton

	DrawRetryButton    bool
	RetryButtonScale   float64
	RetryButtonOffsetX float64
	RetryButtonOffsetY float64

	GameState GameState

	WaterAlpha      float64
	WaterFlowOffset time.Duration

	Particles []TileParticle

	Seed [32]byte

	FlagTutorial *FlagTutorial

	revealdTilesUsingTouch bool
	plantedFlag            bool

	board     Board
	prevBoard Board

	mineCount int

	resetBoardWidth  int
	resetBoardHeight int
	resetMineCount   int

	hadInteraction bool

	playedAddFlagSound    bool
	playedRemoveFlagSound bool

	retryButtonSize float64
	// relative to min(TransformedBoardRect().Dx(), TransformedBoardRect().Dy())
	retryButtonSizeRelative float64

	// relative to center of TransformedBoardRect()
	retryButtonOffsetX float64
	retryButtonOffsetY float64

	viBuffers [3]*VIBuffer

	noInputZone FRectangle
}

func NewGame(boardWidth, boardHeight, mineCount int) *Game {
	g := new(Game)

	g.Zoom = 1

	g.InputHandler = NewGameInputHandler()

	g.FlagTutorial = NewFlagTutorial()

	g.mineCount = mineCount

	g.resetBoardWidth = boardWidth
	g.resetBoardHeight = boardHeight
	g.resetMineCount = mineCount

	g.StyleModifiers = append(g.StyleModifiers, NewTileHighlightModifier())
	g.StyleModifiers = append(g.StyleModifiers, NewFgClickModifier())
	g.StyleModifiers = append(g.StyleModifiers, g.FlagTutorial.GetFlagTutorialStyleModifier())

	g.RetryButton = NewRetryButton()
	g.RetryButton.Disabled = true
	g.RetryButtonScale = 1

	g.RetryButton.OnAny = func(byTouch bool, timing ButtonTiming) {
		trigger := !byTouch && timing == ButtonTimingOnPress
		trigger = trigger || (byTouch && timing == ButtonTimingOnRelease)

		if trigger {
			g.SkipAllAnimationsUntilTag(AnimationTagHideBoard)
			PlaySoundBytes(SeButtonClick, 1.0)
			g.RetryButton.Disabled = true
			g.QueueResetBoardAnimation()
		}
	}

	g.Seed = GetSeed()

	g.GameAnimations = NewCircularQueue[CallbackAnimation](10)

	g.Particles = make([]TileParticle, 0, 256)

	g.ResetBoard()

	return g
}

func (g *Game) SetResetParameter(boardWidth, boardHeight, mineCount int) {
	g.resetBoardWidth = boardWidth
	g.resetBoardHeight = boardHeight
	g.resetMineCount = mineCount
}

func (g *Game) ResetBoardNotStylesEx(newSeed bool) {
	if g.OnBeforeBoardReset != nil {
		g.OnBeforeBoardReset()
	}

	width := g.resetBoardWidth
	height := g.resetBoardHeight
	mineCount := g.resetMineCount

	if newSeed {
		g.Seed = GetSeed()
	}
	InfoLogger.Printf("resetting, seed : %s", SeedToString(g.Seed))

	g.hadInteraction = false

	g.GameState = GameStatePlaying
	g.GameAnimations.Clear()

	g.board = NewBoard(width, height)
	g.prevBoard = NewBoard(width, height)

	g.mineCount = mineCount

	g.DrawRetryButton = false
	g.RetryButton.Disabled = true
	g.RetryButtonScale = 1
	g.RetryButtonOffsetX = 0
	g.RetryButtonOffsetY = 0

	g.Offset = FPt(0, 0)
	g.Zoom = 1
	g.DisableZoomAndPanControl = false
	g.DoingZoomAnimation = false

	g.BaseTileStyles = NewArray2D[TileStyle](width, height)
	g.RenderTileStyles = NewArray2D[TileStyle](width, height)

	g.TileAnimations = NewArray2D[*CircularQueue[CallbackAnimation]](width, height)
	for x := range width {
		for y := range height {
			// TODO : do we need this much queued animation?
			queue := NewCircularQueue[CallbackAnimation](5)
			g.TileAnimations.Set(x, y, &queue)
		}
	}

	for x := range width {
		for y := range height {
			g.BaseTileStyles.Set(x, y, NewTileStyle())
			g.RenderTileStyles.Set(x, y, NewTileStyle())
		}
	}

	g.Particles = g.Particles[:0]

	if g.OnAfterBoardReset != nil {
		g.OnAfterBoardReset()
	}
}

func (g *Game) ResetBoardEx(newSeed bool) {
	g.ResetBoardNotStylesEx(newSeed)

	for x := range g.resetBoardWidth {
		for y := range g.resetBoardHeight {
			targetStyle := GetAnimationTargetTileStyle(g.board, x, y)
			g.BaseTileStyles.Set(x, y, targetStyle)
			g.RenderTileStyles.Set(x, y, targetStyle)
		}
	}
}

func (g *Game) ResetBoardNotStyles() {
	g.ResetBoardNotStylesEx(true)
}

func (g *Game) ResetBoard() {
	g.ResetBoardEx(true)
}

func (g *Game) Update() {
	g.playedAddFlagSound = false
	g.playedRemoveFlagSound = false

	// =============
	// update input
	// =============
	{
		noInputZones := make([]FRectangle, 0, 2)
		noInputZones = append(noInputZones, g.noInputZone)

		if g.DrawRetryButton && !g.InputHandler.IsDragging() {
			noInputZones = append(noInputZones, g.TransformedRetryButtonRect())
		}

		g.InputHandler.NoInputZones = noInputZones
	}
	g.InputHandler.Update(g.board, g.TransformedBoardRect(), g.GameState)

	gi := g.InputHandler.GetGameInput()

	if !g.DisableZoomAndPanControl {
		g.Zoom, g.Offset = g.InputHandler.GetZoomAndOffset(g.Zoom, g.Offset, FRectangleCenter(g.TransformedBoardRect()))
	}

	if (g.InputHandler.IsPinching() || g.InputHandler.IsDragging()) && !g.DisableZoomAndPanControl {
		SetRedraw()
	}

	if g.DoingZoomAnimation {
		SetRedraw()
	}

	// =================================
	// update flag tutorial
	// =================================
	g.FlagTutorial.Update(
		g.board,
		g.TransformedBoardRect(),
		g.MaxRect,
	)

	// =================================
	// handle board interaction
	// =================================

	// =======================================
	prevState := g.GameState
	g.board.SaveTo(g.prevBoard)

	var needToCheckStateChange bool = false

	// true if board or game state has changed
	var stateChanged bool = false

	var interaction BoardInteractionType = InteractionTypeNone
	// =======================================

	if g.GameState == GameStatePlaying && gi.Type != InputTypeNone {
		if gi.Type == InputTypeCheck {
			interaction = InteractionTypeCheck
		} else if gi.Type == InputTypeFlag {
			interaction = InteractionTypeFlag
		} else if gi.Type == InputTypeStep {
			interaction = InteractionTypeStep
		}

		if interaction != InteractionTypeNone {
			g.GameState = g.board.InteractAt(
				gi.BoardX, gi.BoardY, interaction, g.GameState, g.mineCount, g.Seed)

			needToCheckStateChange = true
		}
	}

	// ======================================
	// changing board for debugging purpose
	// ======================================
	if IsKeyJustPressed(SetToDecoBoardKey) && IsDevVersion {
		g.SetDebugBoardForDecoration()
		needToCheckStateChange = true
	}
	if IsKeyJustPressed(InstantWinKey) && IsDevVersion {
		g.SetBoardForInstantWin()
		needToCheckStateChange = true
	}

	// ==============================
	// check if state has changed
	// ==============================
	if needToCheckStateChange {
		// first check game state
		stateChanged = prevState != g.GameState

		// then check board state
		iter := NewBoardIterator(0, 0, g.board.Width-1, g.board.Height-1)

		for iter.HasNext() {
			x, y := iter.GetNext()

			if g.board.Mines.Get(x, y) != g.prevBoard.Mines.Get(x, y) {
				stateChanged = true
				break
			}

			if g.board.Flags.Get(x, y) != g.prevBoard.Flags.Get(x, y) {
				stateChanged = true
				break
			}

			if g.board.Revealed.Get(x, y) != g.prevBoard.Revealed.Get(x, y) {
				stateChanged = true
				break
			}
		}
	}

	// ==============================
	// on state changes
	// ==============================
	// skipping animations
	if prevState == GameStateLost || prevState == GameStateWon {
		// all animations are skippable except AnimationTagRetryButtonReveal
		pressedAny := IsMouseButtonJustPressed(eb.MouseButtonLeft)
		pressedAny = pressedAny || IsMouseButtonJustPressed(eb.MouseButtonRight)
		pressedAny = pressedAny || IsMouseButtonJustPressed(eb.MouseButtonMiddle)
		pressedAny = pressedAny || IsTouchJustPressed(FRectWH(ScreenWidth, ScreenHeight), nil)
		if pressedAny {
			g.SkipAllAnimationsUntilTag(AnimationTagRetryButtonReveal)
		}
	}

	// ======================
	var newTilesRevealed = false
	var newFlagsPlanted = false
	// ======================

	if stateChanged {
		g.SkipAllAnimations()

		SetRedraw() // just do it!!

		iter := NewBoardIterator(0, 0, g.board.Width-1, g.board.Height-1)

		// update flag
		iter.Reset()
		for iter.HasNext() {
			x, y := iter.GetNext()
			if g.prevBoard.Flags.Get(x, y) != g.board.Flags.Get(x, y) {
				newFlagsPlanted = true
				if g.board.Flags.Get(x, y) {
					g.QueueAddFlagAnimation(x, y)
				} else {
					g.QueueRemoveFlagAnimation(x, y)
				}
			}
		}

		// check if we board has been revealed
		iter.Reset()
		for iter.HasNext() {
			x, y := iter.GetNext()
			if g.board.Revealed.Get(x, y) && !g.prevBoard.Revealed.Get(x, y) {
				// on reveal
				newTilesRevealed = true
				g.QueueRevealAnimation(
					g.prevBoard.Revealed, g.board.Revealed,
					Clamp(gi.BoardX, 0, g.board.Width-1),
					Clamp(gi.BoardY, 0, g.board.Height-1),
				)
				break
			}
		}

		if prevState != g.GameState {
			if g.GameState == GameStateLost { // on loss
				g.QueueDefeatAnimation(gi.BoardX, gi.BoardY)
			} else if g.GameState == GameStateWon { // on win
				g.QueueWinAnimation(gi.BoardX, gi.BoardY)
			}
		}

		// call OnFirstInteraction
		if !g.hadInteraction {
			g.hadInteraction = true
			if g.OnFirstInteraction != nil {
				g.OnFirstInteraction()
			}
		}

		// call OnGameEnd
		if g.GameState != prevState && (g.GameState == GameStateWon || g.GameState == GameStateLost) {
			if g.OnGameEnd != nil {
				g.OnGameEnd(g.GameState == GameStateWon)
			}
		}
	}

	if interaction != InteractionTypeNone {
		if !stateChanged { // user wanted to do something but nothing happened
			// pass
		} else { // something did happened
			// pass
		}
	}

	// ============================================
	// check if we need to show flag tutorial
	// ============================================
	if newTilesRevealed && gi.ByTouch {
		g.revealdTilesUsingTouch = true
	}
	if newFlagsPlanted {
		g.plantedFlag = true
	}

	if g.revealdTilesUsingTouch &&
		!g.plantedFlag &&
		gi.Type == InputTypeNone &&
		g.GameState == GameStatePlaying {
		g.FlagTutorial.ShowFlagTutorial = true
	} else {
		g.FlagTutorial.ShowFlagTutorial = false
	}

	// ============================
	// update animations
	// ============================

	// update GameAnimations
	AnimationQueueUpdate(&g.GameAnimations)

	// update BaseTileStyles
	for x := range g.board.Width {
		for y := range g.board.Height {
			AnimationQueueUpdate(g.TileAnimations.Get(x, y))
		}
	}

	// copy it over to RenderTileStyles
	for x := range g.board.Width {
		for y := range g.board.Height {
			g.RenderTileStyles.Set(x, y, g.BaseTileStyles.Get(x, y))
		}
	}

	// if animations queue is not empty, we need to redraw
	if !g.GameAnimations.IsEmpty() {
		SetRedraw()
	}
	{
	REDRAW_CHECK_LOOP:
		for x := range g.board.Width {
			for y := range g.board.Height {
				if !g.TileAnimations.Get(x, y).IsEmpty() {
					SetRedraw()
					break REDRAW_CHECK_LOOP
				}
			}
		}
	}

	// ===================================
	// apply style modifiers
	// ===================================
	for i := 0; i < len(g.StyleModifiers); i++ {
		doRedraw := g.StyleModifiers[i](
			g.prevBoard, g.board,
			g.TransformedBoardRect(),
			interaction,
			stateChanged,
			prevState, g.GameState,
			g.RenderTileStyles,
			gi,
		)

		if doRedraw {
			SetRedraw()
		}
	}

	// =================================
	// update particles
	// =================================
	{
		tc := TileParticleUnitConverter{
			BoardWidth: g.board.Width, BoardHeight: g.board.Height,
			BoardRect: g.TransformedBoardRect(),
		}

		foundAlive := false

		for i, p := range g.Particles {
			if !p.Dead {
				foundAlive = true

				p.OffsetX += p.VelocityX
				p.OffsetY += p.VelocityY

				p.VelocityX += p.GravityX
				p.VelocityY += p.GravityY

				p.Rotation += p.RotVelocity

				p.Timer.TickUp()

				// we consider it dead if it's below screen
				// TODO : I'm half assing this check
				// be more thorough if it ever becomes a problem
				whMax := tc.ToPx(max(p.Width, p.Height))
				_, sy := tc.OffsetToScreen(p)
				if sy > ScreenHeight+whMax {
					p.Dead = true
				}

				g.Particles[i] = p
			} else {
				continue
			}
		}

		if foundAlive {
			SetRedraw()
		}
	}

	// ===================================
	// update RetryButton
	// ===================================
	g.RetryButton.NoInputZone = g.noInputZone

	if !g.DrawRetryButton {
		g.retryButtonSizeRelative = g.retryButtonSize / min(g.Rect.Dx(), g.Rect.Dy())
	}

	g.RetryButton.Rect = g.TransformedRetryButtonRect()
	if !g.DrawRetryButton {
		g.RetryButton.Disabled = true
	}
	g.RetryButton.Update()

	if IsKeyJustPressed(ResetBoardKey) && IsDevVersion {
		g.ResetBoard()
		SetRedraw()
	}
	if IsKeyJustPressed(ResetToSameBoardKey) && IsDevVersion {
		g.ResetBoardEx(false)
		SetRedraw()
	}
}

func (g *Game) Draw(dst *eb.Image) {
	// background
	dst.Fill(TheColorTable[ColorBg])

	doWaterEffect := g.GameState == GameStateWon

	if doWaterEffect {
		SetRedraw()
	}

	DrawBoard(
		dst,

		g.board.Width, g.board.Height,
		g.TransformedBoardRect(),
		g.RenderTileStyles,

		doWaterEffect, g.WaterAlpha, g.WaterFlowOffset,

		(g.InputHandler.IsPinching() && !g.DisableZoomAndPanControl) || g.DoingZoomAnimation,
	)

	if g.DrawRetryButton {
		g.RetryButton.DoWaterEffect = doWaterEffect
		g.RetryButton.WaterAlpha = g.WaterAlpha
		g.RetryButton.WaterFlowOffset = g.WaterFlowOffset

		g.RetryButton.Draw(dst)
	}

	DrawParticles(
		dst,
		g.Particles,
		g.board.Width, g.board.Height,
		g.TransformedBoardRect(),
	)

	g.InputHandler.Draw(dst)

	g.FlagTutorial.Draw(
		dst,
		g.board,
		g.TransformedBoardRect(),
	)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) {
}

func (g *Game) MineCount() int {
	return g.mineCount
}

func (g *Game) FlagCount() int {
	flagCount := 0
	for x := range g.board.Width {
		for y := range g.board.Height {
			if g.board.Flags.Get(x, y) {
				flagCount++
			}
		}
	}

	return flagCount
}

func (g *Game) BoardTileCount() (int, int) {
	return g.board.Width, g.board.Height
}

func (g *Game) HadInteraction() bool {
	return g.hadInteraction
}

func (g *Game) NoInputZone() FRectangle {
	return g.noInputZone
}

func (g *Game) SetNoInputZone(rect FRectangle) {
	g.noInputZone = rect
}

func (g *Game) TransformedBoardRect() FRectangle {
	rect := g.Rect
	rect = FRectScaleCentered(rect, g.Zoom, g.Zoom)
	rect = rect.Add(FPt(g.Offset.X, g.Offset.Y))

	return rect
}

func isTileFirmlyPlaced(style TileStyle) bool {
	const e = 0.08
	return style.DrawTile &&
		CloseToEx(style.TileScale, 1, e) &&
		CloseToEx(style.TileOffsetX, 0, e) &&
		CloseToEx(style.TileOffsetY, 0, e) &&
		style.TileAlpha > e
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

func GetBgFillColor(boardWidth, boardHeight, x, y int) color.Color {
	if IsOddTile(boardWidth, boardHeight, x, y) {
		return ColorTileNormal2
	} else {
		return ColorTileNormal1
	}
}

func GetTileFillColor(boardWidth, boardHeight, x, y int) color.Color {
	if IsOddTile(boardWidth, boardHeight, x, y) {
		return ColorTileRevealed2
	} else {
		return ColorTileRevealed1
	}
}

func GetAnimationTargetTileStyle(board Board, x, y int) TileStyle {
	style := NewTileStyle()

	style.DrawBg = true
	style.BgFillColor = GetBgFillColor(board.Width, board.Height, x, y)

	if board.IsPosInBoard(x, y) {
		if board.Revealed.Get(x, y) {
			style.DrawTile = true
			style.DrawBg = false

			style.TileFillColor = GetTileFillColor(board.Width, board.Height, x, y)
			style.TileStrokeColor = ColorTileRevealedStroke

			count := board.GetNeighborMineCount(x, y)

			if 1 <= count && count <= 8 {
				style.DrawFg = true
				style.FgType = TileFgTypeNumber
				style.FgColor = ColorTableGetNumber(count)
				style.FgNumber = count
			}
		}

		if board.Flags.Get(x, y) {
			style.FgType = TileFgTypeFlag
			style.FgColor = ColorFlag
		}
	}

	return style
}

type VIBuffer struct {
	Vertices []eb.Vertex
	Indices  []uint16
}

func NewVIBuffer(vertCap int, indexCap int) *VIBuffer {
	vi := new(VIBuffer)

	vi.Vertices = make([]eb.Vertex, 0, vertCap)
	vi.Indices = make([]uint16, 0, indexCap)

	return vi
}

func (vi *VIBuffer) Reset() {
	vi.Vertices = vi.Vertices[:0]
	vi.Indices = vi.Indices[:0]
}

// assumes you will use WhiteImage
// so it will set SrcX and SrcY to 1
func VIaddRect(buffer *VIBuffer, rect FRectangle, clr color.Color) {
	indexStart := uint16(len(buffer.Vertices))

	r, g, b, a := clr.RGBA()

	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff
	af := float32(a) / 0xffff

	buffer.Vertices = append(
		buffer.Vertices,
		eb.Vertex{
			SrcX: 1, SrcY: 1, DstX: f32(rect.Min.X), DstY: f32(rect.Min.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
		eb.Vertex{
			SrcX: 1, SrcY: 1, DstX: f32(rect.Max.X), DstY: f32(rect.Min.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
		eb.Vertex{
			SrcX: 1, SrcY: 1, DstX: f32(rect.Max.X), DstY: f32(rect.Max.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
		eb.Vertex{
			SrcX: 1, SrcY: 1, DstX: f32(rect.Min.X), DstY: f32(rect.Max.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
	)

	buffer.Indices = append(
		buffer.Indices,
		indexStart+0, indexStart+1, indexStart+2, indexStart+0, indexStart+2, indexStart+3,
	)
}

// assumes you will use WhiteImage
// so it will set SrcX and SrcY to 1
func VIaddRoundRectEx(
	buffer *VIBuffer,
	rect FRectangle,
	radiuses [4]float64,
	radiusInPixels bool,
	segments [4]int,
	clr color.Color,
) {
	buffer.Vertices, buffer.Indices = AddRoundRectVerts(
		buffer.Vertices, buffer.Indices,
		rect,
		radiuses,
		radiusInPixels,
		segments,
		clr,
	)
}

// assumes you will use WhiteImage
// so it will set SrcX and SrcY to 1
func VIaddRoundRect(
	buffer *VIBuffer,
	rect FRectangle,
	radius float64,
	radiusInPixels bool,
	segments int,
	clr color.Color,
) {
	buffer.Vertices, buffer.Indices = AddRoundRectVerts(
		buffer.Vertices, buffer.Indices,
		rect,
		[4]float64{radius, radius, radius, radius},
		radiusInPixels,
		[4]int{segments, segments, segments, segments},
		clr,
	)
}

// assumes you will use TileImage
func VIaddRoundTile(
	buffer *VIBuffer,
	rect FRectangle,
	isRound [4]bool,
	clr color.Color,
) {
	var verts [4]eb.Vertex

	sv, turns := GetRoundTile(isRound)

	verts[0].DstX, verts[0].DstY = f32(rect.Min.X), f32(rect.Min.Y)
	verts[1].DstX, verts[1].DstY = f32(rect.Max.X), f32(rect.Min.Y)
	verts[2].DstX, verts[2].DstY = f32(rect.Max.X), f32(rect.Max.Y)
	verts[3].DstX, verts[3].DstY = f32(rect.Min.X), f32(rect.Max.Y)

	var srcX [4]float32
	var srcY [4]float32

	srcX[0], srcY[0] = f32(sv.Rect.Min.X), f32(sv.Rect.Min.Y)
	srcX[1], srcY[1] = f32(sv.Rect.Max.X), f32(sv.Rect.Min.Y)
	srcX[2], srcY[2] = f32(sv.Rect.Max.X), f32(sv.Rect.Max.Y)
	srcX[3], srcY[3] = f32(sv.Rect.Min.X), f32(sv.Rect.Max.Y)

	for i := range 4 {
		srcI := (i + turns*3) % 4
		verts[i].SrcX, verts[i].SrcY = srcX[srcI], srcY[srcI]
	}

	r, g, b, a := clr.RGBA()

	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff
	af := float32(a) / 0xffff

	for i := range 4 {
		verts[i].ColorR = rf
		verts[i].ColorG = gf
		verts[i].ColorB = bf
		verts[i].ColorA = af
	}

	indexStart := uint16(len(buffer.Vertices))

	buffer.Vertices = append(buffer.Vertices, verts[:]...)
	buffer.Indices = append(
		buffer.Indices,
		indexStart+0, indexStart+1, indexStart+2, indexStart+0, indexStart+2, indexStart+3,
	)
}

// assumes you will use TileImage
func viAddTileWithoutRotImpl(
	buffer *VIBuffer,
	sv SubView,
	rect FRectangle,
	clr color.Color,
) {
	indexStart := uint16(len(buffer.Vertices))

	r, g, b, a := clr.RGBA()

	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff
	af := float32(a) / 0xffff

	buffer.Vertices = append(
		buffer.Vertices,
		eb.Vertex{
			SrcX: f32(sv.Rect.Min.X), SrcY: f32(sv.Rect.Min.Y),
			DstX: f32(rect.Min.X), DstY: f32(rect.Min.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
		eb.Vertex{
			SrcX: f32(sv.Rect.Max.X), SrcY: f32(sv.Rect.Min.Y),
			DstX: f32(rect.Max.X), DstY: f32(rect.Min.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
		eb.Vertex{
			SrcX: f32(sv.Rect.Max.X), SrcY: f32(sv.Rect.Max.Y),
			DstX: f32(rect.Max.X), DstY: f32(rect.Max.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
		eb.Vertex{
			SrcX: f32(sv.Rect.Min.X), SrcY: f32(sv.Rect.Max.Y),
			DstX: f32(rect.Min.X), DstY: f32(rect.Max.Y),
			ColorR: rf, ColorG: gf, ColorB: bf, ColorA: af,
		},
	)

	buffer.Indices = append(
		buffer.Indices,
		indexStart+0, indexStart+1, indexStart+2, indexStart+0, indexStart+2, indexStart+3,
	)
}

// assumes you will use TileImage
func VIaddAllRoundTile(
	buffer *VIBuffer,
	rect FRectangle,
	clr color.Color,
) {
	viAddTileWithoutRotImpl(
		buffer, GetAllRoundTile(), rect, clr,
	)
}

// assumes you will use TileImage
func VIaddRectTile(
	buffer *VIBuffer,
	rect FRectangle,
	clr color.Color,
) {
	viAddTileWithoutRotImpl(
		buffer, GetRectTile(), rect, clr,
	)
}

// assumes you will use WhiteImage
// so it will set SrcX and SrcY to 1
func VIaddFillPath(buffer *VIBuffer, path ebv.Path, clr color.Color) {
	vPrevLen := len(buffer.Vertices)

	buffer.Vertices, buffer.Indices = path.AppendVerticesAndIndicesForFilling(buffer.Vertices, buffer.Indices)

	r, g, b, a := clr.RGBA()

	rf := float32(r) / 0xffff
	gf := float32(g) / 0xffff
	bf := float32(b) / 0xffff
	af := float32(a) / 0xffff

	for i := vPrevLen; i < len(buffer.Vertices); i++ {
		buffer.Vertices[i].SrcX = 1
		buffer.Vertices[i].SrcY = 1
		buffer.Vertices[i].ColorR = rf
		buffer.Vertices[i].ColorG = gf
		buffer.Vertices[i].ColorB = bf
		buffer.Vertices[i].ColorA = af
	}
}

func VIaddSubView(buffer *VIBuffer, sv SubView, options *DrawSubViewOptions) {
	rect := sv.Rect
	rect0 := FRectMoveTo(rect, 0, 0)

	var vs [4]FPoint

	vs[0] = FPt(rect0.Min.X, rect0.Min.Y)
	vs[1] = FPt(rect0.Max.X, rect0.Min.Y)
	vs[2] = FPt(rect0.Max.X, rect0.Max.Y)
	vs[3] = FPt(rect0.Min.X, rect0.Max.Y)

	var xformed [4]FPoint

	xformed[0] = FPointTransform(vs[0], options.GeoM)
	xformed[1] = FPointTransform(vs[1], options.GeoM)
	xformed[2] = FPointTransform(vs[2], options.GeoM)
	xformed[3] = FPointTransform(vs[3], options.GeoM)

	var verts [4]eb.Vertex
	var indices [6]uint16

	verts[0] = eb.Vertex{
		DstX: f32(xformed[0].X), DstY: f32(xformed[0].Y),
		SrcX: f32(rect.Min.X), SrcY: f32(rect.Min.Y),
	}
	verts[1] = eb.Vertex{
		DstX: f32(xformed[1].X), DstY: f32(xformed[1].Y),
		SrcX: f32(rect.Max.X), SrcY: f32(rect.Min.Y),
	}
	verts[2] = eb.Vertex{
		DstX: f32(xformed[2].X), DstY: f32(xformed[2].Y),
		SrcX: f32(rect.Max.X), SrcY: f32(rect.Max.Y),
	}
	verts[3] = eb.Vertex{
		DstX: f32(xformed[3].X), DstY: f32(xformed[3].Y),
		SrcX: f32(rect.Min.X), SrcY: f32(rect.Max.Y),
	}

	rf := options.ColorScale.R()
	gf := options.ColorScale.G()
	bf := options.ColorScale.B()
	af := options.ColorScale.A()

	for i := range 4 {
		verts[i].ColorR = rf
		verts[i].ColorG = gf
		verts[i].ColorB = bf
		verts[i].ColorA = af
	}

	indexStart := uint16(len(buffer.Vertices))

	indices = [6]uint16{
		indexStart + 0, indexStart + 1, indexStart + 2, indexStart + 0, indexStart + 2, indexStart + 3,
	}

	buffer.Vertices = append(buffer.Vertices, verts[:]...)
	buffer.Indices = append(buffer.Indices, indices[:]...)
}

func VIaddSubViewInRect(
	buffer *VIBuffer,
	rect FRectangle,
	scale float64,
	offsetX, offsetY float64,
	clr color.Color,
	view SubView,
) {
	imgSize := view.Rect.Size()
	rectSize := rect.Size()

	drawScale := GetScaleToFitRectInRect(imgSize.X, imgSize.Y, rectSize.X, rectSize.Y)
	drawScale *= scale

	op := &DrawSubViewOptions{}
	op.GeoM.Concat(TransformToCenter(imgSize.X, imgSize.Y, drawScale, drawScale, 0))
	rectCenter := FRectangleCenter(rect)
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)
	op.GeoM.Translate(offsetX, offsetY)
	op.ColorScale.ScaleWithColor(clr)

	VIaddSubView(buffer, view, op)
}

var DBC = struct { // DrawBoard cache
	VIBuffers [2]*VIBuffer

	ShouldDrawBgTile Array2D[bool]
	ShouldDrawTile   Array2D[bool]
	ShouldDrawFgTile Array2D[bool]

	BgTileRects Array2D[FRectangle]

	TileStrokeRects Array2D[FRectangle]
	TileFillRects   Array2D[FRectangle]

	TileFirmlyPlaced Array2D[bool]
	TileRoundness    Array2D[[4]bool]

	WaterRenderTarget *eb.Image

	FixedSizeFace     *ebt.GoTextFace
	FixedSizeFaceSize float64
}{}

func init() {
	// NOTE : hard coded number based on previous run
	DBC.VIBuffers[0] = NewVIBuffer(4096, 4096)
	DBC.VIBuffers[1] = NewVIBuffer(2048, 2048)
}

func DrawBoard(
	dst *eb.Image,

	boardWidth, boardHeight int,
	boardRect FRectangle,
	tileStyles Array2D[TileStyle],

	// params for water effect
	doWaterEffect bool,
	waterAlpha float64,
	waterFlowOffset time.Duration,

	zoomingInOut bool,
) {
	// TODO: don't draw stuff that are outside the screen
	modColor := func(
		c color.Color,
		alpha float64,
		highlight float64,
		highlightColor color.Color,
	) color.Color {
		faded := ColorFade(c, alpha)
		hlFaded := ColorFade(highlightColor, highlight)

		r1, g1, b1, a1 := faded.RGBA()
		r2, g2, b2, a2 := hlFaded.RGBA()

		r1, g1, b1, a1 = r1>>8, g1>>8, b1>>8, a1>>8
		r2, g2, b2, a2 = r2>>8, g2>>8, b2>>8, a2>>8

		r3 := r2 + (r1*(255-a2))*255/(255*255)
		g3 := g2 + (g1*(255-a2))*255/(255*255)
		b3 := b2 + (b1*(255-a2))*255/(255*255)
		a3 := a2 + (a1*(255-a2))*255/(255*255)

		return color.RGBA{
			uint8(r3), uint8(g3), uint8(b3), uint8(a3),
		}
	}

	iter := NewBoardIterator(0, 0, boardWidth-1, boardHeight-1)

	// ======================
	// create WaterRenderTarget
	// ======================
	{
		recreateRenderTarget := DBC.WaterRenderTarget == nil
		recreateRenderTarget = recreateRenderTarget || DBC.WaterRenderTarget.Bounds().Dx() != int(ScreenWidth)
		recreateRenderTarget = recreateRenderTarget || DBC.WaterRenderTarget.Bounds().Dy() != int(ScreenHeight)

		// WaterRenderTarget
		if recreateRenderTarget {
			if DBC.WaterRenderTarget != nil {
				DBC.WaterRenderTarget.Dispose()
			}
			DBC.WaterRenderTarget = eb.NewImageWithOptions(
				RectWH(int(ScreenWidth), int(ScreenHeight)),
				&eb.NewImageOptions{Unmanaged: true},
			)
		}
	}

	// ======================
	// create FixedSizeFace
	// ======================
	if DBC.FixedSizeFace == nil {
		DBC.FixedSizeFaceSize = 128
		DBC.FixedSizeFace = &ebt.GoTextFace{
			Source: FaceSource,
			Size:   DBC.FixedSizeFaceSize,
		}
		DBC.FixedSizeFace.SetVariation(ebt.MustParseTag("wght"), 600)
	}

	// ======================
	// reset VIBuffers
	// ======================
	for i := range DBC.VIBuffers {
		DBC.VIBuffers[i].Reset()
	}

	// ===============================
	// resize cache
	// ===============================
	DBC.ShouldDrawBgTile.Resize(boardWidth, boardHeight)
	DBC.ShouldDrawTile.Resize(boardWidth, boardHeight)
	DBC.ShouldDrawFgTile.Resize(boardWidth, boardHeight)

	DBC.BgTileRects.Resize(boardWidth, boardHeight)

	DBC.TileStrokeRects.Resize(boardWidth, boardHeight)
	DBC.TileFillRects.Resize(boardWidth, boardHeight)

	DBC.TileRoundness.Resize(boardWidth, boardHeight)
	DBC.TileFirmlyPlaced.Resize(boardWidth, boardHeight)

	// ===============================
	// recalculate cache
	// ===============================
	{
		tileSizeW, tileSizeH := GetBoardTileSize(boardRect, boardWidth, boardHeight)

		for iter.HasNext() {
			x, y := iter.GetNext()

			ogTileRect := GetBoardTileRect(boardRect, boardWidth, boardHeight, x, y)
			style := tileStyles.Get(x, y)

			DBC.ShouldDrawBgTile.Set(x, y, ShouldDrawBgTile(style))
			DBC.ShouldDrawTile.Set(x, y, ShouldDrawTile(style))
			DBC.ShouldDrawFgTile.Set(x, y, ShouldDrawFgTile(style))

			// BgTileRects
			if DBC.ShouldDrawBgTile.Get(x, y) {
				// draw background tile
				bgTileRect := ogTileRect
				bgTileRect = bgTileRect.Add(FPt(style.BgOffsetX, style.BgOffsetY))
				bgTileRect = FRectScaleCentered(bgTileRect, style.BgScale, style.BgScale)

				DBC.BgTileRects.Set(x, y, bgTileRect)
			}

			// TileFillRects
			// TileStrokeRects
			if DBC.ShouldDrawTile.Get(x, y) || DBC.ShouldDrawFgTile.Get(x, y) {
				tileRect := ogTileRect

				tileRect = tileRect.Add(FPt(style.TileOffsetX, style.TileOffsetY))
				tileRect = FRectScaleCentered(tileRect, style.TileScale, style.TileScale)

				tileInset := math.Round(min(tileSizeW, tileSizeH) * 0.065)
				tileOffsetY := math.Round(min(tileSizeW, tileSizeH) * 0.015)

				tileInset = max(tileInset, 2)
				tileOffsetY = max(tileOffsetY, 1)

				strokeRect := tileRect.Inset(-tileInset)
				fillRect := tileRect.Add(FPt(0, -tileOffsetY))

				DBC.TileStrokeRects.Set(x, y, strokeRect)
				DBC.TileFillRects.Set(x, y, fillRect)
			}

			// TileFirmlyPlaced
			{
				DBC.TileFirmlyPlaced.Set(x, y, isTileFirmlyPlaced(style))
			}

			// TileRoundness
			{
				isRound := [4]bool{
					true,
					true,
					true,
					true,
				}

				if DBC.TileFirmlyPlaced.Get(x, y) {
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
							if DBC.TileFirmlyPlaced.Get(rx, ry) {
								isRound[i] = false
								isRound[(i+1)%4] = false
							}
						}
					}
				}

				DBC.TileRoundness.Set(x, y, isRound)
			}
		}
	}

	shapeBuf := DBC.VIBuffers[0]
	spriteBuf := DBC.VIBuffers[1]

	const segments = 6

	// ============================
	// draw background tiles
	// ============================
	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		style := tileStyles.Get(x, y)

		if !DBC.ShouldDrawBgTile.Get(x, y) {
			continue
		}

		bgTileRect := DBC.BgTileRects.Get(x, y)

		VIaddRectTile(
			shapeBuf,
			bgTileRect,
			modColor(style.BgFillColor, style.BgAlpha, style.Highlight, ColorBgHighLight),
		)

		// draw bomb animation
		if style.BgBombAnim > 0 {
			t := Clamp(style.BgBombAnim, 0, 1)

			tileW := bgTileRect.Dx()
			tileH := bgTileRect.Dy()

			outerMargin := min(tileW, tileH) * 0.04
			innerMargin := min(tileW, tileH) * 0.06

			outerMargin = max(outerMargin, 1)
			innerMargin = max(innerMargin, 1)

			outerRect := bgTileRect.Inset(outerMargin)
			outerRect = outerRect.Inset(min(outerRect.Dx(), outerRect.Dy()) * 0.5 * (1 - t))
			innerRect := outerRect.Inset(innerMargin)

			innerRect = innerRect.Add(FPt(0, innerMargin))

			VIaddAllRoundTile(
				shapeBuf,
				outerRect,
				modColor(ColorMineBg1, style.BgAlpha, style.Highlight, ColorBgHighLight),
			)
			VIaddAllRoundTile(
				shapeBuf,
				innerRect,
				modColor(ColorMineBg2, style.BgAlpha, style.Highlight, ColorBgHighLight),
			)

			VIaddSubViewInRect(
				spriteBuf,
				innerRect,
				1,
				0, 0,
				modColor(ColorMine, style.BgAlpha, style.Highlight, ColorBgHighLight),
				GetMineTile(),
			)
		}
	}

	// ============================
	// draw tiles
	// ============================
	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		style := tileStyles.Get(x, y)

		if !DBC.ShouldDrawTile.Get(x, y) {
			continue
		}

		strokeColor := ColorFade(style.TileStrokeColor, style.TileAlpha)

		VIaddRoundTile(
			shapeBuf,
			DBC.TileStrokeRects.Get(x, y),
			DBC.TileRoundness.Get(x, y),
			strokeColor,
		)
	}

	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		style := tileStyles.Get(x, y)

		if !DBC.ShouldDrawTile.Get(x, y) {
			continue
		}

		fillColor := modColor(
			style.TileFillColor,
			style.TileAlpha, style.Highlight, ColorTileHighLight,
		)

		VIaddRoundTile(
			shapeBuf,
			DBC.TileFillRects.Get(x, y),
			DBC.TileRoundness.Get(x, y),
			fillColor,
		)
	}

	// ============================
	// draw foreground tiles
	// ============================
	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		style := tileStyles.Get(x, y)

		if !DBC.ShouldDrawFgTile.Get(x, y) {
			continue
		}

		if style.FgType != TileFgTypeFlag {
			continue
		}

		fgRect := DBC.TileFillRects.Get(x, y)
		fgColor := style.FgColor

		VIaddSubViewInRect(
			spriteBuf,
			fgRect,
			style.FgScale,
			style.FgOffsetX, style.FgOffsetY,
			modColor(fgColor, style.FgAlpha, style.Highlight, ColorFgHighLight),
			GetFlagTile(style.FgFlagAnim),
		)
	}

	// ====================
	// flush buffers
	// ====================

	// flush shapes
	shapesRenderTarget := dst

	if DBC.WaterRenderTarget != nil && doWaterEffect {
		DBC.WaterRenderTarget.Clear()
		shapesRenderTarget = DBC.WaterRenderTarget
	}

	BeginAntiAlias(false)
	BeginFilter(eb.FilterLinear)
	BeginMipMap(false)
	{
		op := &DrawTrianglesOptions{}
		op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
		DrawTriangles(shapesRenderTarget, shapeBuf.Vertices, shapeBuf.Indices, TileSprite.Image, op)
	}
	EndMipMap()
	EndFilter()
	EndAntiAlias()

	// draw water effect
	if doWaterEffect && DBC.WaterRenderTarget != nil {
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
			DBC.WaterRenderTarget,
			rect,
			GlobalTimerNow()+waterFlowOffset,
			colors,
			FPt(0, 0),
		)
		EndBlend()

		// draw waterRenderTarget
		BeginAntiAlias(false)
		BeginFilter(eb.FilterNearest)
		BeginMipMap(false)
		DrawImage(dst, DBC.WaterRenderTarget, nil)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	}

	// flush sprites
	{
		BeginAntiAlias(false)
		BeginFilter(eb.FilterLinear)
		BeginMipMap(false)
		op := &DrawTrianglesOptions{}
		op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
		DrawTriangles(dst, spriteBuf.Vertices, spriteBuf.Indices, TileSprite.Image, op)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	}

	/*
		{
			DebugPrint("Vertices 0", len(DBC.VIBuffers[0].Vertices))
			DebugPrint("Indices  0", len(DBC.VIBuffers[0].Indices))

			DebugPrint("Vertices 1", len(DBC.VIBuffers[1].Vertices))
			DebugPrint("Indices  1", len(DBC.VIBuffers[1].Indices))
		}
	*/

	// draw numbers
	var numberFace *ebt.GoTextFace
	var numberFaceScale float64 = 1

	{
		_, tileSizeH := GetBoardTileSize(boardRect, boardWidth, boardHeight)
		faceSize := tileSizeH * 0.95

		if zoomingInOut {
			numberFace = DBC.FixedSizeFace
			numberFaceScale = faceSize / DBC.FixedSizeFaceSize
		} else {
			numberFace = &ebt.GoTextFace{
				Source: FaceSource,
				Size:   faceSize,
			}
			numberFace.SetVariation(ebt.MustParseTag("wght"), 600)
		}
	}

	BeginAntiAlias(false)
	BeginFilter(eb.FilterNearest)
	BeginMipMap(false)

	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		style := tileStyles.Get(x, y)

		if !DBC.ShouldDrawFgTile.Get(x, y) {
			continue
		}

		fgRect := DBC.TileFillRects.Get(x, y)
		fgScale := style.FgScale * style.TileScale
		fgColor := style.FgColor

		if style.FgType == TileFgTypeNumber {
			count := style.FgNumber
			if 1 <= count && count <= 8 {
				op := &DrawTextOptions{}
				op.PrimaryAlign = ebt.AlignCenter

				center := FRectangleCenter(fgRect)

				scale := numberFaceScale * fgScale

				op.GeoM.Translate(0, -numberFace.Size*0.58)
				op.GeoM.Scale(scale, scale)

				op.GeoM.Translate(
					center.X+style.FgOffsetX, center.Y+style.FgOffsetX,
				)
				op.ColorScale.ScaleWithColor(modColor(fgColor, style.FgAlpha, style.Highlight, ColorFgHighLight))

				DrawText(dst, strconv.Itoa(count), numberFace, op)
			}
		}
	}
	EndMipMap()
	EndAntiAlias()
	EndFilter()
}

func DrawParticles(
	dst *eb.Image,
	particles []TileParticle,
	boardWidth, boardHeight int,
	boardRect FRectangle,
) {
	tc := TileParticleUnitConverter{
		BoardWidth: boardWidth, BoardHeight: boardHeight,
		BoardRect: boardRect,
	}

	for _, p := range particles {
		if p.Dead {
			continue
		}

		svSize := p.SubView.Rect.Size()

		scaleX := tc.ToPx(p.Width) / svSize.X
		scaleY := tc.ToPx(p.Height) / svSize.Y

		screenX, screenY := tc.OffsetToScreen(p)

		t := p.Timer.Normalize()

		if p.ColorLerpFunc != nil {
			t = p.ColorLerpFunc(t)
		}

		color := LerpColorRGBA(p.Color1, p.Color2, t)

		op := &DrawSubViewOptions{}

		op.GeoM.Concat(TransformToCenter(
			p.SubView.Rect.Dx(), p.SubView.Rect.Dy(),
			scaleX, scaleY,
			p.Rotation,
		))
		op.GeoM.Translate(screenX, screenY)

		op.ColorScale.ScaleWithColor(color)

		DrawSubView(dst, p.SubView, op)
	}
}

func NewTileHighlightModifier() StyleModifier {
	var hlWide bool
	var hlX, hlY int

	hlTiles := make([]uint64, 0, 9)
	prevHlTiles := make([]uint64, 0, 9)

	pair := func(a, b uint64) uint64 {
		return (a+b)*(a+b+1)/2 + b
	}

	highlightTile := func(tileStyles Array2D[TileStyle], x, y int) {
		tileStyles.Data[x+tileStyles.Width*y].Highlight = 1

		pairN := pair(u64(x), u64(y))

		for _, v := range hlTiles {
			if v == pairN {
				return
			}
		}

		hlTiles = append(hlTiles, pairN)
	}

	return func(
		prevBoard, board Board,
		boardRect FRectangle,
		interaction BoardInteractionType,
		stateChanged bool, // GameState or board has changed
		prevGameState, gameState GameState,
		tileStyles Array2D[TileStyle], // modify these to change style
		gi GameInput,
	) bool {
		if gameState != GameStatePlaying || gi.Type == InputTypeNone {
			hlTiles = hlTiles[:0]
			prevHlTiles = prevHlTiles[:0]

			return prevGameState != gameState
		}

		prevHlTiles = prevHlTiles[:len(hlTiles)]
		copy(prevHlTiles, hlTiles)

		hlTiles = hlTiles[:0]

		goWide := gi.Type == InputTypeHL || gi.Type == InputTypeCheck
		goWide = goWide && board.IsPosInBoard(gi.BoardX, gi.BoardY)
		goWide = goWide && prevBoard.Revealed.Get(gi.BoardX, gi.BoardY)
		goWide = goWide && !stateChanged

		if goWide {
			hlWide = true
		}

		if !(gi.Type == InputTypeHL || gi.Type == InputTypeCheck) {
			hlWide = false
		}

		hlX = gi.BoardX
		hlY = gi.BoardY

		var iter BoardIterator

		if hlWide {
			iter = NewBoardIterator(hlX-1, hlY-1, hlX+1, hlY+1)
		} else {
			iter = NewBoardIterator(hlX, hlY, hlX, hlY)
		}

		for iter.HasNext() {
			x, y := iter.GetNext()

			if !board.IsPosInBoard(x, y) {
				continue
			}

			if !board.Revealed.Get(x, y) {
				if hlWide {
					if !board.Flags.Get(x, y) {
						highlightTile(tileStyles, x, y)
					}
				} else {
					highlightTile(tileStyles, x, y)
				}
			}
		}

		if board.IsPosInBoard(hlX, hlY) && board.Revealed.Get(hlX, hlY) && board.GetNeighborMineCount(hlX, hlY) > 0 {
			highlightTile(tileStyles, hlX, hlY)
		}

		if len(hlTiles) != len(prevHlTiles) {
			return true
		}

		slices.Sort(hlTiles)
		slices.Sort(prevHlTiles)

		for i := 0; i < len(hlTiles); i++ {
			if hlTiles[i] != prevHlTiles[i] {
				return true
			}
		}

		return false
	}
}

func NewFgClickModifier() StyleModifier {
	const clickTimeDuration = time.Millisecond * 100

	clickTimers := make(map[image.Point]Timer)

	return func(
		prevBoard, board Board,
		boardRect FRectangle,
		interaction BoardInteractionType,
		stateChanged bool,
		prevGameState, gameState GameState,
		tileStyles Array2D[TileStyle],
		gi GameInput,
	) bool {
		prevClickTimers := make(map[image.Point]time.Duration)

		for point, timer := range clickTimers {
			prevClickTimers[point] = timer.Current
		}

		if IsAnyMouseButtonPressed() {
			cursor := CursorFPt()
			boardX, boardY := MousePosToBoardPos(
				boardRect,
				board.Width, board.Height,
				cursor,
			)
			clickTimers[image.Pt(boardX, boardY)] = Timer{
				Current:  clickTimeDuration,
				Duration: clickTimeDuration,
			}
		}

		im := &TheInputManager

		for _, id := range im.TouchingBuf {
			if info, ok := GetTouchInfo(id); ok {
				boardX, boardY := MousePosToBoardPos(
					boardRect,
					board.Width, board.Height,
					info.StartedPos,
				)
				clickTimers[image.Pt(boardX, boardY)] = Timer{
					Current:  clickTimeDuration,
					Duration: clickTimeDuration,
				}
			}
		}

		// remove timers outside the board
		for point := range clickTimers {
			if !board.IsPosInBoard(point.X, point.Y) {
				delete(clickTimers, point)
			}
		}

		// apply tile style
		for point, timer := range clickTimers {
			tileStyles.Data[point.X+point.Y*tileStyles.Width].FgScale *= 1 + timer.Normalize()*0.07
		}

		// decrease timers
		for point, timer := range clickTimers {
			timer.TickDown()
			clickTimers[point] = timer
		}
		// remove timers below zero
		for point, timer := range clickTimers {
			if timer.Current < 0 {
				delete(clickTimers, point)
			}
		}

		redraw := false

		if len(clickTimers) != len(prevClickTimers) {
			redraw = true
		}

		if !redraw {
			for point, timer := range clickTimers {
				t := Clamp(timer.Current, 0, timer.Duration)
				otherT := Clamp(prevClickTimers[point], 0, timer.Duration)
				if t != otherT {
					redraw = true
					break
				}
			}
		}

		return redraw
	}
}

func (g *Game) QueueRevealAnimation(revealsBefore, revealsAfter Array2D[bool], originX, originY int) {
	iter := NewBoardIterator(0, 0, g.board.Width-1, g.board.Height-1)

	getDist := func(x, y int) float64 {
		return FPt(f64(originX), f64(originY)).Sub(FPt(f64(x), f64(y))).Length()
	}

	getDistSquared := func(x, y int) int {
		diffX, diffY := Abs(originX-x), Abs(originY-y)
		return diffX*diffX + diffY*diffY
	}

	var playedAt time.Time
	playSound := func() {
		if g.GameState != GameStatePlaying {
			return
		}
		now := time.Now()
		if now.Sub(playedAt) > time.Millisecond*20 {
			PlaySoundBytes(SeTileReveal, 0.8)
			playedAt = now
		}
	}

	minDist := math.MaxFloat64
	distSquaredMax := 0
	revealedTileCount := 0

	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		if !revealsBefore.Get(x, y) && revealsAfter.Get(x, y) {
			distSquaredMax = max(distSquaredMax, getDistSquared(x, y))
			minDist = min(minDist, getDist(x, y))
			revealedTileCount++
		}
	}

	playedFirstSound := false
	playedLastSound := false

	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()

		if !revealsAfter.Get(x, y) {
			continue
		}

		if revealsBefore.Get(x, y) {
			continue
		}

		distSquared := getDistSquared(x, y)
		dist := getDist(x, y)

		distSubMinDist := dist - minDist

		var timer Timer
		timer.Duration = time.Duration((distSubMinDist) * f64(time.Millisecond) * 20)
		timer.Duration += time.Millisecond * 5
		timer.Current = time.Duration(-distSubMinDist * f64(time.Millisecond) * 20)

		targetStyle := GetAnimationTargetTileStyle(g.board, x, y)

		var anim CallbackAnimation
		anim.Tag = AnimationTagTileReveal

		anim.Update = func() {
			style := g.BaseTileStyles.Get(x, y)
			timer.TickUp()

			t := timer.NormalizeUnclamped()

			if t > 0 {
				if revealedTileCount == 1 {
					if !playedFirstSound {
						playedFirstSound = true
						playSound()
					}
				} else {
					if !playedFirstSound {
						playSound()
						playedFirstSound = true
					}

					if !playedLastSound && distSquared == distSquaredMax {
						playSound()
						playedLastSound = true
					}
				}

				style = targetStyle
				style.DrawBg = true

				t = Clamp(t, 0, 1)

				style.TileScale = BezierCurveDataAsGraph(
					TheBezierTable[BezierTileRevealScale], t)

				style.TileOffsetY = BezierCurveDataAsGraph(
					TheBezierTable[BezierTileRevealOffsetY], t) * 10
			}

			g.BaseTileStyles.Set(x, y, style)
		}

		anim.Skip = func() {
			timer.Current = timer.Duration
			anim.Update()
		}

		anim.Done = func() bool {
			return timer.Current >= timer.Duration
		}

		anim.AfterDone = func() {
			g.BaseTileStyles.Set(x, y, targetStyle)
		}

		g.TileAnimations.Get(x, y).Enqueue(anim)
	}

	// =========================================
	// play tile reveal sound effect repeatedly
	// until animation ends
	//=========================================
	if revealedTileCount >= 3 {
		var gameAnim CallbackAnimation
		gameAnim.Tag = AnimationTagTileReveal

		var repeatSoundTimer Timer

		gameAnim.Update = func() {
			if playedFirstSound && !playedLastSound {
				repeatSoundTimer.TickUp()
				if repeatSoundTimer.Current > time.Millisecond*50 {
					playSound()
					repeatSoundTimer.Current = 0
				}
			}
		}

		gameAnim.Skip = func() {
		}

		gameAnim.Done = func() bool {
			return playedFirstSound && playedLastSound
		}

		g.GameAnimations.Enqueue(gameAnim)
	}
}

func (g *Game) QueueAddFlagAnimation(flagX, flagY int) {
	if !g.playedAddFlagSound {
		PlaySoundBytes(SeFlag, 0.6)
		g.playedAddFlagSound = true
	}

	var timer Timer
	timer.Duration = time.Millisecond * 110

	var anim CallbackAnimation
	anim.Tag = AnimationTagAddFlag

	anim.Update = func() {
		style := g.BaseTileStyles.Get(flagX, flagY)

		style.DrawFg = true
		style.FgType = TileFgTypeFlag
		style.FgColor = ColorFlag

		timer.TickUp()

		t := timer.Normalize()

		style.FgFlagAnim = t

		g.BaseTileStyles.Set(flagX, flagY, style)
	}

	anim.Skip = func() {
		timer.Current = timer.Duration
		anim.Update()
	}

	anim.Done = func() bool {
		return timer.Current >= timer.Duration
	}

	anim.AfterDone = func() {
		style := g.BaseTileStyles.Get(flagX, flagY)

		style.FgFlagAnim = 1

		g.BaseTileStyles.Set(flagX, flagY, style)
	}

	g.TileAnimations.Get(flagX, flagY).Enqueue(anim)
}

func (g *Game) QueueRemoveFlagAnimation(flagX, flagY int) {
	if !g.playedRemoveFlagSound {
		PlaySoundBytes(SeUnflag, 0.8)
		g.playedRemoveFlagSound = true
	}

	var anim CallbackAnimation
	anim.Tag = AnimationTagRemoveFlag

	done := false

	anim.Update = func() {
		style := g.BaseTileStyles.Get(flagX, flagY)

		if style.DrawFg && style.FgType == TileFgTypeFlag {
			style.DrawFg = false
			style.FgType = TileFgTypeNone
			g.BaseTileStyles.Set(flagX, flagY, style)
		}

		velocityX := RandF(0.01, 0.03)
		if rand.IntN(100) > 50 {
			velocityX *= -1
		}

		velocityY := RandF(-0.17, -0.2)

		rotVelocity := RandF(-0.1, 0.1)

		p := TileParticle{
			SubView:       GetFlagTile(1),
			Color1:        ColorFlag,
			Color2:        ColorFade(ColorFlag, 0),
			ColorLerpFunc: EaseInQuint,
			Timer:         Timer{Duration: time.Millisecond * 700},
			BoardX:        flagX, BoardY: flagY,
			Width: 1, Height: 1,
			VelocityX: velocityX, VelocityY: velocityY,
			RotVelocity: rotVelocity,
			GravityX:    0, GravityY: 0.007,
		}

		g.Particles = AppendTileParticle(g.Particles, p)

		done = true
	}

	anim.Skip = func() {
		anim.Update()
	}

	anim.Done = func() bool {
		return done
	}

	g.TileAnimations.Get(flagX, flagY).Enqueue(anim)
}

func (g *Game) GetZoomOutAnimation(tag AnimationTag) CallbackAnimation {
	var timer Timer
	timer.Duration = time.Millisecond * 500

	startZoom := g.Zoom
	startOffset := g.Offset

	var gameAnim CallbackAnimation

	gameAnim.Tag = tag

	gameAnim.Update = func() {
		// during the animation, disable zoom control
		// this will be cleare at AnimationTagRetryButtonReveal
		g.DisableZoomAndPanControl = true

		timer.TickUp()

		doAnimation := timer.Current < timer.Duration
		doAnimation = doAnimation && !(CloseToEx(g.Offset.X, 0, 0.1) && CloseToEx(g.Offset.Y, 0, 0.1))
		doAnimation = doAnimation && !CloseToEx(g.Zoom, 0, 0.01)

		if doAnimation {
			g.DoingZoomAnimation = true

			t := timer.Normalize()
			t = BezierCurveDataAsGraph(TheBezierTable[BezierBoardZoomOut], t)

			g.Zoom = Lerp(startZoom, 1, t)
			g.Offset = FPointLerp(startOffset, FPt(0, 0), t)
		} else {
			g.DoingZoomAnimation = false
			g.Zoom = 1
			g.Offset = FPt(0, 0)
		}
	}

	gameAnim.Skip = func() {
		timer.Current = timer.Duration
		gameAnim.Update()
	}

	gameAnim.Done = func() bool {
		return timer.Current >= timer.Duration
	}

	gameAnim.AfterDone = func() {
		g.Zoom = 1
		g.Offset = FPt(0, 0)
	}

	return gameAnim
}

func (g *Game) QueueDefeatAnimation(originX, originY int) {
	// =================================
	// remove wrongly placed flags
	// =================================
	iter := NewBoardIterator(0, 0, g.board.Width-1, g.board.Height-1)

	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		if g.board.Flags.Get(x, y) && !g.board.Mines.Get(x, y) {
			g.QueueRemoveFlagAnimation(x, y)
		}
	}

	// =================================
	// collect mine positions
	// =================================
	var minePoses []image.Point

	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()
		if g.board.Mines.Get(x, y) && !g.board.Flags.Get(x, y) {
			minePoses = append(minePoses, image.Point{X: x, Y: y})
		}
	}

	if len(minePoses) <= 0 {
		return
	}

	// =================================
	// queue animation where mines are
	// =================================
	getDist := func(pos image.Point) int {
		diffX := originX - pos.X
		diffY := originY - pos.Y
		return diffX*diffX + diffY*diffY
	}

	slices.SortFunc(minePoses, func(a, b image.Point) int {
		distA := getDist(a)
		distB := getDist(b)

		return distA - distB
	})

	var defeatDuration time.Duration

	var offset time.Duration
	const offsetInterval time.Duration = -time.Millisecond * 100

	var useLongerOffset bool = true

	for i := 0; i < len(minePoses); i++ {
		dist := getDist(minePoses[i])
		playSound := true

		for ; i < len(minePoses); i++ {
			p := minePoses[i]
			otherDist := getDist(p)

			if otherDist > dist {
				if useLongerOffset {
					useLongerOffset = false
					offset += offsetInterval * 16 / 6
				} else {
					offset += offsetInterval
				}
				i -= 1
				break
			}

			var timer Timer
			timer.Duration = time.Millisecond * 150
			timer.Current = offset

			// update defeatDuration
			defeatDuration = max(defeatDuration, timer.Duration-timer.Current)

			// add new animation
			var anim CallbackAnimation
			anim.Tag = AnimationTagDefeat

			playSoundCopy := playSound // needed for closure
			playSound = false

			anim.Update = func() {
				style := g.BaseTileStyles.Get(p.X, p.Y)

				if timer.Current > 0 && playSoundCopy {
					playSoundCopy = false
					PlaySoundBytes(SePop, 0.3)
				}

				timer.TickUp()
				style.BgBombAnim = timer.Normalize()

				g.BaseTileStyles.Set(p.X, p.Y, style)
			}

			anim.Skip = func() {
				timer.Current = timer.Duration
				playSoundCopy = false
				anim.Update()
			}

			anim.Done = func() bool {
				return timer.Current >= timer.Duration
			}

			anim.AfterDone = func() {
			}

			g.TileAnimations.Get(p.X, p.Y).Enqueue(anim)
		}
	}

	// =================================
	// after the defeat animaiton
	// queue retry button show animation
	// =================================
	defeatDuration += time.Millisecond * 10

	var defeatAnimTimer Timer
	defeatAnimTimer.Duration = defeatDuration

	var anim CallbackAnimation
	anim.Tag = AnimationTagDefeat

	zoomAnim := g.GetZoomOutAnimation(AnimationTagDefeat)

	anim.Update = func() {
		zoomAnim.Update()
		defeatAnimTimer.TickUp()
	}

	anim.Skip = func() {
		zoomAnim.Skip()
		defeatAnimTimer.Current = defeatAnimTimer.Duration
		anim.Update()
	}

	anim.Done = func() bool {
		return defeatAnimTimer.Current >= defeatAnimTimer.Duration && zoomAnim.Done()
	}

	anim.AfterDone = func() {
		zoomAnim.AfterDone()
		g.QueueRetryButtonAnimation()
	}

	g.GameAnimations.Enqueue(anim)
}

func (g *Game) QueueWinAnimation(originX, originY int) {
	PlaySoundBytes(SeVictory, 0.6)
	fw, fh := f64(g.board.Width), f64(g.board.Height)

	originP := FPt(f64(originX), f64(originY))

	maxDist := math.Sqrt(fw*fw + fh*fh)

	const maxDuration = time.Millisecond * 1000
	const minDuration = time.Millisecond * 50
	const distStartOffset = time.Millisecond * 3

	var winDuration time.Duration

	// queue tile animations
	for x := range g.board.Width {
		for y := range g.board.Height {
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
				style := g.BaseTileStyles.Get(x, y)
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

				if g.board.Revealed.Get(x, y) {
					style.FgColor = LerpColorRGBA(ogFgColor, ColorElementWon, colorT)
				} else {
					style.FgAlpha = 1 - colorT
				}

				style.BgAlpha = 1 - colorT

				g.BaseTileStyles.Set(x, y, style)
			}

			anim.Skip = func() {
				timer.Current = timer.Duration
				anim.Update()
			}

			anim.Done = func() bool {
				return timer.Current >= timer.Duration
			}

			anim.AfterDone = func() {
				style := g.BaseTileStyles.Get(x, y)
				if !g.board.Revealed.Get(x, y) {
					style.DrawFg = false
				}
				style.DrawBg = false
				g.BaseTileStyles.Set(x, y, style)
			}

			g.TileAnimations.Get(x, y).Enqueue(anim)
		}
	}

	// queue game animation
	var winAnimTimer Timer

	winAnimTimer.Duration = winDuration + time.Millisecond*100
	winAnimTimer.Current = 0

	var anim CallbackAnimation
	anim.Tag = AnimationTagWin

	zoomAnim := g.GetZoomOutAnimation(AnimationTagWin)

	anim.Update = func() {
		zoomAnim.Update()

		winAnimTimer.TickUp()

		t := winAnimTimer.Normalize()

		g.WaterAlpha = EaseOutQuint(t)

		waterT := EaseOutQuint(t)

		g.WaterFlowOffset = time.Duration(waterT * f64(time.Second) * 10)
	}

	anim.Skip = func() {
		zoomAnim.Skip()

		winAnimTimer.Current = winAnimTimer.Duration
		anim.Update()
	}

	anim.Done = func() bool {
		return winAnimTimer.Current >= winAnimTimer.Duration && zoomAnim.Done()
	}

	anim.AfterDone = func() {
		zoomAnim.AfterDone()
		g.QueueRetryButtonAnimation()
	}

	g.GameAnimations.Enqueue(anim)
}

func (g *Game) QueueRetryButtonAnimation() {
	buttonRect := g.RetryButtonRect()

	// give 5 pixel margin
	buttonRect = buttonRect.Inset(-5)

	// collect tiles to animate
	toAnimate := make([]image.Point, 0)
	{
		minTileX, minTileY := MousePosToBoardPos(
			g.TransformedBoardRect(),
			g.board.Width, g.board.Height,
			buttonRect.Min,
		)
		minTileX = Clamp(minTileX, 0, g.board.Width-1)
		minTileY = Clamp(minTileY, 0, g.board.Height-1)

		maxTileX, maxTileY := MousePosToBoardPos(
			g.TransformedBoardRect(),
			g.board.Width, g.board.Height,
			buttonRect.Max,
		)
		maxTileX = Clamp(maxTileX, 0, g.board.Width-1)
		maxTileY = Clamp(maxTileY, 0, g.board.Height-1)

		iter := NewBoardIterator(
			minTileX, minTileY, maxTileX, maxTileY,
		)

		for iter.HasNext() {
			x, y := iter.GetNext()
			toAnimate = append(toAnimate, image.Pt(x, y))
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
			style := g.BaseTileStyles.Get(p.X, p.Y)
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

			g.BaseTileStyles.Set(p.X, p.Y, style)
		}

		anim.Skip = func() {
			timer.Current = timer.Duration
			anim.Update()
		}

		anim.Done = func() bool {
			return timer.Current >= timer.Duration
		}

		anim.AfterDone = func() {
			style := g.BaseTileStyles.Get(p.X, p.Y)
			style.DrawBg = false
			style.DrawTile = false
			style.DrawFg = false
			g.BaseTileStyles.Set(p.X, p.Y, style)
		}

		g.TileAnimations.Get(p.X, p.Y).Enqueue(anim)

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

		anim.AfterDone = func() {
			g.DisableZoomAndPanControl = false
		}

		g.GameAnimations.Enqueue(anim)
	}
}

func (g *Game) QueueResetBoardAnimation() {
	fw, fh := f64(g.board.Width), f64(g.board.Height)

	centerP := FPt(f64(g.board.Width-1)*0.5, f64(g.board.Height-1)*0.5)

	maxDist := math.Sqrt(fw*0.5*fw*0.5 + fh*0.5*fh*0.5)

	const minDuration = time.Millisecond * 120
	const maxDuration = time.Millisecond * 400

	var tileAnimationTotal time.Duration

	for x := range g.board.Width {
		for y := range g.board.Height {
			pos := FPt(f64(x), f64(y))
			dist := pos.Sub(centerP).Length()
			d := time.Duration(Lerp(f64(minDuration), f64(maxDuration), 1-dist/maxDist))

			var timer Timer
			timer.Duration = d

			var anim CallbackAnimation
			anim.Tag = AnimationTagHideBoard

			anim.Update = func() {
				style := g.BaseTileStyles.Get(x, y)

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

				g.BaseTileStyles.Set(x, y, style)
			}

			anim.Skip = func() {
				timer.Current = timer.Duration
				anim.Update()
			}

			anim.Done = func() bool {
				return timer.Current >= timer.Duration
			}

			anim.AfterDone = func() {
				style := g.BaseTileStyles.Get(x, y)
				style.DrawBg = false
				style.DrawTile = false
				style.DrawFg = false
				g.BaseTileStyles.Set(x, y, style)
			}

			g.TileAnimations.Get(x, y).Enqueue(anim)

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
			g.ResetBoardNotStyles()
			g.QueueShowBoardAnimation(
				g.board.Width/2,
				g.board.Height/2,
			)
		}

		g.GameAnimations.Enqueue(anim)
	}
}

func (g *Game) QueueShowBoardAnimation(originX, originy int) {
	originP := FPt(f64(originX), f64(originy))

	var maxDist float64

	maxDist = max(maxDist, originP.Sub(FPt(0, 0)).Length())
	maxDist = max(maxDist, originP.Sub(FPt(f64(g.board.Width-1), 0)).Length())
	maxDist = max(maxDist, originP.Sub(FPt(0, f64(g.board.Height-1))).Length())
	maxDist = max(maxDist, originP.Sub(FPt(f64(g.board.Width-1), f64(g.board.Height-1))).Length())

	const minDuration = time.Millisecond * 80
	const maxDuration = time.Millisecond * 200

	var tileAnimationTotal time.Duration

	for x := range g.board.Width {
		for y := range g.board.Height {
			pos := FPt(f64(x), f64(y))
			dist := pos.Sub(originP).Length()
			d := time.Duration(Lerp(f64(minDuration), f64(maxDuration), dist/maxDist))

			var timer Timer
			timer.Duration = d

			var anim CallbackAnimation
			anim.Tag = AnimationTagShowBoard

			targetStyle := GetAnimationTargetTileStyle(g.board, x, y)

			anim.Update = func() {
				timer.TickUp()

				t := timer.Normalize()

				yOffsetT := BezierCurveDataAsGraph(TheBezierTable[BezierBoardShowTileOffsetY], t)
				alphaT := BezierCurveDataAsGraph(TheBezierTable[BezierBoardShowTileAlpha], t)
				scaleT := BezierCurveDataAsGraph(TheBezierTable[BezierBoardShowTileScale], t)

				targetStyle.BgOffsetY = yOffsetT * 50
				targetStyle.BgAlpha = alphaT
				targetStyle.BgScale = scaleT

				g.BaseTileStyles.Set(x, y, targetStyle)
			}

			anim.Skip = func() {
				timer.Current = timer.Duration
				anim.Update()
			}

			anim.Done = func() bool {
				return timer.Current >= timer.Duration
			}

			g.TileAnimations.Get(x, y).Enqueue(anim)

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
	for x := range g.board.Width {
		for y := range g.board.Height {
			AnimationQueueSkipAll(g.TileAnimations.Get(x, y))
		}
	}

	AnimationQueueSkipAll(&g.GameAnimations)
}

func (g *Game) SkipAllAnimationsUntilTag(tags ...AnimationTag) {
	for x := range g.board.Width {
		for y := range g.board.Height {
			AnimationQueueSkipUntilTag(g.TileAnimations.Get(x, y), tags...)
		}
	}

	AnimationQueueSkipUntilTag(&g.GameAnimations, tags...)
}

func GetMineTile() SubView {
	return SpriteSubView(TileSprite, 0)
}

func GetFlagTile(animT float64) SubView {
	const flagSpriteSount = 9
	frame := int(math.Round(animT * f64(flagSpriteSount-1)))
	frame = Clamp(frame, 0, flagSpriteSount-1)
	return SpriteSubView(TileSprite, frame+5)
}

// returns rounded cornder rect SubView
// and how many 90 degree turns it need (always from 0 to 3)
func GetRoundTile(isRound [4]bool) (SubView, int) {
	const (
		d0   = 0
		d90  = 1
		d180 = 2
		d270 = 3
	)

	const tileStart = 20

	roundCount := 0

	for _, round := range isRound {
		if round {
			roundCount++
		}
	}

	switch roundCount {
	case 0:
		return SpriteSubView(TileSprite, tileStart), d0
	case 1:
		if isRound[0] {
			return SpriteSubView(TileSprite, tileStart+1), d0
		}
		if isRound[1] {
			return SpriteSubView(TileSprite, tileStart+1), d90
		}
		if isRound[2] {
			return SpriteSubView(TileSprite, tileStart+1), d180
		}
		if isRound[3] {
			return SpriteSubView(TileSprite, tileStart+1), d270
		}
	case 2:
		if !isRound[0] && !isRound[1] {
			return SpriteSubView(TileSprite, tileStart+2), d180
		}
		if !isRound[1] && !isRound[2] {
			return SpriteSubView(TileSprite, tileStart+2), d270
		}
		if !isRound[2] && !isRound[3] {
			return SpriteSubView(TileSprite, tileStart+2), d0 // d360
		}
		if !isRound[3] && !isRound[0] {
			return SpriteSubView(TileSprite, tileStart+2), d90 // d450
		}
	case 3:
		if !isRound[0] {
			return SpriteSubView(TileSprite, tileStart+3), d90
		}
		if !isRound[1] {
			return SpriteSubView(TileSprite, tileStart+3), d180
		}
		if !isRound[2] {
			return SpriteSubView(TileSprite, tileStart+3), d270
		}
		if !isRound[3] {
			return SpriteSubView(TileSprite, tileStart+3), d0 // d360
		}
	case 4:
		return SpriteSubView(TileSprite, tileStart+4), d0
	default:
		panic("UNREACHABLE")
	}

	return SpriteSubView(TileSprite, tileStart+4), d0
}

func GetAllRoundTile() SubView {
	const tileStart = 20
	return SpriteSubView(TileSprite, tileStart+4)
}

func GetRectTile() SubView {
	const tileStart = 20
	return SpriteSubView(TileSprite, tileStart)
}

func MousePosToBoardPos(
	boardRect FRectangle,
	boardWidth, boardHeight int,
	mousePos FPoint,
) (int, int) {
	mousePos.X -= boardRect.Min.X
	mousePos.Y -= boardRect.Min.Y

	boardX := int(math.Floor(mousePos.X / (boardRect.Dx() / float64(boardWidth))))
	boardY := int(math.Floor(mousePos.Y / (boardRect.Dy() / float64(boardHeight))))

	return boardX, boardY
}

func GetBoardTileSize(
	boardRect FRectangle,
	boardWidth, boardHeight int,
) (float64, float64) {
	tileWidth := boardRect.Dx() / f64(boardWidth)
	tileHeight := boardRect.Dy() / f64(boardHeight)

	return tileWidth, tileHeight
}

func GetBoardTileRect(
	boardRect FRectangle,
	boardWidth, boardHeight int,
	boardX, boardY int,
) FRectangle {
	tileWidth := boardRect.Dx() / f64(boardWidth)
	tileHeight := boardRect.Dy() / f64(boardHeight)

	tileRect := FRectangle{
		Min: FPt(f64(boardX)*tileWidth, f64(boardY)*tileHeight).Add(boardRect.Min),
		Max: FPt(f64(boardX+1)*tileWidth, f64(boardY+1)*tileHeight).Add(boardRect.Min),
	}

	return RectToFRect(FRectToRect(tileRect))
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

	op.Uniforms["ScreenHeight"] = ScreenHeight

	imgRect := WaterShaderImage1.Bounds()
	imgFRect := RectToFRect(imgRect)

	op.GeoM.Scale(rect.Dx()/imgFRect.Dx(), rect.Dy()/imgFRect.Dy())
	op.GeoM.Translate(rect.Min.X, rect.Min.Y)

	DrawRectShader(dst, imgRect.Dx(), imgRect.Dy(), WaterShader, op)
}

func (g *Game) RetryButtonRect() FRectangle {
	boardRect := g.TransformedBoardRect()
	size := g.retryButtonSizeRelative * min(g.TransformedBoardRect().Dx(), g.TransformedBoardRect().Dy())
	rect := FRectWH(size, size)
	center := FRectangleCenter(boardRect)
	rect = CenterFRectangle(rect, center.X, center.Y)

	return rect
}

func (g *Game) TransformedRetryButtonRect() FRectangle {
	rect := g.RetryButtonRect()
	rect = FRectScaleCentered(rect, g.RetryButtonScale, g.RetryButtonScale)
	rect = rect.Add(FPt(g.RetryButtonOffsetX, g.RetryButtonOffsetY))
	return rect
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

	g.SetResetParameter(max(g.board.Width, newBoardWidth), max(g.board.Height, newBoardHeight), 0)
	g.ResetBoard()
	g.QueueShowBoardAnimation(
		(g.board.Width-1)/2,
		(g.board.Height-1)/2,
	)

	g.mineCount = 0

	iter := NewBoardIterator(0, 0, newBoardWidth-1, newBoardHeight-1)
	for iter.HasNext() {
		x, y := iter.GetNext()
		if g.board.IsPosInBoard(x, y) {
			char := newBoard[y][x] //yeah y and x is reversed

			switch char {
			case '@':
				g.board.Revealed.Set(x, y, true)
			case '*':
				g.board.Mines.Set(x, y, true)
				g.mineCount++
			case '+':
				g.board.Mines.Set(x, y, true)
				g.board.Flags.Set(x, y, true)
			}
		}
	}

	iter = NewBoardIterator(0, 0, g.board.Width-1, g.board.Height-1)
	for iter.HasNext() {
		x, y := iter.GetNext()
		if x < newBoardWidth+1 && y < newBoardHeight+1 {
			continue
		}

		if rand.Int64N(100) < 30 && !g.board.Mines.Get(x, y) {
			g.board.Mines.Set(x, y, true)
			g.mineCount++
		}
	}

	iter.Reset()

	for iter.HasNext() {
		x, y := iter.GetNext()

		if !g.board.Mines.Get(x, y) {
			if rand.Int64N(100) < 30 {
				// flag the surrounding
				innerIter := NewBoardIterator(x-1, y-1, x+1, y+1)
				for innerIter.HasNext() {
					inX, inY := innerIter.GetNext()
					if g.board.IsPosInBoard(inX, inY) && g.board.Mines.Get(inX, inY) {
						g.board.Flags.Set(inX, inY, true)
					}
				}

				g.board.SpreadSafeArea(x, y)
			}
		}
	}
}

func (g *Game) SetBoardForInstantWin() {
	if g.board.HasNoMines() {
		g.board.PlaceMines(g.mineCount, g.board.Width-1, g.board.Height-1, GetSeed())
	}

	// count how many tiles we have to reveal
	tilesToReveal := 0
	for x := range g.board.Width {
		for y := range g.board.Height {
			if !g.board.Mines.Get(x, y) && !g.board.Revealed.Get(x, y) {
				tilesToReveal++
			}
		}
	}

	// reveal that many tiles EXCEPT ONE
REVEAL_LOOP:
	for x := range g.board.Width {
		for y := range g.board.Height {
			if tilesToReveal <= 1 {
				break REVEAL_LOOP
			}
			if !g.board.Mines.Get(x, y) && !g.board.Revealed.Get(x, y) {
				g.board.Revealed.Set(x, y, true)
				tilesToReveal--
			}
		}
	}

	// flag every mines
	for x := range g.board.Width {
		for y := range g.board.Height {
			if g.board.Mines.Get(x, y) {
				g.board.Flags.Set(x, y, true)
			}
		}
	}
}

// =====================
// RetryButton
// =====================

func (g *Game) SetRetryButtonSize(size float64) {
	g.retryButtonSize = size
}

type RetryButton struct {
	BaseButton

	ButtonHoverOffset float64

	// water stuff
	DoWaterEffect bool

	WaterAlpha      float64
	WaterFlowOffset time.Duration

	waterRenderTarget *eb.Image
}

func NewRetryButton() *RetryButton {
	rb := new(RetryButton)
	rb.BaseButton = NewBaseButton()
	rb.waterRenderTarget = eb.NewImageWithOptions(
		RectWH(256, 256),
		&eb.NewImageOptions{Unmanaged: true},
	)

	return rb
}

func (rb *RetryButton) Update() {
	prevState := rb.BaseButton.State
	prevHover := rb.ButtonHoverOffset

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

	if prevState != rb.State {
		SetRedraw()
	}

	if !CloseToEx(prevHover, rb.ButtonHoverOffset, 0.05) {
		SetRedraw()
	}
}

func (rb *RetryButton) Draw(dst *eb.Image) {
	bottomRect := FRectWH(rb.Rect.Dx(), rb.Rect.Dy()*0.95)
	topRect := bottomRect

	topRect = FRectMoveTo(topRect, rb.Rect.Min.X, rb.Rect.Min.Y)
	bottomRect = FRectMoveTo(bottomRect, rb.Rect.Min.X, rb.Rect.Max.Y-bottomRect.Dy())

	if rb.State == ButtonStateDown || rb.Disabled {
		topRect = FRectMoveTo(topRect, bottomRect.Min.X, bottomRect.Min.Y)
	} else if rb.State == ButtonStateHover {
		topRect = topRect.Add(FPt(0, -topRect.Dy()*0.04*rb.ButtonHoverOffset))
	}

	// calculate colors
	colorT := Clamp(rb.WaterAlpha, 0, 1)
	if !rb.DoWaterEffect {
		colorT = 0
	}
	color1 := LerpColorRGBA(ColorRetryA1, ColorRetryB1, colorT)
	color2 := LerpColorRGBA(ColorRetryA2, ColorRetryB2, colorT)
	color3 := LerpColorRGBA(ColorRetryA3, ColorRetryB3, colorT)
	color4 := LerpColorRGBA(ColorRetryA4, ColorRetryB4, colorT)

	const segments = 6
	const radius = 0.4

	// NOTE : water effect on RetryButton doesn't look so good
	// while being too expensive.
	//
	// Reenable it when needed.

	//if rb.DoWaterEffect {
	if false {
		const renderMargin = 3
		union := bottomRect.Union(topRect).Inset(-renderMargin)
		union = RectToFRect(FRectToRect(union))

		// resize waterRenderTarget if needed
		if union.Dx() > f64(rb.waterRenderTarget.Bounds().Dx()) || union.Dy() > f64(rb.waterRenderTarget.Bounds().Dy()) {
			newW := rb.waterRenderTarget.Bounds().Dx()
			newH := rb.waterRenderTarget.Bounds().Dy()

			for f64(newW) < union.Dx() || f64(newH) < union.Dy() {
				newW *= 2
				newH *= 2
			}

			rb.waterRenderTarget.Dispose()
			rb.waterRenderTarget = eb.NewImageWithOptions(
				RectWH(newW, newH),
				&eb.NewImageOptions{Unmanaged: true},
			)
		}

		rb.waterRenderTarget.Clear()

		bottomRectW := bottomRect.Sub(union.Min)
		topRectW := topRect.Sub(union.Min)

		FillRoundRect(
			rb.waterRenderTarget, bottomRectW, radius, false, color1)

		FillRoundRect(
			rb.waterRenderTarget, topRectW, radius, false, color2)

		waterColors := [4]color.Color{
			ColorRetryWater1,
			ColorRetryWater2,
			ColorRetryWater3,
			ColorRetryWater4,
		}

		for i, c := range waterColors {
			nrgba := ColorToNRGBA(c)
			waterColors[i] = color.NRGBA{nrgba.R, nrgba.G, nrgba.B, uint8(f64(nrgba.A) * rb.WaterAlpha)}
		}

		BeginBlend(eb.BlendSourceAtop)
		DrawWaterRect(
			rb.waterRenderTarget,
			FRectMoveTo(union, 0, 0),
			GlobalTimerNow()+rb.WaterFlowOffset,
			waterColors,
			union.Min,
		)
		EndBlend()

		// draw waterRenderTarget
		op := &DrawImageOptions{}
		op.GeoM.Translate(union.Min.X, union.Min.Y)

		BeginAntiAlias(false)
		BeginFilter(eb.FilterNearest)
		BeginMipMap(false)
		DrawImage(dst, rb.waterRenderTarget, op)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	} else {
		FillRoundRect(
			dst, bottomRect, radius, false, color1)

		FillRoundRect(
			dst, topRect, radius, false, color2)
	}

	imgRect := RectToFRect(RetryButtonImage.Bounds())
	scale := min(topRect.Dx(), topRect.Dy()) / max(imgRect.Dx(), imgRect.Dy())
	scale *= 0.6

	center := FRectangleCenter(topRect)

	op := &DrawImageOptions{}
	op.GeoM.Concat(TransformToCenter(imgRect.Dx(), imgRect.Dy(), scale, scale, 0))
	op.GeoM.Translate(center.X, center.Y-topRect.Dy()*0.02)
	op.ColorScale.ScaleWithColor(color3)

	DrawImage(dst, RetryButtonImage, op)

	op.GeoM.Translate(0, topRect.Dy()*0.02*2)
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(color4)

	DrawImage(dst, RetryButtonImage, op)
}

// =====================
// Tutorial
// =====================

type FadeSegment struct {
	FadeInStart time.Duration
	FadeInEnd   time.Duration

	FadeOutStart time.Duration
	FadeOutEnd   time.Duration
}

func (fs *FadeSegment) GetFade(timer Timer) float64 {
	if timer.Current <= fs.FadeInEnd {
		return timer.NormalizeRange(fs.FadeInStart, fs.FadeInEnd)
	}

	if timer.Current >= fs.FadeOutStart {
		return 1 - timer.NormalizeRange(fs.FadeOutStart, fs.FadeOutEnd)
	}

	return 1
}

type FlagTutorial struct {
	ShowFlagTutorial bool

	foundGoodTile bool

	numberTileX int
	numberTileY int

	flagTileX int
	flagTileY int

	animationTimer Timer

	// animation constants
	animationDelay time.Duration

	fade FadeSegment

	cursorMoveStart time.Duration
	cursorMoveEnd   time.Duration

	cursorFade FadeSegment

	hlFade FadeSegment

	flagAnimationStart time.Duration
	flagAnimationEnd   time.Duration
}

func NewFlagTutorial() *FlagTutorial {
	ft := new(FlagTutorial)
	ft.numberTileX = -1
	ft.numberTileY = -1

	ft.flagTileX = -1
	ft.flagTileY = -1

	ft.animationDelay = time.Millisecond * 500

	ft.fade.FadeInStart = time.Millisecond * 0
	ft.fade.FadeInEnd = ft.fade.FadeInStart + time.Millisecond*100

	ft.fade.FadeOutStart = time.Millisecond * 1800
	ft.fade.FadeOutEnd = ft.fade.FadeOutStart + time.Millisecond*50

	ft.flagAnimationStart = time.Millisecond * 1100
	ft.flagAnimationEnd = time.Millisecond * 1600

	ft.cursorFade.FadeInStart = ft.fade.FadeInStart
	ft.cursorFade.FadeInEnd = ft.fade.FadeInEnd

	ft.cursorFade.FadeOutStart = time.Millisecond * 1200
	ft.cursorFade.FadeOutEnd = time.Millisecond * 1300

	ft.cursorMoveStart = time.Millisecond * 200
	ft.cursorMoveEnd = time.Millisecond * 1100

	ft.hlFade.FadeInStart = ft.fade.FadeInStart
	ft.hlFade.FadeInEnd = ft.fade.FadeInEnd

	ft.hlFade.FadeOutEnd = ft.flagAnimationStart + time.Millisecond*50
	ft.hlFade.FadeOutStart = ft.hlFade.FadeOutEnd - time.Millisecond*10

	ft.animationTimer.Current = -ft.animationDelay
	ft.animationTimer.Duration = max(
		ft.fade.FadeOutEnd,
		ft.cursorFade.FadeOutEnd,
		ft.hlFade.FadeOutEnd,
		ft.flagAnimationEnd,
	) + time.Millisecond*100

	return ft
}

func (ft *FlagTutorial) GetFlagTutorialStyleModifier() StyleModifier {
	return func(
		prevBoard, board Board,
		boardRect FRectangle,
		interaction BoardInteractionType,
		stateChanged bool,
		prevGameState, gameState GameState,
		tileStyles Array2D[TileStyle],
		gi GameInput,
	) bool {
		if !ft.foundGoodTile {
			return false
		}

		if ft.animationTimer.Current < 0 {
			return false
		}

		iter := NewBoardIterator(
			ft.numberTileX-1, ft.numberTileY-1,
			ft.numberTileX+1, ft.numberTileY+1,
		)

		for iter.HasNext() {
			x, y := iter.GetNext()
			if !board.IsPosInBoard(x, y) {
				continue
			}
			if !(x == ft.numberTileX && y == ft.numberTileY) && board.Revealed.Get(x, y) {
				continue
			}
			tileStyles.Data[x+tileStyles.Width*y].Highlight = ft.hlFade.GetFade(ft.animationTimer)
		}

		if ft.animationTimer.Current >= ft.flagAnimationStart {
			t := ft.animationTimer.NormalizeRange(ft.flagAnimationStart, ft.flagAnimationEnd)

			style := tileStyles.Get(ft.flagTileX, ft.flagTileY)

			style.DrawFg = true
			style.FgType = TileFgTypeFlag
			style.FgColor = ColorFade(ColorFlag, ft.fade.GetFade(ft.animationTimer))
			style.FgFlagAnim = t

			style.Highlight = 0

			tileStyles.Set(ft.flagTileX, ft.flagTileY, style)
		}

		return true
	}
}

func (ft *FlagTutorial) Update(
	board Board,
	boardRect FRectangle,
	maxRect FRectangle,
) {
	maxRectCenter := FRectangleCenter(maxRect)

	centerBX, centerBY := MousePosToBoardPos(
		boardRect,
		board.Width, board.Height,
		FPt(ScreenWidth*0.5, ScreenHeight*0.5),
	)

	isGoodNumberTile := func(numberAt image.Point) bool {
		if !board.IsPosInBoard(numberAt.X, numberAt.Y) {
			return false
		}
		if !board.Revealed.Get(numberAt.X, numberAt.Y) {
			return false
		}
		if board.GetNeighborMineCount(numberAt.X, numberAt.Y) <= 0 {
			return false
		}
		numberTile := GetBoardTileRect(
			boardRect,
			board.Width, board.Height,
			numberAt.X, numberAt.Y,
		)
		if !numberTile.In(maxRect) {
			return false
		}
		return true
	}

	isGoodFlagTile := func(flagAt image.Point) bool {
		if !board.IsPosInBoard(flagAt.X, flagAt.Y) {
			return false
		}
		if board.Revealed.Get(flagAt.X, flagAt.Y) {
			return false
		}
		if !board.Mines.Get(flagAt.X, flagAt.Y) {
			return false
		}
		if board.Flags.Get(flagAt.X, flagAt.Y) {
			return false
		}
		tile := GetBoardTileRect(
			boardRect,
			board.Width, board.Height,
			flagAt.X, flagAt.Y,
		)
		if !tile.In(maxRect) {
			return false
		}

		return true
	}

	getGoodFlagTile := func(numberAt image.Point) (bool, image.Point) {
		if !isGoodNumberTile(numberAt) {
			return false, image.Point{}
		}

		toSearch := [...]image.Point{
			// search vertically and horizontally first
			image.Pt(numberAt.X-1, numberAt.Y),
			image.Pt(numberAt.X+1, numberAt.Y),
			image.Pt(numberAt.X, numberAt.Y-1),
			image.Pt(numberAt.X, numberAt.Y+1),

			// search diagonal last
			image.Pt(numberAt.X-1, numberAt.Y-1),
			image.Pt(numberAt.X+1, numberAt.Y-1),
			image.Pt(numberAt.X-1, numberAt.Y+1),
			image.Pt(numberAt.X+1, numberAt.Y+1),
		}

		for _, pt := range toSearch {
			x, y := pt.X, pt.Y
			if !isGoodFlagTile(image.Pt(x, y)) {
				continue
			}

			return true, image.Pt(x, y)
		}

		return false, image.Point{}
	}

	clampX := func(x int) int {
		return Clamp(x, 0, board.Width-1)
	}

	clampY := func(y int) int {
		return Clamp(y, 0, board.Height-1)
	}

	findGoodTile := false

	if !ft.foundGoodTile {
		findGoodTile = true
		ft.foundGoodTile = false
		ft.animationTimer.Current = -ft.animationDelay
	}
	if !isGoodNumberTile(image.Pt(ft.numberTileX, ft.numberTileY)) || !isGoodFlagTile(image.Pt(ft.flagTileX, ft.flagTileY)) {
		findGoodTile = true
		ft.foundGoodTile = false
		ft.animationTimer.Current = -ft.animationDelay
	}
	if !ft.ShowFlagTutorial {
		findGoodTile = false
		ft.foundGoodTile = false
		ft.animationTimer.Current = -ft.animationDelay
	}

	if findGoodTile {
		searchDist := 0
	SEARCH_LOOP:
		for {
			var minPoint, maxPoint image.Point

			minPoint.X = centerBX - searchDist
			minPoint.Y = centerBY - searchDist

			maxPoint.X = centerBX + searchDist
			maxPoint.Y = centerBY + searchDist

			{
				tileW, tileH := GetBoardTileSize(boardRect, board.Width, board.Height)
				size := f64((searchDist * 2) + 1)
				searchRect := FRectWH(size*tileW, size*tileH)
				searchRect = CenterFRectangle(searchRect, maxRectCenter.X, maxRectCenter.Y)
				if maxRect.In(searchRect) {
					break
				}
			}

			minPoint.X = clampX(minPoint.X)
			minPoint.Y = clampY(minPoint.Y)

			maxPoint.X = clampX(maxPoint.X)
			maxPoint.Y = clampY(maxPoint.Y)

			for x := minPoint.X; x <= maxPoint.X; x++ {
				y := minPoint.Y
				if isGood, flagAt := getGoodFlagTile(image.Pt(x, y)); isGood {
					ft.numberTileX, ft.numberTileY = x, y
					ft.flagTileX, ft.flagTileY = flagAt.X, flagAt.Y
					ft.foundGoodTile = true
					break SEARCH_LOOP
				}
			}
			for x := minPoint.X; x <= maxPoint.X; x++ {
				y := maxPoint.Y
				if isGood, flagAt := getGoodFlagTile(image.Pt(x, y)); isGood {
					ft.numberTileX, ft.numberTileY = x, y
					ft.flagTileX, ft.flagTileY = flagAt.X, flagAt.Y
					ft.foundGoodTile = true
					break SEARCH_LOOP
				}
			}
			for y := minPoint.Y; y <= maxPoint.Y; y++ {
				x := minPoint.X
				if isGood, flagAt := getGoodFlagTile(image.Pt(x, y)); isGood {
					ft.numberTileX, ft.numberTileY = x, y
					ft.flagTileX, ft.flagTileY = flagAt.X, flagAt.Y
					ft.foundGoodTile = true
					break SEARCH_LOOP
				}
			}
			for y := minPoint.Y; y <= maxPoint.Y; y++ {
				x := maxPoint.X
				if isGood, flagAt := getGoodFlagTile(image.Pt(x, y)); isGood {
					ft.numberTileX, ft.numberTileY = x, y
					ft.flagTileX, ft.flagTileY = flagAt.X, flagAt.Y
					ft.foundGoodTile = true
					break SEARCH_LOOP
				}
			}

			searchDist += 1
		}
	}

	if ft.foundGoodTile {
		ft.animationTimer.TickUp()
		if ft.animationTimer.Current > ft.animationTimer.Duration {
			ft.animationTimer.Current = -ft.animationDelay
		}
	} else {
		ft.animationTimer.Current = -ft.animationDelay
	}
}

func (ft *FlagTutorial) Draw(
	dst *eb.Image,
	board Board,
	boardRect FRectangle,
) {
	if ft.foundGoodTile && ft.animationTimer.Current >= 0 && ft.ShowFlagTutorial {
		tileW, tileH := GetBoardTileSize(boardRect, board.Width, board.Height)

		numberTile := GetBoardTileRect(
			boardRect,
			board.Width, board.Height,
			ft.numberTileX, ft.numberTileY,
		)
		flagTile := GetBoardTileRect(
			boardRect,
			board.Width, board.Height,
			ft.flagTileX, ft.flagTileY,
		)

		lerpedPos := FPointLerp(
			FRectangleCenter(numberTile),
			FRectangleCenter(flagTile),
			EaseInCirc(ft.animationTimer.NormalizeRange(
				ft.cursorMoveStart,
				ft.cursorMoveEnd,
			)),
		)

		// =======================
		// draw cursor
		// =======================
		cursorScale := min(tileW/f64(CursorSprite.Width), tileH/f64(CursorSprite.Height)) * 1.2
		cursorOp := &DrawSubViewOptions{}
		cursorOp.GeoM.Concat(TransformToCenter(
			f64(CursorSprite.Width), f64(CursorSprite.Height),
			cursorScale, cursorScale,
			0,
		))

		cursorCenter := lerpedPos.Add(FPt(0, f64(CursorSprite.Height)*cursorScale*0.6))

		cursorOp.GeoM.Translate(
			cursorCenter.X,
			cursorCenter.Y,
		)

		BeginFilter(eb.FilterLinear)
		BeginMipMap(true)
		BeginAntiAlias(true)

		cursorOp.ColorScale.ScaleWithColor(
			ColorFade(
				ColorFlagTutorialStroke,
				ft.cursorFade.GetFade(ft.animationTimer),
			),
		)
		DrawSubView(dst, SpriteSubView(CursorSprite, 0), cursorOp)
		cursorOp.ColorScale.Reset()
		cursorOp.ColorScale.ScaleWithColor(
			ColorFade(
				ColorFlagTutorialFill,
				ft.cursorFade.GetFade(ft.animationTimer),
			),
		)
		DrawSubView(dst, SpriteSubView(CursorSprite, 1), cursorOp)

		EndAntiAlias()
		EndMipMap()
		EndFilter()

		// =======================
		// draw drag sign
		// =======================
		signPos := FRectangleCenter(numberTile)

		signHeight := min(tileH, tileW) * 0.8

		if ft.flagTileY > ft.numberTileY { // flag is below the number
			// display the drag text above
			signPos.Y -= tileH * 0.5      // above tile
			signPos.Y -= signHeight * 1.2 // account for text height
		} else {
			// display the drag text below
			signPos.Y += f64(CursorSprite.Height) * cursorScale * (0.6 + 0.5) // below cursor
			signPos.Y += tileH * 0.1                                          // with extra margin
		}

		signScale := signHeight / f64(DragSignSprite.Height)

		signOp := &DrawSubViewOptions{}
		signOp.GeoM.Translate(-f64(DragSignSprite.Width)*0.5, 0)
		signOp.GeoM.Scale(signScale, signScale)
		signOp.GeoM.Translate(signPos.X, signPos.Y)

		BeginFilter(eb.FilterLinear)
		BeginMipMap(true)
		BeginAntiAlias(true)
		signOp.ColorScale.ScaleWithColor(
			ColorFade(
				ColorFlagTutorialStroke,
				ft.fade.GetFade(ft.animationTimer),
			),
		)
		DrawSubView(dst, SpriteSubView(DragSignSprite, 1), signOp)

		signOp.ColorScale.Reset()
		signOp.ColorScale.ScaleWithColor(
			ColorFade(
				ColorFlagTutorialFill,
				ft.fade.GetFade(ft.animationTimer),
			),
		)
		DrawSubView(dst, SpriteSubView(DragSignSprite, 0), signOp)

		EndAntiAlias()
		EndMipMap()
		EndFilter()

		SetRedraw()
	}
}
