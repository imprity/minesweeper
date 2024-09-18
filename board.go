package main

import (
	"math/rand"
)

//==============================================
// BOARD STUFFS
//==============================================

type Board struct {
	Width  int
	Height int

	Mines [][]bool

	Revealed [][]bool
	Flags    [][]bool
}

func NewBoard(width int, height int) Board {
	var board Board

	board.Width = width
	board.Height = height

	board.Mines = New2DArray[bool](width, height)
	board.Revealed = New2DArray[bool](width, height)
	board.Flags = New2DArray[bool](width, height)

	return board
}

func (board *Board) PlaceMines(count, exceptX, exceptY int) {
	maxCount := board.Width*board.Height - 1
	count = min(count, maxCount)

	minePlaces := make([][2]int, maxCount)
	iterCounter := 0
	iter := NewBoardIterator(0, 0, board.Width-1, board.Height-1)
	for iter.HasNext() {
		x, y := iter.GetNext()
		if !(x == exceptX && y == exceptY) {
			minePlaces[iterCounter] = [2]int{x, y}
			iterCounter += 1
		}
	}
	rand.Shuffle(maxCount, func(i, j int) {
		minePlaces[i], minePlaces[j] = minePlaces[j], minePlaces[i]
	})

	for i := 0; i < count; i++ {
		board.Mines[minePlaces[i][0]][minePlaces[i][1]] = true
	}
}

func (board *Board) Copy() Board {
	copy := NewBoard(board.Width, board.Height)

	iterator := NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	for iterator.HasNext() {
		x, y := iterator.GetNext()

		copy.Mines[x][y] = board.Mines[x][y]
		copy.Revealed[x][y] = board.Revealed[x][y]
		copy.Flags[x][y] = board.Flags[x][y]
	}

	return copy
}

func (board *Board) IsPosInBoard(posX int, posY int) bool {
	return posX >= 0 && posX < board.Width && posY >= 0 && posY < board.Height
}

func (board *Board) SpreadSafeArea(posX int, posY int) {
	if !board.IsPosInBoard(posX, posY) {
		return
	}

	if board.Revealed[posX][posY] {
		return
	}

	if board.Mines[posX][posY] {
		return
	}

	board.Revealed[posX][posY] = true

	if board.GetNeighborMineCount(posX, posY) > 0 {
		return
	}

	iterator := NewBoardIterator(posX-1, posY-1, posX+1, posY+1)
	for iterator.HasNext() {
		x, y := iterator.GetNext()
		if board.IsPosInBoard(x, y) {
			board.SpreadSafeArea(x, y)
		}
	}
}

func (board *Board) GetNeighborMineCount(posX int, posY int) int {
	var mineCount int = 0
	for x := max(posX-1, 0); x < min(posX+2, board.Width); x++ {
		for y := max(posY-1, 0); y < min(posY+2, board.Height); y++ {
			if board.Mines[x][y] {
				mineCount += 1
			}
		}
	}

	return mineCount
}

func (board *Board) GetNeighborFlagCount(posX int, posY int) int {
	var flagCount int = 0
	for x := max(posX-1, 0); x < min(posX+2, board.Width); x++ {
		for y := max(posY-1, 0); y < min(posY+2, board.Height); y++ {
			if board.Flags[x][y] {
				flagCount += 1
			}
		}
	}

	return flagCount
}

func (board *Board) CheckWin() bool {
	var iterator BoardIterator = NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	for iterator.HasNext() {
		x, y := iterator.GetNext()

		if !board.Mines[x][y] && !board.Revealed[x][y] {
			return false
		}
	}

	return true
}

// ==============================================
// BOARD ITERACTION
// ==============================================
/*
type BoardInteractionType int

const (
	InteractionTypeStep BoardInteractionType = iota
	InteractionTypeFlag
	InteractionTypeCheck
)

type BoardInteractionResult int

const (
	InteractionResultContinue BoardInteractionResult = iota
	InteractionResultFail
	InteractionResultWin
)

func (board *Board) InteractAt(posX int, posY int, interaction BoardInteractionType) BoardInteractionResult {
	defer func() {
		// remove flags where it's revealed
		for x := 0; x < board.Width; x++ {
			for y := 0; y < board.Height; y++ {
				if board.Revealed[x][y] {
					board.Flags[x][y] = false
				}
			}
		}
	}()

	switch interaction {
	case InteractionTypeStep:
		{
			if !board.Touched {
				board.PlaceMines(board.MineCount, posX, posY)
				board.Touched = true
			}
			if !board.Revealed[posX][posY] {
				if board.Mines[posX][posY] {
					return InteractionResultFail // user stepped on a mine
				}
				//we have to spread out
				board.SpreadSafeArea(posX, posY)
			}
			if board.CheckWin() {
				return InteractionResultContinue
			} else {
				return InteractionResultContinue
			}
		}
	case InteractionTypeFlag:
		{
			if !board.Revealed[posX][posY] {
				board.Flags[posX][posY] = !board.Flags[posX][posY]
			}
			return InteractionResultContinue
		}
	case InteractionTypeCheck:
		{
			if board.Revealed[posX][posY] && board.GetNeighborMineCount(posX, posY) > 0 {
				var flagCount int = board.GetNeighborFlagCount(posX, posY)
				if board.GetNeighborMineCount(posX, posY) == flagCount {
					//check if user flagged it correctly
					iterator := NewBoardIterator(posX-1, posY-1, posX+1, posY+1)

					for iterator.HasNext() {
						x, y := iterator.GetNext()
						if board.IsPosInBoard(x, y) {
							// user flagged it incorrectly
							if board.Flags[x][y] && !board.Mines[x][y] {
								return InteractionResultFail
							}
						}
					}

					//reset iterator
					iterator = NewBoardIterator(posX-1, posY-1, posX+1, posY+1)

					for iterator.HasNext() {
						x, y := iterator.GetNext()
						if board.IsPosInBoard(x, y) {
							board.SpreadSafeArea(x, y)
						}
					}

				}
			}

			if board.CheckWin() {
				return InteractionResultContinue
			} else {
				return InteractionResultContinue
			}
		}
	default:
		panic("UNREACHABLE")
	}
}
*/

//==============================================
// board iterator
//==============================================

type BoardIterator struct {
	MinX int
	MinY int
	MaxX int
	MaxY int

	CurrentX int
	CurrentY int
}

// inclusive
func NewBoardIterator(x1 int, y1 int, x2 int, y2 int) BoardIterator {
	iterator := BoardIterator{
		MinX: min(x1, x2),
		MinY: min(y1, y2),

		MaxX: max(x1, x2),
		MaxY: max(y1, y2),
	}

	iterator.CurrentX = iterator.MinX
	iterator.CurrentY = iterator.MinY

	return iterator
}

func (bi *BoardIterator) HasNext() bool {
	return bi.CurrentY <= bi.MaxY
}

func (bi *BoardIterator) GetNext() (int, int) {
	x := bi.CurrentX
	y := bi.CurrentY

	bi.CurrentX++
	if bi.CurrentX > bi.MaxX {
		bi.CurrentX = bi.MinX
		bi.CurrentY++
	}

	return x, y
}

func (bi *BoardIterator) Reset() {
	bi.CurrentX = bi.MinX
	bi.CurrentY = bi.MinY
}
