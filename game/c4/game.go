package c4

import (
	"fmt"
	"hash/fnv"

	"github.com/gorgonia/agogo/game"
)

var (
	_ game.State = &Game{}
)

type Game struct {
	b          *Board
	history    []game.PlayerMove
	historical [][]game.Colour
	nextToMove game.Player
	histPtr    int
	moveCount  int
	passCount  int
}

//New creates a new game with a board of (rows,cols) and N to win (connect4 being 4 to win)
func New(rows, cols, N int) *Game {
	b := newBoard(rows, cols, N)
	history := make([]game.PlayerMove, 0, rows*cols)
	historical := make([][]game.Colour, 0, rows*cols/2)
	return &Game{
		b:          b,
		history:    history,
		historical: historical,
	}
}

func (g *Game) BoardSize() (int, int) { return g.b.data.Shape()[0], g.b.data.Shape()[1] }

func (g *Game) SetToMove(p game.Player) { g.nextToMove = p }

func (g *Game) ToMove() game.Player { return g.nextToMove }

func (g *Game) LastMove() game.PlayerMove {
	if len(g.history) > 0 {
		return g.history[g.histPtr-1]
	}
	return game.PlayerMove{Player: game.Player(game.None), Single: -1}
}

// Passes will always return 0
func (g *Game) Passes() int { return 0 }

func (g *Game) MoveNumber() int { return g.moveCount + 1 }

func (g *Game) Check(m game.PlayerMove) bool { _, _, err := g.b.check(m); return err == nil }

func (g *Game) Apply(m game.PlayerMove) game.State {
	board := g.b.data.Data().([]game.Colour)
	historicalBoard := make([]game.Colour, len(board))
	copy(historicalBoard, board)

	if err := g.b.Apply(m); err == nil {
		g.history = append(g.history, m)
		g.historical = append(g.historical, historicalBoard)
		g.histPtr++
	}

	if m.IsPass() {
		g.passCount++
	} else {
		g.passCount = 0
	}
	return g
}

func (g *Game) Score(p game.Player) float32 {
	winning := g.b.checkWin()
	if game.Player(winning) == p {
		return 1
	}
	if winning == game.None {
		return 0
	}
	return -1
}

func (g *Game) UndoLastMove() {
	g.histPtr--
	lastMove := g.history[g.histPtr-1]
	col := int(lastMove.Single)
	var row int
	for row = len(g.b.it) - 1; row >= 0; row-- {
		if g.b.it[row][col] == game.None {
			row--
			break
		}
	}
	g.b.it[row][col] = game.None
}

func (g *Game) Fwd() {
	if len(g.history) > 0 {
		g.histPtr++
	}
}

func (g *Game) Eq(other game.State) bool {
	ot, ok := other.(*Game)
	if !ok {
		return false
	}
	if !ot.b.data.Eq(ot.b.data) {
		return false
	}
	if g.histPtr != ot.histPtr {
		return false
	}
	if g.moveCount != ot.moveCount {
		return false
	}
	if len(g.history) != len(ot.history) {
		return false
	}
	if len(g.historical) != len(ot.historical) {
		return false
	}

	for i := range g.history {
		if ot.history[i] != g.history[i] {
			return false
		}
	}
	for i := range g.historical {
		for j := range g.historical[i] {
			if ot.historical[i][j] != g.historical[i][j] {
				return false
			}
		}
	}
	return true
}

func (g *Game) Clone() game.State {
	b2 := g.b.clone()
	history2 := make([]game.PlayerMove, len(g.history)+2)
	copy(history2, g.history)
	historical2 := make([][]game.Colour, len(g.historical)+2)
	for i := range g.historical {
		historical2[i] = make([]game.Colour, len(g.historical[i]))
		copy(historical2[i], g.historical[i])
	}
	return &Game{
		b:          b2,
		history:    history2,
		historical: historical2,
		nextToMove: g.nextToMove,
		histPtr:    g.histPtr,
		moveCount:  g.moveCount,
		passCount:  g.passCount,
	}
}

func (g *Game) AdditionalScore() float32 { return 0 }

func (g *Game) Ended() (bool, game.Player) {
	winner := g.b.checkWin()
	if winner != game.None {
		return true, game.Player(winner)
	}

	if g.passCount > 2 {
		return true, game.Player(game.None)
	}

	// ended due to full board
	raw := g.b.data.Data().([]game.Colour)
	for i := range raw {
		if raw[i] == game.None {
			return false, game.Player(game.None)
		}
	}
	return true, game.Player(game.None)
}

func (g *Game) Handicap() int { return 0 }

func (g *Game) Reset() {
	data := g.b.data.Data().([]game.Colour)
	for i := range data {
		data[i] = game.None
	}
	g.historical = g.historical[:0]
	g.history = g.history[:0]
	g.histPtr = 0
	g.moveCount = 0
	g.passCount = 0
	g.nextToMove = 0
}

func (g *Game) ActionSpace() int { return g.b.data.Shape()[1] }

func (g *Game) Board() []game.Colour { return g.b.data.Data().([]game.Colour) }

func (g *Game) Hash() game.Zobrist {
	h := fnv.New32a()
	data := g.b.data.Data().([]game.Colour)
	for i := range data {
		fmt.Fprintf(h, "%v", data[i])
	}
	return game.Zobrist(h.Sum32())
}

func (g *Game) Historical(i int) []game.Colour { return g.historical[i] }

func (g *Game) Format(s fmt.State, c rune) { g.b.Format(s, c) }
