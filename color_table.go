package main

import (
	"encoding/json"
	"image/color"
)

type ColorTableIndex int

const (
	ColorBg ColorTableIndex = iota

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

func init() {
	ColorTable[ColorBg] = color.NRGBA{10, 10, 10, 255}

	ColorTable[ColorTileNormal1] = color.NRGBA{30, 30, 30, 255}
	ColorTable[ColorTileNormal2] = color.NRGBA{50, 50, 50, 255}
	ColorTable[ColorTileNormalStroke] = color.NRGBA{150, 150, 150, 255}

	ColorTable[ColorTileRevealed1] = color.NRGBA{255, 255, 255, 255}
	ColorTable[ColorTileRevealed2] = color.NRGBA{255, 255, 255, 255}
	ColorTable[ColorTileRevealedStroke] = color.NRGBA{150, 150, 150, 255}

	ColorTable[ColorNumber] = color.NRGBA{10, 10, 10, 255}

	ColorTable[ColorMine] = color.NRGBA{255, 255, 255, 255}
	ColorTable[ColorFlag] = color.NRGBA{255, 200, 200, 255}
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

	for k, v := range tableMap {
		if index, ok := stringToIndex[k]; ok {
			colorTable[index] = v
		}
	}

	return colorTable, nil
}
