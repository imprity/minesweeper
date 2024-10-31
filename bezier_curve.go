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
const BezierDrawerControlPointScale = 3

type BezierDrawer struct {
	Rect FRectangle

	Points  [4]FPoint
	Focused int

	SnapTo0ButtonP0 *TextButton
	SnapTo0ButtonP3 *TextButton

	SnapToOriginP1 *TextButton
	SnapToOriginP2 *TextButton

	CopyButton *TextButton
	PasteButton *TextButton
}

func NewBezierDrawer() *BezierDrawer {
	bd := new(BezierDrawer)

	bd.Points[0] = FPt(0, 0)
	bd.Points[1] = FPt(0.3, 0)
	bd.Points[2] = FPt(0.7, 1)
	bd.Points[3] = FPt(1, 1)

	bd.SnapTo0ButtonP0 = NewTextButton()
	bd.SnapTo0ButtonP0.Text = "Set To 0"
	bd.SnapTo0ButtonP0.OnClick = func() {
		bd.Points[0].Y = 0
	}

	bd.SnapTo0ButtonP3 = NewTextButton()
	bd.SnapTo0ButtonP3.Text = "Set To 0"
	bd.SnapTo0ButtonP3.OnClick = func() {
		bd.Points[3].Y = 0
	}

	bd.SnapToOriginP1 = NewTextButton()
	bd.SnapToOriginP1.Text = "To Origin"
	bd.SnapToOriginP1.OnClick = func() {
		bd.Points[1] = bd.Points[0]
	}

	bd.SnapToOriginP2 = NewTextButton()
	bd.SnapToOriginP2.Text = "To Origin"
	bd.SnapToOriginP2.OnClick = func() {
		bd.Points[2] = bd.Points[3]
	}

	bd.CopyButton = NewTextButton()
	bd.CopyButton.Text = "Copy"
	bd.CopyButton.OnClick = func() {
		ErrorLogger.Print("TODO: not implemented")
	}

	bd.PasteButton = NewTextButton()
	bd.PasteButton.Text = "Paste"
	bd.PasteButton.OnClick = func() {
		ErrorLogger.Print("TODO: not implemented")
	}

	bd.Focused = -1

	return bd
}

func (bd *BezierDrawer) CurveRect() FRectangle {
	return FRect(
		bd.Rect.Max.X-bd.Rect.Dx()*0.55, bd.Rect.Min.Y,
		bd.Rect.Max.X, bd.Rect.Max.Y,
	).Inset(1)
}

func (bd *BezierDrawer) Update() error {
	// ==================
	// update rects
	// ==================

	// TEST TEST TEST TEST TEST TEST
	// this should be done by others, not this class
	bd.Rect = FRectWH(ScreenWidth*0.6, ScreenHeight*0.6)
	bd.Rect = CenterFRectangle(bd.Rect, ScreenWidth*0.5, ScreenHeight*0.5)
	// TEST TEST TEST TEST TEST TEST

	// update button rects
	bd.SnapTo0ButtonP0.Rect = FRectXYWH(bd.Rect.Min.X, bd.Rect.Min.Y, bd.Rect.Dx()*0.3, bd.Rect.Dy()*0.15).Inset(1)
	bd.SnapToOriginP1.Rect = FRectXYWH(bd.Rect.Min.X, bd.Rect.Min.Y+bd.Rect.Dy()*0.15, bd.Rect.Dx()*0.3, bd.Rect.Dy()*0.15).Inset(1)

	bd.SnapTo0ButtonP3.Rect = FRectXYWH(bd.Rect.Min.X, bd.Rect.Min.Y+bd.Rect.Dy()*0.3, bd.Rect.Dx()*0.3, bd.Rect.Dy()*0.15).Inset(1)
	bd.SnapToOriginP2.Rect = FRectXYWH(bd.Rect.Min.X, bd.Rect.Min.Y+bd.Rect.Dy()*0.45, bd.Rect.Dx()*0.3, bd.Rect.Dy()*0.15).Inset(1)

	bd.PasteButton.Rect = FRectXYWH(bd.Rect.Min.X, bd.Rect.Min.Y+bd.Rect.Dy()*0.8, bd.Rect.Dx()*0.2, bd.Rect.Dy()*0.2).Inset(1)
	bd.CopyButton.Rect = FRectXYWH(bd.Rect.Min.X + bd.Rect.Dx() * 0.2, bd.Rect.Min.Y+bd.Rect.Dy()*0.8, bd.Rect.Dx()*0.2, bd.Rect.Dy()*0.2).Inset(1)

	bd.SnapTo0ButtonP0.Disabled = bd.Focused >= 0
	bd.SnapTo0ButtonP3.Disabled = bd.Focused >= 0

	bd.SnapToOriginP1.Disabled = bd.Focused >= 0
	bd.SnapToOriginP2.Disabled = bd.Focused >= 0

	bd.CopyButton.Disabled = bd.Focused >= 0
	bd.PasteButton.Disabled = bd.Focused >= 0

	bd.PasteButton.Update()

	prevP0 := bd.Points[0]
	prevP3 := bd.Points[3]

	cursor := CursorFPt()

	if IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		focusPriority := [4]int{1, 2, 0, 3}

		for _, i := range focusPriority {
			sp := bd.ControlPosToScreenPos(bd.Points[i])
			if sp.Sub(cursor).LengthSquared() < 20*20 {
				bd.Focused = i
				break
			}
		}
	}

	if bd.Focused >= 0 {
		bd.Points[bd.Focused] = bd.ScreenPosToControlPos(cursor)
	}

	if !IsMouseButtonPressed(eb.MouseButtonLeft) {
		bd.Focused = -1
	}

	bd.SnapTo0ButtonP0.Update()
	bd.SnapTo0ButtonP3.Update()

	bd.SnapToOriginP1.Update()
	bd.SnapToOriginP2.Update()

	// clamp control points
	bd.Points[0].X = 0
	bd.Points[3].X = 1
	bd.Points[0].Y = Clamp(bd.Points[0].Y, -1, 1)
	bd.Points[3].Y = Clamp(bd.Points[3].Y, -1, 1)

	bd.Points[1].X = Clamp(bd.Points[1].X, 0, 1.0/BezierDrawerControlPointScale)
	bd.Points[2].X = Clamp(bd.Points[2].X, 1-1.0/BezierDrawerControlPointScale, 1)

	delta0 := bd.Points[0].Sub(prevP0)
	delta3 := bd.Points[3].Sub(prevP3)

	bd.Points[1] = bd.Points[1].Add(delta0)
	bd.Points[2] = bd.Points[2].Add(delta3)

	bd.CopyButton.Update()

	// TEST TEST TEST TEST TEST TEST TEST TEST
	DebugPrint("p0", fmt.Sprintf("%.2f, %.2f", bd.Points[0].X, bd.Points[0].Y))
	DebugPrint("p1", fmt.Sprintf("%.2f, %.2f", bd.Points[1].X, bd.Points[1].Y))
	DebugPrint("p2", fmt.Sprintf("%.2f, %.2f", bd.Points[2].X, bd.Points[2].Y))
	DebugPrint("p3", fmt.Sprintf("%.2f, %.2f", bd.Points[3].X, bd.Points[3].Y))
	// TEST TEST TEST TEST TEST TEST TEST TEST

	return nil
}

func (bd *BezierDrawer) Draw(dst *eb.Image) {
	// =========================
	// draw CurveRect
	// =========================
	{
		rect := bd.CurveRect()
		StrokeRect(dst, rect, 3, color.NRGBA{255, 255, 255, 255})
		center := FRectangleCenter(rect)
		StrokeLine(
			dst,
			rect.Min.X, center.Y,
			rect.Max.X, center.Y,
			3,
			color.NRGBA{255, 255, 255, 255},
		)
	}

	cps := [4]FPoint{}

	cps[0], cps[3] = bd.Points[0], bd.Points[3]

	cps[1] = bd.Points[1].Sub(bd.Points[0]).Scale(BezierDrawerControlPointScale).Add(bd.Points[0])
	cps[2] = bd.Points[2].Sub(bd.Points[3]).Scale(BezierDrawerControlPointScale).Add(bd.Points[3])

	// =========================
	// draw BezierCurve
	// =========================
	{
		t := f64(0)

		for t < 1 {
			p0 := BezierCurveFPt(cps[0], cps[1], cps[2], cps[3], t)
			t += 0.02
			if t > 1 {
				t = 1
			}
			p1 := BezierCurveFPt(cps[0], cps[1], cps[2], cps[3], t)

			p0 = bd.ControlPosToScreenPos(p0)
			p1 = bd.ControlPosToScreenPos(p1)

			StrokeLine(
				dst,
				p0.X, p0.Y,
				p1.X, p1.Y,
				4,
				color.NRGBA{255, 0, 0, 255},
			)
		}
	}

	// =========================
	// draw using newton
	// =========================
	{
		t := f64(0)

		for t < 1 {
			var p0 FPoint
			var p1 FPoint

			p0.X = Lerp(cps[0].X, cps[3].X, t)
			newtonT := BezierCurveNewton(cps[0].X, cps[1].X, cps[2].X, cps[3].X, p0.X)
			p0.Y = BezierCurve(cps[0].Y, cps[1].Y, cps[2].Y, cps[3].Y, newtonT)
			t += 0.01
			if t > 1 {
				t = 1
			}
			p1.X = Lerp(cps[0].X, cps[3].X, t)
			newtonT = BezierCurveNewton(cps[0].X, cps[1].X, cps[2].X, cps[3].X, p1.X)
			p1.Y = BezierCurve(cps[0].Y, cps[1].Y, cps[2].Y, cps[3].Y, newtonT)

			p0 = bd.ControlPosToScreenPos(p0)
			p1 = bd.ControlPosToScreenPos(p1)

			StrokeLine(
				dst,
				p0.X, p0.Y,
				p1.X, p1.Y,
				4,
				color.NRGBA{0, 0, 255, 255},
			)
		}
	}

	// =========================
	// draw control points
	// =========================
	{
		sps := [4]FPoint{} // screen sps

		for i, p := range bd.Points {
			sps[i] = bd.ControlPosToScreenPos(p)
		}

		DrawFilledCircle(dst, sps[0].X, sps[0].Y, 7, color.NRGBA{255, 255, 255, 255})
		StrokeCircle(dst, sps[0].X, sps[0].Y, 7, 2, color.NRGBA{255, 0, 0, 255})

		DrawFilledCircle(dst, sps[3].X, sps[3].Y, 7, color.NRGBA{255, 255, 255, 255})
		StrokeCircle(dst, sps[3].X, sps[3].Y, 7, 2, color.NRGBA{255, 0, 0, 255})

		StrokeLine(
			dst,
			sps[0].X, sps[0].Y,
			sps[1].X, sps[1].Y,
			2,
			color.NRGBA{0, 255, 0, 255},
		)

		StrokeLine(
			dst,
			sps[2].X, sps[2].Y,
			sps[3].X, sps[3].Y,
			2,
			color.NRGBA{0, 255, 0, 255},
		)

		DrawFilledCircle(dst, sps[1].X, sps[1].Y, 7, color.NRGBA{255, 255, 255, 255})
		StrokeCircle(dst, sps[1].X, sps[1].Y, 7, 2, color.NRGBA{255, 1, 1, 255})

		DrawFilledCircle(dst, sps[2].X, sps[2].Y, 7, color.NRGBA{255, 255, 255, 255})
		StrokeCircle(dst, sps[2].X, sps[2].Y, 7, 2, color.NRGBA{255, 0, 0, 255})
	}

	bd.SnapTo0ButtonP0.Draw(dst)
	bd.SnapTo0ButtonP3.Draw(dst)

	bd.SnapToOriginP1.Draw(dst)
	bd.SnapToOriginP2.Draw(dst)

	bd.CopyButton.Draw(dst)
	bd.PasteButton.Draw(dst)
}

func (bd *BezierDrawer) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (bd *BezierDrawer) ScreenPosToControlPos(pos FPoint) FPoint {
	rect := bd.CurveRect()

	pos.X -= rect.Min.X
	pos.Y -= (rect.Min.Y + rect.Max.Y) * 0.5

	pos.X /= rect.Dx()
	pos.Y /= rect.Dy() * 0.5

	pos.Y = -pos.Y

	return pos
}

func (bd *BezierDrawer) ControlPosToScreenPos(pos FPoint) FPoint {
	rect := bd.CurveRect()

	pos.Y = -pos.Y
	pos.X = pos.X*rect.Dx() + rect.Min.X
	pos.Y = pos.Y*rect.Dy()*0.5 + (rect.Min.Y+rect.Max.Y)*0.5

	return pos
}

func BezierCurve(p0, p1, p2, p3, t float64) float64 {
	it := 1 - t
	return it*it*it*p0 + 3*it*it*t*p1 + 3*it*t*t*p2 + t*t*t*p3
}

func BezierCurveFPt(p0, p1, p2, p3 FPoint, t float64) FPoint {
	return FPt(
		BezierCurve(p0.X, p1.X, p2.X, p3.X, t),
		BezierCurve(p0.Y, p1.Y, p2.Y, p3.Y, t),
	)
}

// approximates t for given n in bezier curve using Newton's method
// hard coded to only support 0 - 1
func BezierCurveNewton(p0, p1, p2, p3, n float64) float64 {
	n = Clamp(n, 0, 1)
	t := n
	for range 4 {
		it := 1 - t
		f := BezierCurve(p0, p1, p2, p3, t) - n
		fd := 3*it*it*(p1-p0) + 6*it*t*(p2-p1) + 3*t*t*(p3-p2)
		if Abs(fd) < 0.0001 {
			break
		}
		if Abs(f) < 0.0001 {
			break
		}
		t = t - f/fd
		t = Clamp(t, 0, 1)
	}

	return Clamp(t, 0, 1)
}
