// package 围碁 implements Go (the board game) related code
//
// 围碁 is a bastardized word.
// The first character is read "wei" in Chinese. The second is read "qi" in Chinese.
// However, the charcter 碁 is no longer actively used in Chinese.
// It is however, actively used in Japanese. Specifically, it's read "go" in Japanese.
//
// The main reason why this package is named with unicode characters instead of `package go`
// is because the standard library of the Go language have the prefix "go"
package 围碁

import (
	"fmt"

	"github.com/gorgonia/agogo/game"
	"github.com/pkg/errors"
)

const (
	// for fast calculation of neighbours
	shift = 4
	mask  = (1 << shift) - 1
)

const (
	None  = game.None
	Black = game.Black
	White = game.White

	BlackP = game.Player(game.Black)
	WhiteP = game.Player(game.White)
)

// Opponent returns the colour of the opponent player
func Opponent(p game.Player) game.Player {
	switch game.Colour(p) {
	case game.White:
		return game.Player(game.Black)
	case game.Black:
		return game.Player(game.White)
	}
	panic("Unreachaable")
}

// IsValid checks that a player is indeed valid
func IsValid(p game.Player) bool { return game.Colour(p) == game.Black || game.Colour(p) == game.White }

// Board represetns a board.
//
// The board was originally implemented with a *tensor.Dense. However it turns out that
// if many iterations of the board were needed, we would quite quickly run out of memory
// because the *tensor.Dense data structure is quite huge.
//
// Given we know the stride of the board, a decision was made to cull the fat from the data structure.
type Board struct {
	size    int32
	data    []game.Colour   // backing data
	it      [][]game.Colour // iterator for quick access
	zobrist                 // hashing of the board
}

func newBoard(size int) *Board {
	data, it := makeBoard(size)
	z := makeZobrist(size)
	return &Board{
		size:    int32(size),
		data:    data,
		it:      it,
		zobrist: z,
	}
}

// Clone clones the board
func (b *Board) Clone() *Board {
	data, it := makeBoard(int(b.size))
	z := makeZobrist(int(b.size))
	z.hash = b.hash
	copy(data, b.data)
	copy(z.table, b.table)
	return &Board{
		size:    b.size,
		data:    data,
		it:      it,
		zobrist: z,
	}
}

// Eq checks that both are equal
func (b *Board) Eq(other *Board) bool {
	if b == other {
		return true
	}
	// easy to check stuff
	if b.size != other.size ||
		b.hash != other.hash ||
		len(b.data) != len(other.data) ||
		len(b.zobrist.table) != len(other.zobrist.table) {
		return false
	}

	for i, c := range b.data {
		if c != other.data[i] {
			return false
		}
	}

	for i, r := range b.zobrist.table {
		if r != other.zobrist.table[i] {
			return false
		}
	}
	return true
}

// Format implements fmt.Formatter
func (b *Board) Format(s fmt.State, c rune) {
	switch c {
	case 's':
		for _, row := range b.it {
			fmt.Fprint(s, "⎢ ")
			for _, col := range row {
				fmt.Fprintf(s, "%s ", col)
			}
			fmt.Fprint(s, "⎥\n")
		}
	}
}

// Reset resets the board state
func (b *Board) Reset() {
	for i := range b.data {
		b.data[i] = game.None
	}
	b.zobrist.hash = 0
}

// Hash returns the calculated hash of the board
func (b *Board) Hash() int32 { return b.hash }

// Apply returns the number of captures or an error, if a move were to be applied
func (b *Board) Apply(m game.PlayerMove) (byte, error) {
	if !IsValid(m.Player) {
		return 0, errors.WithMessage(moveError(m), "Impossible player")
	}

	if int32(m.Single) >= b.size*b.size { // don't check for negative moves. the special moves are to be made at the Game level
		return 0, errors.WithMessage(moveError(m), "Impossible move")
	}

	// if the board location is not empty, then clearly we can't apply
	if b.data[m.Single] != game.None {
		return 0, errors.WithMessage(moveError(m), "Application Failure - board location not empty.")
	}

	captures, err := b.check(m)
	if err != nil {
		return 0, errors.WithMessage(err, "Application Failure.")
	}

	// the move is valid.
	// make the move then update zobrist hash
	b.data[m.Single] = game.Colour(m.Player)
	b.zobrist.update(m)

	// remove prisoners
	for _, prisoner := range captures {
		b.data[prisoner] = game.None
		b.zobrist.update(game.PlayerMove{Player: Opponent(m.Player), Single: prisoner}) // Xoring the original colour
	}
	return byte(len(captures)), nil
}

func (b *Board) Score(player game.Player) float32 {
	colour := game.Colour(player)
	bd := make([]bool, len(b.data))
	q := make(chan int32, b.size*b.size)
	adjacents := [4]int32{-b.size, 1, b.size, 1}

	var reachable float32
	for i := int32(0); i < int32(len(b.data)); i++ {
		if b.data[i] == colour {
			reachable++
			bd[i] = true
			q <- i
		}
	}
	for len(q) > 0 {
		i := <-q
		for _, adj := range adjacents {
			a := i + adj
			if a >= b.size || a < 0 {
				continue
			}
			if !bd[a] && b.data[a] == None {
				reachable++
				bd[a] = true
				q <- a
			}
		}
	}
	return reachable
}

// check will find the captures (if any) if the move is valid. If the move is invalid, an error will be returned
func (b *Board) check(m game.PlayerMove) (captures []game.Single, err error) {
	x := int16(int32(m.Single) / b.size)
	y := int16(int32(m.Single) % b.size)
	c := game.Coord{x, y}

	adj := b.adjacentsCoord(c)
	for _, a := range adj {
		if !b.isCoordValid(a) {
			continue
		}

		if b.it[a.X][a.Y] == game.Colour(Opponent(m.Player)) {
			// find opponent stones with no liberties
			nolibs := b.nolib(a, c)
			for _, nl := range nolibs {
				captures = append(captures, b.ltoi(nl))
			}
		}
	}
	if len(captures) > 0 {
		return
	}

	// check for suicide moves
	suicides := b.nolib(c, game.Coord{-5, -5}) // purposefully incomparable
	if len(suicides) > 0 {
		return nil, errors.WithMessage(moveError(m), "Suicide is not a valid option.")
	}
	return
}

// c is the position of the stone, potential is where a potential stone could be placed
func (b *Board) nolib(c, potential game.Coord) (retVal []game.Coord) {
	found := true
	founds := []game.Coord{c}
	for found {
		found = false
		var group []game.Coord

		for _, f := range founds {
			adj := b.adjacentsCoord(f)

			for _, a := range adj {
				if !b.isCoordValid(a) {
					continue
				}
				// does f have a free liberty
				if b.it[a.X][a.Y] == game.None && !a.Eq(potential) {
					return nil
				}

				// if the found node is not the same colour as its adjacent
				if b.it[f.X][f.Y] != b.it[a.X][a.Y] {
					continue
				}

				// check if we have a group
				potentialGroup := true
				for _, g := range group {
					if g.Eq(a) {
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

// ltoi takes a coordinate and return a single
func (b *Board) ltoi(c game.Coord) game.Single { return game.Single(int32(c.X)*b.size + int32(c.Y)) }

// adjacentsCoord returns the adjacent positions given a coord
func (b *Board) adjacentsCoord(c game.Coord) (retVal [4]game.Coord) {
	for i := range retVal {
		retVal[i] = c.Add(adjacents[i])
	}
	return retVal
}

func (b *Board) isCoordValid(c game.Coord) bool {
	x, y := int32(c.X), int32(c.Y)
	// check if valid
	if x >= b.size || x < 0 {
		return false
	}

	if y >= b.size || y < 0 {
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
