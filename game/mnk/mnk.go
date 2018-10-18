package mnk

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"

	"github.com/gorgonia/agogo/game"
)

var (
	Pass = game.Single(-1)

	Cross  = game.Player(game.Black)
	Nought = game.Player(game.White)

	r = rand.New(rand.NewSource(1337))
)

var _ game.State = &MNK{}

// MNK is a representation of M,N,K games - a game is played on a MxN board. K moves to win.
type MNK struct {
	sync.Mutex
	board   []game.Colour
	m, n, k int

	nextToMove game.Player
	history    []game.PlayerMove
	historical [][]game.Colour
	histPtr    int
}

// New creates a new MNK game
func New(m, n, k int) *MNK {
	return &MNK{
		board:      make([]game.Colour, m*n),
		history:    make([]game.PlayerMove, 0, m*n),
		historical: make([][]game.Colour, 0, m*n),
		m:          m,
		n:          n,
		k:          k,
	}
}

// TicTacToe creates a new MNK game for Tic Tac Toe
func TicTacToe() *MNK {
	return &MNK{
		board:      make([]game.Colour, 9),
		history:    make([]game.PlayerMove, 0, 9),
		historical: make([][]game.Colour, 0, 9),
		m:          3,
		n:          3,
		k:          3,
	}
}

func (g *MNK) Format(s fmt.State, c rune) {
	for i, c := range g.board {
		if i%g.n == 0 {
			fmt.Fprint(s, "⎢ ")
		}
		fmt.Fprintf(s, "%s ", c)
		if (i+1)%g.n == 0 && i != 0 {
			fmt.Fprint(s, "⎥\n")
		}
	}
}

func (g *MNK) BoardSize() (int, int) { return g.m, g.n }
func (g *MNK) Board() []game.Colour  { return g.board }

func (g *MNK) Historical(i int) []game.Colour { return g.historical[i] }

func (g *MNK) Hash() game.Zobrist {
	h := fnv.New32a()
	for _, v := range g.board {
		fmt.Fprintf(h, "%v", v)
	}
	return game.Zobrist(h.Sum32())
}

func (g *MNK) ActionSpace() int { return g.m * g.n }

func (g *MNK) SetToMove(p game.Player) { g.Lock(); g.nextToMove = p; g.Unlock() }

func (g *MNK) ToMove() game.Player { return g.nextToMove }

func (g *MNK) LastMove() game.PlayerMove {
	if len(g.history) > 0 {
		return g.history[g.histPtr-1]
	}
	return game.PlayerMove{game.Player(game.None), Pass}
}

// Passes always returns -1. You can't pass in tic-tac-toe
func (g *MNK) Passes() int { return -1 }

func (g *MNK) MoveNumber() int { return len(g.history) }

func (g *MNK) Check(m game.PlayerMove) bool {
	if m.Single.IsResignation() {
		return true
	}

	// Pass not allowed!
	if m.Single.IsPass() {
		return false
	}

	if int(m.Single) >= len(g.board) {
		return false
	}

	if g.board[int(m.Single)] != game.None {
		return false
	}
	return true
}

func (g *MNK) Apply(m game.PlayerMove) game.State {
	if !g.Check(m) {
		return g // no change to the state
	}

	hb := make([]game.Colour, len(g.board))
	g.Lock()
	copy(hb, g.board)
	g.board[int(m.Single)] = game.Colour(m.Player)
	g.histPtr++
	if len(g.history) < g.histPtr {
		g.history = append(g.history, m)
	} else {
		g.history[g.histPtr-1] = m
	}
	g.historical = append(g.historical, hb)
	g.nextToMove = opponent(m.Player)

	g.Unlock()
	return g
}

// Handicap always returns 0.
func (g *MNK) Handicap() int { return 0 }

func (g *MNK) Score(p game.Player) float32 {
	if g.isWinner(p) {
		return 1
	}
	if g.isWinner(opponent(p)) {
		return -2
	}
	return 0 // draw or incomplete
}

// AdditionalScore returns 0 always. No Komi
func (g *MNK) AdditionalScore() float32 { return 0 }

// Ended checks if the game has ended. If it has, who is the winner?
func (g *MNK) Ended() (ended bool, winner game.Player) {
	if g.isWinner(Cross) {
		return true, Cross
	}
	if g.isWinner(Nought) {
		return true, Nought
	}
	for _, c := range g.board {
		if c == game.None {
			return false, game.Player(game.None)
		}
	}
	return true, game.Player(game.None)
}

func (g *MNK) Reset() {
	for i := range g.board {
		g.board[i] = game.None
	}
	g.history = g.history[:0]
	g.histPtr = 0
}

func (g *MNK) UndoLastMove() {
	if len(g.history) > 0 {
		g.board[int(g.history[g.histPtr-1].Single)] = game.None
		g.histPtr--
	}
}

func (g *MNK) Fwd() {
	if len(g.history) > 0 {
		g.histPtr++
	}
}

func (g *MNK) Eq(other game.State) bool {
	ot, ok := other.(*MNK)
	if !ok {
		return false
	}
	if len(g.board) != len(ot.board) {
		return false
	}
	for i := range g.board {
		if g.board[i] != ot.board[i] {
			return false
		}
	}
	return true
}

func (g *MNK) Clone() game.State {
	retVal := New(g.m, g.n, g.k)
	g.Lock()
	copy(retVal.board, g.board)
	retVal.history = make([]game.PlayerMove, len(g.history), g.m*g.n)
	copy(retVal.history, g.history)
	copy(retVal.historical, g.historical)
	retVal.nextToMove = g.nextToMove
	retVal.histPtr = g.histPtr
	g.Unlock()
	return retVal
}

func (g *MNK) isWinner(p game.Player) bool {
	colour := game.Colour(p)
	// check rows
	for i := 0; i < g.m; i++ {
		var rowCount int
		for j := 0; j < g.n; j++ {
			if g.board[i*g.n+j] == colour {
				rowCount++
			} else {
				rowCount--
			}

		}
		if rowCount >= g.k {
			return true
		}
	}
	// check cols
	for j := 0; j < g.n; j++ {
		var count int
		for i := 0; i*g.n+j < len(g.board); i++ {
			if g.board[i*g.n+j] == colour {
				count++
			} else {
				count = 0
			}
		}
		if count >= g.k {
			return true
		}
	}

	for i := 0; i < g.m; i++ {
		for j := 0; g.n-j > g.n-g.k && j < g.n; j++ {
			idx := i*g.n + j
			var diagCount int
			for g.board[idx] == colour {
				diagCount++
				if diagCount >= g.k {
					return true
				}

				idx = idx + g.n + 1
				if idx >= g.m*g.n {
					break
				}
			}
		}
	}

	for i := 0; i < g.m; i++ {
		for j := g.n - 1; j >= g.k-1; j-- {
			idx := i*g.n + j
			var diagCount int
			for g.board[idx] == colour {
				diagCount++

				if diagCount >= g.k {
					return true
				}

				idx = idx + g.n - 1
				if idx >= g.m*g.n {
					break
				}
			}
		}
	}
	return false
}

func opponent(p game.Player) game.Player {
	switch p {
	case Cross:
		return Nought
	case Nought:
		return Cross
	}
	panic("Unreachable")
}
