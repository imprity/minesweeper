package main

import (
	"fmt"
	"image/color"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type ResourceEditor struct {
	ColorPicker  *ColorPicker
	BezierEditor *BezierEditor
	HSVmodEditor *HSVmodEditor

	ColorTableIndex  ColorTableIndex
	BezierTableIndex BezierTableIndex
	HSVmodTableIndex HSVmodTableIndex

	DoShow     bool
	wasShowing bool

	// 0 : showing color picker
	// 1 : showing bezier editor
	// 2 : showing hsv_mod editor
	ShowingTable int
}

func NewResourceEditor() *ResourceEditor {
	re := new(ResourceEditor)

	re.ColorPicker = NewColorPicker()
	re.BezierEditor = NewBezierEditor()
	re.HSVmodEditor = NewHSVmodEditor()

	return re
}

func (re *ResourceEditor) Update() {
	setEditor := func() {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorPicker.SetColor(TheColorTable[re.ColorTableIndex])
		} else if re.ShowingTable == 1 { // showing bezier table
			re.BezierEditor.SetToBezierCurveData(TheBezierTable[re.BezierTableIndex])
		} else if re.ShowingTable == 2 { // showing hsv_mod table
			re.HSVmodEditor.SetToHSVmod(TheHSVmodTable[re.HSVmodTableIndex])
		}
	}

	if !re.wasShowing && re.DoShow {
		setEditor()
	}

	re.wasShowing = re.DoShow

	if !re.DoShow {
		return
	}

	if re.ShowingTable == 0 { // showing color picker
		re.ColorPicker.Rect = FRectWH(200, 400)
		re.ColorPicker.Rect = FRectMoveTo(re.ColorPicker.Rect, ScreenWidth-210, 10)
		re.ColorPicker.Update()
	} else if re.ShowingTable == 1 { // showing bezier table
		re.BezierEditor.Rect = FRectWH(350, 350)
		re.BezierEditor.Rect = FRectMoveTo(re.BezierEditor.Rect, ScreenWidth-(350+25), 70)
		re.BezierEditor.Update()
	} else if re.ShowingTable == 2 { // showing hsv_mod table
		re.HSVmodEditor.Rect = FRectWH(300, 200)
		re.HSVmodEditor.Rect = FRectMoveTo(re.HSVmodEditor.Rect, ScreenWidth-(300+10), 10)
		re.HSVmodEditor.Update()
	}

	const firstRate = 200 * time.Millisecond
	const repeatRate = 50 * time.Millisecond
	changed := false

	if HandleKeyRepeat(firstRate, repeatRate, ResourceEditorUpKey) {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorTableIndex--
			changed = true
		} else if re.ShowingTable == 1 { // showing color table
			re.BezierTableIndex--
			changed = true
		} else if re.ShowingTable == 2 { // showing hsv_mod table
			re.HSVmodTableIndex--
			changed = true
		}
	}
	if HandleKeyRepeat(firstRate, repeatRate, ResourceEditorDownKey) {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorTableIndex++
			changed = true
		} else if re.ShowingTable == 1 { // showing color table
			re.BezierTableIndex++
			changed = true
		} else if re.ShowingTable == 2 { // showing hsv_mod table
			re.HSVmodTableIndex++
			changed = true
		}
	}

	re.ColorTableIndex = Clamp(re.ColorTableIndex, 0, ColorTableSize-1)
	re.BezierTableIndex = Clamp(re.BezierTableIndex, 0, BezierTableSize-1)
	re.HSVmodTableIndex = Clamp(re.HSVmodTableIndex, 0, HSVmodTableSize-1)

	if changed {
		setEditor()
	}

	prevShowingTable := re.ShowingTable

	if IsKeyJustPressed(eb.KeyA) {
		re.ShowingTable--
	}
	if IsKeyJustPressed(eb.KeyD) {
		re.ShowingTable++
	}

	if re.ShowingTable < 0 {
		re.ShowingTable = 2
	}
	if re.ShowingTable > 2 {
		re.ShowingTable = 0
	}

	if re.ShowingTable != prevShowingTable {
		setEditor()
	}

	if re.ShowingTable == 0 { // showing color picker
		TheColorTable[re.ColorTableIndex] = re.ColorPicker.Color()
	} else if re.ShowingTable == 1 { // showing color table
		TheBezierTable[re.BezierTableIndex] = re.BezierEditor.GetBezierCurveData()
	} else if re.ShowingTable == 2 { // showing hsv_mod table
		TheHSVmodTable[re.HSVmodTableIndex] = re.HSVmodEditor.GetHSVmod()
	}
}

func (re *ResourceEditor) Draw(dst *eb.Image) {
	if !re.DoShow {
		return
	}

	if re.ShowingTable == 0 { // showing color picker
		re.ColorPicker.Draw(dst)
	} else if re.ShowingTable == 1 { // showing bezier table
		re.BezierEditor.Draw(dst)
	} else if re.ShowingTable == 2 { // showing hsv_mod table
		re.HSVmodEditor.Draw(dst)
	}

	helpText := fmt.Sprintf(
		"press %s to reload assets\n"+
			"press %s to save edited assets",
		ReloadAssetsKey.String(),
		SaveAssetsKey.String(),
	)

	// draw list of table entries
	{
		getTableText := func(i int) string {
			if re.ShowingTable == 0 { // showing color picker
				return ColorTableIndex(i).String()
			} else if re.ShowingTable == 1 { // showing bezier table
				return BezierTableIndex(i).String()
			} else if re.ShowingTable == 2 { // showing hsv_mod table
				return HSVmodTableIndex(i).String()
			}

			return ""
		}

		isSelectedIndex := func(i int) bool {
			if re.ShowingTable == 0 { // showing color picker
				if i == int(re.ColorTableIndex) {
					return true
				}
			} else if re.ShowingTable == 1 { // showing bezier table
				if i == int(re.BezierTableIndex) {
					return true
				}
			} else if re.ShowingTable == 2 { // showing hsv_mod table
				if i == int(re.HSVmodTableIndex) {
					return true
				}
			}
			return false
		}

		const textScale = 0.3

		lineSpacing := FontLineSpacing(ClearFace)

		var bgWidth float64
		var bgHeight float64

		var tableEntryStart float64

		{
			w, _ := ebt.Measure(helpText, ClearFace, lineSpacing)

			bgWidth = max(bgWidth, w*textScale)
			bgHeight += lineSpacing * 3 * textScale

			tableEntryStart = bgHeight
		}

		var indexY float64
		var index int
		var indexLimit int

		indexY = tableEntryStart

		if re.ShowingTable == 0 { // showing color picker
			index = int(re.ColorTableIndex)
			indexLimit = int(ColorTableSize)
		} else if re.ShowingTable == 1 { // showing bezier table
			index = int(re.BezierTableIndex)
			indexLimit = int(BezierTableSize)
		} else if re.ShowingTable == 2 { // showing hsv_mod table
			index = int(re.HSVmodTableIndex)
			indexLimit = int(HSVmodTableSize)
		}

		for i := 0; i < indexLimit; i++ {
			text := getTableText(i)

			if i <= index {
				indexY += lineSpacing * textScale
			}

			w, _ := ebt.Measure(text, ClearFace, lineSpacing)

			bgWidth = max(bgWidth, w*textScale)
			bgHeight += lineSpacing * textScale
		}

		bgWidth += 20
		bgHeight += 20

		// draw bg
		FillRect(
			dst, FRectWH(bgWidth, bgHeight), color.NRGBA{0, 0, 0, 150},
		)

		{ // draw help texts
			op := &DrawTextOptions{}

			op.GeoM.Scale(textScale, textScale)
			op.LineSpacing = lineSpacing

			DrawText(dst, helpText, ClearFace, op)
		}

		overflows := indexY > ScreenHeight

		{
			i := 0
			if overflows {
				i = index
			}

			offsetY := tableEntryStart
			if overflows {
				offsetY = ScreenHeight - lineSpacing*textScale
			}

			for {
				text := getTableText(i)

				op := &DrawTextOptions{}

				op.GeoM.Scale(textScale, textScale)
				op.GeoM.Translate(0, offsetY)

				if isSelectedIndex(i) {
					op.ColorScale.ScaleWithColor(color.NRGBA{255, 0, 0, 255})
				} else {
					op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
				}

				DrawText(dst, text, ClearFace, op)
				if overflows {
					i--
					offsetY -= lineSpacing * textScale
					if i < 0 {
						break
					}
					if offsetY < tableEntryStart {
						break
					}
				} else {
					i++
					offsetY += lineSpacing * textScale
					if i >= indexLimit {
						break
					}
					if offsetY > ScreenHeight {
						break
					}
				}
			}
		}
	}
}
