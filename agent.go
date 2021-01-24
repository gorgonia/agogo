package agogo

import (
	"log"
	"runtime"
	"sync"

	dual "github.com/gorgonia/agogo/dualnet"
	"github.com/gorgonia/agogo/game"
	"github.com/gorgonia/agogo/mcts"
)

// An Agent is a player, AI or Human
type Agent struct {
	NN     *dual.Dual
	MCTS   *mcts.MCTS
	Player game.Player
	Enc    GameEncoder

	// Statistics
	Wins float32
	Loss float32
	Draw float32
	sync.Mutex

	name     string
	actions  int
	inferer  chan Inferer
	err      error
	inferers []Inferer
}

func newAgent(a Dualer) *Agent {
	retVal := &Agent{
		NN:       a.Dual(),
		inferers: make([]Inferer, 0),
	}
	return retVal
}

// SwitchToInference uses the inference mode neural network.
func (a *Agent) SwitchToInference(g game.State) (err error) {
	a.Lock()
	a.inferer = make(chan Inferer, numCPU)

	for i := 0; i < numCPU; i++ {
		var inf Inferer
		if inf, err = dual.Infer(a.NN, g.ActionSpace(), false); err != nil {
			return err
		}
		a.inferers = append(a.inferers, inf)
		a.inferer <- inf
	}
	// a.NN = nil // remove old NN
	a.Unlock()
	return nil
}

// Infer infers a bunch of moves based on the game state. This is mainly used to implement a Inferer such that the MCTS search can use it.
func (a *Agent) Infer(g game.State) (policy []float32, value float32) {
	input := a.Enc(g)
	inf := <-a.inferer

	var err error
	policy, value, err = inf.Infer(input)
	if err != nil {
		if el, ok := inf.(ExecLogger); ok {
			log.Println(el.ExecLog())
		}
		panic(err)
	}
	a.inferer <- inf
	return
}

// Search searches the game state and returns a suggested coordinate.
func (a *Agent) Search(g game.State) game.Single {
	a.MCTS.SetGame(g)
	return a.MCTS.Search(a.Player)
}

// NNOutput returns the output of the neural network
func (a *Agent) NNOutput(g game.State) (policy []float32, value float32, err error) {
	input := a.Enc(g)
	inf := <-a.inferer
	policy, value, err = inf.Infer(input)
	a.inferer <- inf
	return
}

func (a *Agent) Close() error {
	close(a.inferer)
	var allErrs manyErr
	for _, inferer := range a.inferers {
		if err := inferer.Close(); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	if len(allErrs) > 0 {
		return allErrs
	}
	return nil
}

func (a *Agent) useDummy(g game.State) {
	a.inferer = make(chan Inferer, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		a.inferer <- dummyInferer{
			outputSize:    g.ActionSpace(),
			currentPlayer: a.Player,
		}
	}
}

func (a *Agent) resetStats() {
	a.Lock()
	a.Wins = 0
	a.Loss = 0
	a.Draw = 0
	a.Unlock()
}
