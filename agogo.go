package agogo

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	dual "github.com/gorgonia/agogo/dualnet"
	"github.com/gorgonia/agogo/game"
	"github.com/gorgonia/agogo/mcts"
	"github.com/pkg/errors"
	"gorgonia.org/tensor"
)

// AZ is the top level structure and the entry point of the API.
// It it a wrapper around the MTCS and the NeeuralNework that composes the algorithm.
// AZ stands for AlphaZero
type AZ struct {
	// state
	Arena
	Statistics
	useDummy bool

	// config
	nnConf          dual.Config
	mctsConf        mcts.Config
	enc             GameEncoder
	aug             Augmenter
	updateThreshold float32
	maxExamples     int

	// io
	outEnc OutputEncoder
}

// New AlphaZero structure. It takes a game state (implementing the board, rules, etc.)
// and a configuration to apply to the MCTS and the neural network
func New(g game.State, conf Config) *AZ {
	if !conf.NNConf.IsValid() {
		panic("NNConf is not valid. Unable to proceed")
	}
	if !conf.MCTSConf.IsValid() {
		panic("MCTSConf is not valid. Unable to proceed")
	}

	a := dual.New(conf.NNConf)
	b := dual.New(conf.NNConf)

	if err := a.Init(); err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	if err := b.Init(); err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	retVal := &AZ{
		Arena:           MakeArena(g, a, b, conf.MCTSConf, conf.Encoder, conf.Augmenter, conf.Name),
		nnConf:          conf.NNConf,
		mctsConf:        conf.MCTSConf,
		enc:             conf.Encoder,
		outEnc:          conf.OutputEncoder,
		aug:             conf.Augmenter,
		updateThreshold: float32(conf.UpdateThreshold),
		maxExamples:     conf.MaxExamples,
		Statistics:      makeStatistics(),
		useDummy:        true,
	}
	retVal.logger = log.New(&retVal.buf, "", log.Ltime)
	return retVal
}

func (a *AZ) setupSelfPlay(iter int) {
	var err error
	if err = a.A.SwitchToInference(a.game); err != nil {
		// DO SOMETHING WITH ERROR
	}
	if err = a.B.SwitchToInference(a.game); err != nil {
		// DO SOMETHING WITH ERROR
	}
	if iter == 0 && a.useDummy {
		log.Printf("Using Dummy")
		a.A.useDummy(a.game)
		a.B.useDummy(a.game)
	}
	log.Printf("Set up selfplay: Switch To inference for A. A.NN %p (%T)", a.A.NN, a.A.NN)
	log.Printf("Set up selfplay: Switch To inference for B. B.NN %p (%T)", a.B.NN, a.B.NN)
}

// SelfPlay plays an episode
func (a *AZ) SelfPlay() []Example {
	_, examples := a.Play(true, nil, a.aug) // don't encode images while selfplay... that'd be boring to watch
	a.game.Reset()
	return examples
}

// Learn learns for iters. It self-plays for episodes, and then trains a new NN from the self play example.
func (a *AZ) Learn(iters, episodes, nniters, arenaGames int) error {
	var err error
	for a.epoch = 0; a.epoch < iters; a.epoch++ {
		var ex []Example
		log.Printf("Self Play for epoch %d. Player A %p, Player B %p", a.epoch, a.A, a.B)

		a.buf.Reset()
		a.logger.Printf("Self Play for epoch %d. Player A %p, Player B %p", a.epoch, a.A, a.B)
		a.logger.SetPrefix("\t")
		a.setupSelfPlay(a.epoch)
		for e := 0; e < episodes; e++ {
			log.Printf("\tEpisode %v", e)
			a.logger.Printf("Episode %v\n", e)
			ex = append(ex, a.SelfPlay()...)
		}
		a.logger.SetPrefix("")
		a.buf.Reset()

		if a.maxExamples > 0 && len(ex) > a.maxExamples {
			shuffleExamples(ex)
			ex = ex[:a.maxExamples]
		}
		Xs, Policies, Values, batches := a.prepareExamples(ex)

		// // create a new DualNet for B
		// a.B.NN = dual.New(a.nnConf)
		// if err = a.B.NN.Dual().Init(); err != nil {
		// 	return errors.WithMessage(err, "Unable to create new DualNet for B")
		// }

		if err = dual.Train(a.B.NN, Xs, Policies, Values, batches, nniters); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("Train fail"))
		}

		a.B.SwitchToInference(a.game)

		a.A.resetStats()
		a.B.resetStats()

		a.logger.Printf("Playing Arena")
		a.logger.SetPrefix("\t")
		for a.gameNumber = 0; a.gameNumber < arenaGames; a.gameNumber++ {
			a.logger.Printf("Playing game number %d", a.gameNumber)
			a.Play(false, a.outEnc, nil)
			a.game.Reset()
		}
		a.logger.SetPrefix("")

		var killedA bool
		log.Printf("A wins %v, loss %v, draw %v\nB wins %v, loss %v, draw %v", a.A.Wins, a.A.Loss, a.A.Draw, a.B.Wins, a.B.Loss, a.B.Draw)

		// if a.B.Wins/(a.B.Wins+a.B.Loss+a.B.Draw) > a.updateThreshold {
		if a.B.Wins/(a.B.Wins+a.A.Wins) > a.updateThreshold {
			// B wins. Kill A, clean up its resources.
			log.Printf("Kill A %p. New A's NN is %p", a.A.NN, a.B.NN)
			if err = a.A.Close(); err != nil {
				return err
			}
			a.A.NN = a.B.NN
			// clear examples
			ex = ex[:0]
			killedA = true
		}
		a.update(a.A)
		if err = a.newB(a.nnConf, killedA); err != nil {
			return err
		}
	}
	return nil
}

// Save learning into filenamee
func (a *AZ) Save(filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0544)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	return enc.Encode(a.A.NN)
}

// Load the Alpha Zero structure from a filename
func (a *AZ) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	a.A.NN = dual.New(a.nnConf)
	a.B.NN = dual.New(a.nnConf)

	dec := gob.NewDecoder(f)
	if err = dec.Decode(a.A.NN); err != nil {
		return errors.WithStack(err)
	}

	f.Seek(0, 0)
	dec = gob.NewDecoder(f)
	if err = dec.Decode(a.B.NN); err != nil {
		return errors.WithStack(err)
	}
	a.useDummy = false
	return nil
}

func (a *AZ) prepareExamples(examples []Example) (Xs, Policies, Values *tensor.Dense, batches int) {
	shuffleExamples(examples)
	batches = len(examples) / a.nnConf.BatchSize
	total := batches * a.nnConf.BatchSize
	var XsBacking, PoliciesBacking, ValuesBacking []float32
	for i, ex := range examples {
		if i >= total {
			break
		}
		XsBacking = append(XsBacking, ex.Board...)

		start := len(PoliciesBacking)
		PoliciesBacking = append(PoliciesBacking, make([]float32, len(ex.Policy))...)
		copy(PoliciesBacking[start:], ex.Policy)

		ValuesBacking = append(ValuesBacking, ex.Value)
	}
	// padd out anythihng that is not full
	// board0 := examples[0].Board
	// policy0 := examples[0].Policy
	// rem := len(examples) % a.nnConf.BatchSize
	// if rem != 0 {
	// 	diff := a.nnConf.BatchSize - rem

	// 	// add padded data
	// 	XsBacking = append(XsBacking, make([]float32, diff*len(board0))...)
	// 	PoliciesBacking = append(PoliciesBacking, make([]float32, diff*len(policy0))...)
	// 	ValuesBacking = append(ValuesBacking, make([]float32, diff)...)
	// }
	// if rem > 0 {
	// 	batches++
	// }

	actionSpace := a.Arena.game.ActionSpace() + 1 // allow passes
	Xs = tensor.New(tensor.WithBacking(XsBacking), tensor.WithShape(a.nnConf.BatchSize*batches, a.nnConf.Features, a.nnConf.Height, a.nnConf.Width))
	Policies = tensor.New(tensor.WithBacking(PoliciesBacking), tensor.WithShape(a.nnConf.BatchSize*batches, actionSpace))
	Values = tensor.New(tensor.WithBacking(ValuesBacking), tensor.WithShape(a.nnConf.BatchSize*batches))
	return
}

func shuffleExamples(examples []Example) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range examples {
		j := r.Intn(i + 1)
		examples[i], examples[j] = examples[j], examples[i]
	}
}
