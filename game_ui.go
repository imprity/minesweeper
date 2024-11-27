package main

import (
	"fmt"
	"image"
	"time"
	//"image/color"

	eb "github.com/hajimehoshi/ebiten/v2"
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
	gu.Game.OnFirstInteraction = func() {
		gu.TopUI.TimerUI.Start()
	}
	gu.Game.OnGameEnd = func(didWin bool) {
		gu.TopUI.TimerUI.Pause()
	}
	gu.Game.OnBoardReset = func() {
		gu.TopUI.TimerUI.Reset()
	}

	gu.TopUI = NewTopUI()
	gu.TopUI.DifficultySelectUI.OnDifficultyChange = func(newDifficulty Difficulty) {
		gu.Difficulty = newDifficulty
		gu.Game.ResetBoard(
			gu.BoardTileCounts[newDifficulty].X, gu.BoardTileCounts[newDifficulty].Y,
			gu.MineCounts[newDifficulty],
		)
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

	gu.TopUI.FlagUI.FlagCount = gu.Game.MineCount() - gu.Game.FlagCount()

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

	FlagUI             *FlagUI
	DifficultySelectUI *DifficultySelectUI
	TimerUI            *TimerUI

	UIScale float64

	FlagUIRect             FRectangle
	DifficultySelectUIRect FRectangle
	TimerUIRect            FRectangle
}

func NewTopUI() *TopUI {
	tu := new(TopUI)

	tu.FlagUI = NewFlagUI()
	tu.DifficultySelectUI = NewDifficultySelectUI()
	tu.TimerUI = NewTimerUI()

	return tu
}

func (tu *TopUI) Update() {
	var totalIdealWidth float64

	const idealMargin = 10

	idealFlagW := tu.FlagUI.GetIdealWidth()
	idealDifficultyW := tu.DifficultySelectUI.GetIdealWidth()
	idealTimerW := tu.TimerUI.GetIdealWidth()

	totalIdealWidth = max(
		idealDifficultyW*0.5+idealMargin+idealFlagW+idealMargin,
		idealDifficultyW*0.5+idealMargin+idealTimerW+idealMargin,
	) * 2

	tu.UIScale = min(tu.Rect.Dx()/totalIdealWidth, tu.Rect.Dy()/TopUIElementIdealHeight)

	flagW := idealFlagW * tu.UIScale
	difficultyW := idealDifficultyW * tu.UIScale
	timerW := idealTimerW * tu.UIScale

	uiHeight := TopUIElementIdealHeight * tu.UIScale

	// update rectangles
	tu.FlagUIRect = FRectXYWH(
		tu.Rect.Min.X+idealMargin*tu.UIScale, tu.Rect.Min.Y,
		flagW, uiHeight,
	)
	tu.DifficultySelectUIRect = FRectXYWH(
		tu.Rect.Min.X+tu.Rect.Dx()*0.5-difficultyW*0.5, tu.Rect.Min.Y,
		difficultyW, uiHeight,
	)
	tu.TimerUIRect = FRectXYWH(
		tu.Rect.Max.X-timerW-idealMargin*tu.UIScale, tu.Rect.Min.Y,
		timerW, uiHeight,
	)

	tu.TimerUI.OnUpdate(tu.TimerUIRect, tu.UIScale)
	tu.DifficultySelectUI.OnUpdate(tu.DifficultySelectUIRect, tu.UIScale)
	tu.FlagUI.OnUpdate(tu.FlagUIRect, tu.UIScale)
}

func (tu *TopUI) Draw(dst *eb.Image) {
	tu.TimerUI.OnDraw(dst, tu.TimerUIRect, tu.UIScale)
	tu.DifficultySelectUI.OnDraw(dst, tu.DifficultySelectUIRect, tu.UIScale)
	tu.FlagUI.OnDraw(dst, tu.FlagUIRect, tu.UIScale)
}

const (
	TopUIElementIdealHeight   = 100
	TopUIElementIdealFontSize = 80
	TopUIElementIdealTextY    = 48
)

type TopUIElement struct {
	// given the ideal height, what width does this element would be?
	GetIdealWidth func() float64

	// actual update and draw function
	OnUpdate func(actualRect FRectangle, scale float64)
	OnDraw   func(dst *eb.Image, actualRect FRectangle, scale float64)
}

type DifficultySelectUI struct {
	TopUIElement

	DifficultyButtonLeft  *ImageButton
	DifficultyButtonRight *ImageButton

	Difficulty         Difficulty
	OnDifficultyChange func(difficulty Difficulty)
}

func NewDifficultySelectUI() *DifficultySelectUI {
	ds := new(DifficultySelectUI)

	// ==============================
	// create difficulty buttons
	// ==============================
	{
		// DifficultyButtonLeft
		ds.DifficultyButtonLeft = NewImageButton()

		ds.DifficultyButtonLeft.OnPress = func(bool) {
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

		ds.DifficultyButtonRight.OnPress = func(bool) {
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

	var idealBtnRectLeft FRectangle
	var idealBtnRectRight FRectangle

	var idealMaxTextWidth float64
	var idealTextWidths [DifficultySize]float64
	var idealTextCenterX float64

	idealFontSize := float64(TopUIElementIdealFontSize)

	for d := Difficulty(0); d < DifficultySize; d++ {
		str := DifficultyStrs[d]
		w, _ := MeasureTextSized(
			str,
			BoldFace,
			idealFontSize,
			FontLineSpacingSized(BoldFace, idealFontSize),
		)
		idealTextWidths[d] = w
		idealMaxTextWidth = max(w, idealMaxTextWidth)
	}

	const idealMargin = 10
	var idealBtnSize FPoint = FPt(70, 70)

	var idealWidth float64 = idealBtnSize.X + idealMargin + idealMaxTextWidth + idealMargin + idealBtnSize.X

	ds.GetIdealWidth = func() float64 {
		return idealWidth
	}

	idealBtnRectLeft = FRectXYWH(
		0, TopUIElementIdealHeight*0.5-idealBtnSize.Y*0.5,
		idealBtnSize.X, idealBtnSize.Y,
	)
	idealBtnRectRight = FRectXYWH(
		idealWidth-idealBtnSize.X, TopUIElementIdealHeight*0.5-idealBtnSize.Y*0.5,
		idealBtnSize.X, idealBtnSize.Y,
	)

	idealTextCenterX = idealWidth * 0.5

	ds.OnUpdate = func(actualRect FRectangle, scale float64) {
		btnRectLeft := FRectScale(idealBtnRectLeft, scale).Add(actualRect.Min)
		btnRectRight := FRectScale(idealBtnRectRight, scale).Add(actualRect.Min)

		ds.DifficultyButtonLeft.Rect = btnRectLeft
		ds.DifficultyButtonRight.Rect = btnRectRight

		ds.DifficultyButtonLeft.Update()
		ds.DifficultyButtonRight.Update()
	}

	ds.OnDraw = func(dst *eb.Image, actualRect FRectangle, scale float64) {
		ds.DifficultyButtonLeft.Draw(dst)
		ds.DifficultyButtonRight.Draw(dst)

		// draw text
		textY := TopUIElementIdealTextY*scale + actualRect.Min.Y
		textCenterX := idealTextCenterX*scale + actualRect.Min.X
		textX := textCenterX - idealTextWidths[ds.Difficulty]*scale*0.5

		fontSize := idealFontSize * scale

		op := &DrawTextOptions{}
		op.GeoM.Concat(TextToYcenter(
			BoldFace,
			fontSize,
			textX, textY,
		))
		op.ColorScale.ScaleWithColor(ColorTopUITitle)

		DrawText(dst, DifficultyStrs[ds.Difficulty], BoldFace, op)
	}

	return ds
}

type FlagUI struct {
	TopUIElement

	FlagCount int
}

func NewFlagUI() *FlagUI {
	fu := new(FlagUI)

	const idealFlagSize = 80

	var idealFlagRect FRectangle = FRectXYWH(
		0, 0,
		idealFlagSize, idealFlagSize,
	)

	idealFontSize := TopUIElementIdealFontSize * 0.85

	const idealMargin = 6

	var idealTextX float64 = idealFlagRect.Dx() + idealMargin

	var idealMaxTextWidth float64
	{
		w, _ := MeasureTextSized(
			"000", RegularFace, idealFontSize, FontLineSpacingSized(RegularFace, idealFontSize))
		idealMaxTextWidth = w
	}

	fu.GetIdealWidth = func() float64 {
		return idealFlagRect.Dx() + idealMargin + idealMaxTextWidth
	}

	fu.OnUpdate = func(actualRect FRectangle, scale float64) {
		// pass
	}

	fu.OnDraw = func(dst *eb.Image, actualRect FRectangle, scale float64) {
		flagRect := FRectScale(idealFlagRect, scale).Add(actualRect.Min)

		// draw flag icon
		DrawSubViewInRect(
			dst, flagRect, 1.1, 0, -scale*2, ColorTopUITitle, GetFlagTile(),
		)

		textX := idealTextX*scale + actualRect.Min.X
		textY := TopUIElementIdealTextY*scale + actualRect.Min.Y

		text := fmt.Sprintf("%d", fu.FlagCount)

		fontSize := idealFontSize * scale

		op := &DrawTextOptions{}
		op.GeoM.Concat(TextToYcenterLimitWidth(
			text,
			RegularFace,
			fontSize,
			textX, textY,
			idealMaxTextWidth*scale,
		))
		op.ColorScale.ScaleWithColor(ColorTopUITitle)

		DrawText(dst, text, RegularFace, op)
	}

	return fu
}

type TimerUI struct {
	TopUIElement

	ticking       bool
	startTime     time.Time
	timeStartFrom time.Duration
}

func NewTimerUI() *TimerUI {
	tu := new(TimerUI)

	const idealTimerSize = 80

	var idealTimerRect FRectangle = FRectXYWH(
		0, TopUIElementIdealHeight*0.5-idealTimerSize*0.5,
		idealTimerSize, idealTimerSize,
	)

	const idealMargin = 10

	var idealTextX float64 = idealTimerRect.Dx() + idealMargin

	var idealFontSizeNormal float64 = TopUIElementIdealFontSize * 0.8

	var idealMaxTextWidth float64
	{
		w, _ := MeasureTextSized(
			"00:00",
			RegularFace, idealFontSizeNormal,
			FontLineSpacingSized(RegularFace, idealFontSizeNormal),
		)
		idealMaxTextWidth = w
	}

	var idealFontSizeSmall float64
	{
		w, _ := MeasureTextSized(
			"00:00:00",
			RegularFace, idealFontSizeNormal,
			FontLineSpacingSized(RegularFace, idealFontSizeNormal),
		)
		idealFontSizeSmall = (idealMaxTextWidth / w)
	}

	tu.GetIdealWidth = func() float64 {
		return idealTimerRect.Dx() + idealMargin + idealMaxTextWidth
	}

	tu.OnUpdate = func(actualRect FRectangle, scale float64) {
		// pass
	}

	tu.OnDraw = func(dst *eb.Image, actualRect FRectangle, scale float64) {
		timerRect := FRectScale(idealTimerRect, scale).Add(actualRect.Min)

		// draw timer icon
		DrawSubViewInRect(
			dst, timerRect, 1.0, 0, 0, ColorTopUITitle, SpriteSubView(TileSprite, 15),
		)

		textX := idealTextX*scale + actualRect.Min.X
		textY := TopUIElementIdealTextY*scale + actualRect.Min.Y

		currentTime := tu.CurrentTime()

		hours := currentTime / time.Hour
		minutes := (currentTime % time.Hour) / time.Minute
		seconds := (currentTime % time.Minute) / time.Second

		fontSize := idealFontSizeNormal
		if hours > 0 {
			fontSize = idealFontSizeSmall
		}
		fontSize *= scale

		var text string
		if hours > 0 {
			text = fmt.Sprintf(
				"%02d:%02d:%02d",
				hours, minutes, seconds,
			)
		} else {
			text = fmt.Sprintf(
				"%02d:%02d",
				minutes, seconds,
			)
		}

		op := &DrawTextOptions{}
		op.GeoM.Concat(TextToYcenterLimitWidth(
			text,
			RegularFace,
			fontSize,
			textX, textY,
			idealMaxTextWidth*scale,
		))
		op.ColorScale.ScaleWithColor(ColorTopUITitle)

		DrawText(dst, text, RegularFace, op)
	}

	return tu
}

func (tu *TimerUI) Start() {
	tu.ticking = true
	tu.startTime = time.Now()
}

func (tu *TimerUI) Pause() {
	tu.ticking = false
	tu.timeStartFrom = time.Now().Sub(tu.startTime)
}

func (tu *TimerUI) Reset() {
	tu.ticking = false
	tu.timeStartFrom = 0
}

func (tu *TimerUI) CurrentTime() time.Duration {
	if !tu.ticking {
		return tu.timeStartFrom
	}
	return tu.timeStartFrom + time.Now().Sub(tu.startTime)
}
