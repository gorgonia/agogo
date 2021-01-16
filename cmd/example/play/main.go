package main

import (
	"fmt"

	"github.com/gorgonia/agogo"
	dual "github.com/gorgonia/agogo/dualnet"
	"github.com/gorgonia/agogo/game"
	"github.com/gorgonia/agogo/game/mnk"
	"github.com/gorgonia/agogo/mcts"

	_ "net/http/pprof"
)

func encodeBoard(a game.State) []float32 {
	board := agogo.EncodeTwoPlayerBoard(a.Board(), nil)
	for i := range board {
		if board[i] == 0 {
			board[i] = 0.001
		}
	}
	playerLayer := make([]float32, len(a.Board()))
	next := a.ToMove()
	if next == game.Player(game.Black) {
		for i := range playerLayer {
			playerLayer[i] = 1
		}
	} else if next == game.Player(game.White) {
		// vecf32.Scale(board, -1)
		for i := range playerLayer {
			playerLayer[i] = -1
		}
	}
	retVal := append(board, playerLayer...)
	return retVal
}

func main() {
	conf := agogo.Config{
		Name:     "Tic Tac Toe",
		NNConf:   dual.DefaultConf(3, 3, 10),
		MCTSConf: mcts.DefaultConfig(3),
	}
	conf.Encoder = encodeBoard

	g := mnk.TicTacToe()
	a := agogo.New(g, conf)
	a.Load("example.model")
	a.A.Player = mnk.Cross
	a.B.Player = mnk.Nought
	a.B.SwitchToInference(g)
	a.A.SwitchToInference(g)
	// Put x int the center
	stateAfterFirstPlay := g.Apply(game.PlayerMove{
		Player: mnk.Cross,
		Single: 4,
	})
	fmt.Println(stateAfterFirstPlay)
	// ⎢ · · · ⎥
	// ⎢ · X · ⎥
	// ⎢ · · · ⎥

	// What to do next
	move := a.B.Search(stateAfterFirstPlay)
	fmt.Println(move)
	// 1
	g.Apply(game.PlayerMove{
		Player: mnk.Nought,
		Single: move,
	})
	fmt.Println(stateAfterFirstPlay)
	// ⎢ · O · ⎥
	// ⎢ · X · ⎥
	// ⎢ · · · ⎥
}
