package c4

import (
	"testing"

	"github.com/gorgonia/agogo/game"
)

func TestGame_Ended(t *testing.T) {
	X := game.Black
	O := game.White
	Z := game.None
	g := New(6, 7, 4)
	data := g.b.data.Data().([]game.Colour)

	var a []game.Colour
	a = []game.Colour{
		X, Z, Z, Z, Z, Z, Z,
		O, Z, Z, Z, Z, Z, Z,
		O, Z, Z, Z, Z, Z, Z,
		X, Z, Z, Z, Z, Z, Z,
		O, O, Z, Z, X, Z, X,
		X, O, Z, O, X, Z, X,
	}
	copy(data, a)

	if ended, winner := g.Ended(); ended {
		t.Errorf("Game is not supposed to end! Winner: %v\n%v", winner, g)
	}

	// fullboard
	a = []game.Colour{
		X, O, X, O, X, O, X,
		O, O, X, O, X, O, O,
		X, X, X, O, X, O, X,
		O, X, O, X, O, X, O,
		X, O, O, X, O, O, X,
		X, X, O, X, O, X, X,
	}
	copy(data, a)

	if ended, winner := g.Ended(); !ended || (ended && winner != game.Player(Z)) {
		t.Errorf("Game is supposed to end with Winner = %v. Got %v\n%v", Z, winner, g)
	}

	// diagonal
	a = []game.Colour{
		X, Z, Z, Z, Z, Z, Z,
		O, Z, Z, Z, Z, Z, Z,
		O, Z, Z, X, Z, Z, Z,
		X, Z, X, Z, Z, Z, Z,
		O, X, Z, Z, X, Z, X,
		X, O, Z, O, X, Z, X,
	}
	copy(data, a)
	if ended, winner := g.Ended(); !ended || (ended && winner != game.Player(X)) {
		t.Errorf("Game is supposed to end with Winner = %v. Got %v\n%v", X, winner, g)
	}

	// diagonal2
	a = []game.Colour{
		X, Z, Z, Z, Z, Z, Z,
		O, Z, Z, Z, Z, Z, Z,
		O, X, Z, O, Z, Z, Z,
		X, Z, X, Z, Z, Z, Z,
		O, X, Z, X, X, Z, X,
		X, O, Z, O, X, Z, X,
	}
	copy(data, a)
	if ended, winner := g.Ended(); !ended || (ended && winner != game.Player(X)) {
		t.Errorf("Game is supposed to end with Winner = %v. Got %v\n%v", X, winner, g)
	}

	// vertical
	a = []game.Colour{
		X, Z, Z, Z, Z, Z, Z,
		O, Z, Z, Z, X, Z, Z,
		O, Z, Z, X, X, Z, Z,
		X, Z, Z, Z, X, Z, Z,
		O, X, Z, Z, X, Z, X,
		X, O, Z, O, O, Z, X,
	}
	copy(data, a)
	if ended, winner := g.Ended(); !ended || (ended && winner != game.Player(X)) {
		t.Errorf("Game is supposed to end with Winner = %v. Got %v\n%v", X, winner, g)
	}

	// horizontal
	a = []game.Colour{
		X, Z, Z, Z, Z, Z, Z,
		O, Z, Z, Z, Z, Z, Z,
		O, Z, Z, X, Z, Z, Z,
		X, Z, Z, Z, X, Z, Z,
		O, X, Z, Z, X, Z, X,
		O, X, X, X, X, Z, X,
	}
	copy(data, a)
	if ended, winner := g.Ended(); !ended || (ended && winner != game.Player(X)) {
		t.Errorf("Game is supposed to end with Winner = %v. Got %v\n%v", X, winner, g)
	}
}
