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

	svFocused bool
	hFocused  bool
	aFocused  bool

	Hue        float64
	Saturation float64
	Value      float64
	Alpha      float64

	CopyButton  *TextButton
	PasteButton *TextButton
}

func NewColorPicker() *ColorPicker {
	cp := new(ColorPicker)

	cp.CopyButton = NewTextButton()
	cp.PasteButton = NewTextButton()

	cp.CopyButton.Text = "copy"
	cp.CopyButton.OnClick = func() {
		str := ColorToString(cp.Color())
		ClipboardWriteText(str)
	}
	cp.PasteButton.Text = "paste"
	cp.PasteButton.OnClick = func() {
		str := ClipboardReadText()
		if c, err := ParseColorString(str); err == nil {
			cp.SetColor(c)
		}
	}

	return cp
}

func (cp *ColorPicker) Color() color.NRGBA {
	c := ColorFromHSV(cp.Hue, cp.Saturation, cp.Value)
	c.A = uint8(255 * cp.Alpha)
	return c
}

func (cp *ColorPicker) SetColor(c color.NRGBA) {
	hsv := ColorToHSV(c)
	cp.Hue = hsv[0]
	cp.Saturation = hsv[1]
	cp.Value = hsv[2]
	cp.Alpha = f64(c.A) / 255
}

// rectType
// 0 - saturation value rect
// 1 - hue rect
// 2 - alpha rect
// 3 - text rect
// 4 - copy button
// 5 - paste button
func (cp *ColorPicker) layoutRectImpl(rectType int) FRectangle {
	heights := [...]float64{
		0.4,  // saturation value rect
		0.1,  // hue rect
		0.1,  // alpha rect
		0.25, // text rect
		0.15, // copy paste rect
	}

	y := cp.Rect.Min.Y

	for i, h := range heights {
		if i >= rectType {
			break
		}
		if i >= 4 {
			break
		}
		y += h * cp.Rect.Dy()
	}

	height := heights[min(rectType, len(heights)-1)] * cp.Rect.Dy()
	width := cp.Rect.Dx()

	if rectType == 4 || rectType == 5 {
		width = cp.Rect.Dx() * 0.5
	}

	x := cp.Rect.Min.X
	if rectType == 5 {
		x = cp.Rect.Max.X - width
	}

	rect := FRectXYWH(x, y, width, height)

	return rect.Inset(3)
}

func (cp *ColorPicker) svRect() FRectangle {
	return cp.layoutRectImpl(0)
}

func (cp *ColorPicker) hRect() FRectangle {
	return cp.layoutRectImpl(1)
}

func (cp *ColorPicker) aRect() FRectangle {
	return cp.layoutRectImpl(2)
}

func (cp *ColorPicker) textRect() FRectangle {
	return cp.layoutRectImpl(3)
}

func (cp *ColorPicker) copyButtonRect() FRectangle {
	return cp.layoutRectImpl(4)
}

func (cp *ColorPicker) pasteButtonRect() FRectangle {
	return cp.layoutRectImpl(5)
}

func (cp *ColorPicker) Update() {
	pt := CursorFPt()

	svRect := cp.svRect()
	hRect := cp.hRect()
	aRect := cp.aRect()

	if pt.In(svRect) && IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		cp.svFocused = true
		cp.hFocused = false
		cp.aFocused = false
	}

	if pt.In(hRect) && IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		cp.svFocused = false
		cp.hFocused = true
		cp.aFocused = false
	}

	if pt.In(aRect) && IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		cp.svFocused = false
		cp.hFocused = false
		cp.aFocused = true
	}

	if !IsMouseButtonPressed(eb.MouseButtonLeft) {
		cp.svFocused = false
		cp.hFocused = false
		cp.aFocused = false
	}

	if cp.svFocused {
		s := (pt.X - svRect.Min.X) / svRect.Dx()
		v := (pt.Y - svRect.Min.Y) / svRect.Dy()
		s = Clamp(s, 0, 1)
		v = Clamp(v, 0, 1)

		v = 1 - v

		cp.Saturation = s
		cp.Value = v
	}

	if cp.hFocused {
		h := (pt.X - hRect.Min.X) / hRect.Dx()
		h = Clamp(h, 0, 1)

		cp.Hue = h * Pi * 2
	}

	if cp.aFocused {
		a := (pt.X - aRect.Min.X) / aRect.Dx()
		a = Clamp(a, 0, 1)

		cp.Alpha = a
	}

	cp.CopyButton.Rect = cp.copyButtonRect()
	cp.PasteButton.Rect = cp.pasteButtonRect()

	cp.CopyButton.Update()
	cp.PasteButton.Update()
}

func (cp *ColorPicker) Draw(dst *eb.Image) {
	// draw bg rect
	{
		DrawFilledRect(dst, cp.Rect, color.NRGBA{0, 0, 0, 130}, true)
	}
	svRect := cp.svRect()
	hRect := cp.hRect()
	aRect := cp.aRect()
	textRect := cp.textRect()

	// draw sv rect
	{
		hue01 := f32(cp.Hue / (Pi * 2))

		verts := [4]eb.Vertex{
			{
				DstX: f32(svRect.Min.X), DstY: f32(svRect.Min.Y),
				ColorR: hue01, ColorG: 0, ColorB: 1, ColorA: 1,
			},
			{
				DstX: f32(svRect.Max.X), DstY: f32(svRect.Min.Y),
				ColorR: hue01, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				DstX: f32(svRect.Max.X), DstY: f32(svRect.Max.Y),
				ColorR: hue01, ColorG: 1, ColorB: 0, ColorA: 1,
			},
			{
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
		verts := [4]eb.Vertex{
			{
				DstX: f32(hRect.Min.X), DstY: f32(hRect.Min.Y),
				ColorR: 0, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				DstX: f32(hRect.Max.X), DstY: f32(hRect.Min.Y),
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				DstX: f32(hRect.Max.X), DstY: f32(hRect.Max.Y),
				ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1,
			},
			{
				DstX: f32(hRect.Min.X), DstY: f32(hRect.Max.Y),
				ColorR: 0, ColorG: 1, ColorB: 1, ColorA: 1,
			},
		}

		indices := [6]uint16{0, 1, 2, 0, 2, 3}

		op := &eb.DrawTrianglesShaderOptions{}
		op.AntiAlias = true

		dst.DrawTrianglesShader(verts[:], indices[:], hsvShader, op)
	}

	// draw alpha rect
	{
		verts := [4]eb.Vertex{
			{
				DstX: f32(aRect.Min.X), DstY: f32(aRect.Min.Y),
				ColorR: 0, ColorG: 0, ColorB: 0, ColorA: 1,
			},
			{
				DstX: f32(aRect.Max.X), DstY: f32(aRect.Min.Y),
				ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
			},
			{
				DstX: f32(aRect.Max.X), DstY: f32(aRect.Max.Y),
				ColorR: 0, ColorG: 0, ColorB: 1, ColorA: 1,
			},
			{
				DstX: f32(aRect.Min.X), DstY: f32(aRect.Max.Y),
				ColorR: 0, ColorG: 0, ColorB: 0, ColorA: 1,
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

	// draw alpha cursor
	{
		cursorX := aRect.Min.X + aRect.Dx()*cp.Alpha
		cursorY := aRect.Min.Y + aRect.Dy()*0.5

		cursorRect := FRectWH(12, aRect.Dy()+5)
		cursorRect = CenterFRectangle(cursorRect, cursorX, cursorY)

		alpha := uint8(cp.Alpha * 255)

		DrawFilledRect(dst, cursorRect, color.NRGBA{alpha, alpha, alpha, 255}, true)
		StrokeRect(dst, cursorRect, 3, color.NRGBA{255, 255, 255, 255}, true)
	}

	// draw text
	{
		textRectHeight := textRect.Dy() / 3

		// TODO : this can be cached
		hsvSizeX, hsvSizeY := ebt.Measure("1.00 1.00 1.00 1.00", ClearFace, FontLineSpacing(ClearFace))
		rgbSizeX, rgbSizeY := ebt.Measure("255 255 255 255", ClearFace, FontLineSpacing(ClearFace))
		hexSizeX, hexSizeY := ebt.Measure("#FFFFFFFF", ClearFace, FontLineSpacing(ClearFace))

		sizeX, sizeY := max(hsvSizeX, rgbSizeX, hexSizeX), max(hsvSizeY, rgbSizeY, hexSizeY)

		scale := min(textRect.Dx()/sizeX, textRectHeight/sizeY)

		op := &ebt.DrawOptions{}
		op.ColorScale.ScaleWithColor(color.NRGBA{255, 255, 255, 255})
		op.Filter = eb.FilterLinear

		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(textRect.Min.X, textRect.Min.Y+textRectHeight*0)

		ebt.Draw(dst, fmt.Sprintf("%.2f %.2f %.2f %.2f", cp.Hue, cp.Saturation, cp.Value, cp.Alpha), ClearFace, op)

		op.GeoM.Reset()
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(textRect.Min.X, textRect.Min.Y+textRectHeight*1)

		c := cp.Color()
		ebt.Draw(dst, fmt.Sprintf("%d %d %d %d", c.R, c.G, c.B, c.A), ClearFace, op)

		op.GeoM.Reset()
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(textRect.Min.X, textRect.Min.Y+textRectHeight*2)

		ebt.Draw(dst, ColorToString(c), ClearFace, op)
	}

	cp.CopyButton.Draw(dst)
	cp.PasteButton.Draw(dst)
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
