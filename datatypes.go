package agogo

import (
	"io"

	dual "github.com/gorgonia/agogo/dualnet"
	"github.com/gorgonia/agogo/game"
	"github.com/gorgonia/agogo/mcts"
)

type Config struct {
	Name            string
	NNConf          dual.Config
	MCTSConf        mcts.Config
	UpdateThreshold float64
	MaxExamples     int // maximum number of examples

	// extensions
	Encoder       GameEncoder
	OutputEncoder OutputEncoder
	Augmenter     Augmenter
}

// GameEncoder encodes a game state as a slice of floats
type GameEncoder func(a game.State) []float32

// OutputEncoder encodes the entire meta state as whatever.
//
// An example OutputEncoder is the GifEncoder. Another example would be a logger.
type OutputEncoder interface {
	Encode(ms game.MetaState) error
	Flush() error
}

// Augmenter takes an example, and creates more examples from it.
type Augmenter func(a Example) []Example

// Example is a representation of an example.
type Example struct {
	Board  []float32
	Policy []float32
	Value  float32
}

// Dualer is an interface for anything that allows getting out a *Dual.
//
// Its sole purpose is to form a monoid-ish data structure for Agent.NN
type Dualer interface {
	Dual() *dual.Dual
}

// Inferer is anything that can infer given an input.
type Inferer interface {
	Infer(a []float32) (policy []float32, value float32, err error)
	io.Closer
}

// ExecLogger is anything that can return the execution log.
type ExecLogger interface {
	ExecLog() string
}
