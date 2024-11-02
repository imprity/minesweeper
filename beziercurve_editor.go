package main

import (
	"fmt"
	eb "github.com/hajimehoshi/ebiten/v2"
	"image/color"
)

var _ = fmt.Printf

//	    2 - 3
//	  *-----*
//	 /
//	/
//	*-----*
//	 0 - 1
//
// Distance between 0-1 and 2-3 are scaled to this
const BezierEditorControlPointScale = 3.0

type BezierEditor struct {
	Rect FRectangle

	Points  [4]FPoint
	Focused int

	SnapTo0ButtonP0 *TextButton
	SnapTo0ButtonP3 *TextButton

	SnapToOriginP1 *TextButton
	SnapToOriginP2 *TextButton

	CopyButton  *TextButton
	PasteButton *TextButton
}

func NewBezierEditor() *BezierEditor {
	be := new(BezierEditor)

	be.Points[0] = FPt(0, 0)
	be.Points[1] = FPt(0.3, 0)
	be.Points[2] = FPt(0.7, 1)
	be.Points[3] = FPt(1, 1)

	be.SnapTo0ButtonP0 = NewTextButton()
	be.SnapTo0ButtonP0.Text = "Set To 0"
	be.SnapTo0ButtonP0.OnClick = func() {
		be.Points[0].Y = 0
	}

	be.SnapTo0ButtonP3 = NewTextButton()
	be.SnapTo0ButtonP3.Text = "Set To 0"
	be.SnapTo0ButtonP3.OnClick = func() {
		be.Points[3].Y = 0
	}

	be.SnapToOriginP1 = NewTextButton()
	be.SnapToOriginP1.Text = "To Origin"
	be.SnapToOriginP1.OnClick = func() {
		be.Points[1] = be.Points[0]
	}

	be.SnapToOriginP2 = NewTextButton()
	be.SnapToOriginP2.Text = "To Origin"
	be.SnapToOriginP2.OnClick = func() {
		be.Points[2] = be.Points[3]
	}

	be.CopyButton = NewTextButton()
	be.CopyButton.Text = "Copy"
	be.CopyButton.OnClick = func() {
		ErrorLogger.Print("TODO: not implemented")
	}

	be.PasteButton = NewTextButton()
	be.PasteButton.Text = "Paste"
	be.PasteButton.OnClick = func() {
		ErrorLogger.Print("TODO: not implemented")
	}

	be.Focused = -1

	return be
}

func (be *BezierEditor) CurveRect() FRectangle {
	return FRect(
		be.Rect.Max.X-be.Rect.Dx()*0.65, be.Rect.Min.Y,
		be.Rect.Max.X, be.Rect.Max.Y,
	).Inset(1)
}

func (be *BezierEditor) Update() error {
	// ==================
	// update rects
	// ==================

	// update button rects
	be.SnapTo0ButtonP0.Rect = FRectXYWH(be.Rect.Min.X, be.Rect.Min.Y, be.Rect.Dx()*0.3, be.Rect.Dy()*0.15).Inset(1)
	be.SnapToOriginP1.Rect = FRectXYWH(be.Rect.Min.X, be.Rect.Min.Y+be.Rect.Dy()*0.15, be.Rect.Dx()*0.3, be.Rect.Dy()*0.15).Inset(1)

	be.SnapTo0ButtonP3.Rect = FRectXYWH(be.Rect.Min.X, be.Rect.Min.Y+be.Rect.Dy()*0.3, be.Rect.Dx()*0.3, be.Rect.Dy()*0.15).Inset(1)
	be.SnapToOriginP2.Rect = FRectXYWH(be.Rect.Min.X, be.Rect.Min.Y+be.Rect.Dy()*0.45, be.Rect.Dx()*0.3, be.Rect.Dy()*0.15).Inset(1)

	be.PasteButton.Rect = FRectXYWH(be.Rect.Min.X, be.Rect.Max.Y-be.Rect.Dy()*0.3, be.Rect.Dx()*0.3, be.Rect.Dy()*0.15).Inset(1)
	be.CopyButton.Rect = FRectXYWH(be.Rect.Min.X, be.Rect.Max.Y-be.Rect.Dy()*0.15, be.Rect.Dx()*0.3, be.Rect.Dy()*0.15).Inset(1)

	be.SnapTo0ButtonP0.Disabled = be.Focused >= 0
	be.SnapTo0ButtonP3.Disabled = be.Focused >= 0

	be.SnapToOriginP1.Disabled = be.Focused >= 0
	be.SnapToOriginP2.Disabled = be.Focused >= 0

	be.CopyButton.Disabled = be.Focused >= 0
	be.PasteButton.Disabled = be.Focused >= 0

	be.PasteButton.Update()

	prevP0 := be.Points[0]
	prevP3 := be.Points[3]

	cursor := CursorFPt()

	if IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		focusPriority := [4]int{1, 2, 0, 3}

		for _, i := range focusPriority {
			sp := be.ControlPosToScreenPos(be.Points[i])
			if sp.Sub(cursor).LengthSquared() < 20*20 {
				be.Focused = i
				break
			}
		}
	}

	if be.Focused >= 0 {
		be.Points[be.Focused] = be.ScreenPosToControlPos(cursor)
	}

	if !IsMouseButtonPressed(eb.MouseButtonLeft) {
		be.Focused = -1
	}

	be.SnapTo0ButtonP0.Update()
	be.SnapTo0ButtonP3.Update()

	be.SnapToOriginP1.Update()
	be.SnapToOriginP2.Update()

	// clamp control points
	be.Points[0].X = Clamp(be.Points[0].X, 0, 1)
	be.Points[3].X = Clamp(be.Points[3].X, 0, 1)
	be.Points[0].Y = Clamp(be.Points[0].Y, -1, 1)
	be.Points[3].Y = Clamp(be.Points[3].Y, -1, 1)

	be.Points[1].X = Clamp(be.Points[1].X, be.Points[0].X+0.0001, 1)
	be.Points[2].X = Clamp(be.Points[2].X, 0, be.Points[3].X-0.0001)

	delta0 := be.Points[0].Sub(prevP0)
	delta3 := be.Points[3].Sub(prevP3)

	be.Points[1] = be.Points[1].Add(delta0)
	be.Points[2] = be.Points[2].Add(delta3)

	be.CopyButton.Update()

	return nil
}

func (be *BezierEditor) Draw(dst *eb.Image) {
	data := be.GetBezierCurveData()

	// screen sps
	sps := [4]FPoint{}
	for i, p := range be.Points {
		sps[i] = be.ControlPosToScreenPos(p)
	}

	curveRect := be.CurveRect()

	// =========================
	// draw background
	// =========================
	DrawFilledRect(dst, be.Rect, color.NRGBA{0, 0, 0, 130})

	// =========================
	// draw CurveRect
	// =========================
	{
		StrokeRect(dst, curveRect, 2, color.NRGBA{255, 255, 255, 100})
		center := FRectangleCenter(curveRect)
		StrokeLine(
			dst,
			curveRect.Min.X, center.Y,
			curveRect.Max.X, center.Y,
			2,
			color.NRGBA{255, 255, 255, 100},
		)
	}

	// =========================
	// draw BezierCurve
	// =========================
	{
		strokeColor := color.NRGBA{255, 0, 0, 255}
		const strokeWidth = 3.8

		firstPoint := be.ControlPosToScreenPos(data.Points[0])

		StrokeLine(
			dst,
			curveRect.Min.X, firstPoint.Y,
			firstPoint.X, firstPoint.Y,
			strokeWidth,
			strokeColor,
		)

		t := f64(0)

		for t < 1 {
			p0 := BezierCurveFPt(data.Points[0], data.Points[1], data.Points[2], data.Points[3], t)
			t += 0.02
			if t > 1 {
				t = 1
			}
			p1 := BezierCurveFPt(data.Points[0], data.Points[1], data.Points[2], data.Points[3], t)

			p0 = be.ControlPosToScreenPos(p0)
			p1 = be.ControlPosToScreenPos(p1)

			StrokeLine(
				dst,
				p0.X, p0.Y,
				p1.X, p1.Y,
				strokeWidth,
				strokeColor,
			)
		}

		lastPoint := be.ControlPosToScreenPos(data.Points[3])

		StrokeLine(
			dst,
			lastPoint.X, lastPoint.Y,
			curveRect.Max.X, lastPoint.Y,
			strokeWidth,
			strokeColor,
		)
	}

	// =========================
	// draw using newton
	// =========================
	{
		strokeColor := color.NRGBA{0, 0, 255, 255}
		const strokeWidth = 4

		x := f64(0)

		for x < 1 {
			var p0 FPoint
			var p1 FPoint

			p0.X = x
			p0.Y = BezierCurveDataAsGraph(data, x)
			x += 0.01
			if x > 1 {
				x = 1
			}
			p1.X = x
			p1.Y = BezierCurveDataAsGraph(data, x)

			p0 = be.ControlPosToScreenPos(p0)
			p1 = be.ControlPosToScreenPos(p1)

			StrokeLine(
				dst,
				p0.X, p0.Y,
				p1.X, p1.Y,
				strokeWidth,
				strokeColor,
			)
		}
	}

	// =========================
	// draw control points
	// =========================
	{
		// draw bars
		StrokeLine(
			dst,
			sps[0].X, curveRect.Min.Y,
			sps[0].X, curveRect.Max.Y,
			1,
			color.NRGBA{255, 255, 255, 100},
		)

		StrokeLine(
			dst,
			sps[3].X, curveRect.Min.Y,
			sps[3].X, curveRect.Max.Y,
			1,
			color.NRGBA{255, 255, 255, 100},
		)

		circleFill := color.NRGBA{0, 0, 0, 255}
		circleStroke := color.NRGBA{255, 255, 255, 255}

		StrokeLine(
			dst,
			sps[0].X, sps[0].Y,
			sps[1].X, sps[1].Y,
			1,
			color.NRGBA{255, 255, 255, 255},
		)

		StrokeLine(
			dst,
			sps[2].X, sps[2].Y,
			sps[3].X, sps[3].Y,
			1,
			color.NRGBA{255, 255, 255, 255},
		)

		// draw circles
		DrawFilledCircle(dst, sps[0].X, sps[0].Y, 7, circleFill)
		StrokeCircle(dst, sps[0].X, sps[0].Y, 7, 2, circleStroke)

		DrawFilledCircle(dst, sps[3].X, sps[3].Y, 7, circleFill)
		StrokeCircle(dst, sps[3].X, sps[3].Y, 7, 2, circleStroke)

		DrawFilledCircle(dst, sps[1].X, sps[1].Y, 7, circleFill)
		StrokeCircle(dst, sps[1].X, sps[1].Y, 7, 2, circleStroke)

		DrawFilledCircle(dst, sps[2].X, sps[2].Y, 7, circleFill)
		StrokeCircle(dst, sps[2].X, sps[2].Y, 7, 2, circleStroke)
	}

	be.SnapTo0ButtonP0.Draw(dst)
	be.SnapTo0ButtonP3.Draw(dst)

	be.SnapToOriginP1.Draw(dst)
	be.SnapToOriginP2.Draw(dst)

	be.CopyButton.Draw(dst)
	be.PasteButton.Draw(dst)
}

func (be *BezierEditor) GetBezierCurveData() BezierCurveData {
	data := BezierCurveData{}

	data.Points[0], data.Points[3] = be.Points[0], be.Points[3]

	data.Points[1] = be.Points[1].Sub(be.Points[0]).Scale(BezierEditorControlPointScale).Add(be.Points[0])
	data.Points[2] = be.Points[2].Sub(be.Points[3]).Scale(BezierEditorControlPointScale).Add(be.Points[3])

	return data
}

func (be *BezierEditor) SetToBezierCurveData(data BezierCurveData) {
	// clamp the data
	data.Points[0].X = 0
	data.Points[3].X = 1
	data.Points[0].Y = Clamp(data.Points[0].Y, -1, 1)
	data.Points[3].Y = Clamp(data.Points[3].Y, -1, 1)

	data.Points[1].X = Clamp(data.Points[1].X, 0, 1)
	data.Points[2].X = Clamp(data.Points[2].X, 0, 1)

	// shrink points 1 and 2
	data.Points[1] = data.Points[1].Sub(data.Points[0]).Scale(1.0 / BezierEditorControlPointScale).Add(data.Points[0])
	data.Points[2] = data.Points[2].Sub(data.Points[3]).Scale(1.0 / BezierEditorControlPointScale).Add(data.Points[3])

	for i, dataP := range data.Points {
		be.Points[i] = dataP
	}
}

func (be *BezierEditor) ScreenPosToControlPos(pos FPoint) FPoint {
	rect := be.CurveRect()

	pos.X -= rect.Min.X
	pos.Y -= (rect.Min.Y + rect.Max.Y) * 0.5

	pos.X /= rect.Dx()
	pos.Y /= rect.Dy() * 0.5

	pos.Y = -pos.Y

	return pos
}

func (be *BezierEditor) ControlPosToScreenPos(pos FPoint) FPoint {
	rect := be.CurveRect()

	pos.Y = -pos.Y
	pos.X = pos.X*rect.Dx() + rect.Min.X
	pos.Y = pos.Y*rect.Dy()*0.5 + (rect.Min.Y+rect.Max.Y)*0.5

	return pos
}
