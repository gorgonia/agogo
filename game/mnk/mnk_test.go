package mnk

import (
	"testing"

	"github.com/gorgonia/agogo/game"
)

func TestTicTacToe(t *testing.T) {
	var X = game.Colour(Cross)
	var O = game.Colour(Nought)
	TTT := TicTacToe()
	TTT.board = []game.Colour{
		X, O, X,
		O, X, O,
		O, O, X,
	}
	if !TTT.isWinner(Cross) {
		t.Error("expected X to be winner")
	}
	if ended, _ := TTT.Ended(); !ended {
		t.Error("expected game to be ended")
	}

	TTT.board = []game.Colour{
		X, O, O,
		X, O, X,
		O, X, X,
	}
	if !TTT.isWinner(Nought) {
		t.Error("expected X to be winner")
	}
}

func TestGomoku(t *testing.T) {
	var X = game.Colour(Cross)
	var O = game.Colour(Nought)
	var Z = game.None
	g := New(7, 7, 5)
	g.board = []game.Colour{
		Z, X, Z, Z, Z, Z, Z,
		Z, Z, X, Z, Z, Z, Z,
		Z, Z, Z, X, Z, Z, Z,
		Z, Z, Z, Z, X, Z, Z,
		Z, Z, Z, Z, Z, X, Z,
		Z, Z, Z, Z, Z, X, Z,
		Z, Z, Z, Z, Z, X, Z,
	}
	if !g.isWinner(Cross) {
		t.Error("expected X to be winner")
	}
	if ended, _ := g.Ended(); !ended {
		t.Error("expected game to be ended")
	}

	g.board = []game.Colour{
		Z, Z, Z, Z, Z, Z, Z,
		Z, Z, Z, Z, Z, O, Z,
		Z, Z, Z, Z, O, Z, Z,
		Z, Z, Z, O, Z, Z, Z,
		Z, Z, O, Z, Z, Z, Z,
		Z, O, Z, Z, Z, Z, Z,
		Z, Z, Z, Z, Z, Z, Z,
	}
	if !g.isWinner(Nought) {
		t.Error("expected O to be winner")
	}
	if ended, _ := g.Ended(); !ended {
		t.Error("expected game to be ended")
	}
}

func TestTicTacToeEnded(t *testing.T) {
	var X = game.Colour(Cross)
	var O = game.Colour(Nought)
	var Z = game.None
	TTT := TicTacToe()
	TTT.board = []game.Colour{
		O, Z, X,
		Z, Z, X,
		Z, O, X,
	}
	ended, winner := TTT.Ended()
	if !ended {
		t.Error("Expected game to have ended")
	}
	if winner != game.Player(X) {
		t.Error("Expected winner to be X")
	}

	TTT.board = []game.Colour{
		O, O, O,
		Z, Z, X,
		X, O, X,
	}
	ended, winner = TTT.Ended()
	if !ended {
		t.Error("Expected game to have ended")
	}
	if winner != game.Player(O) {
		t.Error("Expected winner to be O")
	}

	TTT.board = []game.Colour{
		Z, Z, X,
		X, O, X,
		O, O, O,
	}
	ended, winner = TTT.Ended()
	if !ended {
		t.Error("Expected game to have ended")
	}
	if winner != game.Player(O) {
		t.Error("Expected winner to be O")
	}

	TTT.board = []game.Colour{
		O, Z, X,
		X, O, X,
		O, Z, O,
	}
	ended, winner = TTT.Ended()
	if !ended {
		t.Error("Expected game to have ended")
	}
	if winner != game.Player(O) {
		t.Error("Expected winner to be O")
	}
}
