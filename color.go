package main

import (
	"image/color"
	"math"
)

func ColorNormalized(clr color.NRGBA, multiplyAlpha bool) [4]float64 {
	r, g, b, a := f64(clr.R)/255, f64(clr.G)/255, f64(clr.B)/255, f64(clr.A)/255

	if multiplyAlpha {
		r *= a
		g *= a
		b *= a
	}

	return [4]float64{r, g, b, a}
}

func ColorToHSV(color color.NRGBA) [3]float64 {
	r, g, b := f64(color.R)/255, f64(color.G)/255, f64(color.B)/255

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

	return [3]float64{hue, saturation, brightness}
}

func ColorFromHSV(hue, saturation, value float64) color.NRGBA {
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

	return color.NRGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}
