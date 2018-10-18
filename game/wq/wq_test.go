package 围碁

import (
	"fmt"
	"testing"

	"github.com/gorgonia/agogo/game"
)

func sqrt(a int) int {
	if a == 0 || a == 1 {
		return a
	}
	start := 1
	end := a / 2
	var retVal int
	for start <= end {
		mid := (start + end) / 2
		sq := mid * mid
		if sq == a {
			return mid
		}
		if sq < a {
			start = mid + 1
			retVal = mid
		} else {
			end = mid - 1
		}
	}
	return retVal
}

var applyTests = []struct {
	board      []game.Colour
	move       game.PlayerMove
	board2     []game.Colour // nil if invalid
	taken      byte
	whiteScore float32
	blackScore float32
	willErr    bool
}{
	// placing on an empty
	{
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
		blackScore: 3, // TODO: CHECK
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
		whiteScore: 6, // TODO: CHECK
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
		whiteScore: 9, // TODO: CHECK
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
		blackScore: 4, // TODO: CHECK -  this should just be 2 unless my understanding of Go (the game) is wrong
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

func TestBoard_Apply(t *testing.T) {
	for testID, at := range applyTests {
		size := sqrt(len(at.board))
		board := newBoard(size)
		data := board.data
		copy(data, at.board)

		taken, err := board.Apply(at.move)

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
	board := newBoard(3)
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
	if &board.data[0] == &board2.data[0] {
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
	b := newBoard(7)
	b.it[1][1] = White
	b.it[3][3] = Black
	b.it[1][5] = White
	b.it[5][5] = Black
	s := fmt.Sprintf("%s", b)
	t.Logf("\n%v", s)
}
