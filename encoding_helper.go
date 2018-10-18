package agogo

import (
	"github.com/gorgonia/agogo/game"
	"github.com/pkg/errors"
	"gorgonia.org/vecf32"
)

// EncodeTwoPlayerBoard encodes black as 1, white as -1 for each stone placed
func EncodeTwoPlayerBoard(a []game.Colour, prealloc []float32) []float32 {
	if len(prealloc) != len(a) {
		prealloc = make([]float32, len(a))
	}

	for i := range a {
		switch a[i] {
		case game.Black:
			prealloc[i] = 1
		case game.White:
			prealloc[i] = -1
		default:
			prealloc[i] = 0
		}
	}
	return prealloc
}

// WQEncoder encodes a Go board
func WQEncoder(a game.State) []float32 {
	const lookback = 8
	const features = 2*lookback + 2
	board := a.Board()
	size := len(board)
	retVal := make([]float32, size*features)

	next := a.ToMove()
	encodedPlayer := float32(1)
	var blackStart, whiteStart, nextStart int
	if next == game.Player(game.Black) {
		blackStart = 0
		whiteStart = lookback * size
		nextStart = 2 * lookback * size
	} else {
		blackStart = lookback * size
		whiteStart = 0
		nextStart = (2*lookback + 1) * size
		encodedPlayer = -1
	}

	current := a.MoveNumber() - 1
	for i := 1; i < lookback; i++ {
		h := current - i
		if h > 0 && h < current {
			past := a.Historical(h)
			encodeBlack(past, retVal[blackStart:blackStart+size])
			encodeWhite(past, retVal[whiteStart:whiteStart+size])
		}

		blackStart += size
		whiteStart += size
	}

	for i := nextStart; i < nextStart+size; i++ {
		retVal[i] = encodedPlayer
	}

	return retVal
}

func encodeBlack(a []game.Colour, prealloc []float32) []float32 {
	return EncodeTwoPlayerBoard(a, prealloc)
}

func encodeWhite(a []game.Colour, prealloc []float32) []float32 {
	retVal := EncodeTwoPlayerBoard(a, prealloc)
	vecf32.Scale(retVal, -1)
	return retVal
}

func RotateBoard(board []float32, m, n int) ([]float32, error) {
	if m != n {
		return nil, errors.Errorf("Cannot handle m %d, n %d. This function only takes square boards", m, n)
	}
	copied := make([]float32, len(board))
	copy(copied, board)
	it := MakeIterator(copied, m, n)
	for i := 0; i < m/2; i++ {
		mi1 := m - i - 1
		for j := i; j < mi1; j++ {
			mj1 := m - j - 1
			tmp := it[i][j]
			// right to top
			it[i][j] = it[j][mi1]

			// bottom to right
			it[j][mi1] = it[mi1][mj1]

			// left to bottom
			it[mi1][mj1] = it[mj1][i]

			// tmp is left
			it[mj1][i] = tmp
		}
	}
	ReturnIterator(m, n, it)
	return copied, nil
}
