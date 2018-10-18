package c4

import (
	"errors"
	"fmt"
	// "log"

	"github.com/gorgonia/agogo/game"
	"gorgonia.org/tensor"
	"gorgonia.org/tensor/native"
)

type Board struct {
	data *tensor.Dense
	it   [][]game.Colour
	n    int // how many to be considered a win?
}

func newBoard(rows, cols, n int) *Board {
	backing := make([]game.Colour, rows*cols)
	data := tensor.New(tensor.WithShape(rows, cols), tensor.WithBacking(backing))
	iter, err := native.Matrix(data)
	if err != nil {
		panic(err)
	}
	it := iter.([][]game.Colour)
	return &Board{
		data: data,
		it:   it,
		n:    n,
	}
}

func (b *Board) Format(s fmt.State, c rune) {
	switch c {
	case 's', 'v':
		for _, row := range b.it {
			fmt.Fprint(s, "⎢ ")
			for _, col := range row {
				fmt.Fprintf(s, "%s ", col)
			}
			fmt.Fprint(s, "⎥\n")
		}
	}
}

func (b *Board) Apply(m game.PlayerMove) error {
	if m.Single.IsPass() {
		return nil
	}
	row, col, err := b.check(m)
	if err != nil {
		return err
	}
	b.it[row][col] = game.Colour(m.Player)
	return nil
}

func (b *Board) check(m game.PlayerMove) (row, col int, err error) {
	if m.Single.IsPass() {
		return -1, -1, nil
	}
	col = int(m.Single)
	for row = len(b.it) - 1; row >= 0; row-- {
		if b.it[row][col] == game.None {
			return row, col, nil
		}
	}
	return -1, -1, errors.New("Selected column is full")
}

func (b *Board) clone() *Board {
	sh := b.data.Shape()

	b2 := newBoard(sh[0], sh[1], b.n)
	raw2 := b2.data.Data().([]game.Colour)
	raw := b.data.Data().([]game.Colour)
	copy(raw2, raw)
	return b2
}

func (b *Board) checkWin() game.Colour {
	rows, cols := b.data.Shape()[0], b.data.Shape()[1]
	if winner := b.checkVertical(rows, cols); winner != game.None {
		return winner
	}
	if winner := b.checkHorizontal(rows, cols); winner != game.None {
		return winner
	}
	if winner := b.checkTLBR(rows, cols); winner != game.None {
		return winner
	}
	return b.checkTRBL(rows, cols)
}

// checkVertical checks downwards
func (b *Board) checkVertical(rows, cols int) game.Colour {
	for x := 0; x < cols; x++ {
		for y := 0; y < rows; y++ {
			c := b.it[y][x]
			winning := true
			if c != game.None {
				for i := 0; i < b.n; i++ {
					if y+i < rows {
						if b.it[y+i][x] != c {
							winning = false
						}
					} else {
						winning = false
					}
				}
				if winning {
					return c
				}
			}
		}
	}
	return game.None
}

// checkHorizontal checks rightwards
func (b *Board) checkHorizontal(rows, cols int) game.Colour {
	for x := 0; x < cols; x++ {
		for y := 0; y < rows; y++ {
			c := b.it[y][x]
			winning := true
			if c != game.None {
				for i := 0; i < b.n; i++ {
					if x+i < cols {
						if b.it[y][x+i] != c {
							winning = false
						}
					} else {
						winning = false
					}
				}
				if winning {
					return c
				}
			}
		}
	}
	return game.None
}

func (b *Board) checkTLBR(rows, cols int) game.Colour {
	for x := 0; x < cols; x++ {
		for y := 0; y < rows; y++ {
			c := b.it[y][x]
			winning := true
			if c != game.None {
				for i := 0; i < b.n; i++ {
					if x-i >= 0 && y+i < rows {
						if b.it[y+i][x-i] != c {
							winning = false
						}
					} else {
						winning = false
					}
				}
				if winning {
					return c
				}
			}
		}
	}
	return game.None
}

func (b *Board) checkTRBL(rows, cols int) game.Colour {
	for x := 0; x < cols; x++ {
		for y := 0; y < rows; y++ {
			c := b.it[y][x]
			winning := true
			if c != game.None {
				for i := 0; i < b.n; i++ {
					if x+i < cols && y+i < rows {
						if b.it[y+i][x+i] != c {
							winning = false
						}
					} else {
						winning = false
					}
				}
				if winning {
					return c
				}
			}
		}
	}
	return game.None
}
