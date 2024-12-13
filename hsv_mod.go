package main

import (
	"encoding/json"
)

type HSVmod struct {
	Hue        float64 // -Pi to Pi
	Saturation float64 // -1 to 1
	Value      float64 // -1 to 1
}

type HSVmodTableIndex int

const (
	HSVmodBg HSVmodTableIndex = iota

	HSVmodTableSize
)

var TheHSVmodTable [HSVmodTableSize]HSVmod

func HSVmodTableToJson(table [HSVmodTableSize]HSVmod) ([]byte, error) {
	tableMap := make(map[string]HSVmod)

	for i := HSVmodTableIndex(0); i < HSVmodTableSize; i++ {
		tableMap[i.String()] = table[i]
	}

	jsonBytes, err := json.MarshalIndent(tableMap, "", "    ")
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

func HSVmodTableFromJson(tableJson []byte) ([HSVmodTableSize]HSVmod, error) {
	var hsvTable [HSVmodTableSize]HSVmod

	var tableMap map[string]HSVmod

	err := json.Unmarshal(tableJson, &tableMap)
	if err != nil {
		return hsvTable, err
	}

	stringToIndex := make(map[string]int)
	for i := HSVmodTableIndex(0); i < HSVmodTableSize; i++ {
		stringToIndex[i.String()] = int(i)
	}

	for indexName, index := range stringToIndex {
		if hsvMod, ok := tableMap[indexName]; ok {
			hsvTable[index] = hsvMod
		}
	}

	return hsvTable, nil
}
