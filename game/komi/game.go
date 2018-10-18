package komi

import (
	"fmt"
	"sync"

	"github.com/gorgonia/agogo/game"
	"github.com/pkg/errors"
)

const (
	None  = game.None
	Black = game.Black
	White = game.White

	BlackP = game.Player(game.Black)
	WhiteP = game.Player(game.White)

	Pass = game.Single(-1)
)

var _ game.State = &Game{}
var _ game.CoordConverter = &Game{}

type Game struct {
	sync.Mutex
	board []game.Colour
	m, n  int32 // board size is mxn, k surrounded to win
	k     float32

	nextToMove game.Player
	history    []game.PlayerMove
	historical [][]game.Colour
	histPtr    int
	ws, bs     float32 // white score and black score
	z          zobrist

	// transient state
	taken int
	err   error
}

func New(m, n, k int) *Game {
	return &Game{
		m:          int32(m),
		n:          int32(n),
		k:          float32(k),
		board:      make([]game.Colour, m*n),
		nextToMove: BlackP,
		z:          makeZobrist(m, n),
	}
}

func (g *Game) BoardSize() (int, int) { return int(g.m), int(g.n) }

func (g *Game) Board() []game.Colour { return g.board }

func (g *Game) Historical(i int) []game.Colour { return g.historical[i] }

func (g *Game) Hash() game.Zobrist { return game.Zobrist(g.z.hash) }

func (g *Game) ActionSpace() int { return int(g.m * g.n) }

func (g *Game) SetToMove(p game.Player) { g.Lock(); g.nextToMove = p; g.Unlock() }

func (g *Game) ToMove() game.Player { return g.nextToMove }

func (g *Game) LastMove() game.PlayerMove {
	if len(g.history) > 0 {
		return g.history[g.histPtr-1]
	}
	return game.PlayerMove{game.Player(game.None), Pass}
}

// Passes always returns -1. You can't pass in Komi
func (g *Game) Passes() int { return -1 }

func (g *Game) MoveNumber() int { return len(g.history) }

func (g *Game) Check(m game.PlayerMove) bool {
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
	if _, err := g.check(m); err != nil {
		// log.Printf("Checking %v. OK", m)
		return false
	}
	return true
}

func (g *Game) Apply(m game.PlayerMove) game.State {
	hb := make([]game.Colour, len(g.board))

	g.taken, g.err = g.apply(m)
	if g.err != nil {
		return g
	}
	g.Lock()
	copy(hb, g.board)
	g.histPtr++
	if len(g.history) < g.histPtr {
		g.history = append(g.history, m)
	} else {
		g.history[g.histPtr-1] = m
	}
	g.historical = append(g.historical, hb)
	g.nextToMove = Opponent(m.Player)
	g.Unlock()

	switch m.Player {
	case BlackP:
		g.bs += float32(g.taken)
	case WhiteP:
		g.ws += float32(g.taken)
	}
	return g
}

func (g *Game) Handicap() int { return 0 }

func (g *Game) Score(p game.Player) float32 {
	if p == WhiteP {
		return g.ws
	} else if p == BlackP {
		return g.bs
	}
	panic("unreachable")
}

func (g *Game) AdditionalScore() float32 { return 0 }

func (g *Game) Ended() (ended bool, winner game.Player) {
	if g.ws >= g.k {
		return true, WhiteP
	}
	if g.bs >= g.k {
		return true, BlackP
	}

	var potentials []game.Single
	for i := 0; i < len(g.board); i++ {
		if g.board[i] == None {
			potentials = append(potentials, game.Single(i))
		}
	}

	var currHasMoveLeft, oppHasMoveLeft bool
	for _, pot := range potentials {
		pm := game.PlayerMove{g.nextToMove, game.Single(pot)}
		if g.Check(pm) {
			currHasMoveLeft = true
			break
		}
	}
	for _, pot := range potentials {
		pm := game.PlayerMove{Opponent(g.nextToMove), game.Single(pot)}
		if g.Check(pm) {
			oppHasMoveLeft = true
			break
		}
	}

	if currHasMoveLeft && oppHasMoveLeft {
		return false, game.Player(game.None)
	}
	if g.ws > g.bs {
		return true, game.Player(game.White)
	}
	if g.bs > g.ws {
		return true, game.Player(game.Black)
	}
	return true, game.Player(game.None)

}

func (g *Game) Reset() {
	for i := range g.board {
		g.board[i] = game.None
	}
	g.history = g.history[:0]
	g.historical = g.historical[:0]
	g.histPtr = 0
	g.nextToMove = BlackP
	g.ws = 0
	g.bs = 0
	g.z = makeZobrist(int(g.m), int(g.n))
}

func (g *Game) UndoLastMove() {
	if len(g.history) > 0 {
		g.board[int(g.history[g.histPtr-1].Single)] = game.None
		g.histPtr--
	}
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
	if g.nextToMove != ot.nextToMove ||
		len(g.board) != len(ot.board) ||
		len(g.history) != len(ot.history) &&
			(len(g.history) > 0 && len(ot.history) > 0 && len(g.history[:g.histPtr-1]) != len(ot.history[:ot.histPtr-1])) {
		return false
	}
	for i := range g.board {
		if g.board[i] != ot.board[i] {
			return false
		}
	}
	return true
}

func (g *Game) Clone() game.State {
	retVal := &Game{
		m:     g.m,
		n:     g.n,
		k:     g.k,
		board: make([]game.Colour, len(g.board)),
	}
	g.Lock()
	copy(retVal.board, g.board)
	retVal.history = make([]game.PlayerMove, len(g.history), len(g.history)+4)
	retVal.historical = make([][]game.Colour, len(g.historical), len(g.historical)+4)
	copy(retVal.history, g.history)
	copy(retVal.historical, g.historical)
	retVal.nextToMove = g.nextToMove
	retVal.histPtr = g.histPtr
	retVal.z = g.z.clone()
	g.Unlock()
	return retVal
}

func (g *Game) Format(s fmt.State, c rune) {
	it := game.MakeIterator(g.board, g.m, g.n)
	defer game.ReturnIterator(g.m, g.n, it)
	switch c {
	case 's', 'v':
		for _, row := range it {
			fmt.Fprint(s, "⎢ ")
			for _, col := range row {
				fmt.Fprintf(s, "%s ", col)
			}
			fmt.Fprint(s, "⎥\n")
		}
	}
}

func (g *Game) Err() error { return g.err }

func (g *Game) Itol(c game.Single) game.Coord {
	x := int16(int32(c) / int32(g.m))
	y := int16(int32(c) % int32(g.m))
	return game.Coord{x, y}
}

// Ltoi takes a coordinate and return a single
func (g *Game) Ltoi(c game.Coord) game.Single { return game.Single(int32(c.X)*g.m + int32(c.Y)) }

func (g *Game) apply(m game.PlayerMove) (int, error) {
	if !isValid(m.Player) {
		return 0, errors.WithMessage(moveError(m), "Impossible player")
	}
	if m.Single.IsPass() {
		return 0, errors.WithMessage(moveError(m), "Cannot pass")
	}

	if int32(m.Single) >= g.m*g.m { // don't check for negative moves. the special moves are to be made at the Game level
		return 0, errors.WithMessage(moveError(m), "Impossible move")
	}

	// if the board location is not empty, then clearly we can't apply
	if g.board[m.Single] != game.None {
		return 0, errors.WithMessage(moveError(m), "Application Failure - board location not empty.")
	}

	captures, err := g.check(m)
	if err != nil {
		return 0, errors.WithMessage(err, "Application Failure.")
	}

	// the move is valid.
	// make the move then update zobrist hash
	g.board[m.Single] = game.Colour(m.Player)
	g.z.update(m)

	// remove prisoners
	for _, prisoner := range captures {
		g.board[prisoner] = game.None
		g.z.update(game.PlayerMove{Player: Opponent(m.Player), Single: prisoner}) // Xoring the original colour
	}
	return len(captures), nil
}

// check will find the captures (if any) if the move is valid. If the move is invalid, an error will be returned
func (g *Game) check(m game.PlayerMove) (captures []game.Single, err error) {
	if m.Single.IsPass() {
		return nil, errors.New("Cannot pass")
	}

	c := g.Itol(m.Single)
	it := game.MakeIterator(g.board, g.m, g.n)
	defer game.ReturnIterator(g.m, g.n, it)

	adj := g.adjacentsCoord(c)
	for _, a := range adj {
		if !g.isCoordValid(a) {
			continue
		}

		if it[a.X][a.Y] == game.Colour(Opponent(m.Player)) {
			// find Opponent stones with no liberties
			nolibs := g.nolib(it, a, c)
			for _, nl := range nolibs {
				captures = append(captures, g.Ltoi(nl))
			}
		}
	}
	if len(captures) > 0 {
		return
	}
	// check for suicide moves
	suicides := g.nolib(it, c, game.Coord{-5, -5}) // purposefully incomparable
	if len(suicides) > 0 {
		return nil, errors.WithMessage(moveError(m), "Suicide is not a valid option.")
	}
	return
}

// c is the position of the stone, potential is where a potential stone could be placed
func (g *Game) nolib(it [][]game.Colour, c, potential game.Coord) (retVal []game.Coord) {
	found := true
	founds := []game.Coord{c}
	for found {
		found = false
		var group []game.Coord
		for _, f := range founds {
			adj := g.adjacentsCoord(f)

			for _, a := range adj {
				if !g.isCoordValid(a) {
					continue
				}
				// does f have a free liberty
				if it[a.X][a.Y] == game.None && !a.Eq(potential) {
					return nil
				}
				// if the found node is not the same colour as its adjacent
				if it[f.X][f.Y] != it[a.X][a.Y] {
					continue
				}

				// check if we have a group
				potentialGroup := true
				for _, gr := range group {
					if gr.Eq(a) {
						potentialGroup = false
						break
					}
				}

				if potentialGroup {
					for _, l := range retVal {
						if l.Eq(a) {
							potentialGroup = false
							break
						}
					}
				}

				if potentialGroup {
					group = append(group, a)
					found = true
				}

			}
		}
		retVal = append(retVal, founds...)
		founds = group
	}
	return retVal
}

// adjacentsCoord returns the adjacent positions given a coord
func (g *Game) adjacentsCoord(c game.Coord) (retVal [4]game.Coord) {
	for i := range retVal {
		retVal[i] = c.Add(adjacents[i])
	}
	return retVal
}

func (g *Game) isCoordValid(c game.Coord) bool {
	x, y := int32(c.X), int32(c.Y)
	// check if valid
	if x >= g.m || x < 0 {
		return false
	}

	if y >= g.n || y < 0 {
		return false
	}
	return true
}

var adjacents = [4]game.Coord{
	{0, 1},
	{1, 0},
	{0, -1},
	{-1, 0},
}

// Opponent returns the colour of the Opponent player
func Opponent(p game.Player) game.Player {
	switch game.Colour(p) {
	case game.White:
		return game.Player(game.Black)
	case game.Black:
		return game.Player(game.White)
	}
	panic("Unreachaable")
}

// isValid checks that a player is indeed valid
func isValid(p game.Player) bool { return game.Colour(p) == game.Black || game.Colour(p) == game.White }
