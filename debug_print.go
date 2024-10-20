package main

import (
	"image/color"
	"strings"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

type DebugMsg struct {
	Key   string
	Value string
}

var TheDebugPrintManager struct {
	DebugMsgs           []DebugMsg
	PersistentDebugMsgs []DebugMsg

	builder strings.Builder
}

func DebugPrint(key, value string) {
	dm := &TheDebugPrintManager

	for i, msg := range dm.DebugMsgs {
		if msg.Key == key {
			dm.DebugMsgs[i].Value = value
			return
		}
	}

	dm.DebugMsgs = append(dm.DebugMsgs, DebugMsg{
		Key:   key,
		Value: value,
	})
}

func DebugPrintPersist(key, value string) {
	dm := &TheDebugPrintManager

	for i, msg := range dm.PersistentDebugMsgs {
		if msg.Key == key {
			dm.DebugMsgs[i].Value = value
			return
		}
	}

	dm.PersistentDebugMsgs = append(dm.PersistentDebugMsgs, DebugMsg{
		Key:   key,
		Value: value,
	})
}

func DrawDebugMsgs(dst *eb.Image) {
	dm := &TheDebugPrintManager

	dm.builder.Reset()

	msgCounter := 0

	for _, msg := range dm.PersistentDebugMsgs {
		// builder doesn't actually errors out
		// no need to check error
		dm.builder.WriteString(msg.Key)
		dm.builder.WriteString(": ")
		dm.builder.WriteString(msg.Value)

		msgCounter++
		if msgCounter != len(dm.PersistentDebugMsgs)+len(dm.DebugMsgs) {
			dm.builder.WriteString("\n")
		}
	}

	for _, msg := range dm.DebugMsgs {
		dm.builder.WriteString(msg.Key)
		dm.builder.WriteString(": ")
		dm.builder.WriteString(msg.Value)

		msgCounter++
		if msgCounter != len(dm.PersistentDebugMsgs)+len(dm.DebugMsgs) {
			dm.builder.WriteString("\n")
		}
	}

	const fontSize = 20
	const hozMargin = 5
	const vertMargin = 5

	scale := fontSize / FontSize(ClearFace)
	fontLineSpacing := FontLineSpacing(ClearFace) + 3

	text := dm.builder.String()

	w, h := ebt.Measure(text, ClearFace, fontLineSpacing)

	dstFRect := RectToFRect(dst.Bounds())

	// update width and height of
	boxW, boxH := w*scale+hozMargin*2, h*scale+vertMargin*2

	rect := FRectangle{
		Min: FPt(dstFRect.Max.X-boxW, dstFRect.Max.Y-boxH),
		Max: dstFRect.Max,
	}

	// draw background
	DrawFilledRect(
		dst,
		rect,
		color.NRGBA{0, 0, 0, 100},
	)

	// draw text
	op := &DrawTextOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(
		rect.Min.X+hozMargin, rect.Min.Y+vertMargin,
	)
	op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
	op.LayoutOptions.LineSpacing = fontLineSpacing

	DrawText(dst, text, ClearFace, op)
}

func ClearDebugMsgs() {
	dm := &TheDebugPrintManager

	dm.DebugMsgs = dm.DebugMsgs[:0]
}
