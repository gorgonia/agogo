package agogo

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/chewxy/math32"
	dual "github.com/gorgonia/agogo/dualnet"
	"github.com/gorgonia/agogo/game"
	"github.com/gorgonia/agogo/mcts"
)

type Arena struct {
	r    *rand.Rand
	game game.State
	A, B *Agent

	// state
	currentPlayer *Agent
	conf          mcts.Config
	buf           bytes.Buffer
	logger        *log.Logger

	// only relevant to training
	name       string
	epoch      int // training epoch
	gameNumber int // which game is this in

	// when to screw it all and just reinit a new NN
	oldThresh int
	oldCount  int
}

// MakeArena makes an arena given a game.
func MakeArena(g game.State, a, b Dualer, conf mcts.Config, enc GameEncoder, aug Augmenter, name string) Arena {
	A := &Agent{
		NN:   a.Dual(),
		Enc:  enc,
		name: "A",
	}
	A.MCTS = mcts.New(g, conf, A)
	B := &Agent{
		NN:   b.Dual(),
		Enc:  enc,
		name: "B",
	}
	B.MCTS = mcts.New(g, conf, B)

	if name == "" {
		name = "UNKNOWN GAME"
	}

	return Arena{
		r:    rand.New(rand.NewSource(time.Now().UnixNano())),
		game: g,
		A:    A,
		B:    B,
		conf: conf,
		name: name,

		oldThresh: 10,
	}
}

func NewArena(g game.State, a, b Dualer, conf mcts.Config, enc GameEncoder, aug Augmenter, name string) *Arena {
	ar := MakeArena(g, a, b, conf, enc, aug, name)
	ar.logger = log.New(&ar.buf, "", log.Ltime)
	return &ar
}

// Play plays a game, and retrns a winner. If it is a draw, the returned colour is None.
func (a *Arena) Play(record bool, enc OutputEncoder, aug Augmenter) (winner game.Player, examples []Example) {
	if a.r.Intn(2) == 0 {
		a.A.Player = game.Player(game.Black)
		a.B.Player = game.Player(game.White)
		a.currentPlayer = a.A
	} else {
		a.A.Player = game.Player(game.White)
		a.B.Player = game.Player(game.Black)
		a.currentPlayer = a.B
	}

	a.game.SetToMove(a.currentPlayer.Player)
	a.logger.Printf("Playing. Recording %t\n", record)
	a.logger.SetPrefix("\t\t")
	var ended bool
	var passCount int
	for ended, winner = a.game.Ended(); !ended; ended, winner = a.game.Ended() {

		best := a.currentPlayer.Search(a.game)
		if best.IsPass() {
			passCount++
		} else {
			passCount = 0
		}
		a.logger.Printf("Current Player: %v. Best Move %v\n", a.currentPlayer.Player, best)
		if record {
			boards := a.currentPlayer.Enc(a.game)
			policies := a.currentPlayer.MCTS.Policies(a.game)
			ex := Example{
				Board:  boards,
				Policy: policies,
				// THIS IS A HACK.
				// The value is 1 or -1 depending on player colour, but for now we store the player colour
				Value: float32(a.currentPlayer.Player),
			}
			if validPolicies(policies) {
				if aug != nil {
					examples = append(examples, aug(ex)...)
				} else {
					examples = append(examples, ex)
				}
			}

		}

		// policy, value := a.currentPlayer.Infer(a.game)
		// log.Printf("\t\tPlayer %v made Move %v | %1.1v %1.1v", a.currentPlayer.Player, best, policy, value)
		a.game = a.game.Apply(game.PlayerMove{a.currentPlayer.Player, best})
		a.switchPlayer()
		if enc != nil {
			enc.Encode(a)
		}
		if passCount >= 2 {
			break
		}
	}
	a.logger.SetPrefix("\t")
	a.A.MCTS.Reset()
	a.B.MCTS.Reset()
	if enc != nil {
		log.Printf("\tDone playing")
	}

	for i := range examples {
		switch {
		case winner == game.Player(game.None):
			examples[i].Value = 0
		case examples[i].Value == float32(winner):
			examples[i].Value = 1
		default:
			examples[i].Value = -1
		}
	}
	var winningAgent *Agent
	switch {
	case winner == game.Player(game.None):
		a.A.Draw++
		a.B.Draw++
	case winner == a.A.Player:
		a.A.Wins++
		a.B.Loss++
		winningAgent = a.A
	case winner == a.B.Player:
		a.B.Wins++
		a.A.Loss++
		winningAgent = a.B
	}
	if !record {
		log.Printf("Winner %v | %p", winner, winningAgent)
	}
	// a.A.MCTS.Reset()
	// a.B.MCTS.Reset()
	a.A.MCTS = mcts.New(a.game, a.conf, a.A)
	a.B.MCTS = mcts.New(a.game, a.conf, a.B)
	runtime.GC()
	return game.Player(game.None), examples
}

func (a *Arena) Epoch() int                  { return a.epoch }
func (a *Arena) GameNumber() int             { return a.gameNumber }
func (a *Arena) Name() string                { return a.name }
func (a *Arena) Score(p game.Player) float64 { return float64(a.game.Score(p)) }
func (a *Arena) State() game.State           { return a.game }

func (a *Arena) Log(w io.Writer) {
	fmt.Fprintf(w, a.buf.String())
	fmt.Fprintln(w, "\nA:\n")
	fmt.Fprintln(w, a.A.MCTS.Log())
	fmt.Fprintln(w, "\nB:\n")
	fmt.Fprintln(w, a.B.MCTS.Log())
}

func (a *Arena) newB(conf dual.Config, killedA bool) (err error) {
	if killedA {
		a.oldCount = 0
	}

	// if a.oldCount >= a.oldThresh {
	// 	a.B.NN = dual.New(conf)
	// 	err = a.B.NN.Init()
	// 	a.oldCount = 0
	// } else {
	// 	a.B.NN, err = a.B.NN.Clone()
	// }

	a.B.NN = dual.New(conf)
	err = a.B.NN.Init()

	a.oldCount++
	log.Printf("NewB NN %p", a.B.NN)
	return err
}

func (a *Arena) switchPlayer() {
	switch a.currentPlayer {
	case a.A:
		a.currentPlayer = a.B
	case a.B:
		a.currentPlayer = a.A
	}
}

func cloneBoard(a []game.Colour) []game.Colour {
	retVal := make([]game.Colour, len(a))
	copy(retVal, a)
	return retVal
}

func validPolicies(policy []float32) bool {
	for _, v := range policy {
		if math32.IsInf(v, 0) {
			return false
		}
		if math32.IsNaN(v) {
			return false
		}
	}
	return true
}
