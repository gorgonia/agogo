package 围碁

import (
	"testing"

	"github.com/gorgonia/agogo/game"
)

func TestGameBasics(t *testing.T) {
	g := New(19, 0, 7.5)
	g2 := g.Clone()
	if !g.Eq(g2) {
		t.Fatal("Expected clones to be equal")
	}

	g.SetToMove(game.Player(game.White))
	if g.Eq(g2) {
		t.Fatal("Expected clones to be unequal after the parent object has changed")
	}
}
