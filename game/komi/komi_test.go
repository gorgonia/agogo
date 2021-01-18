package komi

import (
	"fmt"
	"testing"

	"github.com/gorgonia/agogo/game"
)

var applyTests = []struct {
	m, n       int
	board      []game.Colour
	move       game.PlayerMove
	board2     []game.Colour // nil if invalid
	taken      int
	whiteScore float32
	blackScore float32
	willErr    bool
}{
	// placing on an empty
	{
		m: 3, n: 3,
		board: []game.Colour{
			None, None, None,
			None, None, None,
			None, None, None,
		},
		move: game.PlayerMove{game.Player(Black), game.Single(4)}, // {1, 1}
		board2: []game.Colour{
			None, None, None,
			None, Black, None,
			None, None, None,
		},
		taken:      0,
		blackScore: 0, // TODO: CHECK
		willErr:    false,
	},

	// basic capture
	// · O ·
	// O X O
	// · · ·
	//
	// becomes:
	//
	// · O ·
	// O · O
	// · O ·
	{
		m: 3, n: 3,
		board: []game.Colour{
			None, White, None,
			White, Black, White,
			None, None, None,
		},
		move: game.PlayerMove{game.Player(White), game.Single(7)}, // {2, 1}
		board2: []game.Colour{
			None, White, None,
			White, None, White,
			None, White, None,
		},
		taken:      1,
		whiteScore: 1, // TODO: CHECK
		willErr:    false,
	},

	// group capture
	// Note the extra column on the right is because we use sqrt to determine board size
	// · O · ·
	// O X O ·
	// O X O ·
	// · · · ·
	//
	// becomes:
	//
	// · O · ·
	// O · O ·
	// O · O ·
	// · O · ·
	{
		m: 4, n: 4,
		board: []game.Colour{
			None, White, None, None,
			White, Black, White, None,
			White, Black, White, None,
			None, None, None, None,
		},
		move: game.PlayerMove{game.Player(White), game.Single(13)}, // {3, 1}
		board2: []game.Colour{
			None, White, None, None,
			White, None, White, None,
			White, None, White, None,
			None, White, None, None,
		},
		taken:      2,
		whiteScore: 2, // TODO: CHECK
		willErr:    false,
	},

	// edge case (literally AT THE EDGE)
	// · · · ·
	// · · · ·
	// · X X ·
	// X O O ·
	//
	// becomes:
	//
	// · · · ·
	// · · · ·
	// · X X ·
	// X · · X
	{
		m: 4, n: 4,
		board: []game.Colour{
			None, None, None, None,
			None, None, None, None,
			None, Black, Black, None,
			Black, White, White, None,
		},
		move: game.PlayerMove{game.Player(Black), game.Single(15)}, // {3, 3}
		board2: []game.Colour{
			None, None, None, None,
			None, None, None, None,
			None, Black, Black, None,
			Black, None, None, Black,
		},
		taken:      2,
		blackScore: 2, // TODO: CHECK -  this should just be 2 unless my understanding of Go (the game) is wrong
		willErr:    false,
	},

	// Suicide
	// · X ·
	// X · X
	// · X ·
	//
	// Disallowed:
	// · X ·
	// X O X
	// · X ·
	{
		m: 3, n: 3,
		board: []game.Colour{
			None, White, None,
			White, None, White,
			None, White, None,
		},
		move:    game.PlayerMove{game.Player(Black), game.Single(4)}, // {1, 1}
		board2:  nil,
		taken:   0,
		willErr: true,
	},

	// impossible move
	{
		m: 3, n: 3,
		board: []game.Colour{
			None, None, None,
			None, None, None,
			None, None, None,
		},
		move:    game.PlayerMove{game.Player(Black), game.Single(15)}, // {3, 3}
		board2:  nil,
		taken:   0,
		willErr: true,
	},

	// impossible colour
	{
		m: 3, n: 3,
		board: []game.Colour{
			None, None, None,
			None, None, None,
			None, None, None,
		},
		move:    game.PlayerMove{game.Player(None), game.Single(15)}, // {3, 3}
		board2:  nil,
		taken:   0,
		willErr: true,
	},
}

func TestBoard_apply(t *testing.T) {
	for testID, at := range applyTests {
		board := New(at.m, at.n, 3)
		data := board.board
		copy(data, at.board)

		board.Apply(at.move)
		taken := board.taken
		err := board.err

		switch {
		case at.willErr && err == nil:
			t.Errorf("Expected an error for \n%s", board)
			continue
		case at.willErr && err != nil:
			// expected an error
			continue
		case !at.willErr && err != nil:
			t.Errorf("err %v", err)
			continue
		}

		if taken != at.taken {
			t.Errorf("Test %d: Expected %d to be taken. Got %d instead", testID, at.taken, taken)
		}

		for i, v := range data {
			if v != at.board2[i] {
				t.Errorf("Board failure:\n%s", board)
			}
		}

		whiteScore := board.Score(WhiteP)
		blackScore := board.Score(BlackP)
		if whiteScore != at.whiteScore {
			t.Errorf("Expected White Score %v. Got %v. Board\n%s", at.whiteScore, whiteScore, board)
		}
		if blackScore != at.blackScore {
			t.Errorf("Expected Black Score %v. Got %v. Board\n%s", at.blackScore, blackScore, board)
		}
	}
}

func TestCloneEq(t *testing.T) {
	board := New(3, 3, 3)
	if !board.Eq(board) {
		t.Fatal("Failed basic equality")
	}
	// clone a clean board for later
	board3 := board.Clone()
	board.Apply(game.PlayerMove{game.Player(Black), game.Single(2)})
	board.Apply(game.PlayerMove{game.Player(White), game.Single(4)})

	board2 := board.Clone()
	if board2 == board {
		t.Errorf("Cloning should not yield the same address")
	}
	if &board.board[0] == &board2.(*Game).board[0] {
		t.Errorf("Cloning should not yield the same underlying backing")
	}
	if !board.Eq(board2) {
		t.Fatal("Cloning failed")
	}

	board.Reset()
	if !board.Eq(board3) {
		t.Fatalf("Reset board should be the same as newBoard\n%s\n%s", board, board3)
	}
}

func TestBoard_Format(t *testing.T) {
	g := New(7, 7, 3)
	it := game.MakeIterator(g.board, g.m, g.n)
	it[1][1] = White
	it[3][3] = Black
	it[1][5] = White
	it[5][5] = Black
	s := fmt.Sprintf("%s", g)
	t.Logf("\n%v", s)
}

func TestKomi_Ended(t *testing.T) {
	//
	// ⎢ O X X · X ⎥
	// ⎢ · X X X X ⎥
	// ⎢ X X · O X ⎥
	// ⎢ X · O · O ⎥
	// ⎢ · O · X · ⎥

	m, n := 5, 5
	board := []game.Colour{
		White, Black, Black, None, Black,
		None, Black, Black, Black, Black,
		Black, Black, None, White, Black,
		Black, None, White, None, White,
		None, White, None, Black, None,
	}
	g := New(m, n, 3)
	g.nextToMove = WhiteP
	data := g.board
	copy(data, board)

	ended, _ := g.Ended()
	if !ended {
		t.Error("Game is supposed to have ended")
	}

}

func TestCheck(t *testing.T) {
	// ⎢ X · X · X X O ⎥
	// ⎢ · · O X X O · ⎥
	// ⎢ O O · X X O O ⎥
	// ...

	m, n := 3, 7
	board := []game.Colour{
		Black, None, Black, None, Black, Black, White,
		None, None, White, Black, Black, White, None,
		White, White, None, Black, Black, White, White,
	}
	g := New(m, n, 3)
	g.nextToMove = WhiteP
	data := g.board
	copy(data, board)

	if g.Check(game.PlayerMove{Player: WhiteP, Single: game.Single(3)}) {
		t.Error("Expect to not be able to put a stone there")
	}

}
