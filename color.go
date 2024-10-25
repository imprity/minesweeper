package main

import (
	"fmt"
	css "github.com/mazznoer/csscolorparser"
	"image/color"
	"math"
)

func ColorNormalized(clr color.Color, multiplyAlpha bool) [4]float64 {
	c := ColorToNRGBA(clr)
	r, g, b, a := f64(c.R)/255, f64(c.G)/255, f64(c.B)/255, f64(c.A)/255

	if multiplyAlpha {
		r *= a
		g *= a
		b *= a
	}

	return [4]float64{r, g, b, a}
}

func ColorToHSV(clr color.Color) [3]float64 {
	c := ColorToNRGBA(clr)
	r, g, b := f64(c.R)/255, f64(c.G)/255, f64(c.B)/255

	cMax := max(r, g, b)
	cMin := min(r, g, b)

	dist := cMax - cMin

	var hue float64

	if dist == 0 {
		hue = 0
	} else {
		if cMax == r {
			hue = math.Mod((g-b)/dist, 6)
		} else if cMax == g {
			hue = ((b - r) / dist) + 2
		} else {
			hue = ((r - g) / dist) + 4
		}
	}

	hue *= 60 * math.Pi / 180

	var saturation float64
	if cMax > 0 {
		saturation = dist / cMax
	}

	brightness := cMax

	// just in case
	saturation = Clamp(saturation, 0, 1)
	brightness = Clamp(brightness, 0, 1)

	for hue < 0 {
		hue += math.Pi * 2
	}

	for hue > math.Pi*2 {
		hue -= math.Pi * 2
	}

	return [3]float64{hue, saturation, brightness}
}

func ColorFromHSV(hue, saturation, value float64) color.NRGBA {
	for hue < 0 {
		hue += math.Pi * 2
	}

	for hue > math.Pi*2 {
		hue -= math.Pi * 2
	}

	saturation = Clamp(saturation, 0, 1)
	value = Clamp(value, 0, 1)

	c := saturation * value
	h := hue / (60 * math.Pi / 180)
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))

	var r, g, b float64
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

	r = Clamp(r, 0, 1)
	g = Clamp(g, 0, 1)
	b = Clamp(b, 0, 1)

	return color.NRGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func ColorToNRGBA(clr color.Color) color.NRGBA {
	if clr == nil {
		return color.NRGBA{}
	}
	return color.NRGBAModel.Convert(clr).(color.NRGBA)
}

func LerpColorRGB(c1, c2 color.Color, t float64) color.NRGBA {
	c1f := ColorNormalized(c1, false)
	c2f := ColorNormalized(c2, false)

	r := Lerp(c1f[0], c2f[0], t)
	g := Lerp(c1f[1], c2f[1], t)
	b := Lerp(c1f[2], c2f[2], t)

	return color.NRGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func LerpColorRGBA(c1, c2 color.Color, t float64) color.NRGBA {
	c1f := ColorNormalized(c1, false)
	c2f := ColorNormalized(c2, false)

	r := Lerp(c1f[0], c2f[0], t)
	g := Lerp(c1f[1], c2f[1], t)
	b := Lerp(c1f[2], c2f[2], t)
	a := Lerp(c1f[3], c2f[3], t)

	return color.NRGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), uint8(a * 255)}
}

func ColorFade(c color.Color, a float64) color.NRGBA {
	nc := ColorNormalized(c, false)
	return color.NRGBA{
		uint8(255 * nc[0]),
		uint8(255 * nc[1]),
		uint8(255 * nc[2]),
		uint8(255 * nc[3] * a),
	}
}

func ColorToString(clr color.Color) string {
	c := ColorToNRGBA(clr)
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

func ParseColorString(str string) (color.NRGBA, error) {
	c, err := css.Parse(str)

	if err != nil {
		return color.NRGBA{}, err
	}

	nrgba := color.NRGBA{
		R: uint8(255 * c.R),
		G: uint8(255 * c.G),
		B: uint8(255 * c.B),
		A: uint8(255 * c.A),
	}

	return nrgba, nil
}
