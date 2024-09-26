package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	//go:embed assets/spritesheet-100x100-5x5.png
	tileSpriteImageData []byte
	//go:embed assets/spritesheet-100x100-5x5.json
	tileSpriteJson []byte
)
var TileSprite Sprite

var (
	//go:embed "assets/COOPBL.TTF"
	decoFontFile []byte
	DecoFace     *ebt.GoTextFace
)
var (
	//go:embed dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf
	clearFontFile []byte
	ClearFace     *ebt.GoTextFace
)

var WhiteImage *eb.Image

func init() {
	whiteImg := image.NewNRGBA(RectWH(3, 3))
	for x := range 3 {
		for y := range 3 {
			whiteImg.Set(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	wholeWhiteImage := eb.NewImageFromImage(whiteImg)
	WhiteImage = wholeWhiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*eb.Image)
}

func LoadAssets() {
	// load tile sprite
	{
		var err error
		TileSprite, err = ParseSpriteJsonMetadata(bytes.NewReader(tileSpriteJson))
		if err != nil {
			ErrorLogger.Fatalf("failed to load sprite : %v", err)
		}

		image, _, err := image.Decode(bytes.NewReader(tileSpriteImageData))
		if err != nil {
			ErrorLogger.Fatalf("failed to load sprite : %v", err)
		}
		TileSprite.Image = eb.NewImageFromImage(image)
	}

	// load fonts
	{
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(decoFontFile))
		if err != nil {
			ErrorLogger.Fatalf("failed to load font : %v", err)
		}

		DecoFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}
	}
	{
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(clearFontFile))
		if err != nil {
			ErrorLogger.Fatalf("failed to load font : %v", err)
		}

		ClearFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}
	}
}
