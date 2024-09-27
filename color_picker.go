package main

import (
	"fmt"
	"image/color"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var hsvShader *eb.Shader

func init() {
	shader, err := eb.NewShader([]byte(hsvShaderCode))
	if err != nil {
		ErrorLogger.Fatalf("failed to load the shader %v", err)
	}

	hsvShader = shader
}

type ColorPicker struct {
	Rect FRectangle

	SvFocused bool
	HFocused  bool

	Hue        float64
	Saturation float64
	Value      float64
}

func (cp *ColorPicker) Color() color.NRGBA {
	return ColorFromHSV(cp.Hue, cp.Saturation, cp.Value)
}

func (cp *ColorPicker) SetColor(c color.NRGBA) {
	hsv := ColorToHSV(c)
	cp.Hue = hsv[0]
	cp.Saturation = hsv[1]
	cp.Value = hsv[2]
}

func (cp *ColorPicker) SvRect() FRectangle {
	return FRect(
		cp.Rect.Min.X, cp.Rect.Min.Y,
		cp.Rect.Max.X, cp.Rect.Min.Y+cp.Rect.Dy()*0.6,
	)
}

func (cp *ColorPicker) HRect() FRectangle {
	svRect := cp.SvRect()

	return FRectXYWH(
		cp.Rect.Min.X, svRect.Max.Y+cp.Rect.Dy()*0.05,
		cp.Rect.Dx(), cp.Rect.Dy()*0.1,
	)
}

func (cp *ColorPicker) TextRect() FRectangle {
	return FRect(
		cp.Rect.Min.X, cp.Rect.Max.Y-cp.Rect.Dy()*0.2,
		cp.Rect.Max.X, cp.Rect.Max.Y,
	)
}

func (cp *ColorPicker) Update() {
	pt := CursorFPt()

	svRect := cp.SvRect()
	hRect := cp.HRect()

	if pt.In(svRect) && IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		cp.SvFocused = true
		cp.HFocused = false
	}

	if pt.In(hRect) && IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		cp.SvFocused = false
		cp.HFocused = true
	}

	if !IsMouseButtonPressed(eb.MouseButtonLeft) {
		cp.SvFocused = false
		cp.HFocused = false
	}

	if cp.SvFocused {
		s := (pt.X - svRect.Min.X) / svRect.Dx()
		v := (pt.Y - svRect.Min.Y) / svRect.Dy()
		s = Clamp(s, 0, 1)
		v = Clamp(v, 0, 1)

		v = 1 - v

		cp.Saturation = s
		cp.Value = v
	}

	if cp.HFocused {
		h := (pt.X - hRect.Min.X) / hRect.Dx()
		h = Clamp(h, 0, 1)

		cp.Hue = h * Pi * 2
	}
}

func (cp *ColorPicker) Draw(dst *eb.Image) {
	svRect := cp.SvRect()
	hRect := cp.HRect()

	// draw sv rect
	{
		wRect := WhiteImage.Bounds()

		hue01 := f32(cp.Hue / (Pi * 2))

		verts := [4]eb.Vertex{
			{
				SrcX: f32(wRect.Min.X), SrcY: f32(wRect.Min.Y),
				DstX: f32(svRect.Min.X), DstY: f32(svRect.Min.Y),
				ColorR: hue01, ColorG: 0, ColorB: 1, ColorA: 1,
			},
			{
				SrcX: f32(wRect.Max.X), SrcY: f32(wRect.Min.Y),
				DstX: f32(svRect.Max.X), DstY: f32(svRect.Min.Y),
				ColorR: hue01, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				SrcX: f32(wRect.Max.X), SrcY: f32(wRect.Max.Y),
				DstX: f32(svRect.Max.X), DstY: f32(svRect.Max.Y),
				ColorR: hue01, ColorG: 1, ColorB: 0, ColorA: 1,
			},
			{
				SrcX: f32(wRect.Min.X), SrcY: f32(wRect.Max.Y),
				DstX: f32(svRect.Min.X), DstY: f32(svRect.Max.Y),
				ColorR: hue01, ColorG: 0, ColorB: 0, ColorA: 1,
			},
		}

		indices := [6]uint16{0, 1, 2, 0, 2, 3}

		op := &eb.DrawTrianglesShaderOptions{}
		op.AntiAlias = true

		dst.DrawTrianglesShader(verts[:], indices[:], hsvShader, op)
	}

	// draw hue rect
	{
		wRect := WhiteImage.Bounds()

		verts := [4]eb.Vertex{
			{
				SrcX: f32(wRect.Min.X), SrcY: f32(wRect.Min.Y),
				DstX: f32(hRect.Min.X), DstY: f32(hRect.Min.Y),
				ColorR: 0, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				SrcX: f32(wRect.Max.X), SrcY: f32(wRect.Min.Y),
				DstX: f32(hRect.Max.X), DstY: f32(hRect.Min.Y),
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				SrcX: f32(wRect.Max.X), SrcY: f32(wRect.Max.Y),
				DstX: f32(hRect.Max.X), DstY: f32(hRect.Max.Y),
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				SrcX: f32(wRect.Min.X), SrcY: f32(wRect.Max.Y),
				DstX: f32(hRect.Min.X), DstY: f32(hRect.Max.Y),
				ColorR: 0, ColorG: 1, ColorB: 1, ColorA: 1,
			},
		}

		indices := [6]uint16{0, 1, 2, 0, 2, 3}

		op := &eb.DrawTrianglesShaderOptions{}
		op.AntiAlias = true

		dst.DrawTrianglesShader(verts[:], indices[:], hsvShader, op)
	}

	// draw sv cursor
	{
		const cursorSize = 20

		cursorX := svRect.Min.X + svRect.Dx()*cp.Saturation
		cursorY := svRect.Min.Y + svRect.Dy()*(1-cp.Value)

		DrawFilledCircle(
			dst, cursorX, cursorY, cursorSize, cp.Color(), true)
		StrokeCircle(
			dst, cursorX, cursorY, cursorSize, 3, color.NRGBA{255, 255, 255, 255}, true)
	}

	// draw hue cursor
	{
		cursorX := hRect.Min.X + hRect.Dx()*(cp.Hue/(Pi*2))
		cursorY := hRect.Min.Y + hRect.Dy()*0.5

		cursorRect := FRectWH(12, hRect.Dy()+5)
		cursorRect = CenterFRectangle(cursorRect, cursorX, cursorY)

		DrawFilledRect(dst, cursorRect, ColorFromHSV(cp.Hue, 1, 1), true)
		StrokeRect(dst, cursorRect, 3, color.NRGBA{255, 255, 255, 255}, true)
	}

	// draw text
	{
		textRect := cp.TextRect()
		DrawFilledRect(dst, textRect, color.NRGBA{0, 0, 0, 130}, true)

		// TODO : this can be cached
		hsvSizeX, hsvSizeY := ebt.Measure("1.00 1.00 1.00", ClearFace, FontLineSpacing(ClearFace))
		rgbSizeX, rgbSizeY := ebt.Measure("255 255 255", ClearFace, FontLineSpacing(ClearFace))

		sizeX, sizeY := max(hsvSizeX, rgbSizeX), max(hsvSizeY, rgbSizeY)

		scale := min(textRect.Dx()/sizeX, textRect.Dy()/sizeY)

		op := &ebt.DrawOptions{}
		op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
		op.Filter = eb.FilterLinear

		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(textRect.Min.X, textRect.Min.Y)

		ebt.Draw(dst, fmt.Sprintf("%.2f %.2f %.2f", cp.Hue, cp.Saturation, cp.Value), ClearFace, op)

		op.GeoM.Reset()
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(textRect.Min.X, textRect.Min.Y+textRect.Dy()*0.5)

		c := cp.Color()
		ebt.Draw(dst, fmt.Sprintf("%d %d %d", c.R, c.G, c.B), ClearFace, op)
	}
}

var hsvShaderCode string = `
//kage:unit pixels
package main

const Pi = 3.14159265358979323846264338327950288419716939937510582097494459

func colorFromHSV(hue, saturation, value float) vec3 {
	c := saturation * value
	h := hue / (60 * Pi / 180)
	x := c * (1 - abs(mod(h, 2)-1))

	var r, g, b float
	if h < 1 {
		r, g, b = c, x, 0
	} else if h < 2 {
		r, g, b = x, c, 0
	} else if h < 3 {
		r, g, b = 0, c, x
	} else if h < 4 {
		r, g, b = 0, x, c
	} else if h < 5 {
		r, g, b = x, 0, c
	} else {
		r, g, b = c, 0, x
	}

	m := value - c

	r, g, b = r+m, g+m, b+m

	return vec3(r, g, b)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	hue := color.r * 2 * Pi
	saturation := color.g
	value := color.b

	c := colorFromHSV(hue, saturation, value)

	c.rgb *= color.a

	return vec4(c, color.a)
}
`
