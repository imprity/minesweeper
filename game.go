package main

import (
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	_ "github.com/silbinarywolf/preferdiscretegpu"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
)

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

	rp.Button = NewImageButton()

	rp.Button.Image = SpriteSubView(TileSprite, 17)
	rp.Button.ImageOnHover = SpriteSubView(TileSprite, 17)
	rp.Button.ImageOnDown = SpriteSubView(TileSprite, 17)

	rp.Button.ImageColor = color.NRGBA{0, 255, 0, 100}
	rp.Button.ImageColorOnHover = color.NRGBA{0, 255, 0, 255}
	rp.Button.ImageColorOnDown = color.NRGBA{0, 255, 0, 255}

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

			textW, textH := ebt.Measure(text, DecoFace, FontLineSpacing(DecoFace))
			textRect := rp.TextRect()
			scale := min(textRect.Dx()/textW, textRect.Dy()/textH)

			op := &ebt.DrawOptions{}
			op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
			center := FRectangleCenter(textRect)
			op.GeoM.Translate(center.X, center.Y)
			op.ColorScale.ScaleWithColor(color.NRGBA{0, 0, 0, 255})
			op.Filter = eb.FilterLinear

			ebt.Draw(dst, text, DecoFace, op)
		}

		// draw button
		rp.Button.Draw(dst)
	}
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
		ct.ColorPicker.SetColor(ColorTable[ct.TableIndex])
	}
	ct.wasShowing = ct.DoShow

	ct.ColorPicker.Rect = FRectWH(200, 400)
	ct.ColorPicker.Rect = FRectMoveTo(ct.ColorPicker.Rect, ScreenWidth-210, 10)
	ct.ColorPicker.Update()

	const firstRate = 200 * time.Millisecond
	const repeatRate = 50 * time.Millisecond
	changed := false

	if HandleKeyRepeat(firstRate, repeatRate, eb.KeyW) {
		ct.TableIndex--
		changed = true
	}
	if HandleKeyRepeat(firstRate, repeatRate, eb.KeyS) {
		ct.TableIndex++
		changed = true
	}
	ct.TableIndex = Clamp(ct.TableIndex, 0, ColorTableSize-1)

	if changed {
		ct.ColorPicker.SetColor(ColorTable[ct.TableIndex])
	}

	ColorTable[ct.TableIndex] = ct.ColorPicker.Color()
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
			dst, FRectWH(bgWidth, bgHeight), color.NRGBA{0, 0, 0, 150}, true,
		)

		// draw list texts
		offsetY := float64(0)

		for i := ColorTableIndex(0); i < ColorTableSize; i++ {
			text := i.String()
			op := &ebt.DrawOptions{}

			op.GeoM.Scale(textScale, textScale)
			op.GeoM.Translate(0, offsetY)
			if i == ct.TableIndex {
				op.ColorScale.ScaleWithColor(color.NRGBA{255, 0, 0, 255})
			} else {
				op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
			}
			op.Filter = eb.FilterLinear

			ebt.Draw(dst, text, ClearFace, op)

			offsetY += lineSpacing * textScale
		}
	}
}

type Game struct {
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

	DebugMode       bool
	ColorPickerMode bool

	ColorTablePicker *ColorTablePicker
}

func NewGame() *Game {
	g := new(Game)

	g.RetryPopup = NewRetryPopup()

	g.ColorTablePicker = NewColorTablePicker()

	g.TopMenuShowAnimTimer = Timer{
		Duration: time.Millisecond * 200,
	}
	g.TopMenuShowAnimTimer.Current = g.TopMenuShowAnimTimer.Duration

	g.HighlightDuraiton = time.Millisecond * 100

	g.MineCount = [DifficultySize]int{
		10, 20, 30,
	}
	g.BoardTileCount = [DifficultySize]int{
		10, 15, 20,
	}

	g.BoardSizeRatio = 0.8

	g.TopUIMarginHorizontal = 5
	g.TopUIMarginTop = 5
	g.TopUIMarginBottom = 5

	g.TopUIButtonButtonRatio = 0.2
	g.TopUIButtonTextRatio = 0.5

	initBoard := func() {
		g.Board = NewBoard(
			g.BoardTileCount[g.Difficulty],
			g.BoardTileCount[g.Difficulty],
		)

		g.TileHighLights = New2DArray[TileHighLight](g.Board.Width, g.Board.Height)
		g.BoardTouched = false
	}
	initBoard()

	g.RetryPopup.RegisterButtonCallback(func() {
		initBoard()
		g.GameState = GameStatePlaying
	})

	// ==============================
	// create difficulty buttons
	// ==============================
	{
		leftRect := g.GetDifficultyButtonRect(false)
		rightRect := g.GetDifficultyButtonRect(true)

		// DifficultyButtonLeft
		g.DifficultyButtonLeft = NewImageButton()

		g.DifficultyButtonLeft.Rect = leftRect
		g.DifficultyButtonLeft.OnClick = func() {
			g.Difficulty -= 1
			g.Difficulty = max(g.Difficulty, 0)
			initBoard()
		}

		g.DifficultyButtonLeft.Image = SpriteSubView(TileSprite, 11)
		g.DifficultyButtonLeft.ImageOnHover = SpriteSubView(TileSprite, 11)
		g.DifficultyButtonLeft.ImageOnDown = SpriteSubView(TileSprite, 13)

		g.DifficultyButtonLeft.ImageColor = ColorTopUIButton
		g.DifficultyButtonLeft.ImageColorOnHover = ColorTopUIButtonOnHover
		g.DifficultyButtonLeft.ImageColorOnDown = ColorTopUIButtonOnDown

		// DifficultyButtonRight
		g.DifficultyButtonRight = NewImageButton()

		g.DifficultyButtonRight.Rect = rightRect
		g.DifficultyButtonRight.OnClick = func() {
			g.Difficulty += 1
			g.Difficulty = min(g.Difficulty, DifficultySize-1)
			initBoard()
		}

		g.DifficultyButtonRight.Image = SpriteSubView(TileSprite, 12)
		g.DifficultyButtonRight.ImageOnHover = SpriteSubView(TileSprite, 12)
		g.DifficultyButtonRight.ImageOnDown = SpriteSubView(TileSprite, 14)

		g.DifficultyButtonRight.ImageColor = ColorTopUIButton
		g.DifficultyButtonRight.ImageColorOnHover = ColorTopUIButtonOnHover
		g.DifficultyButtonRight.ImageColorOnDown = ColorTopUIButtonOnDown
	}

	return g
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

func (g *Game) GetTopUIRect() FRectangle {
	boardRect := g.BoardRect()
	return FRect(
		boardRect.Min.X+g.TopUIMarginHorizontal,
		g.TopUIMarginTop,
		boardRect.Max.X-g.TopUIMarginHorizontal,
		boardRect.Min.Y-g.TopUIMarginBottom,
	)
}

func (g *Game) GetDifficultyButtonRect(forRight bool) FRectangle {
	parentRect := g.GetTopUIRect()
	width := parentRect.Dx() * g.TopUIButtonButtonRatio

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

func (g *Game) GetDifficultyTextRect() FRectangle {
	parentRect := g.GetTopUIRect()
	width := parentRect.Dx() * g.TopUIButtonTextRatio

	rect := FRectWH(width, parentRect.Dy())

	pCenter := FRectangleCenter(parentRect)
	rect = CenterFRectangle(rect, pCenter.X, pCenter.Y)

	return rect
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

func (g *Game) SetTileHightlight(boardX, boardY int) {
	g.TileHighLights[boardX][boardY].Brightness = 1
}

func (g *Game) Update() error {
	cursor := CursorFPt()

	boardX, boardY := g.MousePosToBoardPos(cursor)

	prevState := g.GameState
	_ = prevState // might be handy later

	// =================================
	// handle board interaction
	// =================================
	if g.GameState == GameStatePlaying && boardX >= 0 && boardY >= 0 {
		if g.Board.Revealed[boardX][boardY] {
			pressedL := IsMouseButtonPressed(eb.MouseButtonLeft)
			justPressedL := IsMouseButtonJustPressed(eb.MouseButtonLeft)

			pressedR := IsMouseButtonPressed(eb.MouseButtonRight)
			justPressedR := IsMouseButtonJustPressed(eb.MouseButtonRight)

			if (justPressedL && pressedR) || (justPressedR && pressedL) {
				flagCount := g.Board.GetNeighborFlagCount(boardX, boardY)
				mineCount := g.Board.GetNeighborMineCount(boardX, boardY)

				if flagCount == mineCount { // check if flagged correctly
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

					if flaggedCorrectly {
						iter.Reset()
						for iter.HasNext() {
							x, y := iter.GetNext()
							if g.Board.IsPosInBoard(x, y) {
								g.Board.SpreadSafeArea(x, y)
							}
						}
					} else {
						if missedMine {
							g.GameState = GameStateLost
						}
					}
				} else { // if not flagged correctly just highlight the area
					iter := NewBoardIterator(boardX-1, boardY-1, boardX+1, boardY+1)

					for iter.HasNext() {
						x, y := iter.GetNext()

						if g.Board.IsPosInBoard(x, y) && !g.Board.Revealed[x][y] && !g.Board.Flags[x][y] {
							g.SetTileHightlight(x, y)
						}
					}
				}
			}
		} else { // not revealed
			if IsMouseButtonJustPressed(eb.MouseButtonLeft) {
				if !g.Board.Flags[boardX][boardY] {
					if !g.BoardTouched { // mine is not placed
						g.Board.PlaceMines(g.MineCount[g.Difficulty], boardX, boardY)
						g.Board.SpreadSafeArea(boardX, boardY)

						iter := NewBoardIterator(0, 0, g.Board.Width-1, g.Board.Height-1)
						// remove any flags that might have been placed
						for iter.HasNext() {
							x, y := iter.GetNext()
							g.Board.Flags[x][y] = false
						}
						g.BoardTouched = true
					} else { // mine has been placed
						if !g.Board.Mines[boardX][boardY] {
							g.Board.SpreadSafeArea(boardX, boardY)
						} else {
							g.GameState = GameStateLost
						}
					}
				}
			} else if IsMouseButtonJustPressed(eb.MouseButtonRight) {
				g.Board.Flags[boardX][boardY] = !g.Board.Flags[boardX][boardY]
			}
		}

		justPressedL := IsMouseButtonJustPressed(eb.MouseButtonLeft)
		justPressedR := IsMouseButtonJustPressed(eb.MouseButtonRight)

		if justPressedL || justPressedR {
			// check if user has won the game
			if g.Board.IsAllSafeTileRevealed() {
				g.GameState = GameStateWon
			}
		}
	}

	// update highlights
	if !(IsMouseButtonPressed(eb.MouseButtonLeft) || IsMouseButtonPressed(eb.MouseButtonRight)) {
		for y := 0; y < g.Board.Height; y++ {
			for x := 0; x < g.Board.Width; x++ {
				g.TileHighLights[x][y].Brightness -= f64(UpdateDelta()) / f64(g.HighlightDuraiton)
				g.TileHighLights[x][y].Brightness = max(g.TileHighLights[x][y].Brightness, 0)
			}
		}
	}

	// ==========================
	// update TopMenuShowAnimT
	// ==========================
	if g.BoardTouched {
		g.TopMenuShowAnimTimer.TickDown()
	} else {
		g.TopMenuShowAnimTimer.TickUp()
	}

	// ==========================
	// update top menu buttons
	// ==========================
	g.DifficultyButtonLeft.Disabled = g.BoardTouched
	g.DifficultyButtonRight.Disabled = g.BoardTouched

	// update button rect
	{
		lRect := g.GetDifficultyButtonRect(false)
		rRect := g.GetDifficultyButtonRect(true)

		t := f64(g.TopMenuShowAnimTimer.Current) / f64(g.TopMenuShowAnimTimer.Duration)
		t = Clamp(t, 0, 1)

		lRectY := Lerp(-lRect.Dy()-10, lRect.Min.Y, t)
		rRectY := Lerp(-rRect.Dy()-10, rRect.Min.Y, t)

		lRect = FRectMoveTo(lRect, lRect.Min.X, lRectY)
		rRect = FRectMoveTo(rRect, rRect.Min.X, rRectY)

		g.DifficultyButtonLeft.Rect = lRect
		g.DifficultyButtonRight.Rect = rRect
	}

	g.DifficultyButtonLeft.Update()
	g.DifficultyButtonRight.Update()
	// ==========================

	g.RetryPopup.DoShow = g.GameState == GameStateLost || g.GameState == GameStateWon
	g.RetryPopup.DidWin = g.GameState == GameStateWon

	g.RetryPopup.Update()

	// ==========================
	// debug mode
	// ==========================
	if IsKeyJustPressed(eb.KeyF1) {
		g.DebugMode = !g.DebugMode
	}
	if IsKeyJustPressed(eb.KeyF10) {
		g.SetDebugBoardForDecoration()
	}

	// ==========================
	// color table picker
	// ==========================
	if IsKeyJustPressed(eb.KeyF2) {
		g.ColorTablePicker.DoShow = !g.ColorTablePicker.DoShow
	}
	g.ColorTablePicker.Update()

	return nil
}

func (g *Game) SetDebugBoardForDecoration() {
	//MINE_COUNT doesn't really matter here
	g.Board = NewBoard(15, 15)
	g.TileHighLights = New2DArray[TileHighLight](g.Board.Width, g.Board.Height)

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

func (g *Game) GetTileRect(boardX, boardY int) FRectangle {
	boardRect := g.BoardRect()

	tileWidth := boardRect.Dx() / f64(g.Board.Width)
	tileHeight := boardRect.Dy() / f64(g.Board.Height)

	return FRectangle{
		Min: FPt(f64(boardX)*tileWidth, f64(boardY)*tileHeight).Add(boardRect.Min),
		Max: FPt(f64(boardX+1)*tileWidth, f64(boardY+1)*tileHeight).Add(boardRect.Min),
	}
}

func (g *Game) DrawTile(dst *eb.Image, boardX, boardY int, tile SubView, clr color.Color) {
	tileRect := g.GetTileRect(boardX, boardY)

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

func (g *Game) DrawDifficultyText(dst *eb.Image) {
	var maxW, maxH float64
	var textW, textH float64

	// TODO : cache this if you can
	for d := Difficulty(0); d < DifficultySize; d++ {
		str := DifficultyStrs[d]
		w, h := ebt.Measure(str, DecoFace, FontLineSpacing(DecoFace))
		maxW = max(w, maxW)
		maxH = max(h, maxH)

		if d == g.Difficulty {
			textW, textH = w, h
		}
	}

	rect := g.GetDifficultyTextRect()

	t := f64(g.TopMenuShowAnimTimer.Current) / f64(g.TopMenuShowAnimTimer.Duration)
	t = Clamp(t, 0, 1)

	rectY := Lerp(-rect.Dy()-10, rect.Min.Y, t)
	rect = FRectMoveTo(rect, rect.Min.X, rectY)

	scale := min(rect.Dx()/maxW, rect.Dy()/maxH)

	rectCenter := FRectangleCenter(rect)

	op := &ebt.DrawOptions{}
	op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)

	op.Filter = eb.FilterLinear

	op.ColorScale.ScaleWithColor(ColorTable[ColorTopUITitle])

	ebt.Draw(dst, DifficultyStrs[g.Difficulty], DecoFace, op)
}

func (g *Game) DrawRoundBoard(dst *eb.Image) {
	isRevealed := func(x, y int) bool {
		if !g.Board.IsPosInBoard(x, y) {
			return false
		}
		return g.Board.Revealed[x][y]
	}

	const radius = 0.8

	isOddTile := func(x, y int) bool {
		index := x + g.Board.Height*y
		if g.Board.Width%2 == 0 {
			if y%2 == 0 {
				return index%2 != 0
			} else {
				return index%2 == 0
			}
		} else {
			return index%2 != 0
		}
	}

	iter := NewBoardIterator(0, 0, g.Board.Width, g.Board.Height)
	for iter.HasNext() {
		x, y := iter.GetNext()

		boardTileRect := g.GetTileRect(x, y)
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

		radiusPx := min(rect.Dx()*0.5, rect.Dy()*0.5) * radius

		halfDx := rect.Dx() * 0.5
		halfDy := rect.Dy() * 0.5

		cornerRects := [4]FRectangle{
			FRectXYWH(rect.Min.X, rect.Min.Y, halfDx, halfDy),
			FRectXYWH(rect.Min.X+halfDx, rect.Min.Y, halfDx, halfDy),
			FRectXYWH(rect.Min.X+halfDx, rect.Min.Y+halfDx, halfDx, halfDy),
			FRectXYWH(rect.Min.X, rect.Min.Y+halfDx, halfDx, halfDy),
		}

		isOddCorner := func(corner int) bool {
			if isOddTile(x, y) {
				return corner%2 == 0
			} else {
				return corner%2 != 0
			}
		}

		cornerColor := func(corner int) color.Color {
			if isOddCorner(corner) {
				return ColorTileRevealed1
			} else {
				return ColorTileRevealed2
			}
		}

		cornerVerts := [4]FPoint{
			rect.Min,
			{rect.Max.X, rect.Min.Y},
			rect.Max,
			{rect.Min.X, rect.Max.Y},
		}

		// +--1--+
		// |     |
		// 0     2
		// |     |
		// +--3--+
		centerVerts := [4]FPoint{
			{rect.Min.X, rect.Min.Y + halfDy},
			{rect.Min.X + halfDx, rect.Min.Y},
			{rect.Max.X, rect.Min.Y + halfDy},
			{rect.Min.X + halfDx, rect.Max.Y},
		}

		getConcaveOrSharpCornersRectPath := func(
			isConcave [4]bool,
			concaveRadius [4]float64,
		) *ebv.Path {
			p := &ebv.Path{}

			p.MoveTo(f32(centerVerts[0].X), f32(centerVerts[0].Y))

			for i := range 4 {
				next := (i + 1) % 4
				if isConcave[i] {
					var arcCenter FPoint
					{
						rectCenter := FRectangleCenter(rect)
						switch i {
						case 0:
							arcCenter.X = rectCenter.X - concaveRadius[i]
							arcCenter.Y = rectCenter.Y - concaveRadius[i]
						case 1:
							arcCenter.X = rectCenter.X + concaveRadius[i]
							arcCenter.Y = rectCenter.Y - concaveRadius[i]
						case 2:
							arcCenter.X = rectCenter.X + concaveRadius[i]
							arcCenter.Y = rectCenter.Y + concaveRadius[i]
						case 3:
							arcCenter.X = rectCenter.X - concaveRadius[i]
							arcCenter.Y = rectCenter.Y + concaveRadius[i]
						}
					}

					startAngle := Pi*0.5 + Pi*0.5*f32(i)
					endAngle := startAngle - Pi*0.5
					p.Arc(f32(arcCenter.X), f32(arcCenter.Y), f32(concaveRadius[i]), startAngle, endAngle, ebv.CounterClockwise)
					p.LineTo(f32(centerVerts[next].X), f32(centerVerts[next].Y))
				} else {
					p.LineTo(f32(cornerVerts[i].X), f32(cornerVerts[i].Y))
					p.LineTo(f32(centerVerts[next].X), f32(centerVerts[next].Y))
				}
			}

			p.Close()

			return p
		}

		drawCorner := func(corner int) {
			switch corner {
			case 0:
				cornerRect := cornerRects[corner]
				DrawFilledRoundRectEx(dst, cornerRect,
					[4]float64{0, 0, radiusPx, 0},
					cornerColor(corner), true,
				)
			case 1:
				cornerRect := cornerRects[corner]
				DrawFilledRoundRectEx(dst, cornerRect,
					[4]float64{0, 0, 0, radiusPx},
					cornerColor(corner), true,
				)
			case 2:
				cornerRect := cornerRects[corner]
				DrawFilledRoundRectEx(dst, cornerRect,
					[4]float64{radiusPx, 0, 0, 0},
					cornerColor(corner), true,
				)
			case 3:
				cornerRect := cornerRects[corner]
				DrawFilledRoundRectEx(dst, cornerRect,
					[4]float64{0, radiusPx, 0, 0},
					cornerColor(corner), true,
				)
			}
		}

		switch revealCount {
		case 0:
			// pass
		case 1:
			for i, v := range revealed {
				if v {
					drawCorner(i)
				}
			}
		case 2:
			{
				/*
					var isConcave [4]bool
					var concaveRadius [4]float64

					for i, v := range revealed {
						if !v {
							isConcave[i] = true
							concaveRadius[i] = radiusPx * 1.5
						}
					}
					p := getConcaveOrSharpCornersRectPath(
						isConcave, concaveRadius,
					)

					vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
					DrawVerticies(dst, vs, is, ColorTileRevealed1, true)
				*/
				for i, v := range revealed {
					if v {
						//DrawFilledRect(dst, cornerRects[i], cornerColor(i), true)
						drawCorner(i)
					}
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

			isConcave[unRevealed] = true
			concaveRadius[unRevealed] = radiusPx

			p := getConcaveOrSharpCornersRectPath(
				isConcave, concaveRadius,
			)

			vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
			DrawVerticies(dst, vs, is, cornerColor((unRevealed+1)%4), true)

			radiuses := [4]float64{}
			radiuses[unRevealed] = radiusPx

			DrawFilledRoundRectEx(dst, cornerRects[(unRevealed+2)%4], radiuses, cornerColor((unRevealed+2)%4), true)
			//DrawFilledRect(dst, cornerRects[(unRevealed+2)%4], cornerColor((unRevealed+2)%4), true)
		case 4:
			for i, v := range revealed {
				if v {
					DrawFilledRect(dst, cornerRects[i], cornerColor(i), true)
				}
			}
		}
	}
}

func (g *Game) Draw(dst *eb.Image) {
	// background
	dst.Fill(ColorTable[ColorBg])

	isOddTile := func(x, y int) bool {
		index := x + g.Board.Height*y
		if g.Board.Width%2 == 0 {
			if y%2 == 0 {
				return index%2 != 0
			} else {
				return index%2 == 0
			}
		} else {
			return index%2 != 0
		}
	}

	// ===========================
	// draw board
	// ===========================
	iter := NewBoardIterator(0, 0, g.Board.Width-1, g.Board.Height-1)

	// draw regular tile background
	for iter.HasNext() {
		x, y := iter.GetNext()
		tileRect := g.GetTileRect(x, y)

		if !g.Board.Revealed[x][y] {
			bgColor := ColorTable[ColorTileNormal1]
			if isOddTile(x, y) {
				bgColor = ColorTable[ColorTileNormal2]
			}

			DrawFilledRect(dst, tileRect, bgColor, true)
			StrokeRect(dst, tileRect, 1, ColorTable[ColorTileNormalStroke], true)
		}

		// draw highlight
		if g.TileHighLights[x][y].Brightness > 0 {
			t := g.TileHighLights[x][y].Brightness
			c := ColorTable[ColorTileHighLight]
			DrawFilledRect(
				dst,
				tileRect,
				color.NRGBA{c.R, c.G, c.B, uint8(f64(c.A) * t)},
				true,
			)
		}
	}

	// draw revealed tiles
	/*
		iter.Reset()
		for iter.HasNext() {
			x, y := iter.GetNext()
			tileRect := g.GetTileRect(x, y)

			if g.Board.Revealed[x][y] {
				bgColor := ColorTable[ColorTileRevealed1]
				if isOddTile(x, y) {
					bgColor = ColorTable[ColorTileRevealed2]
				}

				DrawFilledRect(dst, tileRect, bgColor, true)
				StrokeRect(dst, tileRect, 1, ColorTable[ColorTileRevealedStroke], true)
			}
		}
	*/
	g.DrawRoundBoard(dst)

	// draw foreground elements
	iter.Reset()
	for iter.HasNext() {
		x, y := iter.GetNext()

		// draw flags
		if g.Board.Flags[x][y] {
			g.DrawTile(dst, x, y, GetFlagTile(), ColorTable[ColorFlag])
		}

		// draw mines
		if g.GameState == GameStateLost && g.Board.Mines[x][y] && !g.Board.Flags[x][y] {
			g.DrawTile(dst, x, y, GetMineTile(), ColorTable[ColorMine])
		}

		// draw number
		if g.Board.Revealed[x][y] {
			if count := g.Board.GetNeighborMineCount(x, y); count > 0 {
				g.DrawTile(dst, x, y, GetNumberTile(count), ColorTableGetNumber(count))
			}
		}

		if g.DebugMode && g.Board.Mines[x][y] {
			g.DrawTile(dst, x, y, GetMineTile(), color.NRGBA{255, 0, 0, 255})
		}
	}

	g.DifficultyButtonLeft.Draw(dst)
	g.DifficultyButtonRight.Draw(dst)

	g.DrawDifficultyText(dst)

	g.RetryPopup.Draw(dst)

	g.ColorTablePicker.Draw(dst)
}

func (a *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
