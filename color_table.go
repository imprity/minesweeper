package main

import (
	"encoding/json"
	"image/color"
)

type ColorTableIndex int

const (
	ColorBg ColorTableIndex = iota

	ColorTopUITitle
	ColorTopUIButton
	ColorTopUIButtonOnHover
	ColorTopUIButtonOnDown

	ColorTileNormal1
	ColorTileNormal2
	ColorTileNormalStroke

	ColorTileRevealed1
	ColorTileRevealed2
	ColorTileRevealedStroke

	ColorNumber

	ColorMine
	ColorFlag

	ColorTableSize
)

var ColorTable [ColorTableSize]color.NRGBA
var DefaultcolorTable [ColorTableSize]color.NRGBA

func init() {
	var colorSet [ColorTableSize]bool

	setColor := func(index ColorTableIndex, c color.NRGBA) {
		colorSet[index] = true
		DefaultcolorTable[index] = c
	}

	setColor(ColorBg, color.NRGBA{10, 10, 10, 255})

	setColor(ColorTopUITitle, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTopUIButton, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTopUIButtonOnHover, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTopUIButtonOnDown, color.NRGBA{255, 255, 255, 255})

	setColor(ColorTileNormal1, color.NRGBA{30, 30, 30, 255})
	setColor(ColorTileNormal2, color.NRGBA{50, 50, 50, 255})
	setColor(ColorTileNormalStroke, color.NRGBA{150, 150, 150, 255})

	setColor(ColorTileRevealed1, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTileRevealed2, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTileRevealedStroke, color.NRGBA{150, 150, 150, 255})

	setColor(ColorNumber, color.NRGBA{10, 10, 10, 255})

	setColor(ColorMine, color.NRGBA{255, 255, 255, 255})
	setColor(ColorFlag, color.NRGBA{255, 200, 200, 255})

	for i := ColorTableIndex(0); i < ColorTableSize; i++ {
		if !colorSet[i] {
			ErrorLogger.Fatalf("color for %s has no default value", i.String())
		}
	}

	ColorTable = DefaultcolorTable
}

func ColorTableToJson(table [ColorTableSize]color.NRGBA) ([]byte, error) {
	tableMap := make(map[string]color.NRGBA)

	for i := ColorTableIndex(0); i < ColorTableSize; i++ {
		tableMap[i.String()] = table[i]
	}

	jsonBytes, err := json.MarshalIndent(tableMap, "", "    ")
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

func ColorTableFromJson(tableJson []byte) ([ColorTableSize]color.NRGBA, error) {
	var colorTable [ColorTableSize]color.NRGBA

	var tableMap map[string]color.NRGBA

	err := json.Unmarshal(tableJson, &tableMap)
	if err != nil {
		return colorTable, err
	}

	stringToIndex := make(map[string]int)
	for i := ColorTableIndex(0); i < ColorTableSize; i++ {
		stringToIndex[i.String()] = int(i)
	}

	for indexName, index := range stringToIndex {
		if clr, ok := tableMap[indexName]; ok {
			colorTable[index] = clr
		} else {
			colorTable[index] = DefaultcolorTable[index]
		}
	}

	return colorTable, nil
}
