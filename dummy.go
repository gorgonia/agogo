package agogo

import "github.com/gorgonia/agogo/game"

type dummyInferer struct {
	outputSize    int
	currentPlayer game.Player
}

func (d dummyInferer) Infer(a []float32) (policy []float32, value float32, err error) {
	switch d.currentPlayer {
	case 1:
		value = 1
	case 2:
		value = -1
	}
	policy = make([]float32, d.outputSize)
	for i := range policy {
		policy[i] = 1 / float32(d.outputSize)
	}

	return policy, value, nil
}

func (d dummyInferer) Close() error { return nil }
