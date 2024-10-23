package main

import (
	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var TheGraphicsContext struct {
	BlendStack  []eb.Blend
	FilterStack []eb.Filter
	AntiAlias   bool
}

func init() {
	ctx := &TheGraphicsContext

	ctx.BlendStack = append(ctx.BlendStack, eb.Blend{})
	ctx.FilterStack = append(ctx.FilterStack, eb.FilterLinear)
	ctx.AntiAlias = true
}

func BeginBlend(filter eb.Blend) {
	ctx := &TheGraphicsContext

	ctx.BlendStack = append(ctx.BlendStack, filter)
}

func EndBlend() {
	ctx := &TheGraphicsContext

	ctx.BlendStack = ctx.BlendStack[0 : len(ctx.BlendStack)-1]
}

func CurrentBlend() eb.Blend {
	ctx := &TheGraphicsContext

	return ctx.BlendStack[len(ctx.BlendStack)-1]
}

func BeginFilter(filter eb.Filter) {
	ctx := &TheGraphicsContext

	ctx.FilterStack = append(ctx.FilterStack, filter)
}

func EndFilter() {
	ctx := &TheGraphicsContext

	ctx.FilterStack = ctx.FilterStack[0 : len(ctx.FilterStack)-1]
}

func CurrentFilter() eb.Filter {
	ctx := &TheGraphicsContext

	return ctx.FilterStack[len(ctx.FilterStack)-1]
}

func IsAntiAliasOn() bool {
	return TheGraphicsContext.AntiAlias
}

func SetAntiAlias(onOff bool) {
	TheGraphicsContext.AntiAlias = onOff
}

type DrawImageOptions struct {
	GeoM eb.GeoM

	ColorScale eb.ColorScale
}

type DrawRectShaderOptions struct {
	GeoM eb.GeoM

	ColorScale eb.ColorScale

	Uniforms map[string]any

	Images [4]*eb.Image
}

type DrawTrianglesOptions struct {
	ColorScaleMode eb.ColorScaleMode

	Address eb.Address

	FillRule eb.FillRule
}

type DrawTrianglesShaderOptions struct {
	Uniforms map[string]any

	Images [4]*eb.Image

	FillRule eb.FillRule
}

type DrawTextOptions struct {
	DrawImageOptions
	ebt.LayoutOptions
}

func DrawImage(dst *eb.Image, src *eb.Image, options *DrawImageOptions) {
	if options == nil {
		options = &DrawImageOptions{}
	}
	op := &eb.DrawImageOptions{}
	op.GeoM = options.GeoM
	op.ColorScale = options.ColorScale
	op.Blend = CurrentBlend()
	op.Filter = CurrentFilter()
	dst.DrawImage(src, op)
}

func DrawRectShader(
	dst *eb.Image,
	width, height int,
	shader *eb.Shader,
	options *DrawRectShaderOptions,
) {
	if options == nil {
		options = &DrawRectShaderOptions{}
	}
	op := &eb.DrawRectShaderOptions{}
	op.GeoM = options.GeoM
	op.ColorScale = options.ColorScale
	op.Blend = CurrentBlend()
	op.Uniforms = options.Uniforms
	op.Images = options.Images
	dst.DrawRectShader(width, height, shader, op)
}

func DrawTriangles(
	dst *eb.Image,
	vertices []eb.Vertex, indices []uint16,
	img *eb.Image,
	options *DrawTrianglesOptions,
) {
	if options == nil {
		options = &DrawTrianglesOptions{}
	}
	op := &eb.DrawTrianglesOptions{}
	op.ColorScaleMode = options.ColorScaleMode
	op.Blend = CurrentBlend()
	op.Filter = CurrentFilter()
	op.Address = options.Address
	op.FillRule = options.FillRule
	op.AntiAlias = TheGraphicsContext.AntiAlias

	dst.DrawTriangles(vertices, indices, img, op)
}

func DrawTrianglesShader(
	dst *eb.Image,
	vertices []eb.Vertex, indices []uint16,
	shader *eb.Shader,
	options *DrawTrianglesShaderOptions,
) {
	if options == nil {
		options = &DrawTrianglesShaderOptions{}
	}
	op := &eb.DrawTrianglesShaderOptions{}
	op.Blend = CurrentBlend()
	op.Uniforms = options.Uniforms
	op.Images = options.Images
	op.FillRule = options.FillRule
	op.AntiAlias = TheGraphicsContext.AntiAlias

	dst.DrawTrianglesShader(vertices, indices, shader, op)
}

func DrawText(
	dst *eb.Image,
	text string,
	face ebt.Face,
	options *DrawTextOptions,
) {
	if options == nil {
		options = &DrawTextOptions{}
	}
	op := &ebt.DrawOptions{}
	op.GeoM = options.GeoM
	op.ColorScale = options.ColorScale
	op.Blend = CurrentBlend()
	op.Filter = CurrentFilter()
	ebt.Draw(dst, text, face, op)
}
