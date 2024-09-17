package main

import (
	"encoding/json"
	"fmt"
	"image"
	"io"

	eb "github.com/hajimehoshi/ebiten/v2"
)

type Sprite struct {
	*eb.Image

	Width, Height int

	Margin int

	Count int
}

func SpriteBounds(sprite Sprite, spriteN int) image.Rectangle {
	if spriteN < 0 || spriteN >= sprite.Count {
		panicMsg := fmt.Sprintf("index out of range [%d] with length %d", spriteN, sprite.Count)
		panic(panicMsg)
	}

	w := sprite.Width + sprite.Margin
	h := sprite.Height + sprite.Margin

	spriteW, spriteH := ImageSize(sprite)

	colCount := spriteW / w
	rowCount := spriteH / h

	_ = rowCount // might use this later

	// prevent dvidision by zero
	// (it also makes no sense for col and row count to be zero)
	colCount = max(colCount, 1)
	rowCount = max(rowCount, 1)

	col := spriteN % colCount
	row := spriteN / colCount

	imageMin := sprite.Bounds().Min

	return image.Rectangle{
		Min: image.Pt(col*w, row*h).Add(imageMin),
		Max: image.Pt(col*w+sprite.Width, row*h+sprite.Height).Add(imageMin),
	}
}

func SpriteSubImage(sprite Sprite, spriteN int) *eb.Image {
	return sprite.SubImage(SpriteBounds(sprite, spriteN)).(*eb.Image)
}

type spriteJsonMetadata struct {
	SpriteWidth  int
	SpriteHeight int

	SpriteCount int

	SpriteMargin int
}

// Parse sprite json metadata.
// Parsed sprite doen't contain image.
func ParseSpriteJsonMetadata(jsonReader io.Reader) (Sprite, error) {
	sprite := Sprite{}
	metadata := spriteJsonMetadata{}

	decoder := json.NewDecoder(jsonReader)

	if err := decoder.Decode(&metadata); err != nil {
		return sprite, err
	}

	sprite.Width = metadata.SpriteWidth
	sprite.Height = metadata.SpriteHeight
	sprite.Margin = metadata.SpriteMargin
	sprite.Count = metadata.SpriteCount

	return sprite, nil
}
