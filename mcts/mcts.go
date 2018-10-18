package mcts

import "github.com/gorgonia/agogo/game"

// type GameState interface {
// 	game.State

// 	// Go specific
// 	Komi() float32
// 	SuperKo() bool
// 	IsEye(m game.PlayerMove) bool
// }

// Inferencer is essentially the neural network
type Inferencer interface {
	Infer(state game.State) (policy []float32, value float32)
	// Log() string // in debug mode, the Log() method should return the neural network log
}

const (
	Pass   game.Single = -1
	Resign game.Single = -2

	White game.Player = game.Player(game.White)
	Black game.Player = game.Player(game.Black)

	virtualLoss1       = 0x40400000 // 3 in float32
	defaultMinPsaRatio = 0x40000000 // 2 in float32
)

// PassPreference
type PassPreference int

const (
	DontPreferPass PassPreference = iota
	PreferPass
	DontResign
	MAXPASSPREFERENCE
)

func init() {
	if !Pass.IsPass() {
		panic("Pass has  to be Pass")
	}

	if !Resign.IsResignation() {
		panic("Resign  has to be Resign")
	}
}
