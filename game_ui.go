package minesweeper

import (
	"fmt"
	"image"
	"image/color"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var _ = color.White

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

	MineCounts [DifficultySize]int

	BoardTileCountsNormal [DifficultySize]image.Point // constant
	BoardTileCountsMobile [DifficultySize]image.Point // constant

	BoardSizeRatiosNormal [DifficultySize]float64 // constant, relative to board area
	BoardSizeRatiosMobile [DifficultySize]float64 // constant, relative to board area

	ButtonSizeRatioNormal float64 // constant, relative to min(ScreenWidth, ScreenHeight)
	ButtonSizeRatioMobile float64 // constant, relative to min(ScreenWidth, ScreenHeight)

	BoardMarginTop        float64 // constant
	BoardMarginBottom     float64 // constant
	BoardMarginHorizontal float64 // constant

	TopUI *TopUI

	TopUIHeight float64 // constant, relative to ScreenHeight

	ResourceEditor *ResourceEditor

	wasOnMobile bool
}

func NewGameUI() *GameUI {
	gu := new(GameUI)

	gu.wasOnMobile = ProbablyOnMobile()

	// set constants
	gu.MineCounts = [DifficultySize]int{10, 40, 99}

	gu.BoardTileCountsNormal = [DifficultySize]image.Point{
		image.Pt(10, 10), image.Pt(16, 16), image.Pt(22, 22),
	}
	gu.BoardTileCountsMobile = [DifficultySize]image.Point{
		image.Pt(10, 10), image.Pt(13, 20), image.Pt(18, 27),
	}

	gu.BoardSizeRatiosNormal = [DifficultySize]float64{0.75, 0.9, 1}
	gu.BoardSizeRatiosMobile = [DifficultySize]float64{1, 1, 1}

	gu.ButtonSizeRatioNormal = 0.2
	gu.ButtonSizeRatioMobile = 0.33

	gu.TopUIHeight = 0.075

	gu.BoardMarginTop = 10
	gu.BoardMarginBottom = 10
	gu.BoardMarginHorizontal = 10

	gu.Game = NewGame(
		gu.BoardTileCount(DifficultyEasy).X, gu.BoardTileCount(DifficultyEasy).Y,
		gu.MineCounts[DifficultyEasy],
	)
	gu.Game.OnFirstInteraction = func() {
		gu.TopUI.TimerUI.Start()
	}
	gu.Game.OnGameEnd = func(didWin bool) {
		gu.TopUI.TimerUI.Pause()
	}
	gu.Game.OnBeforeBoardReset = func() {
		gu.Game.SetResetParameter(
			gu.BoardTileCount(gu.Difficulty).X, gu.BoardTileCount(gu.Difficulty).Y,
			gu.MineCounts[gu.Difficulty],
		)
		gu.TopUI.TimerUI.Reset()
	}

	gu.TopUI = NewTopUI()
	gu.TopUI.DifficultySelectUI.OnDifficultyChange = func(newDifficulty Difficulty) {
		gu.Difficulty = newDifficulty
		gu.Game.SetResetParameter(
			gu.BoardTileCount(gu.Difficulty).X, gu.BoardTileCount(gu.Difficulty).Y,
			gu.MineCounts[gu.Difficulty],
		)
		gu.Game.ResetBoard()
	}

	gu.ResourceEditor = NewResourceEditor()

	return gu
}

func (gu *GameUI) Update() {
	if gu.wasOnMobile != ProbablyOnMobile() {
		gu.wasOnMobile = ProbablyOnMobile()

		if !gu.Game.HadInteraction() {
			gu.Game.SetResetParameter(
				gu.BoardTileCount(gu.Difficulty).X, gu.BoardTileCount(gu.Difficulty).Y,
				gu.MineCounts[gu.Difficulty],
			)
			gu.Game.ResetBoard()
		}
	}

	gu.TopUI.Rect = gu.TopUIRect()
	gu.TopUI.Update()

	gu.Game.SetNoInputZone(gu.TopUI.Rect)

	gu.Game.Rect = gu.BoardRect()
	gu.Game.SetRetryButtonSize(min(ScreenWidth, ScreenHeight) * gu.ButtonSizeRatio())
	gu.Game.Update()

	gu.TopUI.FlagUI.FlagCount = gu.Game.MineCount() - gu.Game.FlagCount()

	if IsKeyJustPressed(ShowResourceEditorKey) && IsDevVersion {
		gu.ResourceEditor.DoShow = !gu.ResourceEditor.DoShow
	}
	if gu.ResourceEditor.DoShow {
		SetRedraw()
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

func (gu *GameUI) BoardTileCount(difficulty Difficulty) image.Point {
	if ProbablyOnMobile() {
		return gu.BoardTileCountsMobile[difficulty]
	} else {
		return gu.BoardTileCountsNormal[difficulty]
	}
}

func (gu *GameUI) BoardSizeRatio(difficulty Difficulty) float64 {
	if ProbablyOnMobile() {
		return gu.BoardSizeRatiosMobile[difficulty]
	} else {
		return gu.BoardSizeRatiosNormal[difficulty]
	}
}

func (gu *GameUI) ButtonSizeRatio() float64 {
	if ProbablyOnMobile() {
		return gu.ButtonSizeRatioMobile
	} else {
		return gu.ButtonSizeRatioNormal
	}
}

func (gu *GameUI) BoardRect() FRectangle {
	topRect := gu.TopUI.RenderedRect()

	parentRect := FRect(
		gu.BoardMarginHorizontal, topRect.Max.Y+gu.BoardMarginTop,
		ScreenWidth-gu.BoardMarginHorizontal, ScreenHeight-gu.BoardMarginBottom,
	)

	/*
		boardTileWidth = gu.BoardTileCounts[difficulty].X
		boardTileHeight = gu.BoardTileCounts[difficulty].Y
	*/
	boardTileWidth, boardTileHeight := gu.Game.BoardTileCount()

	scale := min(
		parentRect.Dx()*gu.BoardSizeRatio(gu.Difficulty)/f64(boardTileWidth),
		parentRect.Dy()*gu.BoardSizeRatio(gu.Difficulty)/f64(boardTileHeight),
	)

	boardWidth := f64(boardTileWidth) * scale
	boardHeight := f64(boardTileHeight) * scale

	boardRect := FRectWH(
		boardWidth, boardHeight,
	)
	pCenter := FRectangleCenter(parentRect)
	boardRect = CenterFRectangle(boardRect, pCenter.X, pCenter.Y)

	return boardRect
}

func (gu *GameUI) TopUIRect() FRectangle {
	w := ScreenWidth
	h := ScreenHeight * gu.TopUIHeight

	x := ScreenWidth*0.5 - w*0.5
	y := float64(0)

	return FRectXYWH(x, y, w, h)
}

type TopUI struct {
	Rect FRectangle

	IdealTopMargin    float64 // constant, relative to TopUIIdealHeight
	IdealBottomMargin float64 // constant, relative to TopUIIdealHeight

	MuteButtonUI       *MuteButtonUI
	FlagUI             *FlagUI
	DifficultySelectUI *DifficultySelectUI
	TimerUI            *TimerUI

	UIScale float64

	MuteButtonUIRect       FRectangle
	FlagUIRect             FRectangle
	DifficultySelectUIRect FRectangle
	TimerUIRect            FRectangle
}

func NewTopUI() *TopUI {
	tu := new(TopUI)

	tu.IdealTopMargin = 7
	tu.IdealBottomMargin = 10

	tu.MuteButtonUI = NewMuteButtonUI()
	tu.FlagUI = NewFlagUI()
	tu.DifficultySelectUI = NewDifficultySelectUI()
	tu.TimerUI = NewTimerUI()

	return tu
}

func (tu *TopUI) Update() {
	var totalIdealWidth float64

	const idealMargin = 5
	const idealMuteMargin = 27

	idealMuteW := tu.MuteButtonUI.GetIdealWidth()
	idealFlagW := tu.FlagUI.GetIdealWidth()
	idealDifficultyW := tu.DifficultySelectUI.GetIdealWidth()
	idealTimerW := tu.TimerUI.GetIdealWidth()

	totalIdealWidth = max(
		idealMargin+idealTimerW+idealMargin+idealDifficultyW*0.5,
		idealDifficultyW*0.5+idealMargin+idealFlagW+idealMargin+idealMuteW+idealMuteMargin,
	) * 2

	tu.UIScale = min(
		tu.Rect.Dx()/totalIdealWidth,
		tu.Rect.Dy()/(TopUIIdealHeight+tu.IdealTopMargin+tu.IdealBottomMargin),
	)

	uiRect := FRect(
		tu.Rect.Min.X, tu.Rect.Min.Y+tu.IdealTopMargin*tu.UIScale,
		tu.Rect.Max.X, tu.Rect.Max.Y+tu.IdealBottomMargin*tu.UIScale,
	)

	margin := idealMargin * tu.UIScale
	muteMargin := idealMuteMargin * tu.UIScale

	_ = margin

	muteW := idealMuteW * tu.UIScale
	flagW := idealFlagW * tu.UIScale
	difficultyW := idealDifficultyW * tu.UIScale
	timerW := idealTimerW * tu.UIScale

	uiHeight := TopUIIdealHeight * tu.UIScale

	// update rectangles
	tu.MuteButtonUIRect = FRectXYWH(
		uiRect.Max.X-muteMargin-muteW, uiRect.Min.Y,
		muteW, uiHeight,
	)
	tu.DifficultySelectUIRect = FRectXYWH(
		uiRect.Min.X+uiRect.Dx()*0.5-difficultyW*0.5, uiRect.Min.Y,
		difficultyW, uiHeight,
	)
	timerMinX := uiRect.Min.X
	timerMaxX := tu.DifficultySelectUIRect.Min.X - timerW
	tu.TimerUIRect = FRectXYWH(
		Lerp(timerMinX, timerMaxX, 0.53),
		uiRect.Min.Y,
		timerW, uiHeight,
	)
	flagMinX := (tu.DifficultySelectUIRect.Max.X)
	flagMaxX := (tu.MuteButtonUIRect.Min.X) - flagW
	tu.FlagUIRect = FRectXYWH(
		Lerp(flagMinX, flagMaxX, 0.6),
		uiRect.Min.Y,
		flagW, uiHeight,
	)

	tu.MuteButtonUI.OnUpdate(tu.MuteButtonUIRect, tu.UIScale)
	tu.TimerUI.OnUpdate(tu.TimerUIRect, tu.UIScale)
	tu.DifficultySelectUI.OnUpdate(tu.DifficultySelectUIRect, tu.UIScale)
	tu.FlagUI.OnUpdate(tu.FlagUIRect, tu.UIScale)
}

func (tu *TopUI) Draw(dst *eb.Image) {
	FillRect(
		dst,
		tu.RenderedRect(),
		ColorTopUIBg,
		//color.NRGBA{188, 188, 188, 255},
	)
	tu.MuteButtonUI.OnDraw(dst, tu.MuteButtonUIRect, tu.UIScale)
	tu.TimerUI.OnDraw(dst, tu.TimerUIRect, tu.UIScale)
	tu.DifficultySelectUI.OnDraw(dst, tu.DifficultySelectUIRect, tu.UIScale)
	tu.FlagUI.OnDraw(dst, tu.FlagUIRect, tu.UIScale)
}

// TopUI's display rect might be smaller than
// Rect field due to various layouts
// this function gives you that rect
func (tu *TopUI) RenderedRect() FRectangle {
	return FRectXYWH(
		tu.Rect.Min.X, tu.Rect.Min.Y,
		tu.Rect.Dx(),
		(TopUIIdealHeight+tu.IdealTopMargin+tu.IdealBottomMargin)*tu.UIScale,
	)
}

const (
	TopUIIdealHeight = 100
	/*
		TopUIIdealFaceSize = 71
		TopUIIdealTextY    = 46
	*/
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

		ds.DifficultyButtonLeft.Image = SpriteSubView(UISprite, 0)
		ds.DifficultyButtonLeft.ImageOnHover = SpriteSubView(UISprite, 0)
		ds.DifficultyButtonLeft.ImageOnDown = SpriteSubView(UISprite, 2)

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

		ds.DifficultyButtonRight.Image = SpriteSubView(UISprite, 1)
		ds.DifficultyButtonRight.ImageOnHover = SpriteSubView(UISprite, 1)
		ds.DifficultyButtonRight.ImageOnDown = SpriteSubView(UISprite, 3)

		ds.DifficultyButtonRight.ImageColor = ColorTopUIButton
		ds.DifficultyButtonRight.ImageColorOnHover = ColorTopUIButtonOnHover
		ds.DifficultyButtonRight.ImageColorOnDown = ColorTopUIButtonOnDown
	}

	var idealBtnRectLeft FRectangle
	var idealBtnRectRight FRectangle

	var idealMaxTextWidth float64
	var idealTextWidths [DifficultySize]float64
	var idealTextCenterX float64

	const idealFaceSize = 71

	idealFace := &ebt.GoTextFace{
		Source: FaceSource,
		Size:   idealFaceSize,
	}
	idealFace.SetVariation(ebt.MustParseTag("wght"), 700)

	for d := Difficulty(0); d < DifficultySize; d++ {
		str := DifficultyStrs[d]
		w, _ := ebt.Measure(
			str,
			idealFace,
			FaceLineSpacing(idealFace),
		)
		idealTextWidths[d] = w
		idealMaxTextWidth = max(w, idealMaxTextWidth)
	}

	const idealMargin = 15
	var idealBtnSize FPoint = FPt(70, 70)

	var idealWidth float64 = idealBtnSize.X + idealMargin + idealMaxTextWidth + idealMargin + idealBtnSize.X

	ds.GetIdealWidth = func() float64 {
		return idealWidth
	}

	idealBtnRectLeft = FRectXYWH(
		0, TopUIIdealHeight*0.5-idealBtnSize.Y*0.5,
		idealBtnSize.X, idealBtnSize.Y,
	)
	idealBtnRectRight = FRectXYWH(
		idealWidth-idealBtnSize.X, TopUIIdealHeight*0.5-idealBtnSize.Y*0.5,
		idealBtnSize.X, idealBtnSize.Y,
	)

	idealBtnRectLeft = idealBtnRectLeft.Add(FPt(0, 4))
	idealBtnRectRight = idealBtnRectRight.Add(FPt(0, 4))

	idealTextCenterX = idealWidth * 0.5

	ds.OnUpdate = func(actualRect FRectangle, scale float64) {
		btnRectLeft := FRectScale(idealBtnRectLeft, scale).Add(actualRect.Min)
		btnRectRight := FRectScale(idealBtnRectRight, scale).Add(actualRect.Min)

		ds.DifficultyButtonLeft.Rect = btnRectLeft
		ds.DifficultyButtonRight.Rect = btnRectRight

		ds.DifficultyButtonLeft.Update()
		ds.DifficultyButtonRight.Update()
	}

	difficultyTextOffsetsY := [DifficultySize]float64{
		-1.6,
		0,
		1,
	}

	ds.OnDraw = func(dst *eb.Image, actualRect FRectangle, scale float64) {
		ds.DifficultyButtonLeft.Draw(dst)
		ds.DifficultyButtonRight.Draw(dst)

		// draw text
		textCenterY := (actualRect.Min.Y + actualRect.Max.Y) * 0.5
		textCenterX := idealTextCenterX*scale + actualRect.Min.X

		textCenterY += difficultyTextOffsetsY[ds.Difficulty] * scale

		faceSize := idealFaceSize * scale
		face := &ebt.GoTextFace{
			Source: FaceSource,
			Size:   faceSize,
		}
		face.SetVariation(ebt.MustParseTag("wght"), 700)

		op := &DrawTextOptions{}
		op.PrimaryAlign = ebt.AlignCenter
		op.GeoM.Translate(
			textCenterX,
			textCenterY-FaceSize(face)*0.5,
		)
		op.ColorScale.ScaleWithColor(ColorTopUITitle)

		DrawText(dst, DifficultyStrs[ds.Difficulty], face, op)
	}

	return ds
}

type FlagUI struct {
	TopUIElement

	FlagCount int
}

func NewFlagUI() *FlagUI {
	fu := new(FlagUI)

	const idealFlagSize = 84

	var idealFlagRect FRectangle = FRectXYWH(
		0, TopUIIdealHeight*0.5-idealFlagSize*0.5,
		idealFlagSize, idealFlagSize,
	)
	idealFlagRect = idealFlagRect.Add(FPt(0, -5))

	const idealFaceSize = 62

	const idealMargin = 6

	var idealTextX float64 = idealFlagRect.Dx() + idealMargin
	const idealTextY = 54

	var idealMaxTextWidth float64
	{
		idealFace := &ebt.GoTextFace{
			Source: FaceSource,
			Size:   idealFaceSize,
		}
		idealFace.SetVariation(ebt.MustParseTag("wght"), 400)

		w, _ := ebt.Measure(
			"000", idealFace, FaceLineSpacing(idealFace))
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
			dst, flagRect, 1.1, 0, 0, ColorTopUIFlag, GetFlagTile(1.0),
		)

		textX := idealTextX*scale + actualRect.Min.X
		textCenterY := idealTextY*scale + actualRect.Min.Y

		text := fmt.Sprintf("%d", fu.FlagCount)

		faceSize := idealFaceSize * scale
		face := &ebt.GoTextFace{
			Source: FaceSource,
			Size:   faceSize,
		}
		face.SetVariation(ebt.MustParseTag("wght"), 400)
		WidthLimitFace(text, face, idealMaxTextWidth*scale)

		op := &DrawTextOptions{}
		op.GeoM.Translate(textX, textCenterY-FaceSize(face)*0.5)
		op.ColorScale.ScaleWithColor(ColorTopUITitle)

		DrawText(dst, text, face, op)
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
		0, TopUIIdealHeight*0.5-idealTimerSize*0.5,
		idealTimerSize, idealTimerSize,
	)

	const idealMargin = 10

	var idealTextX float64 = idealTimerRect.Dx() + idealMargin
	const idealTextY = 51

	var idealFaceSizeNormal float64 = 58
	var idealFaceSizeSmall float64

	var idealMaxTextWidth float64
	{
		idealFace := &ebt.GoTextFace{
			Source: FaceSource,
			Size:   idealFaceSizeNormal,
		}
		idealFace.SetVariation(ebt.MustParseTag("wght"), 400)

		w, _ := ebt.Measure(
			"00:00",
			idealFace,
			FaceLineSpacing(idealFace),
		)
		idealMaxTextWidth = w

		w, _ = ebt.Measure(
			"00:00:00",
			idealFace,
			FaceLineSpacing(idealFace),
		)
		idealFaceSizeSmall = idealFaceSizeNormal * (idealMaxTextWidth / w)
	}

	tu.GetIdealWidth = func() float64 {
		return idealTimerRect.Dx() + idealMargin + idealMaxTextWidth
	}

	var prevTime time.Duration

	tu.OnUpdate = func(actualRect FRectangle, scale float64) {
		currentTime := tu.CurrentTime()

		hours, minutes, seconds := GetHourMinuteSeconds(currentTime)
		prevHours, prevMinutes, prevSeconds := GetHourMinuteSeconds(prevTime)

		if prevHours != hours || prevMinutes != minutes || prevSeconds != seconds {
			SetRedraw()
			prevTime = currentTime
		}
	}

	tu.OnDraw = func(dst *eb.Image, actualRect FRectangle, scale float64) {
		timerRect := FRectScale(idealTimerRect, scale).Add(actualRect.Min)

		// draw timer icon
		DrawSubViewInRect(
			dst, timerRect, 1.0, 0, 0, ColorTopUITitle, SpriteSubView(UISprite, 4),
		)

		textX := idealTextX*scale + actualRect.Min.X
		textCenterY := idealTextY*scale + actualRect.Min.Y

		currentTime := tu.CurrentTime()

		hours, minutes, seconds := GetHourMinuteSeconds(currentTime)

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

		faceSize := idealFaceSizeNormal
		if hours > 0 {
			faceSize = idealFaceSizeSmall
		}
		faceSize *= scale

		face := &ebt.GoTextFace{
			Source: FaceSource,
			Size:   faceSize,
		}
		face.SetVariation(ebt.MustParseTag("wght"), 400)
		WidthLimitFace(text, face, idealMaxTextWidth*scale)

		op := &DrawTextOptions{}
		op.GeoM.Translate(textX, textCenterY-FaceSize(face)*0.5)
		op.ColorScale.ScaleWithColor(ColorTopUITitle)

		DrawText(dst, text, face, op)
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

type MuteButtonUI struct {
	TopUIElement

	MuteButton *ImageButton

	IsMute bool
}

func NewMuteButtonUI() *MuteButtonUI {
	mu := new(MuteButtonUI)

	// ==============================
	// create mute button
	// ==============================
	{
		mu.MuteButton = NewImageButton()

		mu.MuteButton.Image = SpriteSubView(UISprite, 6)
		mu.MuteButton.ImageOnHover = SpriteSubView(UISprite, 6)
		mu.MuteButton.ImageOnDown = SpriteSubView(UISprite, 6)

		mu.MuteButton.ImageColor = ColorTopUIButton
		mu.MuteButton.ImageColorOnHover = ColorTopUIButtonOnHover
		mu.MuteButton.ImageColorOnDown = ColorTopUIButtonOnDown

		mu.MuteButton.OnPress = func(bool) {
			mu.IsMute = !mu.IsMute
			if mu.IsMute {
				mu.MuteButton.Image = SpriteSubView(UISprite, 5)
				mu.MuteButton.ImageOnHover = SpriteSubView(UISprite, 5)
				mu.MuteButton.ImageOnDown = SpriteSubView(UISprite, 5)
				SetGlobalVolume(0)
			} else {
				mu.MuteButton.Image = SpriteSubView(UISprite, 6)
				mu.MuteButton.ImageOnHover = SpriteSubView(UISprite, 6)
				mu.MuteButton.ImageOnDown = SpriteSubView(UISprite, 6)
				SetGlobalVolume(1)
			}
		}
	}

	const idealBtnSize = 82

	var idealBtnRect FRectangle = FRectXYWH(
		0, TopUIIdealHeight*0.5-idealBtnSize*0.5,
		idealBtnSize, idealBtnSize,
	)

	mu.GetIdealWidth = func() float64 {
		return idealBtnRect.Dx()
	}

	mu.OnUpdate = func(actualRect FRectangle, scale float64) {
		rect := FRectScale(idealBtnRect, scale).Add(actualRect.Min)
		mu.MuteButton.Rect = rect
		mu.MuteButton.Update()
	}

	mu.OnDraw = func(dst *eb.Image, actualRect FRectangle, scale float64) {
		mu.MuteButton.Draw(dst)
	}

	return mu
}
