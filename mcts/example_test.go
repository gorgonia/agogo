package mcts_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/gorgonia/agogo/game"
	"github.com/gorgonia/agogo/game/komi"
	"github.com/gorgonia/agogo/game/mnk"
	"github.com/gorgonia/agogo/mcts"
)

var (
	Pass = game.Single(-1)

	Cross  = game.Player(game.Black)
	Nought = game.Player(game.White)

	r = rand.New(rand.NewSource(1337))
)

func opponent(p game.Player) game.Player {
	switch p {
	case Cross:
		return Nought
	case Nought:
		return Cross
	}
	panic("Unreachable")
}

type dummyNN struct{}

func (dummyNN) Infer(state game.State) (policy []float32, value float32) {
	policy = make([]float32, 10) // 10 because last one is a pass
	switch state.MoveNumber() {
	case 0:
		policy[4] = 0.9
		value = 0.5
	case 1:
		policy[0] = 0.1 // switch colours remember?
		value = 0.5
	case 2:
		policy[2] = 0.9
		value = 8 / 9
	case 3:
		policy[6] = 0.1
		value = 8 / 9
	case 4:
		policy[3] = 0.9
		value = 8 / 9
	case 5:
		policy[5] = 0.1
		value = 0.5
	case 6:
		policy[1] = 0.9
		value = 8 / 9
	case 7:
		policy[7] = 0.1
		value = 0
	case 8:
		policy[8] = 0.9
		value = 0
	}
	return
}

func Example() {
	g := mnk.TicTacToe()
	conf := mcts.Config{
		PUCT:           1.0,
		M:              3,
		N:              3,
		Timeout:        500 * time.Millisecond,
		PassPreference: mcts.DontPreferPass,
		Budget:         10000,
		DumbPass:       true,
		RandomCount:    0, // this is a deterministic example
	}
	nn := dummyNN{}
	t := mcts.New(g, conf, nn)
	player := Cross

	var buf bytes.Buffer
	var ended bool
	var winner game.Player
	for ended, winner = g.Ended(); !ended; ended, winner = g.Ended() {
		moveNum := g.MoveNumber()
		best := t.Search(player)
		g = g.Apply(game.PlayerMove{player, best}).(*mnk.MNK)
		fmt.Fprintf(&buf, "Turn %d\n%v---\n", moveNum, g)
		if moveNum == 2 {
			ioutil.WriteFile("fullGraph_tictactoe.dot", []byte(t.ToDot()), 0644)
		}
		player = opponent(player)
	}

	log.Printf("Playout:\n%v", buf.String())
	fmt.Printf("WINNER %v\n", winner)

	// the outputs should look something like this (may dfiffer due to random numbers)
	// Turn 0
	// ⎢ · · · ⎥
	// ⎢ · X · ⎥
	// ⎢ · · · ⎥
	// ---
	// Turn 1
	// ⎢ O · · ⎥
	// ⎢ · X · ⎥
	// ⎢ · · · ⎥
	// ---
	// Turn 2
	// ⎢ O · X ⎥
	// ⎢ · X · ⎥
	// ⎢ · · · ⎥
	// ---
	// Turn 3
	// ⎢ O · X ⎥
	// ⎢ · X · ⎥
	// ⎢ O · · ⎥
	// ---
	// Turn 4
	// ⎢ O · X ⎥
	// ⎢ X X · ⎥
	// ⎢ O · · ⎥
	// ---
	// Turn 5
	// ⎢ O · X ⎥
	// ⎢ X X O ⎥
	// ⎢ O · · ⎥
	// ---
	// Turn 6
	// ⎢ O X X ⎥
	// ⎢ X X O ⎥
	// ⎢ O · · ⎥
	// ---
	// Turn 7
	// ⎢ O X X ⎥
	// ⎢ X X O ⎥
	// ⎢ O O · ⎥
	// ---
	// Turn 8
	// ⎢ O X X ⎥
	// ⎢ X X O ⎥
	// ⎢ O O X ⎥
	// ---

	// Output:
	// WINNER None
}

type dummyNN2 struct{}

func (dummyNN2) Infer(state game.State) (policy []float32, value float32) {
	policy = make([]float32, 25)
	for i := range policy {
		policy[i] = 1 / 25.0
	}
	return policy, 1 / 25.0
}

func Example_Komi() {
	g := komi.New(5, 5, 3)
	conf := mcts.Config{
		PUCT:           1.0,
		M:              5,
		N:              5,
		Timeout:        500 * time.Millisecond,
		PassPreference: mcts.DontPreferPass,
		Budget:         100,
		DumbPass:       true,
		RandomCount:    0, // this is a deterministic example
	}

	nn := dummyNN2{}
	t := mcts.New(g, conf, nn)
	player := game.Player(game.White)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			// sig is a ^C, handle it
			log.Println(t.Log())
			// ioutil.WriteFile("fullGraph.dot", []byte(t.ToDot()), 0644)
			os.Exit(1)
		}
	}()

	var buf bytes.Buffer
	var ended bool
	var winner game.Player
	for ended, winner = g.Ended(); !ended; ended, winner = g.Ended() {
		moveNum := g.MoveNumber()
		player = opponent(player)
		best := t.Search(player)
		g = g.Apply(game.PlayerMove{player, best}).(*komi.Game)
		fmt.Fprintf(&buf, "Turn %d\n%v---\n", moveNum, g)
	}

	log.Printf("Playout:\n%v", buf.String())
	fmt.Printf("WINNER %v\n", winner)

	// Output:
	// 0
}
