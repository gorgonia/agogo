package agogo

import (
	"testing"

	"github.com/gorgonia/agogo/game"
	"github.com/stretchr/testify/assert"
)

func TestRotateBoard(t *testing.T) {
	//
	// ⎢ O · · · X ⎥
	// ⎢ · O · X · ⎥ // this line is to break rotational symmetry
	// ⎢ · · · · · ⎥
	// ⎢ · · · · · ⎥
	// ⎢ X · · · O ⎥

	m, n := 5, 5
	board := []game.Colour{
		White, None, None, None, Black,
		None, White, None, Black, None,
		None, None, None, None, None,
		None, None, None, None, None,
		Black, None, None, None, White,
	}
	t.Logf("0:\n%v", board)

	rot1, err := RotateBoard(board, m, n)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("1:\n%v", rot1)

	rot2, err := RotateBoard(rot1, m, n)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("2:\n%v", rot2)

	rot3, err := RotateBoard(rot2, m, n)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("3:\n%v", rot3)

	rot4, err := RotateBoard(rot3, m, n)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("4:\n%v", rot4)

	assert.Equal(t, board, rot4, "After 4 rotations the board should be the same")
}
