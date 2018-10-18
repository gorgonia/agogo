package 围碁

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/gorgonia/agogo/game"
)

// Note this is a default implementation that has not yet been customised for our purposes
// GTPBoard will need to be replaced with AgogoBoard and have the associated GTP commands integrated with it
// Need to add func to agogo.go that calls StartGTP when new game is initialised
// Quoting chewxy, need a `GTPCapableState` struct that embeds `game.State` or `*wq.Game` and upon which we can Apply state changes
// Possibly split file out into gtp_wrapper where StartGTP lives and gtp_engine with the interface between agogo & GTP clients, similar to minigo

const (
	BORDER = -1
	EMPTY  = 0
	BLACK  = 1 // Don't change
	WHITE  = 2 // these now
)

type Point struct {
	X int
	Y int
}

type AgogoBoard struct {
	State game.State
}

type GTPBoard struct {
	State       [][]int
	Ko          Point
	Size        int
	Komi        float64
	NextPlayer  int
	CapsByBlack int
	CapsByWhite int
}

var known_commands = []string{
	"boardsize", "clear_board", "genmove", "known_command", "komi", "list_commands",
	"name", "play", "protocol_version", "quit", "showboard", "undo", "version",
}

var bw_strings = []string{"??", "Black", "White"} // Relies on BLACK == 1 and WHITE == 2

func NewGTPBoard(size int, komi float64) *GTPBoard {
	var board GTPBoard
	board.Size = size
	board.Komi = komi
	board.Clear()
	return &board
}

func (b *GTPBoard) Clear() {

	// GTPBoard arrays are 2D arrays of (size + 2) x (size + 2)
	// with explicit borders.

	b.State = make([][]int, b.Size+2)
	for i := 0; i < b.Size+2; i++ {
		b.State[i] = make([]int, b.Size+2)
	}
	for i := 0; i < b.Size+2; i++ {
		b.State[i][0] = BORDER
		b.State[i][b.Size+1] = BORDER
		b.State[0][i] = BORDER
		b.State[b.Size+1][i] = BORDER
	}
	b.Ko.X = -1
	b.Ko.Y = -1
	b.NextPlayer = BLACK
	b.CapsByBlack = 0
	b.CapsByWhite = 0
}

func (b *GTPBoard) Copy() *GTPBoard {
	newboard := NewGTPBoard(b.Size, b.Komi) // Does the borders for us, as well as size and komi
	for y := 1; y <= b.Size; y++ {
		for x := 1; x <= b.Size; x++ {
			newboard.State[x][y] = b.State[x][y]
		}
	}
	newboard.Ko.X = b.Ko.X
	newboard.Ko.Y = b.Ko.Y
	newboard.NextPlayer = b.NextPlayer
	newboard.CapsByBlack = b.CapsByBlack
	newboard.CapsByWhite = b.CapsByWhite
	return newboard
}

func (b *GTPBoard) String() string {
	s := "Current board:\n"
	for y := 1; y <= b.Size; y++ {
		for x := 1; x <= b.Size; x++ {
			c := '.'
			if b.Ko.X == x && b.Ko.Y == y {
				c = '*'
			}
			if b.State[x][y] == BLACK {
				c = 'X'
			} else if b.State[x][y] == WHITE {
				c = 'O'
			}
			s += fmt.Sprintf("%c ", c)
		}
		s += "\n"
	}
	s += fmt.Sprintf("Captures by Black: %d\n", b.CapsByBlack)
	s += fmt.Sprintf("Captures by White: %d\n", b.CapsByWhite)
	s += fmt.Sprintf("Komi: %.1f\n", b.Komi)
	s += fmt.Sprintf("Next: %s\n", bw_strings[b.NextPlayer])
	s += "\n"
	return s
}

func (b *GTPBoard) Dump() { // For debug only
	fmt.Printf(b.String())
}

func (b *GTPBoard) PlayMove(colour, x, y int) error {

	if colour != BLACK && colour != WHITE {
		return fmt.Errorf("colour neither black nor white")
	}

	var opponent_colour int

	if colour == BLACK {
		opponent_colour = WHITE
	} else {
		opponent_colour = BLACK
	}

	if x < 1 || x > b.Size || y < 1 || y > b.Size {
		return fmt.Errorf("coordinate off board")
	}

	if b.State[x][y] != EMPTY {
		return fmt.Errorf("coordinate not empty")
	}

	// Disallow playing on the ko square...

	if colour == b.NextPlayer && b.Ko.X == x && b.Ko.Y == y {
		return fmt.Errorf("illegal ko recapture")
	}

	// We must make the move here, AFTER the ko check...

	b.State[x][y] = colour

	// Normal captures...

	last_point_captured := Point{-1, -1} // If we captured exactly 1 stone, this will record it

	stones_destroyed := 0
	adj_points := AdjacentPoints(x, y)

	for _, point := range adj_points {
		if b.State[point.X][point.Y] == opponent_colour {
			if b.GroupHasLiberties(point.X, point.Y) == false {
				stones_destroyed += b.destroy_group(point.X, point.Y)
				last_point_captured = Point{point.X, point.Y}
			}
		}
	}

	// Disallow moves with no liberties (obviously after captures have been done)...

	if b.GroupHasLiberties(x, y) == false {
		b.State[x][y] = EMPTY
		return fmt.Errorf("move is suicidal")
	}

	// A square is a ko square if:
	//    - It was the site of the only stone captured this turn
	//    - The capturing stone has no friendly neighbours
	//    - The capturing stone has one liberty

	b.Ko.X = -1
	b.Ko.Y = -1

	if stones_destroyed == 1 {

		// Provisonally set the ko square to be the captured square...

		b.Ko.X = last_point_captured.X
		b.Ko.Y = last_point_captured.Y

		// But unset it if the capturing stone has any friendly neighbours or > 1 liberty

		liberties := 0
		friend_flag := false

		for _, point := range adj_points {
			if b.State[point.X][point.Y] == EMPTY {
				liberties += 1
			}
			if b.State[point.X][point.Y] == colour {
				friend_flag = true
				break
			}
		}

		if friend_flag || liberties > 1 {
			b.Ko.X = -1
			b.Ko.Y = -1
		}
	}

	// Update some board info...

	if colour == BLACK {
		b.NextPlayer = WHITE
		b.CapsByBlack += stones_destroyed
	} else {
		b.NextPlayer = BLACK
		b.CapsByWhite += stones_destroyed
	}

	return nil
}

func (b *GTPBoard) GroupHasLiberties(x, y int) bool {

	if x < 1 || y < 1 || x > b.Size || y > b.Size {
		panic("GroupHasLiberties() called with illegal x,y")
	}

	checked_stones := make(map[Point]bool)
	return b.group_has_liberties(x, y, checked_stones)
}

func (b *GTPBoard) group_has_liberties(x, y int, checked_stones map[Point]bool) bool {

	checked_stones[Point{x, y}] = true

	adj_points := AdjacentPoints(x, y)

	for _, adj := range adj_points {
		if b.State[adj.X][adj.Y] == EMPTY {
			return true
		}
	}

	for _, adj := range adj_points {
		if b.State[adj.X][adj.Y] == b.State[x][y] {
			if checked_stones[Point{adj.X, adj.Y}] == false {
				if b.group_has_liberties(adj.X, adj.Y, checked_stones) {
					return true
				}
			}
		}
	}

	return false
}

func (b *GTPBoard) destroy_group(x, y int) int {

	if x < 1 || y < 1 || x > b.Size || y > b.Size {
		panic("destroy_group() called with illegal x,y")
	}

	stones_destroyed := 1
	colour := b.State[x][y]
	b.State[x][y] = EMPTY

	for _, adj := range AdjacentPoints(x, y) {
		if b.State[adj.X][adj.Y] == colour {
			stones_destroyed += b.destroy_group(adj.X, adj.Y)
		}
	}

	return stones_destroyed
}

func (b *GTPBoard) Pass(colour int) error {

	if colour != BLACK && colour != WHITE {
		return fmt.Errorf("colour neither black nor white")
	}

	b.Ko.X = -1
	b.Ko.Y = -1
	if colour == BLACK {
		b.NextPlayer = WHITE
	} else {
		b.NextPlayer = BLACK
	}

	return nil
}

func (b *GTPBoard) NewFromMove(colour, x, y int) (*GTPBoard, error) {
	newboard := b.Copy()
	err := newboard.PlayMove(colour, x, y)
	if err != nil {
		return nil, err
	}
	return newboard, nil
}

func (b *GTPBoard) NewFromPass(colour int) (*GTPBoard, error) {
	newboard := b.Copy()
	err := newboard.Pass(colour)
	if err != nil {
		return nil, err
	}
	return newboard, nil
}

func (b *GTPBoard) AllLegalMoves(colour int) []Point {

	if colour != BLACK && colour != WHITE {
		return nil
	}

	var all_possible []Point

	for x := 1; x <= b.Size; x++ {

	Y_LOOP:
		for y := 1; y <= b.Size; y++ {

			if b.State[x][y] != EMPTY {
				continue
			}

			for _, point := range AdjacentPoints(x, y) {
				if b.State[point.X][point.Y] == EMPTY {
					all_possible = append(all_possible, Point{x, y}) // Move is clearly legal since some of its neighbours are empty
					continue Y_LOOP
				}
			}

			// The move we are playing will have no liberties of its own.
			// So check it by trying it. This is crude...

			_, err := b.NewFromMove(colour, x, y)
			if err == nil {
				all_possible = append(all_possible, Point{x, y})
			}
		}
	}

	return all_possible
}

func (b *GTPBoard) StringFromXY(x, y int) string {
	letter := 'A' + x - 1
	if letter >= 'I' {
		letter += 1
	}
	number := b.Size + 1 - y
	return fmt.Sprintf("%c%d", letter, number)
}

func (b *GTPBoard) StringFromPoint(p Point) string {
	return b.StringFromXY(p.X, p.Y)
}

func (b *GTPBoard) XYFromString(s string) (int, int, error) {

	if len(s) < 2 {
		return -1, -1, fmt.Errorf("coordinate string too short")
	}

	letter := strings.ToLower(s)[0]

	if letter < 'a' || letter > 'z' {
		return -1, -1, fmt.Errorf("letter part of coordinate not in range a-z")
	}

	if letter == 'i' {
		return -1, -1, fmt.Errorf("letter i not permitted")
	}

	x := int((letter - 'a') + 1)
	if letter > 'i' {
		x -= 1
	}

	tmp, err := strconv.Atoi(s[1:])
	if err != nil {
		return -1, -1, fmt.Errorf("couldn't parse number part of coordinate")
	}
	y := (b.Size + 1 - tmp)

	if x > b.Size || y > b.Size || x < 1 || y < 1 {
		return -1, -1, fmt.Errorf("coordinate off board")
	}

	return x, y, nil
}

func AdjacentPoints(x, y int) []Point {
	return []Point{Point{x - 1, y}, Point{x + 1, y}, Point{x, y - 1}, Point{x, y + 1}}
}

func StartGTP(genmove func(colour int, board *GTPBoard) string, name string, version string) {

	var history []*GTPBoard

	board := NewGTPBoard(19, 0.0)

	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		line := scanner.Text()
		line = strings.TrimSpace(line)
		line = strings.ToLower(line) // Note this lowercase conversion
		tokens := strings.Fields(line)

		if len(tokens) == 0 {
			continue
		}

		var id int = -1

		if unicode.IsDigit(rune(tokens[0][0])) {
			var err error
			id, err = strconv.Atoi(tokens[0])
			if err != nil {
				fmt.Printf("? Couldn't parse ID\n\n")
				continue
			}
			tokens = tokens[1:]
		}

		if len(tokens) == 0 {
			continue // This is GNU Go's behaviour when receiving just an ID
		}

		// So, by now, tokens is a list of the actual command; meanwhile id (if any) is saved
		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "name" {
			print_success(id, name)
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "version" {
			print_success(id, version)
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "protocol_version" {
			print_success(id, "2")
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "list_commands" {
			response := ""
			for _, command := range known_commands {
				response += command + "\n"
			}
			print_success(id, response)
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "known_command" {
			if len(tokens) < 2 {
				print_failure(id, "no argument received for known_command")
				continue
			}
			response := "false"
			for _, command := range known_commands {
				if command == tokens[1] {
					response = "true"
					break
				}
			}
			print_success(id, response)
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "komi" {
			if len(tokens) < 2 {
				print_failure(id, "no argument received for komi")
				continue
			}
			komi, err := strconv.ParseFloat(tokens[1], 64)
			if err != nil {
				print_failure(id, "couldn't parse komi float")
				continue
			}

			board.Komi = komi
			for i := 0; i < len(history); i++ { // Since komi is in the boards, change it through history
				history[i].Komi = komi
			}

			print_success(id, "")
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "clear_board" {
			board.Clear()
			history = nil
			print_success(id, "")
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "quit" {
			print_success(id, "")
			os.Exit(0)
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "showboard" {
			print_success(id, board.String())
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "boardsize" {
			if len(tokens) < 2 {
				print_failure(id, "no argument received for boardsize")
				continue
			}
			size, err := strconv.Atoi(tokens[1])
			if err != nil {
				print_failure(id, "couldn't parse boardsize int")
				continue
			}
			if size < 3 || size > 26 {
				print_failure(id, "boardsize not in range 3 - 26")
				continue
			}
			board = NewGTPBoard(size, board.Komi)
			history = nil
			print_success(id, "")
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "play" {

			if len(tokens) < 3 {
				print_failure(id, "insufficient arguments received for play")
				continue
			}

			if tokens[1] != "black" && tokens[1] != "b" && tokens[1] != "white" && tokens[1] != "w" {
				print_failure(id, "did not understand colour for play")
				continue
			}

			var colour int
			if tokens[1][0] == 'w' {
				colour = WHITE
			} else {
				colour = BLACK
			}

			if tokens[2] == "pass" {

				newboard, _ := board.NewFromPass(colour)
				history = append(history, board)
				board = newboard

			} else {

				x, y, err := board.XYFromString(tokens[2])
				if err != nil {
					print_failure(id, err.Error())
					continue
				}

				newboard, err := board.NewFromMove(colour, x, y)
				if err != nil {
					print_failure(id, err.Error())
					continue
				}

				history = append(history, board)
				board = newboard
			}

			print_success(id, "")
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "genmove" {

			if len(tokens) < 2 {
				print_failure(id, "no argument received for genmove")
				continue
			}

			if tokens[1] != "black" && tokens[1] != "b" && tokens[1] != "white" && tokens[1] != "w" {
				print_failure(id, "did not understand colour for genmove")
				continue
			}

			var colour int
			if tokens[1][0] == 'w' {
				colour = WHITE
			} else {
				colour = BLACK
			}

			s := genmove(colour, board.Copy()) // Send the engine a copy, not the real thing

			if s == "pass" {

				newboard, _ := board.NewFromPass(colour)
				history = append(history, board)
				board = newboard

			} else {

				x, y, err := board.XYFromString(s)
				if err != nil {
					print_failure(id, fmt.Sprintf("illegal move from engine: %s (%v)", s, err))
					continue
				}

				newboard, err := board.NewFromMove(colour, x, y)
				if err != nil {
					print_failure(id, fmt.Sprintf("illegal move from engine: %s (%v)", s, err))
					continue
				}

				history = append(history, board)
				board = newboard
			}

			print_success(id, s)
			continue
		}

		// --------------------------------------------------------------------------------------------------

		if tokens[0] == "undo" {

			if len(history) == 0 {
				print_failure(id, "cannot undo")
				continue
			} else {
				board = history[len(history)-1]
				history = history[0 : len(history)-1]
				print_success(id, "")
				continue
			}
		}

		// --------------------------------------------------------------------------------------------------

		print_failure(id, "unknown command")
	}
}

//func InitGTP(gamestate, game.State) GTPEncoding gtpencoding {
//  return
//}

func print_reply(id int, s string, shebang string) {
	s = strings.TrimSpace(s)
	fmt.Printf(shebang)
	if id != -1 {
		fmt.Printf("%d", id)
	}
	if s != "" {
		fmt.Printf(" %s\n\n", s)
	} else {
		fmt.Printf("\n\n")
	}
}

func print_success(id int, s string) {
	print_reply(id, s, "=")
}

func print_failure(id int, s string) {
	print_reply(id, s, "?")
}
