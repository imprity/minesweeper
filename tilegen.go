//go:build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
)

var (
	TileColor1 = color.RGBA{30, 30, 30, 255}
	TileColor2 = color.RGBA{230, 230, 230, 255}
)

var (
	TileWidth  int
	TileHeight int
)

var (
	TileCountRow    int
	TileCountColumn int
)

func init() {
	flag.IntVar(&TileWidth, "w", 100, "tile width")
	flag.IntVar(&TileHeight, "h", 100, "tile height")

	flag.IntVar(&TileCountColumn, "c", 10, "tile column count")
	flag.IntVar(&TileCountRow, "r", 10, "tile row count")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		_, scriptName := filepath.Split(os.Args[0])

		if _, scriptFile, _, ok := runtime.Caller(0); ok {
			_, scriptName = filepath.Split(scriptFile)
		}

		fmt.Fprintf(out, "Usage of %s:\n", scriptName)
		fmt.Fprintf(out, "\n")
		fmt.Printf("  -w int\n")
		fmt.Printf("	tile width\n")
		fmt.Printf("  -h int\n")
		fmt.Printf("	tile height\n")
		fmt.Printf("  -c int\n")
		fmt.Printf("	tile column count\n")
		fmt.Printf("  -r int\n")
		fmt.Printf("	tile row count\n")
	}
}

func main() {
	flag.Parse()

	img := image.NewRGBA(
		image.Rect(0, 0, TileWidth*TileCountColumn, TileHeight*TileCountRow),
	)

	drawTile(img, TileWidth, TileHeight, TileCountRow, TileCountColumn)

	err := saveImage(img, fmt.Sprintf(
		"tile-%dx%d-%dx%d",
		TileWidth, TileHeight, TileCountColumn, TileCountRow,
	))

	if err != nil {
		fmt.Printf("failed to generate tilemap : %v", err)
	}
}

func drawTile(
	img *image.RGBA,
	tileWidth, tileHeight int,
	tileRow, tileColumn int,
) {
	toggle := false

	for r := 0; r < tileRow; r++ {
		for c := 0; c < tileColumn; c++ {
			var color color.RGBA
			if toggle {
				color = TileColor1
			} else {
				color = TileColor2
			}

			x := c * tileWidth
			y := r * tileHeight

			draw.Draw(
				img,
				image.Rect(x, y, x+TileWidth, y+TileHeight),
				&image.Uniform{color},
				image.ZP, draw.Over)

			toggle = !toggle
		}

		if tileColumn%2 == 0 {
			toggle = !toggle
		}
	}
}

func saveImage(img image.Image, name string) error {
	entries, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	counter := 1

	imagePath := name + ".png"

	for _, entry := range entries {
		if entry.Name() == imagePath {
			imagePath = fmt.Sprintf("%s(%d).png", name, counter)
			counter++
		}
	}

	buffer := new(bytes.Buffer)
	png.Encode(buffer, img)

	err = os.WriteFile(imagePath, buffer.Bytes(), 0664)
	if err != nil {
		return err
	}

	return nil
}
