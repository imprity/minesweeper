package main

import (
	"fmt"
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

	DebugMsgRenderTarget *eb.Image

	builder strings.Builder
}

func DebugPrintf(key, fmtStr string, values ...any) {
	DebugPuts(key, fmt.Sprintf(fmtStr, values...))
}

func DebugPrint(key string, values ...any) {
	DebugPuts(key, fmt.Sprint(values...))
}

func DebugPuts(key, value string) {
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

func DebugPrintfPersist(key, fmtStr string, values ...any) {
	DebugPutsPersist(key, fmt.Sprintf(fmtStr, values...))
}

func DebugPrintPersist(key string, values ...any) {
	DebugPutsPersist(key, fmt.Sprint(values...))
}

func DebugPutsPersist(key, value string) {
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

	// update width and height of background rect
	boxW, boxH := w*scale+hozMargin*2, h*scale+vertMargin*2

	rect := FRectWH(boxW, boxH)

	createBuf := dm.DebugMsgRenderTarget == nil
	createBuf = createBuf || dm.DebugMsgRenderTarget.Bounds().Dx() < int(boxW+1)
	createBuf = createBuf || dm.DebugMsgRenderTarget.Bounds().Dy() < int(boxH+1)

	if createBuf {
		if dm.DebugMsgRenderTarget != nil {
			dm.DebugMsgRenderTarget.Deallocate()
		}
		dm.DebugMsgRenderTarget = eb.NewImageWithOptions(
			RectWH(int(boxW+1), int(boxH+1)),
			&eb.NewImageOptions{Unmanaged: true},
		)
	}

	dm.DebugMsgRenderTarget.Clear()

	// draw background
	FillRect(
		dm.DebugMsgRenderTarget,
		rect,
		color.NRGBA{255, 255, 255, 255},
	)
	FillRect(
		dm.DebugMsgRenderTarget,
		rect.Inset(2),
		color.NRGBA{0, 0, 0, 255},
	)

	// draw text
	{
		op := &DrawTextOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(
			hozMargin, vertMargin,
		)
		op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
		op.LayoutOptions.LineSpacing = fontLineSpacing

		DrawText(dm.DebugMsgRenderTarget, text, ClearFace, op)
	}

	// draw DebugMsgRenderTarget
	{
		dstRect := RectToFRect(dst.Bounds())
		op := &DrawImageOptions{}
		op.GeoM.Translate(dstRect.Max.X-boxW, dstRect.Max.Y-boxH)
		DrawImage(dst, dm.DebugMsgRenderTarget, op)
	}
}

func ClearDebugMsgs() {
	dm := &TheDebugPrintManager

	dm.DebugMsgs = dm.DebugMsgs[:0]
}
