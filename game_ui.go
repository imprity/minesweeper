package main

import (
	"image"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
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

type GameUI struct {
	Game *Game

	Difficulty Difficulty

	MineCounts      [DifficultySize]int         // constant
	BoardTileCounts [DifficultySize]image.Point // constant
	BoardSizeRatios [DifficultySize]float64     // constant, relative to ScreenHeight

	TopUI *TopUI

	TopUIHeight float64 // constant, relative to ScreenHeight

	TopUIInsetTop float64 // constant

	ResourceEditor *ResourceEditor
}

func NewGameUI() *GameUI {
	gu := new(GameUI)

	// set constants
	gu.MineCounts = [DifficultySize]int{10, 30, 70}
	gu.BoardTileCounts = [DifficultySize]image.Point{
		image.Pt(10, 10), image.Pt(15, 13), image.Pt(20, 20),
	}
	gu.BoardSizeRatios = [DifficultySize]float64{0.6, 0.75, 0.85}

	gu.TopUIHeight = 0.09
	gu.TopUIInsetTop = 5

	gu.Game = NewGame(
		gu.BoardTileCounts[DifficultyEasy].X, gu.BoardTileCounts[DifficultyEasy].Y,
		gu.MineCounts[DifficultyEasy],
	)
	gu.TopUI = NewTopUI()
	gu.TopUI.OnDifficultyChange = func(newDifficulty Difficulty) {
		gu.Difficulty = newDifficulty
		gu.Game.MineCount = gu.MineCounts[newDifficulty]
		gu.Game.ResetBoard(gu.BoardTileCounts[newDifficulty].X, gu.BoardTileCounts[newDifficulty].Y)
		gu.Game.Rect = gu.BoardRect(newDifficulty)
	}

	gu.ResourceEditor = NewResourceEditor()

	return gu
}

func (gu *GameUI) Update() {
	gu.Game.Rect = gu.BoardRect(gu.Difficulty)
	gu.Game.RetryButtonSize = gu.BoardSize()
	gu.TopUI.Rect = gu.TopUIRect()

	gu.Game.Update()
	gu.TopUI.Update()

	if IsKeyJustPressed(ShowResourceEditorKey) {
		gu.ResourceEditor.DoShow = !gu.ResourceEditor.DoShow
	}
	gu.ResourceEditor.Update()
}

func (gu *GameUI) Draw(dst *eb.Image) {
	gu.Game.Draw(dst)

	gu.TopUI.Draw(dst)

	gu.ResourceEditor.Draw(dst)
}

func (gu *GameUI) Layout(outsideWidth, outsideHeight int) {
	gu.Game.Layout(outsideWidth, outsideHeight)
}

func (gu *GameUI) BoardRect(difficulty Difficulty) FRectangle {
	parentRect := FRectWH(
		ScreenWidth, ScreenHeight,
	)

	var boardTileWidth, boardTileHeight int

	boardTileWidth = gu.BoardTileCounts[difficulty].X
	boardTileHeight = gu.BoardTileCounts[difficulty].Y

	maxSize := min(parentRect.Dx(), parentRect.Dy()) * gu.BoardSizeRatios[difficulty]

	var boardWidth, boardHeight float64

	if boardTileWidth > boardTileHeight {
		boardWidth = maxSize
		boardHeight = maxSize * f64(boardTileHeight) / f64(boardTileWidth)
	} else {
		boardHeight = maxSize
		boardWidth = maxSize * f64(boardTileWidth) / f64(boardTileHeight)
	}

	boardRect := FRectWH(
		boardWidth, boardHeight,
	)
	pCenter := FRectangleCenter(parentRect)
	boardRect = CenterFRectangle(boardRect, pCenter.X, pCenter.Y)

	if boardRect.Overlaps(gu.TopUIRect()) {
		parentRect := FRect(
			0, gu.TopUIRect().Max.Y,
			ScreenWidth, ScreenHeight,
		)

		pCenter := FRectangleCenter(parentRect)
		boardRect = CenterFRectangle(boardRect, pCenter.X, pCenter.Y)
	}

	return boardRect
}

func (gu *GameUI) BoardSize() float64 {
	return min(ScreenWidth, ScreenHeight) * 0.2
}

func (gu *GameUI) TopUIRect() FRectangle {
	w := ScreenWidth
	h := ScreenHeight * gu.TopUIHeight

	h -= gu.TopUIInsetTop
	h = max(h, 0)

	x := ScreenWidth*0.5 - w*0.5
	y := gu.TopUIInsetTop

	return FRectXYWH(x, y, w, h)
}

type TopUI struct {
	Rect FRectangle

	Disabled bool

	DifficultyButtonLeft  *ImageButton
	DifficultyButtonRight *ImageButton

	UIElementWidth  float64 // constant, ratio of ui elements
	UIElementHeight float64 // constant, ratio of ui elements

	ButtonWidth  float64 // constant, relative to UIWidth
	ButtonHeight float64 // constant, relative to UIWidth

	TextWidth  float64 // constant, relative to UIWidth
	TextHeight float64 // constant, relative to UIWidth

	Difficulty         Difficulty
	OnDifficultyChange func(difficulty Difficulty)
}

func NewTopUI() *TopUI {
	tu := new(TopUI)

	tu.UIElementWidth = 5
	tu.UIElementHeight = 1

	tu.ButtonWidth = 0.15
	tu.ButtonHeight = 0.8

	tu.TextWidth = 0.55
	tu.TextHeight = 0.75

	// ==============================
	// create difficulty buttons
	// ==============================
	{
		leftRect := tu.GetDifficultyButtonRect(false)
		rightRect := tu.GetDifficultyButtonRect(true)

		// DifficultyButtonLeft
		tu.DifficultyButtonLeft = NewImageButton()

		tu.DifficultyButtonLeft.Rect = leftRect
		tu.DifficultyButtonLeft.OnClick = func() {
			prevDifficulty := tu.Difficulty
			tu.Difficulty = max(tu.Difficulty-1, 0)
			if tu.OnDifficultyChange != nil && prevDifficulty != tu.Difficulty {
				tu.OnDifficultyChange(tu.Difficulty)
			}
		}

		tu.DifficultyButtonLeft.Image = SpriteSubView(TileSprite, 11)
		tu.DifficultyButtonLeft.ImageOnHover = SpriteSubView(TileSprite, 11)
		tu.DifficultyButtonLeft.ImageOnDown = SpriteSubView(TileSprite, 13)

		tu.DifficultyButtonLeft.ImageColor = ColorTopUIButton
		tu.DifficultyButtonLeft.ImageColorOnHover = ColorTopUIButtonOnHover
		tu.DifficultyButtonLeft.ImageColorOnDown = ColorTopUIButtonOnDown

		// DifficultyButtonRight
		tu.DifficultyButtonRight = NewImageButton()

		tu.DifficultyButtonRight.Rect = rightRect
		tu.DifficultyButtonRight.OnClick = func() {
			prevDifficulty := tu.Difficulty
			tu.Difficulty = min(tu.Difficulty+1, DifficultySize-1)
			if tu.OnDifficultyChange != nil && prevDifficulty != tu.Difficulty {
				tu.OnDifficultyChange(tu.Difficulty)
			}
		}

		tu.DifficultyButtonRight.Image = SpriteSubView(TileSprite, 12)
		tu.DifficultyButtonRight.ImageOnHover = SpriteSubView(TileSprite, 12)
		tu.DifficultyButtonRight.ImageOnDown = SpriteSubView(TileSprite, 14)

		tu.DifficultyButtonRight.ImageColor = ColorTopUIButton
		tu.DifficultyButtonRight.ImageColorOnHover = ColorTopUIButtonOnHover
		tu.DifficultyButtonRight.ImageColorOnDown = ColorTopUIButtonOnDown
	}

	return tu
}

func (tu *TopUI) Update() {
	tu.DifficultyButtonLeft.Disabled = tu.Disabled
	tu.DifficultyButtonRight.Disabled = tu.Disabled

	// update button rect
	tu.DifficultyButtonLeft.Rect = tu.GetDifficultyButtonRect(false)
	tu.DifficultyButtonRight.Rect = tu.GetDifficultyButtonRect(true)

	tu.DifficultyButtonLeft.Update()
	tu.DifficultyButtonRight.Update()
	// ==========================
}

func (tu *TopUI) Draw(dst *eb.Image) {
	tu.DifficultyButtonLeft.Draw(dst)
	tu.DifficultyButtonRight.Draw(dst)

	tu.DrawDifficultyText(dst)
}

func (tu *TopUI) GetUIElementRect() FRectangle {
	width := tu.UIElementWidth
	height := tu.UIElementHeight

	scale := min(tu.Rect.Dx()/width, tu.Rect.Dy()/height)

	width *= scale
	height *= scale

	x := tu.Rect.Min.X + tu.Rect.Dx()*0.5 - width*0.5
	y := tu.Rect.Max.Y - height

	return FRectXYWH(x, y, width, height)
}

func (tu *TopUI) GetDifficultyButtonRect(forRight bool) FRectangle {
	parentRect := tu.GetUIElementRect()

	width := parentRect.Dx() * tu.ButtonWidth
	height := parentRect.Dy() * tu.ButtonHeight

	if forRight {
		return FRectXYWH(
			parentRect.Max.X-width, (parentRect.Min.Y+parentRect.Max.Y)*0.5-height*0.5,
			width, height,
		)
	} else {
		return FRectXYWH(
			parentRect.Min.X, (parentRect.Min.Y+parentRect.Max.Y)*0.5-height*0.5,
			width, height,
		)
	}
}

func (tu *TopUI) GetDifficultyTextRect() FRectangle {
	parentRect := tu.GetUIElementRect()

	width := parentRect.Dx() * tu.TextWidth
	height := parentRect.Dy() * tu.TextHeight

	rect := FRectWH(width, height)

	pCenter := FRectangleCenter(parentRect)
	rect = CenterFRectangle(rect, pCenter.X, pCenter.Y)

	return rect
}

func (tu *TopUI) DrawDifficultyText(dst *eb.Image) {
	var maxW, maxH float64
	var textW, textH float64

	// TODO : cache this if you can
	for d := Difficulty(0); d < DifficultySize; d++ {
		str := DifficultyStrs[d]
		w, h := ebt.Measure(str, DecoFace, FontLineSpacing(DecoFace))
		maxW = max(w, maxW)
		maxH = max(h, maxH)

		if d == tu.Difficulty {
			textW, textH = w, h
		}
	}

	rect := tu.GetDifficultyTextRect()

	scale := min(rect.Dx()/maxW, rect.Dy()/maxH)

	rectCenter := FRectangleCenter(rect)

	op := &DrawTextOptions{}
	op.GeoM.Concat(TransformToCenter(textW, textH, scale, scale, 0))
	op.GeoM.Translate(rectCenter.X, rectCenter.Y)

	op.ColorScale.ScaleWithColor(TheColorTable[ColorTopUITitle])

	DrawText(dst, DifficultyStrs[tu.Difficulty], DecoFace, op)
}
