package minesweeper

import (
	"encoding/json"
)

type BezierTableIndex int

const (
	BezierTileRevealScale BezierTableIndex = iota
	BezierTileRevealOffsetY

	BezierBoardHideTile
	BezierBoardHideTileAlpha
	BezierBoardHideButton

	BezierBoardShowTileOffsetY
	BezierBoardShowTileAlpha
	BezierBoardShowTileScale

	BezierBoardZoomOut

	BezierTableSize
)

var TheBezierTable [BezierTableSize]BezierCurveData

func init() {
	for i := BezierTableIndex(0); i < BezierTableSize; i++ {
		TheBezierTable[i] = DefaultBezierCurveData
	}
}

func BezierTableToJson(table [BezierTableSize]BezierCurveData) ([]byte, error) {
	tableMap := make(map[string]BezierCurveData)

	for i := BezierTableIndex(0); i < BezierTableSize; i++ {
		tableMap[i.String()] = table[i]
	}

	jsonBytes, err := json.MarshalIndent(tableMap, "", "    ")
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

func BezierTableFromJson(tableJson []byte) ([BezierTableSize]BezierCurveData, error) {
	var bezierTable [BezierTableSize]BezierCurveData

	var tableMap map[string]BezierCurveData

	err := json.Unmarshal(tableJson, &tableMap)
	if err != nil {
		return bezierTable, err
	}

	stringToIndex := make(map[string]int)
	for i := BezierTableIndex(0); i < BezierTableSize; i++ {
		stringToIndex[i.String()] = int(i)
	}

	for indexName, index := range stringToIndex {
		if bezier, ok := tableMap[indexName]; ok {
			bezierTable[index] = bezier
		} else {
			bezierTable[index] = DefaultBezierCurveData
		}
	}

	return bezierTable, nil
}

type BezierCurveData struct {
	Points [4]FPoint
}

var DefaultBezierCurveData BezierCurveData = BezierCurveData{
	Points: [4]FPoint{
		FPt(0, 0),
		FPt(0.3, 0),
		FPt(0.7, 1),
		FPt(1, 1),
	},
}
