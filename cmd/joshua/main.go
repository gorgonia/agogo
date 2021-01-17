package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorgonia/agogo"
	dual "github.com/gorgonia/agogo/dualnet"
	"github.com/gorgonia/agogo/game"
	"github.com/gorgonia/agogo/game/mnk"
	"github.com/gorgonia/agogo/mcts"

	"net/http"
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
		Name:            "Tic Tac Toe",
		NNConf:          dual.DefaultConf(3, 3, 10),
		MCTSConf:        mcts.DefaultConfig(3),
		UpdateThreshold: 0.52,
	}
	conf.NNConf.BatchSize = 100
	conf.NNConf.Features = 2 // write a better encoding of the board, and increase features (and that allows you to increase K as well)
	conf.NNConf.K = 3
	conf.NNConf.SharedLayers = 3
	conf.MCTSConf = mcts.Config{
		PUCT:           1.0,
		M:              3,
		N:              3,
		Timeout:        100 * time.Millisecond,
		PassPreference: mcts.DontPreferPass,
		Budget:         1000,
		DumbPass:       true,
		RandomCount:    0,
	}

	outEnc := NewEncoder()
	go func(h http.Handler) {
		mux := http.NewServeMux()
		mux.Handle("/ws", h)
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./htdocs"))))

		log.Println("http://localhost:8080")
		http.ListenAndServe(":8080", mux)
	}(outEnc)

	conf.Encoder = encodeBoard
	conf.OutputEncoder = outEnc

	g := mnk.TicTacToe()
	a := agogo.New(g, conf)
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("press ented when ready")
	reader.ReadString('\n')

	//a.Learn(5, 30, 200, 30) // 5 epochs, 50 episode, 100 NN iters, 100 games.
	a.Learn(5, 2, 100, 20) // 5 epochs, 50 episode, 100 NN iters, 100 games.
	a.Save("example.model")
}
