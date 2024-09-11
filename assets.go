package main

import (
	_ "embed"
	"bytes"
	"image"
	_ "image/png"
	_ "image/jpeg"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	//go:embed spritesheet-100x100-5x5.png
	tileSpriteImageData []byte
	//go:embed spritesheet-100x100-5x5.json
	tileSpriteJson []byte
)
var TileSprite Sprite

//go:embed COOPBL.TTF
var fontFile []byte
var FontFace *ebt.GoTextFace

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

	// load font
	{
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(fontFile))
		if err != nil {
			ErrorLogger.Fatalf("failed to load font : %v", err)
		}

		FontFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}
	}
}
