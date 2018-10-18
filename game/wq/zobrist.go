package 围碁

import (
	"math/rand"
	"time"

	"github.com/gorgonia/agogo/game"
	"github.com/pkg/errors"
)

// zobrist is a data structure for calculating Zobrist hashes.
// https://en.wikipedia.org/wiki/Zobrist_hashing
//
// The original implementation uses gorgonia's tensor
// Fundamentally it is a (BOARDSIZE, BOARDSIZE 2) 3-Tensor, which stores the hash state.
// The hash is then calculated from that
// But in light of optimizing all the things for memory, it's been stripped down to the absolute fundamentals:
//	- a backing storage
//	- an iterator for quick access
//
// The semantics of the iterator has also been updated.  Given that the board will be updated
// with a game.Single, instead of a game.Coord, another way to think of the table is as a matrix of
// (BOARDSIZE * BOARDSIZE, 2). The design of the iterator is geared around that.
type zobrist struct {
	table  []int32   // backing storage
	it     [][]int32 // iterator for the normal hash
	hash   int32
	koHash int32
}

func makeZobrist(size int) zobrist {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	table, it := makeZobristTable(size) // see naughty.go
	for i := range table {
		table[i] = r.Int31()
	}
	return zobrist{
		table: table,
		it:    it,
	}
}

// update calculates the hash and returns it. As per the namesake, the calculated hash is updaated as a side effect.
func (z *zobrist) update(m game.PlayerMove) (int32, error) {
	switch game.Colour(m.Player) {
	case game.Black:
		z.hash ^= z.it[m.Single][0]
		return z.hash, nil
	case game.White:
		z.hash ^= z.it[m.Single][1]
		return z.hash, nil
	default:
		return 0, errors.Errorf("Cannot update hash for %v", m)
	}
}
