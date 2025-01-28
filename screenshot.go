//go:build screenshot

package minesweeper

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
)

func init() {
	ScreenshotEnabled = true

	DebugPutsPersist("screenshot", "true")
}

func TakeScreenshot(img *eb.Image, dir string) (string, error) {
	dir = filepath.Clean(dir)

	if !filepath.IsLocal(dir) {
		return "", fmt.Errorf("screenshot directory must be local")
	}

	timeStr := time.Now().Format("0102150405")

	dirPath, err := RelativePath(dir)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", err
	}

	var filename = fmt.Sprintf("pic-%s.png", timeStr)

	nameCounter := 1
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry.Name() == filename {
			nameCounter += 1
			filename = fmt.Sprintf("pic-%s-(%d).png", timeStr, nameCounter)
			// do it again!
			i = 0
		}
	}

	fullPath := filepath.Join(dirPath, filename)

	buffer := &bytes.Buffer{}
	imgImg := ImageImageFromEbImage(img)
	err = png.Encode(buffer, imgImg)
	if err != nil {
		return "", err
	}

	toWrite := buffer.Bytes()

	err = os.WriteFile(fullPath, toWrite, 0644)
	if err != nil {
		return "", err
	}

	return fullPath, nil
}
