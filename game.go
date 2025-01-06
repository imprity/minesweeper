package main

// TODO : currently, reveal animation and bomb animation
// needs too many sound players for it's sound effects
//
// causing sound to bug out on chrome
//
// change the animation and sound effects to use less players
// it doesn't even sound that good even with using many players

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

	FlagAnim float64

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
) bool

type PlayerPool struct {
	pool   []*Player
	cursor int
	volume float64
}

func NewPlayerPool(size int, audioName string) PlayerPool {
	pool := PlayerPool{}
	pool.pool = make([]*Player, size)

	for i := 0; i < len(pool.pool); i++ {
		pool.pool[i] = NewPlayer(audioName)
	}

	pool.volume = 1

	return pool
}

func (p *PlayerPool) SetVolume(volume float64) {
	volume = Clamp(volume, 0, 1)
	p.volume = volume
	for _, p := range p.pool {
		p.SetVolume(volume)
	}
}

func (p *PlayerPool) Volume() float64 {
	return p.volume
}

func (p *PlayerPool) Play() {
	if !IsSoundReady() {
		return
	}

	p.pool[p.cursor].SetPosition(0)
	p.pool[p.cursor].Play()
	p.cursor++
	if p.cursor >= len(p.pool) {
		p.cursor = 0
	}
}

type Game struct {
	Rect FRectangle

	OnBoardReset       func()
	OnGameEnd          func(didWin bool)
	OnFirstInteraction func()

	BaseTileStyles   Array2D[TileStyle]
	RenderTileStyles Array2D[TileStyle]

	TileAnimations Array2D[*CircularQueue[CallbackAnimation]]

	GameAnimations CircularQueue[CallbackAnimation]

	StyleModifiers []StyleModifier

	RetryButton *RetryButton

	RetryButtonSize float64

	DrawRetryButton    bool
	RetryButtonScale   float64
	RetryButtonOffsetX float64
	RetryButtonOffsetY float64

	GameState GameState

	WaterAlpha      float64
	WaterFlowOffset time.Duration

	WaterRenderTarget *eb.Image

	Particles []TileParticle

	TileRevealSoundPlayers PlayerPool
	BombSoundPlayers       PlayerPool

	board     Board
	prevBoard Board

	shouldCallOnFirstInteraction bool

	mineCount int

	placedMinesOnBoard bool

	viBuffers [3]*VIBuffer
}

func NewGame(boardWidth, boardHeight, mineCount int) *Game {
	g := new(Game)

	g.mineCount = mineCount

	g.WaterRenderTarget = eb.NewImage(int(ScreenWidth), int(ScreenHeight))

	g.StyleModifiers = append(g.StyleModifiers, NewTileHighlightModifier())
	g.StyleModifiers = append(g.StyleModifiers, NewFgClickModifier())

	g.RetryButton = NewRetryButton()
	g.RetryButton.Disabled = true
	g.RetryButtonScale = 1

	g.RetryButton.OnPress = func(justPressed bool) {
		if justPressed {
			PlaySoundBytes(SeSwitch38, 0.8)
		}
	}

	g.RetryButton.OnRelease = func() {
		g.QueueResetBoardAnimation()
		PlaySoundBytes(SeCut, 0.8)
	}

	g.RetryButton.WaterRenderTarget = g.WaterRenderTarget

	g.GameAnimations = NewCircularQueue[CallbackAnimation](10)

	g.Particles = make([]TileParticle, 0, 256)

	g.TileRevealSoundPlayers = NewPlayerPool(10, SeCut)
	g.TileRevealSoundPlayers.SetVolume(0.3)
	g.BombSoundPlayers = NewPlayerPool(5, SeUnlinkSummer)
	g.BombSoundPlayers.SetVolume(0.3)

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

	g.TileAnimations = New2DArray[*CircularQueue[CallbackAnimation]](width, height)
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

	if g.OnBoardReset != nil {
		g.OnBoardReset()
	}
}

func (g *Game) ResetBoard(width, height, mineCount int) {
	g.ResetBoardWithNoStyles(width, height, mineCount)

	for x := range width {
		for y := range height {
			targetStyle := GetAnimationTargetTileStyle(g.board, x, y)
			g.BaseTileStyles.Set(x, y, targetStyle)
			g.RenderTileStyles.Set(x, y, targetStyle)
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
						if g.board.Mines.Get(x, y) != g.prevBoard.Mines.Get(x, y) {
							stateChanged = true
							break DIFF_CHECK
						}

						if g.board.Flags.Get(x, y) != g.prevBoard.Flags.Get(x, y) {
							stateChanged = true
							break DIFF_CHECK
						}

						if g.board.Revealed.Get(x, y) != g.prevBoard.Revealed.Get(x, y) {
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

	// ==============================
	// on state changes
	// ==============================
	if stateChanged {
		SetRedraw() // just do it!!

		// check if we board has been revealed
	REVEAL_CHECK:
		for x := range g.board.Width {
			for y := range g.board.Height {
				if g.board.Revealed.Get(x, y) && !g.prevBoard.Revealed.Get(x, y) {
					// on reveal
					g.QueueRevealAnimation(
						g.prevBoard.Revealed, g.board.Revealed, ms.BoardX, ms.BoardY)
					//PlaySoundBytes(SoundEffects[15], 0.5)

					break REVEAL_CHECK
				}
			}
		}

		// update flag
		for x := range g.board.Width {
			for y := range g.board.Height {
				if g.prevBoard.Flags.Get(x, y) != g.board.Flags.Get(x, y) {
					if g.board.Flags.Get(x, y) {
						g.QueueAddFlagAnimation(x, y)
					} else {
						g.QueueRemoveFlagAnimation(x, y)
					}
				}
			}
		}

		if prevState != g.GameState {
			if g.GameState == GameStateLost { // on loss
				PlaySoundBytes(SeLinkSummer, 0.7)
				g.QueueDefeatAnimation(ms.BoardX, ms.BoardY)
			} else if g.GameState == GameStateWon { // on win
				g.QueueWinAnimation(ms.BoardX, ms.BoardY)
				PlaySoundBytes(SeWobble2, 0.6)
				//PlaySoundBytes(SoundEffects[SeWobble3], 0.6)
				//PlaySoundBytes(SoundEffects[SeSave], 0.6)
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

	if interaction != InteractionTypeNone {
		if !stateChanged { // user wanted to do something but nothing happened
			// pass
		} else { // something did happened
			// pass
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
			g.Rect,
			interaction,
			stateChanged,
			prevState, g.GameState,
			g.RenderTileStyles,
		)

		if doRedraw {
			SetRedraw()
		}
	}

	// skipping animations
	if prevState == GameStateLost || prevState == GameStateWon {
		// all animations are skippable except AnimationTagRetryButtonReveal
		if ms.JustPressedAny() {
			g.SkipAllAnimationsUntilTag(AnimationTagRetryButtonReveal)
		}
	}

	// =================================
	// update particles
	// =================================
	{
		tc := TileParticleUnitConverter{
			BoardWidth: g.board.Width, BoardHeight: g.board.Height,
			BoardRect: g.Rect,
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
	g.RetryButton.Rect = g.TransformedRetryButtonRect()
	g.RetryButton.Update()
	if !g.DrawRetryButton {
		g.RetryButton.Disabled = true
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

		g.board, g.Rect,
		g.RenderTileStyles,

		doWaterEffect, g.WaterRenderTarget, g.WaterAlpha, g.WaterFlowOffset,
	)

	if g.DrawRetryButton {
		g.RetryButton.WaterRenderTarget = g.WaterRenderTarget
		g.RetryButton.DoWaterEffect = doWaterEffect
		g.RetryButton.WaterAlpha = g.WaterAlpha
		g.RetryButton.WaterFlowOffset = g.WaterFlowOffset

		g.RetryButton.Draw(dst)
	}

	DrawParticles(
		dst,
		g.Particles,
		g.board.Width, g.board.Height,
		g.Rect,
	)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) {
	if g.WaterRenderTarget.Bounds().Dx() != outsideWidth || g.WaterRenderTarget.Bounds().Dy() != outsideHeight {
		g.WaterRenderTarget = eb.NewImage(outsideWidth, outsideHeight)
	}
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

func isTileFirmlyPlaced(style TileStyle) bool {
	const e = 0.08
	return style.DrawTile &&
		CloseToEx(style.TileScale, 1, e) &&
		CloseToEx(style.TileOffsetX, 0, e) &&
		CloseToEx(style.TileOffsetY, 0, e) &&
		style.TileAlpha > e
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

func GetAnimationTargetTileStyle(board Board, x, y int) TileStyle {
	style := NewTileStyle()

	style.DrawBg = true
	style.BgFillColor = ColorTileNormal1
	if IsOddTile(board.Width, board.Height, x, y) {
		style.BgFillColor = ColorTileNormal2
	}

	if board.IsPosInBoard(x, y) {
		if board.Revealed.Get(x, y) {
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
	VIBuffers [3]*VIBuffer

	ShouldDrawBgTile Array2D[bool]
	ShouldDrawTile   Array2D[bool]
	ShouldDrawFgTile Array2D[bool]

	BgTileRects Array2D[FRectangle]

	TileStrokeRects Array2D[FRectangle]
	TileFillRects   Array2D[FRectangle]

	TileFirmlyPlaced Array2D[bool]
	TileRoundness    Array2D[[4]bool]
}{}

func init() {
	// NOTE : hard coded number based on previous run
	DBC.VIBuffers[0] = NewVIBuffer(4096, 16384)
	DBC.VIBuffers[1] = NewVIBuffer(2048, 2048)
	DBC.VIBuffers[2] = NewVIBuffer(2048, 2048)
}

func DrawBoard(
	dst *eb.Image,

	board Board,
	boardRect FRectangle,
	tileStyles Array2D[TileStyle],

	// params for water effect
	doWaterEffect bool,
	waterRenderTarget *eb.Image,
	waterAlpha float64,
	waterFlowOffset time.Duration,
) {
	// TODO : we need flagSpriteBuf only because flag animations are stored in different image
	// merge flag sprite with other sprites.

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

	iter := NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	// ======================
	// reset VIBuffers
	// ======================
	for i := range DBC.VIBuffers {
		DBC.VIBuffers[i].Reset()
	}

	// ===============================
	// resize cache
	// ===============================
	DBC.ShouldDrawBgTile.Resize(board.Width, board.Height)
	DBC.ShouldDrawTile.Resize(board.Width, board.Height)
	DBC.ShouldDrawFgTile.Resize(board.Width, board.Height)

	DBC.BgTileRects.Resize(board.Width, board.Height)

	DBC.TileStrokeRects.Resize(board.Width, board.Height)
	DBC.TileFillRects.Resize(board.Width, board.Height)

	DBC.TileRoundness.Resize(board.Width, board.Height)
	DBC.TileFirmlyPlaced.Resize(board.Width, board.Height)

	// ===============================
	// recalculate cache
	// ===============================
	{
		tileSizeW, tileSizeH := GetBoardTileSize(boardRect, board.Width, board.Height)

		for iter.HasNext() {
			x, y := iter.GetNext()

			ogTileRect := GetBoardTileRect(boardRect, board.Width, board.Height, x, y)
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

						if 0 <= rx && rx < board.Width && 0 <= ry && ry < board.Height {
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
	flagSpriteBuf := DBC.VIBuffers[2]

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

		fgRect := DBC.TileFillRects.Get(x, y)
		fgColor := style.FgColor

		if style.FgType == TileFgTypeNumber {
			count := board.GetNeighborMineCount(x, y)
			if 1 <= count && count <= 8 {
				VIaddSubViewInRect(
					spriteBuf,
					fgRect,
					style.FgScale,
					style.FgOffsetX, style.FgOffsetY,
					modColor(fgColor, style.FgAlpha, style.Highlight, ColorFgHighLight),
					GetNumberTile(count),
				)
			}
		} else if style.FgType == TileFgTypeFlag {
			VIaddSubViewInRect(
				flagSpriteBuf,
				fgRect,
				style.FgScale,
				style.FgOffsetX, style.FgOffsetY,
				modColor(fgColor, style.FgAlpha, style.Highlight, ColorFgHighLight),
				GetFlagTile(style.FlagAnim),
			)
		}
	}

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

		// draw waterRenderTarget
		BeginAntiAlias(false)
		BeginFilter(eb.FilterNearest)
		BeginMipMap(false)
		DrawImage(dst, waterRenderTarget, nil)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	}

	// flush sprites
	{
		BeginAntiAlias(false)
		BeginFilter(eb.FilterLinear)
		BeginMipMap(true)
		op := &DrawTrianglesOptions{}
		op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
		DrawTriangles(dst, spriteBuf.Vertices, spriteBuf.Indices, TileSprite.Image, op)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	}

	// flush flag sprites
	{
		BeginAntiAlias(false)
		BeginFilter(eb.FilterLinear)
		BeginMipMap(true)
		op := &DrawTrianglesOptions{}
		op.ColorScaleMode = eb.ColorScaleModePremultipliedAlpha
		DrawTriangles(dst, flagSpriteBuf.Vertices, flagSpriteBuf.Indices, FlagAnimSprite.Image, op)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	}
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
	) bool {
		if gameState != GameStatePlaying {
			hlTiles = hlTiles[:0]
			prevHlTiles = prevHlTiles[:0]

			return prevGameState != gameState
		}

		prevHlTiles = prevHlTiles[:len(hlTiles)]
		copy(prevHlTiles, hlTiles)

		hlTiles = hlTiles[:0]

		ms := GetMouseState(board, boardRect)

		goWide := interaction == InteractionTypeCheck
		goWide = goWide && board.IsPosInBoard(ms.BoardX, ms.BoardY)
		goWide = goWide && prevBoard.Revealed.Get(ms.BoardX, ms.BoardY)
		goWide = goWide && !stateChanged

		if goWide {
			hlWide = true
		}

		pressingCheck := (ms.PressedL && ms.PressedR) || ms.PressedM

		if !pressingCheck {
			hlWide = false
		}

		hlX = ms.BoardX
		hlY = ms.BoardY

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
		tileStyles Array2D[TileStyle],
	) bool {
		if gameState != GameStatePlaying { // only do this when we are actually playing
			return gameState != prevGameState
		}

		doRedraw := false

		ms := GetMouseState(board, boardRect)

		cursorOnFg := board.IsPosInBoard(ms.BoardX, ms.BoardY)
		cursorOnFg = cursorOnFg && tileStyles.Get(ms.BoardX, ms.BoardY).DrawFg

		if cursorOnFg && ms.JustPressedAny() {
			clickTimer.Current = clickTimer.Duration
			clickX = ms.BoardX
			clickY = ms.BoardY
			focused = true
			doRedraw = true
		}

		if !ms.PressedAny() || !(clickX == ms.BoardX && clickY == ms.BoardY) {
			focused = false
		}

		if !focused {
			if clickTimer.Current >= 0 {
				doRedraw = true
			}
			clickTimer.TickDown()
		}

		if clickTimer.Current > 0 {
			if board.IsPosInBoard(clickX, clickY) {
				tileStyles.Data[clickX+clickY*tileStyles.Width].FgScale *= 1 + clickTimer.Normalize()*0.07
			}
		}

		return doRedraw
	}
}

func (g *Game) QueueRevealAnimation(revealsBefore, revealsAfter Array2D[bool], originX, originy int) {
	g.SkipAllAnimations()

	fw, fh := f64(g.board.Width-1), f64(g.board.Height-1)

	originP := FPt(f64(originX), f64(originy))

	maxDist := math.Sqrt(fw*fw + fh*fh)

	const maxDuration = time.Millisecond * 900
	const minDuration = time.Millisecond * 20

	soundEffectCounter := 0

	for x := range g.board.Width {
		for y := range g.board.Height {
			if !revealsBefore.Get(x, y) && revealsAfter.Get(x, y) {
				pos := FPt(f64(x), f64(y))
				dist := pos.Sub(originP).Length()
				d := time.Duration(f64(maxDuration) * (dist / maxDist))

				var timer Timer

				timer.Duration = max(d, minDuration)
				timer.Current = 0

				targetStyle := GetAnimationTargetTileStyle(g.board, x, y)

				var anim CallbackAnimation
				anim.Tag = AnimationTagTileReveal

				playedSound := false

				if soundEffectCounter%5 != 0 {
					playedSound = true
				}
				soundEffectCounter++

				anim.Update = func() {
					style := g.BaseTileStyles.Get(x, y)
					timer.TickUp()

					t := timer.Normalize()
					t = t * t

					const limit = 0.4

					if t > limit {
						style = targetStyle

						t = ((t - limit) / (1 - limit))
						t = Clamp(t, 0, 1)

						style.TileScale = Lerp(1.2, 1.0, t)

						if !playedSound {
							playedSound = true
							g.TileRevealSoundPlayers.Play()
						}
					} else {
						style.DrawTile = false
						style.DrawFg = false
					}

					g.BaseTileStyles.Set(x, y, style)
				}

				anim.Skip = func() {
					playedSound = true
					timer.Current = timer.Duration
					anim.Update()
				}

				anim.Done = func() bool {
					return timer.Current >= timer.Duration
				}

				anim.AfterDone = func() {
					// always play sound at the end
					// if you didn't play sound before
					if soundEffectCounter >= 2 {
						g.TileRevealSoundPlayers.Play()
					}
					soundEffectCounter = 0
				}

				g.TileAnimations.Get(x, y).Enqueue(anim)
			}
		}
	}
}

func (g *Game) QueueAddFlagAnimation(flagX, flagY int) {
	g.SkipAllAnimations()

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

		style.FlagAnim = t

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

		style.FlagAnim = 1

		g.BaseTileStyles.Set(flagX, flagY, style)
	}

	g.TileAnimations.Get(flagX, flagY).Enqueue(anim)
}

func (g *Game) QueueRemoveFlagAnimation(flagX, flagY int) {
	g.SkipAllAnimations()

	var anim CallbackAnimation
	anim.Tag = AnimationTagRemoveFlag

	done := false

	anim.Update = func() {
		style := g.BaseTileStyles.Get(flagX, flagY)

		style.DrawFg = false
		style.FgType = TileFgTypeNone

		velocityX := RandF(0.02, 0.06)
		if rand.IntN(100) > 50 {
			velocityX *= -1
		}

		velocityY := RandF(-0.25, -0.3)

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
			GravityX:    0, GravityY: 0.02,
		}

		g.Particles = AppendTileParticle(g.Particles, p)

		done = true

		g.BaseTileStyles.Set(flagX, flagY, style)
	}

	anim.Skip = func() {
		anim.Update()
	}

	anim.Done = func() bool {
		return done
	}

	g.TileAnimations.Get(flagX, flagY).Enqueue(anim)
}

func (g *Game) QueueDefeatAnimation(originX, originY int) {
	g.SkipAllAnimations()

	var minePoses []image.Point

	for x := range g.board.Width {
		for y := range g.board.Height {
			if g.board.Mines.Get(x, y) && !g.board.Flags.Get(x, y) {
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

		playedSound := false

		anim.Update = func() {
			style := g.BaseTileStyles.Get(p.X, p.Y)

			if timer.Current > 0 && !playedSound {
				playedSound = true
				g.BombSoundPlayers.Play()
			}

			timer.TickUp()
			style.BgBombAnim = timer.Normalize()

			g.BaseTileStyles.Set(p.X, p.Y, style)
		}

		anim.Skip = func() {
			timer.Current = timer.Duration
			playedSound = true
			anim.Update()
		}

		anim.Done = func() bool {
			return timer.Current >= timer.Duration
		}

		anim.AfterDone = func() {
		}

		g.TileAnimations.Get(p.X, p.Y).Enqueue(anim)
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

func GetNumberTile(number int) SubView {
	if !(1 <= number && number <= 8) {
		ErrLogger.Fatalf("%d is not a valid number", number)
	}

	return SpriteSubView(TileSprite, number-1)
}

func GetMineTile() SubView {
	return SpriteSubView(TileSprite, 9)
}

func GetFlagTile(animT float64) SubView {
	frame := int(math.Round(animT * f64(FlagAnimSprite.Count-1)))
	frame = Clamp(frame, 0, FlagAnimSprite.Count-1)
	return SpriteSubView(FlagAnimSprite, frame)
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

	roundCount := 0

	for _, round := range isRound {
		if round {
			roundCount++
		}
	}

	switch roundCount {
	case 0:
		return SpriteSubView(TileSprite, 20), d0
	case 1:
		if isRound[0] {
			return SpriteSubView(TileSprite, 21), d0
		}
		if isRound[1] {
			return SpriteSubView(TileSprite, 21), d90
		}
		if isRound[2] {
			return SpriteSubView(TileSprite, 21), d180
		}
		if isRound[3] {
			return SpriteSubView(TileSprite, 21), d270
		}
	case 2:
		if !isRound[0] && !isRound[1] {
			return SpriteSubView(TileSprite, 22), d180
		}
		if !isRound[1] && !isRound[2] {
			return SpriteSubView(TileSprite, 22), d270
		}
		if !isRound[2] && !isRound[3] {
			return SpriteSubView(TileSprite, 22), d0 // d360
		}
		if !isRound[3] && !isRound[0] {
			return SpriteSubView(TileSprite, 22), d90 // d450
		}
	case 3:
		if !isRound[0] {
			return SpriteSubView(TileSprite, 23), d90
		}
		if !isRound[1] {
			return SpriteSubView(TileSprite, 23), d180
		}
		if !isRound[2] {
			return SpriteSubView(TileSprite, 23), d270
		}
		if !isRound[3] {
			return SpriteSubView(TileSprite, 23), d0 // d360
		}
	case 4:
		return SpriteSubView(TileSprite, 24), d0
	default:
		panic("UNREACHABLE")
	}

	return SpriteSubView(TileSprite, 24), d0
}

func GetAllRoundTile() SubView {
	return SpriteSubView(TileSprite, 24)
}

func GetRectTile() SubView {
	return SpriteSubView(TileSprite, 20)
}

func MousePosToBoardPos(board Board, boardRect FRectangle, mousePos FPoint) (int, int) {
	mousePos.X -= boardRect.Min.X
	mousePos.Y -= boardRect.Min.Y

	boardX := int(math.Floor(mousePos.X / (boardRect.Dx() / float64(board.Width))))
	boardY := int(math.Floor(mousePos.Y / (boardRect.Dy() / float64(board.Height))))

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

	imgRect := WaterShaderImage1.Bounds()
	imgFRect := RectToFRect(imgRect)

	op.GeoM.Scale(rect.Dx()/imgFRect.Dx(), rect.Dy()/imgFRect.Dy())
	op.GeoM.Translate(rect.Min.X, rect.Min.Y)

	DrawRectShader(dst, imgRect.Dx(), imgRect.Dy(), WaterShader, op)
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
	if !g.placedMinesOnBoard {
		g.board.PlaceMines(g.mineCount, g.board.Width-1, g.board.Height-1)
	}
	g.placedMinesOnBoard = true

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
}

// =====================
// RetryButton
// =====================

type RetryButton struct {
	BaseButton

	ButtonHoverOffset float64

	// water stuff
	DoWaterEffect bool

	WaterAlpha      float64
	WaterFlowOffset time.Duration

	WaterRenderTarget *eb.Image
}

func NewRetryButton() *RetryButton {
	rb := new(RetryButton)

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

	if rb.State == ButtonStateDown {
		topRect = FRectMoveTo(topRect, bottomRect.Min.X, bottomRect.Min.Y)
	} else if rb.State == ButtonStateHover {
		topRect = topRect.Add(FPt(0, -topRect.Dy()*0.025*rb.ButtonHoverOffset))
	}

	const segments = 6
	const radius = 0.4

	renderTarget := dst

	if rb.DoWaterEffect {
		rb.WaterRenderTarget.Clear()
		renderTarget = rb.WaterRenderTarget
	}

	colorT := Clamp(rb.WaterAlpha, 0, 1)
	if !rb.DoWaterEffect {
		colorT = 0
	}
	color1 := LerpColorRGBA(ColorRetryA1, ColorRetryB1, colorT)
	color2 := LerpColorRGBA(ColorRetryA2, ColorRetryB2, colorT)
	color3 := LerpColorRGBA(ColorRetryA3, ColorRetryB3, colorT)
	color4 := LerpColorRGBA(ColorRetryA4, ColorRetryB4, colorT)

	FillRoundRectFast(
		renderTarget,
		bottomRect,
		radius,
		false,
		segments,
		color1,
		//color.NRGBA{0, 0, 0, 255},
	)

	FillRoundRectFast(
		renderTarget,
		topRect,
		radius,
		false,
		segments,
		color2,
		//color.NRGBA{105, 223, 145, 255},
	)

	if rb.DoWaterEffect {
		rect := bottomRect.Union(topRect)
		rect = rect.Inset(-3)

		colors := [4]color.Color{
			ColorRetryWater1,
			ColorRetryWater2,
			ColorRetryWater3,
			ColorRetryWater4,
		}

		for i, c := range colors {
			nrgba := ColorToNRGBA(c)
			colors[i] = color.NRGBA{nrgba.R, nrgba.G, nrgba.B, uint8(f64(nrgba.A) * rb.WaterAlpha)}
		}

		BeginBlend(eb.BlendSourceAtop)
		DrawWaterRect(
			rb.WaterRenderTarget,
			rect,
			GlobalTimerNow()+rb.WaterFlowOffset,
			colors,
			FPt(0, 0),
		)
		EndBlend()

		// draw waterRenderTarget
		BeginAntiAlias(false)
		BeginFilter(eb.FilterNearest)
		BeginMipMap(false)
		DrawImage(dst, rb.WaterRenderTarget, nil)
		EndMipMap()
		EndAntiAlias()
		EndFilter()
	}

	imgRect := RectToFRect(RetryButtonImage.Bounds())
	scale := min(topRect.Dx(), topRect.Dy()) / max(imgRect.Dx(), imgRect.Dy())
	scale *= 0.6

	center := FRectangleCenter(topRect)

	op := &DrawImageOptions{}
	op.GeoM.Concat(TransformToCenter(imgRect.Dx(), imgRect.Dy(), scale, scale, 0))
	op.GeoM.Translate(center.X, center.Y-topRect.Dy()*0.02)
	//op.ColorScale.ScaleWithColor(color.NRGBA{0, 0, 0, 255})
	op.ColorScale.ScaleWithColor(color3)

	DrawImage(dst, RetryButtonImage, op)

	op.GeoM.Translate(0, topRect.Dy()*0.02*2)
	op.ColorScale.Reset()
	//op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
	op.ColorScale.ScaleWithColor(color4)

	DrawImage(dst, RetryButtonImage, op)
}
