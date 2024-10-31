package main

import (
	"image/color"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type ResourceEditor struct {
	ColorPicker  *ColorPicker
	BezierEditor *BezierEditor

	ColorTableIndex  ColorTableIndex
	BezierTableIndex BezierTableIndex

	DoShow     bool
	wasShowing bool

	// 0 : showing color picker
	// 1 : showing bezier editor
	ShowingTable int
}

func NewResourceEditor() *ResourceEditor {
	re := new(ResourceEditor)
	re.ColorPicker = NewColorPicker()
	re.BezierEditor = NewBezierEditor()

	return re
}

func (re *ResourceEditor) Update() {
	if !re.wasShowing && re.DoShow {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorPicker.SetColor(TheColorTable[re.ColorTableIndex])
		} else if re.ShowingTable == 1 { // showing bezier table
			re.BezierEditor.SetToBezierCurveData(TheBezierTable[re.BezierTableIndex])
		}
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
		// TODO: pick better rect
		re.BezierEditor.Rect = FRectWH(350, 350)
		re.BezierEditor.Rect = FRectMoveTo(re.BezierEditor.Rect, ScreenWidth-(350+25), 70)
		re.BezierEditor.Update()
	}

	const firstRate = 200 * time.Millisecond
	const repeatRate = 50 * time.Millisecond
	changed := false

	if HandleKeyRepeat(firstRate, repeatRate, ColorPickerUpKey) {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorTableIndex--
			changed = true
		} else if re.ShowingTable == 1 { // showing color table
			re.BezierTableIndex--
			changed = true
		}
	}
	if HandleKeyRepeat(firstRate, repeatRate, ColorPickerDownKey) {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorTableIndex++
			changed = true
		} else if re.ShowingTable == 1 { // showing color table
			re.BezierTableIndex++
			changed = true
		}
	}

	re.ColorTableIndex = Clamp(re.ColorTableIndex, 0, ColorTableSize-1)
	re.BezierTableIndex = Clamp(re.BezierTableIndex, 0, BezierTableSize-1)

	if changed {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorPicker.SetColor(TheColorTable[re.ColorTableIndex])
		} else if re.ShowingTable == 1 { // showing color table
			re.BezierEditor.SetToBezierCurveData(TheBezierTable[re.BezierTableIndex])
		}
	}

	prevShowingTable := re.ShowingTable

	if IsKeyJustPressed(eb.KeyA) {
		re.ShowingTable--
	}
	if IsKeyJustPressed(eb.KeyD) {
		re.ShowingTable++
	}

	if re.ShowingTable < 0 {
		re.ShowingTable = 1
	}
	if re.ShowingTable > 1 {
		re.ShowingTable = 0
	}

	if re.ShowingTable != prevShowingTable {
		if re.ShowingTable == 0 { // showing color picker
			re.ColorPicker.SetColor(TheColorTable[re.ColorTableIndex])
		} else if re.ShowingTable == 1 { // showing color table
			re.BezierEditor.SetToBezierCurveData(TheBezierTable[re.BezierTableIndex])
		}
	}

	if re.ShowingTable == 0 { // showing color picker
		TheColorTable[re.ColorTableIndex] = re.ColorPicker.Color()
	} else if re.ShowingTable == 1 { // showing color table
		TheBezierTable[re.BezierTableIndex] = re.BezierEditor.GetBezierCurveData()
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
	}

	// draw list of table entries
	{
		const textScale = 0.3

		lineSpacing := FontLineSpacing(ClearFace)

		var indexLimit int

		if re.ShowingTable == 0 { // showing color picker
			indexLimit = int(ColorTableSize)
		} else if re.ShowingTable == 1 { // showing bezier table
			indexLimit = int(BezierTableSize)
		}

		// get bg width
		bgWidth := float64(0)
		for i := 0; i < indexLimit; i++ {
			var text string
			if re.ShowingTable == 0 { // showing color picker
				text = ColorTableIndex(i).String()
			} else if re.ShowingTable == 1 { // showing bezier table
				text = BezierTableIndex(i).String()
			}

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

		for i := 0; i < indexLimit; i++ {
			var text string
			if re.ShowingTable == 0 { // showing color picker
				text = ColorTableIndex(i).String()
			} else if re.ShowingTable == 1 { // showing bezier table
				text = BezierTableIndex(i).String()
			}

			op := &DrawTextOptions{}

			op.GeoM.Scale(textScale, textScale)
			op.GeoM.Translate(0, offsetY)

			doShowRed := false

			if re.ShowingTable == 0 { // showing color picker
				if i == int(re.ColorTableIndex) {
					doShowRed = true
				}
			} else if re.ShowingTable == 1 { // showing bezier table
				if i == int(re.BezierTableIndex) {
					doShowRed = true
				}
			}

			if doShowRed {
				op.ColorScale.ScaleWithColor(color.NRGBA{255, 0, 0, 255})
			} else {
				op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
			}

			DrawText(dst, text, ClearFace, op)

			offsetY += lineSpacing * textScale
		}
	}
}
