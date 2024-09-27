package main

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf
//go:embed assets/COOPBL.TTF
//go:embed assets
var EmbeddedAssets embed.FS

var TileSprite Sprite

var (
	ClearFace *ebt.GoTextFace
	DecoFace  *ebt.GoTextFace
)

var WhiteImage *eb.Image

func init() {
	// create WhiteImage
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
	InfoLogger.Print("loading assets")

	var hotReloadPath string

	if HotReload {
		var err error
		if hotReloadPath, err = RelativePath("./"); err != nil {
			ErrorLogger.Fatalf("failed to get assets path: %v", err)
		}
	}

	loadData := func(filePath string) ([]byte, error) {
		var data []byte
		var err error
		if HotReload {
			data, err = os.ReadFile(filepath.Join(hotReloadPath, filePath))
		} else {
			data, err = fs.ReadFile(EmbeddedAssets, filePath)
		}
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	mustLoadData := func(filepath string) []byte {
		data, err := loadData(filepath)
		if err != nil {
			ErrorLogger.Fatalf("failed to load %s: %v", filepath, err)
		}
		return data
	}

	// load tile sprite
	{
		tileSpriteJson := mustLoadData("assets/spritesheet-100x100-5x5.json")
		tileSprite, err := ParseSpriteJsonMetadata(bytes.NewReader(tileSpriteJson))
		if err != nil {
			ErrorLogger.Fatalf("failed to load sprite: %v", err)
		}

		tileSpriteImageData := mustLoadData("assets/spritesheet-100x100-5x5.png")
		image, _, err := image.Decode(bytes.NewReader(tileSpriteImageData))
		if err != nil {
			ErrorLogger.Fatalf("failed to load sprite: %v", err)
		}

		tileSprite.Image = eb.NewImageFromImage(image)
		TileSprite = tileSprite
	}

	// load fonts
	{
		decoFontFile := mustLoadData("assets/COOPBL.TTF")
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(decoFontFile))
		if err != nil {
			ErrorLogger.Fatalf("failed to load font: %v", err)
		}
		DecoFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}
	}
	{
		clearFontFile := mustLoadData("dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf")
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(clearFontFile))
		if err != nil {
			ErrorLogger.Fatalf("failed to load font: %v", err)
		}

		ClearFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}
	}

	// load color table
	loadColorTable := func() error {
		jsonData, err := loadData("assets/color-table.json")
		if err != nil {
			return err
		}
		table, err := ColorTableFromJson(jsonData)
		if err != nil {
			return err
		}
		ColorTable = table
		return nil
	}
	if err := loadColorTable(); err != nil {
		ErrorLogger.Printf("failed to load color table: %v", err)
	}
}

func SaveColorTable() {
	InfoLogger.Print("saving color table")

	saveImp := func() error {
		jsonBytes, err := ColorTableToJson(ColorTable)
		if err != nil {
			return err
		}
		path, err := RelativePath("assets/color-table.json")
		if err != nil {
			return err
		}
		err = os.WriteFile(path, jsonBytes, 0664)
		if err != nil {
			return err
		}

		return nil
	}

	if err := saveImp(); err != nil {
		ErrorLogger.Printf("failed to save color table: %v", err)
	}
}
