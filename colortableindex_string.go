// Code generated by "stringer -type ColorTableIndex"; DO NOT EDIT.

package main

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ColorBg-0]
	_ = x[ColorTopUIBg-1]
	_ = x[ColorTopUITitle-2]
	_ = x[ColorTopUIButton-3]
	_ = x[ColorTopUIButtonOnHover-4]
	_ = x[ColorTopUIButtonOnDown-5]
	_ = x[ColorTopUIFlag-6]
	_ = x[ColorTileNormal1-7]
	_ = x[ColorTileNormal2-8]
	_ = x[ColorTileNormalStroke-9]
	_ = x[ColorTileRevealed1-10]
	_ = x[ColorTileRevealed2-11]
	_ = x[ColorTileRevealedStroke-12]
	_ = x[ColorNumber1-13]
	_ = x[ColorNumber2-14]
	_ = x[ColorNumber3-15]
	_ = x[ColorNumber4-16]
	_ = x[ColorNumber5-17]
	_ = x[ColorNumber6-18]
	_ = x[ColorNumber7-19]
	_ = x[ColorNumber8-20]
	_ = x[ColorFlag-21]
	_ = x[ColorElementWon-22]
	_ = x[ColorMineBg1-23]
	_ = x[ColorMineBg2-24]
	_ = x[ColorMine-25]
	_ = x[ColorTileHighLight-26]
	_ = x[ColorWater1-27]
	_ = x[ColorWater2-28]
	_ = x[ColorWater3-29]
	_ = x[ColorWater4-30]
	_ = x[ColorTableSize-31]
}

const _ColorTableIndex_name = "ColorBgColorTopUIBgColorTopUITitleColorTopUIButtonColorTopUIButtonOnHoverColorTopUIButtonOnDownColorTopUIFlagColorTileNormal1ColorTileNormal2ColorTileNormalStrokeColorTileRevealed1ColorTileRevealed2ColorTileRevealedStrokeColorNumber1ColorNumber2ColorNumber3ColorNumber4ColorNumber5ColorNumber6ColorNumber7ColorNumber8ColorFlagColorElementWonColorMineBg1ColorMineBg2ColorMineColorTileHighLightColorWater1ColorWater2ColorWater3ColorWater4ColorTableSize"

var _ColorTableIndex_index = [...]uint16{0, 7, 19, 34, 50, 73, 95, 109, 125, 141, 162, 180, 198, 221, 233, 245, 257, 269, 281, 293, 305, 317, 326, 341, 353, 365, 374, 392, 403, 414, 425, 436, 450}

func (i ColorTableIndex) String() string {
	if i < 0 || i >= ColorTableIndex(len(_ColorTableIndex_index)-1) {
		return "ColorTableIndex(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ColorTableIndex_name[_ColorTableIndex_index[i]:_ColorTableIndex_index[i+1]]
}
