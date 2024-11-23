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
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

var _ = fmt.Printf

type GameState int

const (
	GameStatePlaying GameState = iota
	GameStateWon
	GameStateLost
)

type MouseState struct {
	JustPressedL bool
	JustPressedR bool
	JustPressedM bool

	PressedL bool
	PressedR bool
	PressedM bool

	CursorX float64
	CursorY float64

	BoardX int
	BoardY int
}

func (ms *MouseState) PressedAny() bool {
	return ms.PressedL || ms.PressedR || ms.PressedM
}

func (ms *MouseState) JustPressedAny() bool {
	return ms.JustPressedL || ms.JustPressedR || ms.JustPressedM
}

func GetMouseState(board Board, boardRect FRectangle) MouseState {
	var ms MouseState

	ms.JustPressedL = IsMouseButtonJustPressed(eb.MouseButtonLeft)
	ms.JustPressedR = IsMouseButtonJustPressed(eb.MouseButtonRight)
	ms.JustPressedM = IsMouseButtonJustPressed(eb.MouseButtonMiddle)

	ms.PressedL = IsMouseButtonPressed(eb.MouseButtonLeft)
	ms.PressedR = IsMouseButtonPressed(eb.MouseButtonRight)
	ms.PressedM = IsMouseButtonPressed(eb.MouseButtonMiddle)

	cursor := CursorFPt()

	ms.CursorX = cursor.X
	ms.CursorY = cursor.Y

	ms.BoardX, ms.BoardY = MousePosToBoardPos(board, boardRect, cursor)

	return ms
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

	// animations used by Game
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
	tileStyles [][]TileStyle, // modify these to change style
) bool

type Game struct {
	Rect FRectangle

	OnBoardReset       func()
	OnGameEnd          func(didWin bool)
	OnFirstInteraction func()

	BaseTileStyles   [][]TileStyle
	RenderTileStyles [][]TileStyle

	TileAnimations [][]CircularQueue[CallbackAnimation]

	GameAnimations CircularQueue[CallbackAnimation]

	StyleModifiers []StyleModifier

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

	board     Board
	prevBoard Board

	shouldCallOnFirstInteraction bool

	mineCount int

	placedMinesOnBoard bool

	viBuffers [2]*VIBuffer
}

func NewGame(boardWidth, boardHeight, mineCount int) *Game {
	g := new(Game)

	g.viBuffers[0] = new(VIBuffer)
	g.viBuffers[1] = new(VIBuffer)

	g.mineCount = mineCount

	g.WaterRenderTarget = eb.NewImage(int(ScreenWidth), int(ScreenHeight))

	g.StyleModifiers = append(g.StyleModifiers, NewTileHighlightModifier())
	g.StyleModifiers = append(g.StyleModifiers, NewNumberClickModifier())

	g.RetryButton = NewRetryButton()
	g.RetryButton.Disabled = true
	g.RetryButton.ActOnRelease = true
	g.RetryButtonScale = 1

	g.RetryButton.OnClick = func() {
		g.QueueResetBoardAnimation()
	}

	g.GameAnimations = NewCircularQueue[CallbackAnimation](10)

	g.ResetBoard(boardWidth, boardHeight, g.mineCount)

	return g
}

func (g *Game) ResetBoardWithNoStyles(width, height, mineCount int) {
	g.shouldCallOnFirstInteraction = true

	g.placedMinesOnBoard = false

	g.GameState = GameStatePlaying
	g.GameAnimations.Clear()

	g.board = NewBoard(width, height)
	g.prevBoard = NewBoard(width, height)

	g.mineCount = mineCount

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

	if g.OnBoardReset != nil {
		g.OnBoardReset()
	}
}

func (g *Game) ResetBoard(width, height, mineCount int) {
	g.ResetBoardWithNoStyles(width, height, mineCount)

	for x := range width {
		for y := range height {
			targetStyle := GetAnimationTargetTileStyle(g.board, x, y)
			g.BaseTileStyles[x][y] = targetStyle
			g.RenderTileStyles[x][y] = targetStyle
		}
	}
}

func (g *Game) Update() {
	ms := GetMouseState(g.board, g.Rect)

	// =================================
	// handle board interaction
	// =================================

	// =======================================
	prevState := g.GameState
	g.board.SaveTo(g.prevBoard)

	var stateChanged bool = false

	var interaction BoardInteractionType = InteractionTypeNone
	// =======================================

	if g.GameState == GameStatePlaying && g.board.IsPosInBoard(ms.BoardX, ms.BoardY) && ms.JustPressedAny() {
		if ((ms.JustPressedL && ms.PressedR) || (ms.PressedL && ms.JustPressedR)) || ms.JustPressedM {
			interaction = InteractionTypeCheck
		} else if ms.JustPressedR {
			interaction = InteractionTypeFlag
		} else if ms.JustPressedL {
			interaction = InteractionTypeStep
		}

		if interaction != InteractionTypeNone {
			g.GameState = g.board.InteractAt(ms.BoardX, ms.BoardY, interaction, g.mineCount)

			// ==============================
			// check if state has changed
			// ==============================

			// first check game state
			stateChanged = prevState != g.GameState

			// then check board state
			if !stateChanged {
			DIFF_CHECK:
				for x := range g.board.Width {
					for y := range g.board.Height {
						if g.board.Mines[x][y] != g.prevBoard.Mines[x][y] {
							stateChanged = true
							break DIFF_CHECK
						}

						if g.board.Flags[x][y] != g.prevBoard.Flags[x][y] {
							stateChanged = true
							break DIFF_CHECK
						}

						if g.board.Revealed[x][y] != g.prevBoard.Revealed[x][y] {
							stateChanged = true
							break DIFF_CHECK
						}
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
			g.prevBoard.Revealed, g.board.Revealed, 0, 0)
		stateChanged = true
	}
	if IsKeyJustPressed(InstantWinKey) {
		g.SetBoardForInstantWin()
		g.QueueRevealAnimation(
			g.prevBoard.Revealed, g.board.Revealed, 0, 0)
		stateChanged = true
	}

	// reveal the board if mouse button 4 is pressed
	if g.GameState == GameStatePlaying &&
		ms.BoardX >= 0 && ms.BoardY >= 0 &&
		IsMouseButtonJustPressed(eb.MouseButton4) {

		g.placedMinesOnBoard = true

		g.board.Revealed[ms.BoardX][ms.BoardY] = true
		stateChanged = true
	}

	// ==============================
	// on state changes
	// ==============================
	if stateChanged {
		// check if we need to start board reveal animation
	REVEAL_CHECK:
		for x := range g.board.Width {
			for y := range g.board.Height {
				if g.board.Revealed[x][y] && !g.prevBoard.Revealed[x][y] {
					g.QueueRevealAnimation(
						g.prevBoard.Revealed, g.board.Revealed, ms.BoardX, ms.BoardY)

					break REVEAL_CHECK
				}
			}
		}

		if prevState != g.GameState {
			if g.GameState == GameStateLost {
				g.QueueDefeatAnimation(ms.BoardX, ms.BoardY)
			} else if g.GameState == GameStateWon {
				g.QueueWinAnimation(ms.BoardX, ms.BoardY)
			}
		}

		// call OnFirstInteraction
		if g.shouldCallOnFirstInteraction {
			g.shouldCallOnFirstInteraction = false
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

	// ============================
	// update animations
	// ============================

	// update GameAnimations
	AnimationQueueUpdate(&g.GameAnimations)

	// update BaseTileStyles
	for x := range g.board.Width {
		for y := range g.board.Height {
			AnimationQueueUpdate(&g.TileAnimations[x][y])
		}
	}

	// copy it over to RenderTileStyles
	for x := range g.board.Width {
		for y := range g.board.Height {
			g.RenderTileStyles[x][y] = g.BaseTileStyles[x][y]
		}
	}

	// ===================================
	// apply style modifiers
	// ===================================
	for i := 0; i < len(g.StyleModifiers); i++ {
		g.StyleModifiers[i](
			g.prevBoard, g.board,
			g.Rect,
			interaction,
			stateChanged,
			prevState, g.GameState,
			g.RenderTileStyles,
		)
	}

	// ============================
	// update flag drawing
	// ============================
	if stateChanged {
		for x := range g.board.Width {
			for y := range g.board.Height {
				if g.prevBoard.Flags[x][y] != g.board.Flags[x][y] {
					if g.board.Flags[x][y] {
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
		if ms.JustPressedAny() {
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

		g.board, g.Rect,
		g.RenderTileStyles,

		g.GameState == GameStateWon, g.WaterRenderTarget, g.WaterAlpha, g.WaterFlowOffset,

		g.viBuffers,
	)

	if g.DrawRetryButton {
		g.RetryButton.Draw(dst)
	}
}

func (g *Game) MineCount() int {
	return g.mineCount
}

func (g *Game) FlagCount() int {
	flagCount := 0
	for x := range g.board.Width {
		for y := range g.board.Height {
			if g.board.Flags[x][y] {
				flagCount++
			}
		}
	}

	return flagCount
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
	return style.DrawBg && !style.DrawTile && style.BgScale > 0.001 && style.BgAlpha > 0.001
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

type VIBuffer struct {
	Vertices []eb.Vertex
	Indices  []uint16
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
	imgSize := ImageSizeFPt(view)
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

	viBuffers [2]*VIBuffer,
) {
	shapeBuf := viBuffers[0]
	spriteBuf := viBuffers[1]

	shapeBuf.Reset()
	spriteBuf.Reset()

	// ============================
	// draw background tiles
	// ============================
	forEachBgTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, bgTileRect FRectangle) {
			if ShouldDrawBgTile(style) {
				VIaddRect(
					shapeBuf,
					bgTileRect,
					ColorFade(style.BgFillColor, style.BgAlpha),
				)

				// draw highlight
				if style.BgTileHightlight > 0 {
					t := style.BgTileHightlight
					c := TheColorTable[ColorTileHighLight]

					VIaddRect(
						shapeBuf,
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

					outerP := GetRoundRectPathFast(outerRect, outerRadius, true, 5)
					innerP := GetRoundRectPathFast(innerRect, innerRadius, true, 5)

					VIaddFillPath(shapeBuf, *outerP, ColorFade(ColorMineBg1, style.BgAlpha))
					VIaddFillPath(shapeBuf, *innerP, ColorFade(ColorMineBg2, style.BgAlpha))

					VIaddSubViewInRect(spriteBuf, innerRect, 1, 0, 0, ColorFade(ColorMine, style.BgAlpha), GetMineTile())
				}
			}
		},
	)

	// ============================
	// draw tiles
	// ============================
	const segments = 5

	forEachTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, strokeRect, fillRect FRectangle, radiusPx [4]float64) {
			if ShouldDrawTile(style) {
				strokeColor := ColorFade(style.TileStrokeColor, style.TileAlpha)

				p := GetRoundRectPathFastEx(strokeRect, radiusPx, true, [4]int{segments, segments, segments, segments})
				VIaddFillPath(shapeBuf, *p, strokeColor)
			}
		},
	)

	forEachTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, strokeRect, fillRect FRectangle, radiusPx [4]float64) {
			if ShouldDrawTile(style) {
				fillColor := ColorFade(style.TileFillColor, style.TileAlpha)

				p := GetRoundRectPathFastEx(fillRect, radiusPx, true, [4]int{segments, segments, segments, segments})
				VIaddFillPath(shapeBuf, *p, fillColor)
			}
		},
	)

	forEachFgTile(
		board, boardRect, tileStyles,

		func(x, y int, style TileStyle, fgRect FRectangle) {
			if ShouldDrawFgTile(style) {
				fgColor := style.FgColor

				if style.FgType == TileFgTypeNumber {
					count := board.GetNeighborMineCount(x, y)
					if 1 <= count && count <= 8 {
						VIaddSubViewInRect(
							spriteBuf,
							fgRect,
							style.FgScale,
							style.FgOffsetX, style.FgOffsetY,
							ColorFade(fgColor, style.FgAlpha),
							GetNumberTile(count),
						)
					}
				} else if style.FgType == TileFgTypeFlag {
					VIaddSubViewInRect(
						spriteBuf,
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

	// ====================
	// flush buffers
	// ====================

	// flush shapes
	shapesRenderTarget := dst

	if waterRenderTarget != nil && doWaterEffect {
		waterRenderTarget.Clear()
		shapesRenderTarget = waterRenderTarget
	}

	BeginAntiAlias(false)
	BeginFilter(eb.FilterNearest)
	BeginMipMap(false)
	{
		op := &DrawTrianglesOptions{}
		op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
		DrawTriangles(shapesRenderTarget, shapeBuf.Vertices, shapeBuf.Indices, WhiteImage, op)
	}
	EndMipMap()
	EndFilter()
	EndAntiAlias()

	// draw water effect
	if doWaterEffect && waterRenderTarget != nil {
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

		BeginAntiAlias(false)
		BeginFilter(eb.FilterNearest)
		BeginMipMap(false)
		// draw waterRenderTarget
		DrawImage(dst, waterRenderTarget, nil)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	}

	// flush sprites
	{
		op := &DrawTrianglesOptions{}
		op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
		DrawTriangles(dst, spriteBuf.Vertices, spriteBuf.Indices, TileSprite.Image, op)
	}
}

func DrawDummyBgBoard(
	dst *eb.Image,
	boardWidth, boardHeight int,
	boardRect FRectangle,
	buffers [2]*VIBuffer,
) {
	// TODO: this is fucking stupid. We are creating a dummy board only to throw it away
	// each time we draw
	dummyBoard := NewBoard(boardWidth, boardHeight)
	dummyStyles := New2DArray[TileStyle](boardWidth, boardHeight)

	for x := range boardWidth {
		for y := range boardHeight {
			dummyStyles[x][y] = GetAnimationTargetTileStyle(dummyBoard, x, y)
		}
	}

	DrawBoard(
		dst,

		dummyBoard, boardRect,
		dummyStyles,

		false, nil, 0, 0,
		buffers,
	)
}

func MousePosToBoardPos(board Board, boardRect FRectangle, mousePos FPoint) (int, int) {
	mousePos.X -= boardRect.Min.X
	mousePos.Y -= boardRect.Min.Y

	boardX := int(math.Floor(mousePos.X / (boardRect.Dx() / float64(board.Width))))
	boardY := int(math.Floor(mousePos.Y / (boardRect.Dy() / float64(board.Height))))

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

func NewTileHighlightModifier() StyleModifier {
	var highlightTimer Timer

	highlightTimer.Duration = time.Millisecond * 100

	var highlightX, highlightY int

	return func(
		prevBoard, board Board,
		boardRect FRectangle,
		interaction BoardInteractionType,
		stateChanged bool,
		prevGameState, gameState GameState,
		tileStyles [][]TileStyle,
	) bool {
		if gameState != GameStatePlaying { // only do this when we are actually playing
			return false
		}

		ms := GetMouseState(board, boardRect)

		startHL := interaction == InteractionTypeCheck
		startHL = startHL && board.IsPosInBoard(ms.BoardX, ms.BoardY)
		startHL = startHL && prevBoard.Revealed[ms.BoardX][ms.BoardY]
		startHL = startHL && !stateChanged

		if startHL {
			highlightTimer.Current = highlightTimer.Duration
		}

		pressingCheck := (ms.PressedL && ms.PressedR) || ms.PressedM

		if !pressingCheck {
			highlightTimer.TickDown()
		}

		if highlightTimer.Current > 0 {
			if pressingCheck {
				highlightX = ms.BoardX
				highlightY = ms.BoardY
			}

			iter := NewBoardIterator(highlightX-1, highlightY-1, highlightX+1, highlightY+1)
			for iter.HasNext() {
				x, y := iter.GetNext()
				if board.IsPosInBoard(x, y) && !board.Revealed[x][y] && !board.Flags[x][y] {
					t := highlightTimer.Normalize()
					tileStyles[x][y].BgTileHightlight += t
				}
			}
		}

		return highlightTimer.Current > 0
	}
}

func NewNumberClickModifier() StyleModifier {
	var clickTimer Timer

	clickTimer.Duration = time.Millisecond * 100

	var clickX, clickY int

	var focused bool

	return func(
		prevBoard, board Board,
		boardRect FRectangle,
		interaction BoardInteractionType,
		stateChanged bool,
		prevGameState, gameState GameState,
		tileStyles [][]TileStyle,
	) bool {
		if gameState != GameStatePlaying { // only do this when we are actually playing
			return false
		}

		ms := GetMouseState(board, boardRect)

		cursorOnNumber := board.IsPosInBoard(ms.BoardX, ms.BoardY)
		cursorOnNumber = cursorOnNumber && board.Revealed[ms.BoardX][ms.BoardY]
		cursorOnNumber = cursorOnNumber && board.GetNeighborMineCount(ms.BoardX, ms.BoardY) > 0

		if cursorOnNumber && ms.JustPressedAny() {
			clickTimer.Current = clickTimer.Duration
			clickX = ms.BoardX
			clickY = ms.BoardY
			focused = true
		}

		if !ms.PressedAny() || !(clickX == ms.BoardX && clickY == ms.BoardY) {
			focused = false
		}

		if !focused {
			clickTimer.TickDown()
		}

		if clickTimer.Current > 0 {
			if board.IsPosInBoard(clickX, clickY) {
				tileStyles[clickX][clickY].FgScale *= 1 + clickTimer.Normalize()*0.07
			}
		}

		return clickTimer.Current > 0
	}
}

func (g *Game) QueueRevealAnimation(revealsBefore, revealsAfter [][]bool, originX, originy int) {
	g.SkipAllAnimations()

	fw, fh := f64(g.board.Width-1), f64(g.board.Height-1)

	originP := FPt(f64(originX), f64(originy))

	maxDist := math.Sqrt(fw*fw + fh*fh)

	const maxDuration = time.Millisecond * 900
	const minDuration = time.Millisecond * 20

	for x := range g.board.Width {
		for y := range g.board.Height {
			if !revealsBefore[x][y] && revealsAfter[x][y] {
				pos := FPt(f64(x), f64(y))
				dist := pos.Sub(originP).Length()
				d := time.Duration(f64(maxDuration) * (dist / maxDist))

				var timer Timer

				timer.Duration = max(d, minDuration)
				timer.Current = 0

				targetStyle := GetAnimationTargetTileStyle(g.board, x, y)

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

	for x := range g.board.Width {
		for y := range g.board.Height {
			if g.board.Mines[x][y] && !g.board.Flags[x][y] {
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

				if g.board.Revealed[x][y] {
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
				if !g.board.Revealed[x][y] {
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

	for x := range g.board.Width {
		for y := range g.board.Height {
			tileRect := GetBoardTileRect(g.Rect, g.board.Width, g.board.Height, x, y)

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
			g.ResetBoardWithNoStyles(g.board.Width, g.board.Height, g.mineCount)
			g.QueueShowBoardAnimation(
				g.board.Width/2,
				g.board.Height/2,
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
	for x := range g.board.Width {
		for y := range g.board.Height {
			AnimationQueueSkipAll(&g.TileAnimations[x][y])
		}
	}

	AnimationQueueSkipAll(&g.GameAnimations)
}

func (g *Game) SkipAllAnimationsUntilTag(tags ...AnimationTag) {
	for x := range g.board.Width {
		for y := range g.board.Height {
			AnimationQueueSkipUntilTag(&g.TileAnimations[x][y], tags...)
		}
	}

	AnimationQueueSkipUntilTag(&g.GameAnimations, tags...)
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

	tileRect := FRectangle{
		Min: FPt(f64(boardX)*tileWidth, f64(boardY)*tileHeight).Add(boardRect.Min),
		Max: FPt(f64(boardX+1)*tileWidth, f64(boardY+1)*tileHeight).Add(boardRect.Min),
	}

	return RectToFRect(FRectToRect(tileRect))
}

func (g *Game) DrawTile(
	dst *eb.Image,
	boardX, boardY int,
	scale float64,
	offsetX, offsetY float64,
	clr color.Color,
	tile SubView,
) {
	tileRect := GetBoardTileRect(g.Rect, g.board.Width, g.board.Height, boardX, boardY)
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

	g.ResetBoard(max(g.board.Width, newBoardWidth), max(g.board.Height, newBoardHeight), 0)
	g.QueueShowBoardAnimation(
		(g.board.Width-1)/2,
		(g.board.Height-1)/2,
	)

	g.placedMinesOnBoard = true
	g.mineCount = 0

	iter := NewBoardIterator(0, 0, newBoardWidth-1, newBoardHeight-1)
	for iter.HasNext() {
		x, y := iter.GetNext()
		if g.board.IsPosInBoard(x, y) {
			char := newBoard[y][x] //yeah y and x is reversed

			switch char {
			case '@':
				g.board.Revealed[x][y] = true
			case '*':
				g.board.Mines[x][y] = true
				g.mineCount++
			case '+':
				g.board.Mines[x][y] = true
				g.board.Flags[x][y] = true
			}
		}
	}

	iter = NewBoardIterator(0, 0, g.board.Width-1, g.board.Height-1)
	for iter.HasNext() {
		x, y := iter.GetNext()
		if x < newBoardWidth+1 && y < newBoardHeight+1 {
			continue
		}

		if rand.Int64N(100) < 30 && !g.board.Mines[x][y] {
			g.board.Mines[x][y] = true
			g.mineCount++
		}
	}

	iter.Reset()

	for iter.HasNext() {
		x, y := iter.GetNext()

		if !g.board.Mines[x][y] {
			if rand.Int64N(100) < 30 {
				// flag the surrounding
				innerIter := NewBoardIterator(x-1, y-1, x+1, y+1)
				for innerIter.HasNext() {
					inX, inY := innerIter.GetNext()
					if g.board.IsPosInBoard(inX, inY) && g.board.Mines[inX][inY] {
						g.board.Flags[inX][inY] = true
					}
				}

				g.board.SpreadSafeArea(x, y)
			}
		}
	}
}

func (g *Game) SetBoardForInstantWin() {
	if !g.placedMinesOnBoard {
		g.board.PlaceMines(g.mineCount, g.board.Width-1, g.board.Height-1)
	}
	g.placedMinesOnBoard = true

	// count how many tiles we have to reveal
	tilesToReveal := 0
	for x := range g.board.Width {
		for y := range g.board.Height {
			if !g.board.Mines[x][y] && !g.board.Revealed[x][y] {
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
			if !g.board.Mines[x][y] && !g.board.Revealed[x][y] {
				g.board.Revealed[x][y] = true
				tilesToReveal--
			}
		}
	}
}

// =====================
// RetryButton
// =====================

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
