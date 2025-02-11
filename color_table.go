package minesweeper

import (
	"encoding/json"
	"image/color"
)

type ColorTableIndex int

func (c ColorTableIndex) RGBA() (r, g, b, a uint32) {
	return TheColorTable[c].RGBA()
}

const (
	ColorBg ColorTableIndex = iota

	ColorTopUIBg
	ColorTopUITitle
	ColorTopUIButton
	ColorTopUIButtonOnHover
	ColorTopUIButtonOnDown
	ColorTopUIFlag

	ColorTileNormal1
	ColorTileNormal2
	ColorTileNormalStroke

	ColorTileRevealed1
	ColorTileRevealed2
	ColorTileRevealedStroke

	ColorNumber1
	ColorNumber2
	ColorNumber3
	ColorNumber4
	ColorNumber5
	ColorNumber6
	ColorNumber7
	ColorNumber8

	ColorFlag

	ColorElementWon

	ColorMineBg1
	ColorMineBg2
	ColorMine

	ColorBgHighLight
	ColorTileHighLight
	ColorFgHighLight

	ColorWater1
	ColorWater2
	ColorWater3
	ColorWater4

	ColorRetryA1
	ColorRetryA2
	ColorRetryA3
	ColorRetryA4

	ColorRetryB1
	ColorRetryB2
	ColorRetryB3
	ColorRetryB4

	ColorRetryWater1
	ColorRetryWater2
	ColorRetryWater3
	ColorRetryWater4

	ColorFlagTutorialFill
	ColorFlagTutorialStroke

	ColorTableSize
)

var TheColorTable [ColorTableSize]color.NRGBA
var DefaultcolorTable [ColorTableSize]color.NRGBA

func init() {
	var colorSet [ColorTableSize]bool

	setColor := func(index ColorTableIndex, c color.NRGBA) {
		colorSet[index] = true
		DefaultcolorTable[index] = c
	}

	setColor(ColorBg, color.NRGBA{10, 10, 10, 255})

	setColor(ColorTopUIBg, color.NRGBA{188, 188, 188, 255})
	setColor(ColorTopUITitle, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTopUIButton, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTopUIButtonOnHover, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTopUIButtonOnDown, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTopUIFlag, color.NRGBA{218, 26, 26, 255})

	setColor(ColorTileNormal1, color.NRGBA{30, 30, 30, 255})
	setColor(ColorTileNormal2, color.NRGBA{50, 50, 50, 255})
	setColor(ColorTileNormalStroke, color.NRGBA{150, 150, 150, 255})

	setColor(ColorTileRevealed1, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTileRevealed2, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTileRevealedStroke, color.NRGBA{150, 150, 150, 255})

	setColor(ColorNumber1, color.NRGBA{255, 0, 0, 255})
	setColor(ColorNumber2, color.NRGBA{0, 255, 0, 255})
	setColor(ColorNumber3, color.NRGBA{0, 0, 255, 255})
	setColor(ColorNumber4, color.NRGBA{255, 255, 0, 255})
	setColor(ColorNumber5, color.NRGBA{0, 255, 255, 255})
	setColor(ColorNumber6, color.NRGBA{255, 0, 255, 255})
	setColor(ColorNumber7, color.NRGBA{255, 255, 255, 255})
	setColor(ColorNumber8, color.NRGBA{100, 100, 100, 255})

	setColor(ColorFlag, color.NRGBA{255, 200, 200, 255})

	setColor(ColorElementWon, color.NRGBA{0, 0, 0, 255})

	setColor(ColorMineBg1, color.NRGBA{49, 7, 7, 255})
	setColor(ColorMineBg2, color.NRGBA{229, 61, 61, 255})
	setColor(ColorMine, color.NRGBA{255, 255, 255, 255})

	setColor(ColorBgHighLight, color.NRGBA{255, 255, 255, 255})
	setColor(ColorTileHighLight, color.NRGBA{255, 255, 255, 255})
	setColor(ColorFgHighLight, color.NRGBA{255, 255, 255, 255})

	setColor(ColorWater1, color.NRGBA{0x64, 0x39, 0xFF, 0xFF})
	setColor(ColorWater2, color.NRGBA{0x4F, 0x75, 0xFF, 0xFF})
	setColor(ColorWater3, color.NRGBA{0x00, 0xCC, 0xDD, 0xFF})
	setColor(ColorWater4, color.NRGBA{0x7C, 0xF5, 0xFF, 0xFF})

	setColor(ColorRetryA1, color.NRGBA{0, 0, 0, 255})
	setColor(ColorRetryA2, color.NRGBA{105, 223, 145, 255})
	setColor(ColorRetryA3, color.NRGBA{0, 0, 0, 255})
	setColor(ColorRetryA4, color.NRGBA{255, 255, 255, 255})

	setColor(ColorRetryB1, color.NRGBA{0, 0, 0, 255})
	setColor(ColorRetryB2, color.NRGBA{105, 223, 145, 255})
	setColor(ColorRetryB3, color.NRGBA{0, 0, 0, 255})
	setColor(ColorRetryB4, color.NRGBA{255, 255, 255, 255})

	setColor(ColorRetryWater1, color.NRGBA{0x64, 0x39, 0xFF, 0xFF})
	setColor(ColorRetryWater2, color.NRGBA{0x4F, 0x75, 0xFF, 0xFF})
	setColor(ColorRetryWater3, color.NRGBA{0x00, 0xCC, 0xDD, 0xFF})
	setColor(ColorRetryWater4, color.NRGBA{0x7C, 0xF5, 0xFF, 0xFF})

	setColor(ColorFlagTutorialFill, color.NRGBA{0, 0, 0, 0xFF})
	setColor(ColorFlagTutorialStroke, color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF})

	for i := ColorTableIndex(0); i < ColorTableSize; i++ {
		if !colorSet[i] {
			ErrLogger.Fatalf("color for %s has no default value", i.String())
		}
	}

	TheColorTable = DefaultcolorTable
}

func ColorTableGetNumber(i int) ColorTableIndex {
	return ColorNumber1 + ColorTableIndex(i-1)
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
