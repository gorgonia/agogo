package game

import (
	"fmt"
)

type Colour int32

const (
	None Colour = iota
	Black
	White
)

func (cl Colour) Format(s fmt.State, c rune) {
	switch c {
	case 'v': // used in debug
		switch cl {
		case None:
			fmt.Fprint(s, "None")
		case Black:
			fmt.Fprint(s, "Black")
		case White:
			fmt.Fprint(s, "White")
		}
	case 's': // used in board games
		switch cl {
		case None:
			fmt.Fprint(s, "·")
		case Black:
			fmt.Fprint(s, "X")
		case White:
			fmt.Fprint(s, "O")
		}
	}
}

// Player represents a player. It's also a colour.
type Player Colour

func (p Player) Format(s fmt.State, c rune) {
	switch c {
	case 'v': // used in debug
		switch Colour(p) {
		case None:
			fmt.Fprint(s, "None")
		case Black:
			fmt.Fprint(s, "Black")
		case White:
			fmt.Fprint(s, "White")
		}
	case 's': // used in board games
		switch Colour(p) {
		case None:
			fmt.Fprint(s, "·")
		case Black:
			fmt.Fprint(s, "X")
		case White:
			fmt.Fprint(s, "O")
		}
	}
}

// PlayerMove is a tuple indicating the player and the move to be made.
//
// For now, the move is a Single. The original implementation took a Coordinate
type PlayerMove struct {
	Player
	Single
}

// Eq returns true if both are equal
func (p PlayerMove) Eq(other PlayerMove) bool {
	return p.Player == other.Player && p.Single == other.Single
}

func (p PlayerMove) Format(s fmt.State, c rune) { fmt.Fprintf(s, "%v@%d", p.Player, p.Single) }

// Coordinate is a representation of coordinates. This is typically a move
type Coordinate interface {
	IsResignation() bool
	IsPass() bool
}

// Coord represents a (row, col) coordinate.
// Given we're unlikely to actually have a board size of 255x255 or greater,
// a pair of bytes is sufficient to represent the coordinates
//
// The Coord uses a standard computer cartesian coordinates
//		- (0, 0) represents the top left
//		- (18, 18) represents the bottom right of a 19x19 board
//		- (255, 255) represents a "pass" move
// 		- (254, 254) represents a "resign" move
type Coord struct {
	X, Y int16
}

func (c Coord) Add(other Coord) struct{ X, Y int16 } {
	return Coord{c.X + other.X, c.Y + other.Y}
}

func (c Coord) Eq(other Coord) bool { return c.X == other.X && c.Y == other.Y }

// IsResignation returns true when the coordinate represents a "resignation" move
func (c Coord) IsResignation() bool { return c.X == 254 && c.Y == 254 }

// IsPass returns true when the coordinate represents a "pass" move
func (c Coord) IsPass() bool { return c.X == 255 && c.Y == 255 }

// Single represents a coordinate as a single 8-bit number, utilized in a rowmajor fashion.
//		- 0 represents the top left
//		- 18 represents the top right
//		- 19 represents (1, 0)
// 		- -1 represents the "pass" move
//		- -2 represents the "resignation" move
type Single int32

// IsResignation returns true when the coordinate represents a "resignation" move
func (c Single) IsResignation() bool { return c == -2 }

// IsPass returns true when the coordinate represents a "pass" move
func (c Single) IsPass() bool { return c == -1 }

// State is any game that implements these and are able to report back
type State interface {
	// These methods represent the game state
	BoardSize() (int, int) // returns the board size
	Board() []Colour       // returns the board state
	ActionSpace() int      // returns the number of permissible actions
	Hash() Zobrist         // returns the hash of the board
	ToMove() Player        // returns the next player to move (terminology is a bit confusing - this means the current player)
	Passes() int           // returns number of passes that have been made
	MoveNumber() int       // returns count of moves so far that led to this point.
	LastMove() PlayerMove  // returns the last move that was made
	Handicap() int         // returns a handicap (i.e. allow N moves)

	// Meta-game stuff
	Score(p Player) float32             // score of the given player
	AdditionalScore() float32           // additional tie breaking scores (like komi etc)
	Ended() (ended bool, winner Player) // has the game ended? if yes, then who's the winner?

	// interactions
	SetToMove(Player)         // set the next player to move
	Check(m PlayerMove) bool  // check if the placement is legal
	Apply(m PlayerMove) State // should return a GameState. The required side effect is the NextToMove has to change.
	Reset()                   // reset state

	// For MCTS
	Historical(i int) []Colour // returns the board state from history
	UndoLastMove()
	Fwd()

	// generics
	Eq(other State) bool
	Clone() State
}

// KomiSetter is any State that can set a Komi score.
//
// The komi score may be acquired from the State via AdditionalScore()
type KomiSetter interface {
	State
	SetKomi(komi float64) error
}

// Zobrist is a type representing a "zobrist" hash.
// The word "Zobrist" is put in quotes because only Go and chess uses zobrist hashing.
// Other games have different hashes of the boards (because only Go and Chess have subtractive boards)
type Zobrist uint32

type MetaState interface {
	Name() string // name of the game
	Epoch() int
	GameNumber() int
	Score(a Player) float64
	State() State
}

type CoordConverter interface {
	Ltoi(Coord) Single
	Itol(Single) Coord
}
