package 围碁

import "github.com/gorgonia/agogo/game"

var _ game.State = &Game{}

type historicalBoard struct {
	board []game.Colour
	hash  game.Zobrist
}

// Game implements game.State and mcts.GameState
type Game struct {
	board      *Board
	history    []game.PlayerMove
	historical []historicalBoard
	nextToMove game.Player

	komi      float32 // komidashi
	moveCount int     // move number, 1 indexed
	passes    int     // count of passes
	histPtr   int     // pointer at the history (for easy forwarding)
	handicap  int     // duh
	captures  [2]byte // number of captures
	ends      bool    // game ended due to all possible moves being played
}

func New(boardSize, handicap int, komi float64) *Game {
	b := newBoard(boardSize)
	return &Game{
		board:      b,
		nextToMove: game.Player(game.Black),
		komi:       float32(komi),
		handicap:   handicap,
		historical: make([]historicalBoard, 0, int(b.size)),
		history:    make([]game.PlayerMove, 0, int(b.size)),
	}
}

func (g *Game) BoardSize() (int, int) { return int(g.board.size), int(g.board.size) }

func (g *Game) Board() []game.Colour { return g.board.data }

func (g *Game) Historical(i int) []game.Colour { return g.historical[i].board }

func (g *Game) Hash() game.Zobrist { return game.Zobrist(g.board.hash) }

func (g *Game) ActionSpace() int { return len(g.board.data) }

func (g *Game) SetToMove(p game.Player) { g.nextToMove = p }

func (g *Game) ToMove() game.Player { return g.nextToMove }

func (g *Game) LastMove() game.PlayerMove {
	if len(g.history) > 0 {
		return g.history[g.histPtr-1]
	}
	return game.PlayerMove{Player: game.Player(game.None), Single: -1}
}

func (g *Game) Passes() int { return g.passes }

func (g *Game) MoveNumber() int { return len(g.history) }

func (g *Game) Check(m game.PlayerMove) bool {
	if m.Single.IsResignation() {
		return true
	}
	if m.Single.IsPass() {
		return true
	}
	if int(m.Single) >= len(g.board.data) {
		return false
	}
	_, err := g.board.check(m)

	// TODO: SuperKo checks
	return err == nil
}

func (g *Game) Apply(m game.PlayerMove) game.State {
	newState := g.Clone().(*Game)
	// TODO : check for passes etc
	captures, _ := newState.board.Apply(m)

	newState.captures[m.Player-1] += captures
	newState.nextToMove = Opponent(m.Player)
	newState.history = append(newState.history, m)
	newState.histPtr++
	newState.moveCount++
	return newState
}

func (g *Game) Ended() (ended bool, winner game.Player) {
	if g.passes >= 2 {
		ended = true
	}
	if g.ends {
		ended = true
	}
	if !ended {
		return false, game.Player(game.None)
	}

	whiteScore := g.Score(WhiteP)
	blackScore := g.Score(BlackP)
	switch {
	case whiteScore == blackScore:
		return true, game.Player(game.None)
	case whiteScore > blackScore:
		return true, WhiteP
	default:
		return true, BlackP
	}
}

func (g *Game) Reset() { panic("not implemented") }

func (g *Game) UndoLastMove() { panic("not implemented") }

func (g *Game) Fwd() { panic("not implemented") }

func (g *Game) Eq(other game.State) bool {
	ot, ok := other.(*Game)
	if !ok {
		return false
	}

	// easy to check stuff first
	if g.nextToMove != ot.nextToMove ||
		g.komi != ot.komi ||
		g.moveCount != ot.moveCount ||
		g.passes != ot.passes ||
		g.handicap != ot.handicap ||
		len(g.history) != len(ot.history) &&
			(len(g.history) > 0 && len(ot.history) > 0 && len(g.history[:g.histPtr-1]) != len(ot.history[:ot.histPtr-1])) {
		return false
	}

	// specifically unchecked: histPtr

	for i, c := range g.captures {
		if ot.captures[i] != c {
			return false
		}
	}

	// heavier checks

	if !g.board.Eq(ot.board) {
		return false
	}
	for i, j := 0, 0; i < g.histPtr && j < ot.histPtr; i, j = i+1, j+1 {
		pm := g.history[i]
		if !pm.Eq(ot.history[j]) {
			return false
		}
	}

	return true
}

func (g *Game) Clone() game.State {
	newState := &Game{}
	newState.board = g.board.Clone()
	newState.history = make([]game.PlayerMove, len(g.history), len(g.history)+1)
	copy(newState.history, g.history)
	newState.nextToMove = g.nextToMove
	newState.komi = g.komi
	newState.moveCount = g.moveCount
	newState.passes = g.passes
	newState.histPtr = g.histPtr
	newState.captures = g.captures
	return newState
}

func (g *Game) Handicap() int               { return g.handicap }
func (g *Game) Score(p game.Player) float32 { panic("not implemented") }
func (g *Game) AdditionalScore() float32    { return g.komi }
func (g *Game) SuperKo() bool               { panic("not implemented") }

func (g *Game) IsEye(m game.PlayerMove) bool {
	panic("not implemented")
}
