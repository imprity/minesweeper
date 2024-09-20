package main

import (
	eb "github.com/hajimehoshi/ebiten/v2"
	ebv "github.com/hajimehoshi/ebiten/v2/vector"
	"image/color"
)

func DrawFilledRect(
	dst *eb.Image,
	rect FRectangle,
	clr color.Color,
	antialias bool,
) {
	ebv.DrawFilledRect(
		dst,
		f32(rect.Min.X), f32(rect.Min.Y), f32(rect.Dx()), f32(rect.Dy()),
		clr,
		antialias,
	)
}

func StrokeRect(
	dst *eb.Image,
	rect FRectangle,
	strokeWidth float64,
	clr color.Color,
	antialias bool,
) {
	ebv.StrokeRect(
		dst,
		f32(rect.Min.X), f32(rect.Min.Y), f32(rect.Dx()), f32(rect.Dy()),
		f32(strokeWidth),
		clr,
		antialias,
	)
}
