package main

import (
	"golang.org/x/exp/constraints"
	"image"
	"math"

	eb "github.com/hajimehoshi/ebiten/v2"
)

const Pi = math.Pi

// =================================
// FPoint
// =================================

type FPoint struct {
	X, Y float64
}

func FPt(x, y float64) FPoint {
	return FPoint{X: x, Y: y}
}

func (p FPoint) Add(q FPoint) FPoint {
	p.X += q.X
	p.Y += q.Y
	return p
}

func (p FPoint) Sub(q FPoint) FPoint {
	p.X -= q.X
	p.Y -= q.Y
	return p
}

func (p FPoint) Div(q FPoint) FPoint {
	p.X /= q.X
	p.Y /= q.Y
	return p
}

func (p FPoint) Eq(q FPoint) bool {
	return p.X == q.X && p.Y == q.Y
}

func (p FPoint) In(r FRectangle) bool {
	return r.Min.X <= p.X && p.X <= r.Max.X &&
		r.Min.Y <= p.Y && p.Y <= r.Max.Y
}

func (p FPoint) Mul(q FPoint) FPoint {
	p.X *= q.X
	p.Y *= q.Y
	return p
}

func (p FPoint) Rotate(theta float64) FPoint {
	cos := math.Cos(theta)
	sin := math.Sin(theta)

	return FPoint{
		X: cos*p.X - sin*p.Y,
		Y: sin*p.X + cos*p.Y,
	}
}

func (p FPoint) Scale(s float64) FPoint {
	return FPt(p.X*s, p.Y*s)
}

func (p FPoint) LengthSquared() float64 {
	return p.X*p.X + p.Y*p.Y
}

func (p FPoint) Length() float64 {
	return math.Sqrt(p.LengthSquared())
}

func (p FPoint) Normalize() FPoint {
	length := p.Length()
	return FPt(p.X/length, p.Y/length)
}

func FPointTransform(pt FPoint, geom eb.GeoM) FPoint {
	x, y := geom.Apply(pt.X, pt.Y)
	return FPt(x, y)
}

// =================================
// FRectangle
// =================================

type FRectangle struct {
	Min, Max FPoint
}

func FRect(x0, y0, x1, y1 float64) FRectangle {
	return FRectangle{
		Min: FPt(x0, y0),
		Max: FPt(x1, y1),
	}
}

// =================================================
// below is copy pasted frorm go image package
// but modified to be used for FRectangle
// license is at below
// =================================================

// Dx returns r's width.
func (r FRectangle) Dx() float64 {
	return r.Max.X - r.Min.X
}

// Dy returns r's height.
func (r FRectangle) Dy() float64 {
	return r.Max.Y - r.Min.Y
}

// Size returns r's width and height.
func (r FRectangle) Size() FPoint {
	return FPoint{
		r.Max.X - r.Min.X,
		r.Max.Y - r.Min.Y,
	}
}

// Add returns the rectangle r translated by p.
func (r FRectangle) Add(p FPoint) FRectangle {
	return FRectangle{
		FPoint{r.Min.X + p.X, r.Min.Y + p.Y},
		FPoint{r.Max.X + p.X, r.Max.Y + p.Y},
	}
}

// Sub returns the rectangle r translated by -p.
func (r FRectangle) Sub(p FPoint) FRectangle {
	return FRectangle{
		FPoint{r.Min.X - p.X, r.Min.Y - p.Y},
		FPoint{r.Max.X - p.X, r.Max.Y - p.Y},
	}
}

// Inset returns the rectangle r inset by n, which may be negative. If either
// of r's dimensions is less than 2*n then an empty rectangle near the center
// of r will be returned.
func (r FRectangle) Inset(n float64) FRectangle {
	if r.Dx() < 2*n {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += n
		r.Max.X -= n
	}
	if r.Dy() < 2*n {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += n
		r.Max.Y -= n
	}
	return r
}

// Intersect returns the largest rectangle contained by both r and s. If the
// two rectangles do not overlap then the zero rectangle will be returned.
func (r FRectangle) Intersect(s FRectangle) FRectangle {
	if r.Min.X < s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y < s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X > s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y > s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	// Letting r0 and s0 be the values of r and s at the time that the method
	// is called, this next line is equivalent to:
	//
	// if max(r0.Min.X, s0.Min.X) >= min(r0.Max.X, s0.Max.X) || likewiseForY { etc }
	if r.Empty() {
		return FRectangle{}
	}
	return r
}

// Union returns the smallest rectangle that contains both r and s.
func (r FRectangle) Union(s FRectangle) FRectangle {
	if r.Empty() {
		return s
	}
	if s.Empty() {
		return r
	}
	if r.Min.X > s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y > s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X < s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y < s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	return r
}

// Empty reports whether the rectangle contains no points.
func (r FRectangle) Empty() bool {
	return r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y
}

// Eq reports whether r and s contain the same set of points. All empty
// rectangles are considered equal.
func (r FRectangle) Eq(s FRectangle) bool {
	return r == s || r.Empty() && s.Empty()
}

// Overlaps reports whether r and s have a non-empty intersection.
func (r FRectangle) Overlaps(s FRectangle) bool {
	return !r.Empty() && !s.Empty() &&
		r.Min.X < s.Max.X && s.Min.X < r.Max.X &&
		r.Min.Y < s.Max.Y && s.Min.Y < r.Max.Y
}

// In reports whether every point in r is in s.
func (r FRectangle) In(s FRectangle) bool {
	if r.Empty() {
		return true
	}
	// Note that r.Max is an exclusive bound for r, so that r.In(s)
	// does not require that r.Max.In(s).
	return s.Min.X <= r.Min.X && r.Max.X <= s.Max.X &&
		s.Min.Y <= r.Min.Y && r.Max.Y <= s.Max.Y
}

// Canon returns the canonical version of r. The returned rectangle has minimum
// and maximum coordinates swapped if necessary so that it is well-formed.
func (r FRectangle) Canon() FRectangle {
	if r.Max.X < r.Min.X {
		r.Min.X, r.Max.X = r.Max.X, r.Min.X
	}
	if r.Max.Y < r.Min.Y {
		r.Min.Y, r.Max.Y = r.Max.Y, r.Min.Y
	}
	return r
}

// =======================================
// end of things I copied from google
// =======================================

// =================================
// collision checking
// =================================

func CheckCollisionRects(r1, r2 image.Rectangle) bool {
	return r1.Overlaps(r2)
}

func CheckCollisionFRects(r1, r2 FRectangle) bool {
	return r1.Overlaps(r2)
}

func CheckCollisionPtRect(pt image.Point, rect image.Rectangle) bool {
	return pt.In(rect)
}

func CheckCollisionFPtFRect(pt FPoint, rect FRectangle) bool {
	return pt.In(rect)
}

// =================================
// misc
// =================================

func PointToFPoint(p image.Point) FPoint {
	return FPoint{X: float64(p.X), Y: float64(p.Y)}
}

func FPointToPoint(p FPoint) image.Point {
	return image.Point{X: int(p.X), Y: int(p.Y)}
}

func RectToFRect(rect image.Rectangle) FRectangle {
	return FRectangle{
		Min: PointToFPoint(rect.Min),
		Max: PointToFPoint(rect.Max),
	}
}

func FRectToRect(rect FRectangle) image.Rectangle {
	return image.Rectangle{
		Min: FPointToPoint(rect.Min),
		Max: FPointToPoint(rect.Max),
	}
}

func RectWH(w, h int) image.Rectangle {
	return image.Rectangle{
		Min: image.Point{},
		Max: image.Point{w, h},
	}
}

func FRectWH(w, h float64) FRectangle {
	return FRectangle{
		Min: FPoint{0, 0},
		Max: FPoint{w, h},
	}
}

func RectXYWH(x, y, w, h int) image.Rectangle {
	return image.Rectangle{
		Min: image.Point{x, y},
		Max: image.Point{x + w, y + h},
	}
}

func FRectXYWH(x, y, w, h float64) FRectangle {
	return FRectangle{
		Min: FPoint{x, y},
		Max: FPoint{x + w, y + h},
	}
}

func RectangleCenter(rect image.Rectangle) image.Point {
	return image.Point{
		X: (rect.Min.X + rect.Max.X) / 2,
		Y: (rect.Min.Y + rect.Max.Y) / 2,
	}
}

func CenterRectangle(rect image.Rectangle, x, y int) image.Rectangle {
	halfW := rect.Dx() / 2
	halfH := rect.Dy() / 2

	return image.Rectangle{
		Min: image.Pt(x-halfW, y-halfH),
		Max: image.Pt(x+halfW, y+halfH),
	}
}

func FRectangleCenter(rect FRectangle) FPoint {
	return FPoint{
		X: (rect.Min.X + rect.Max.X) * 0.5,
		Y: (rect.Min.Y + rect.Max.Y) * 0.5,
	}
}

func CenterFRectangle(rect FRectangle, x, y float64) FRectangle {
	halfW := rect.Dx() * 0.5
	halfH := rect.Dy() * 0.5

	return FRectangle{
		Min: FPt(x-halfW, y-halfH),
		Max: FPt(x+halfW, y+halfH),
	}
}

func RectMoveTo(rect image.Rectangle, x, y int) image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(x, y),
		Max: image.Pt(x+rect.Dx(), y+rect.Dy()),
	}
}

func FRectMoveTo(rect FRectangle, x, y float64) FRectangle {
	return FRectangle{
		Min: FPt(x, y),
		Max: FPt(x+rect.Dx(), y+rect.Dy()),
	}
}

func FRectScale(rect FRectangle, scale float64) FRectangle {
	return FRectangle{
		Min: rect.Min.Scale(scale),
		Max: rect.Max.Scale(scale),
	}
}

func FRectScaleCentered(rect FRectangle, scale float64) FRectangle {
	newRect := FRectWH(rect.Dx()*scale, rect.Dy()*scale)
	center := FRectangleCenter(rect)
	return CenterFRectangle(newRect, center.X, center.Y)
}

func FRectLerp(rectA, rectB FRectangle, t float64) FRectangle {
	return FRect(
		Lerp(rectA.Min.X, rectB.Min.X, t),
		Lerp(rectA.Min.Y, rectB.Min.Y, t),
		Lerp(rectA.Max.X, rectB.Max.X, t),
		Lerp(rectA.Max.Y, rectB.Max.Y, t),
	)
}

func Clamp[N constraints.Integer | constraints.Float](n, minN, maxN N) N {
	n = min(n, maxN)
	n = max(n, minN)

	return n
}

func Abs[N constraints.Signed | constraints.Float](n N) N {
	if n < 0 {
		return n * -1
	}

	return n
}

func Lerp[F constraints.Float](a, b, t F) F {
	return a + (b-a)*t
}

func CloseTo(a, b float64) bool {
	d := a - b
	if -0.0001 <= d && d <= 0.001 { // very arbitrary epsilon
		return true
	}
	return false
}

func CloseToEx(a, b, errorMargin float64) bool {
	d := a - b
	if -errorMargin <= d && d <= errorMargin {
		return true
	}
	return false
}

// ========================
// bezier curve
// ========================

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

func BezierCurveDataAsGraph(data BezierCurveData, x float64) float64 {
	if x < data.Points[0].X {
		return data.Points[0].Y
	}
	if x > data.Points[3].X {
		return data.Points[3].Y
	}

	t := BezierCurveNewton(data.Points[0].X, data.Points[1].X, data.Points[2].X, data.Points[3].X, x)
	y := BezierCurve(data.Points[0].Y, data.Points[1].Y, data.Points[2].Y, data.Points[3].Y, t)

	return y
}

// ========================
// easing functions
// ========================

// copy pasted from https://easings.net/#

func EaseInCirc(t float64) float64 {
	return math.Pow(t, 2)
}

func EaseOutCirc(t float64) float64 {
	return 1 - math.Pow(1-t, 2)
}

func EaseInOutCirc(t float64) float64 {
	if t < 0.5 {
		return (1 - math.Sqrt(1-math.Pow(2*t, 2))) / 2
	} else {
		return (math.Sqrt(1-math.Pow(-2*t+2, 2)) + 1) / 2
	}
}

func EaseInCubic(t float64) float64 {
	return math.Pow(t, 3)
}

func EaseOutCubic(t float64) float64 {
	return 1 - math.Pow(1-t, 3)
}

func EaseInQuint(t float64) float64 {
	return math.Pow(t, 5)
}

func EaseOutQuint(t float64) float64 {
	return 1 - math.Pow(1-t, 5)
}

func EaseInElastic(t float64) float64 {
	const c4 = (2 * math.Pi) / 3

	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	return -math.Pow(2, 10*t-10) * math.Sin((t*10-10.75)*c4)
}

func EaseOutElastic(t float64) float64 {
	const c4 = (2 * math.Pi) / 3

	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*c4) + 1
}

func EaseInOutElastic(t float64) float64 {
	const c5 = (2 * math.Pi) / 4.5

	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}

	if t < 0.5 {
		return -(math.Pow(2, 20*t-10) * math.Sin((20*t-11.125)*c5)) / 2
	}

	return (math.Pow(2, -20*t+10)*math.Sin((20*t-11.125)*c5))/2 + 1
}

/*
Copyright (c) 2009 The Go Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
