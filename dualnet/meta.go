package dual

import (
	"bytes"
	"log"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	G "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
	"gorgonia.org/tensor/native"
)

// Train is a basic trainer.
func Train(d *Dual, Xs, policies, values *tensor.Dense, batches, iterations int) error {
	var s slicer
	for i := 0; i < iterations; i++ {
		// var cost float32
		for bat := 0; bat < batches; bat++ {
			m := G.NewTapeMachine(d.g, G.BindDualValues(d.Model()...))
			model := G.NodesToValueGrads(d.Model())
			solver := G.NewVanillaSolver(G.WithLearnRate(0.1))
			batchStart := bat * d.Config.BatchSize
			batchEnd := batchStart + d.Config.BatchSize

			Xs2 := s.Slice(Xs, sli(batchStart, batchEnd))
			π := s.Slice(policies, sli(batchStart, batchEnd))
			v := s.Slice(values, sli(batchStart, batchEnd))

			G.Let(d.planes, Xs2)
			G.Let(d.Π, π)
			G.Let(d.V, v)
			if err := m.RunAll(); err != nil {
				return err
			}
			// cost = d.cost.Data().(float32)
			if err := solver.Step(model); err != nil {
				return err
			}
			//m.Reset()
			tensor.ReturnTensor(Xs2)
			tensor.ReturnTensor(π)
			tensor.ReturnTensor(v)
		}
		if err := shuffleBatch(Xs, policies, values); err != nil {
			return err
		}
		// TODO: add a channel to send training  cost data down
		// log.Printf("%d\t%v", i, cost/float32(batches))
	}
	return nil
}

// shuffleBatch shuffles the batches.
func shuffleBatch(Xs, π, v *tensor.Dense) (err error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	oriXs := Xs.Shape().Clone()
	oriPis := π.Shape().Clone()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v %v", Xs.Shape(), π.Shape())
			panic(r)
		}
	}()
	Xs.Reshape(as2D(Xs.Shape())...)
	π.Reshape(as2D(π.Shape())...)

	var matXs, matPis [][]float32
	if matXs, err = native.MatrixF32(Xs); err != nil {
		return errors.Wrapf(err, "shuffle batch failed - matX")
	}
	if matPis, err = native.MatrixF32(π); err != nil {
		return errors.Wrapf(err, "shuffle batch failed - pi")
	}
	vs := v.Data().([]float32)

	tmp := make([]float32, Xs.Shape()[1])
	for i := range matXs {
		j := r.Intn(i + 1)

		rowI := matXs[i]
		rowJ := matXs[j]
		copy(tmp, rowI)
		copy(rowI, rowJ)
		copy(rowJ, tmp)

		piI := matPis[i]
		piJ := matPis[j]
		copy(tmp, piI)
		copy(piI, piJ)
		copy(piJ, tmp)

		vs[i], vs[j] = vs[j], vs[i]
	}
	Xs.Reshape(oriXs...)
	π.Reshape(oriPis...)

	return nil
}

func as2D(s tensor.Shape) tensor.Shape {
	retVal := tensor.BorrowInts(2)
	retVal[0] = s[0]
	retVal[1] = s[1]
	for i := 2; i < len(s); i++ {
		retVal[1] *= s[i]
	}
	return retVal
}

// Inferencer is a struct that holds the state for a *Dual and a VM. By using an Inferece struct,
// there is no longer a need to create a VM every time an inference needs to be done.
type Inferencer struct {
	d *Dual
	m G.VM

	input *tensor.Dense
	buf   *bytes.Buffer
}

// Infer takes a trained *Dual, and creates a interence data structure such that it'd be easy to infer
func Infer(d *Dual, actionSpace int, toLog bool) (*Inferencer, error) {
	conf := d.Config
	conf.FwdOnly = true
	conf.BatchSize = actionSpace
	newShape := d.planes.Shape().Clone()
	newShape[0] = actionSpace
	retVal := &Inferencer{
		d:     New(conf),
		input: tensor.New(tensor.WithShape(newShape...), tensor.Of(Float)),
	}
	if err := retVal.d.Init(); err != nil {
		return nil, err
	}
	retVal.d.SetTesting()
	// G.WithInit(G.Zeroes())(retVal.d.planes)

	infModel := retVal.d.Model()
	for i, n := range d.Model() {
		original := n.Value().Data().([]float32)
		cloned := infModel[i].Value().Data().([]float32)
		copy(cloned, original)
	}

	retVal.buf = new(bytes.Buffer)
	if toLog {
		logger := log.New(retVal.buf, "", 0)
		retVal.m = G.NewTapeMachine(retVal.d.g,
			G.WithLogger(logger),
			G.WithWatchlist(),
			G.TraceExec(),
			G.WithValueFmt("%+1.1v"),
			G.WithNaNWatch(),
		)
	} else {
		retVal.m = G.NewTapeMachine(retVal.d.g)
	}
	return retVal, nil
}

// Dual implements Dualer
func (m *Inferencer) Dual() *Dual { return m.d }

// Infer takes the board, in form of a []float32, and runs inference, and returns the value
func (m *Inferencer) Infer(board []float32) (policy []float32, value float32, err error) {
	m.buf.Reset()
	for _, op := range m.d.ops {
		op.Reset()
	}

	// copy board to the provided preallocated input tensor
	m.input.Zero()
	data := m.input.Data().([]float32)
	copy(data, board)

	m.m.Reset()
	// log.Printf("Let planes %p be input %v", m.d.planes, board)
	m.buf.Reset()
	G.Let(m.d.planes, m.input)
	if err = m.m.RunAll(); err != nil {
		return nil, 0, err
	}
	policy = m.d.policyValue.Data().([]float32)
	value = m.d.value.Data().([]float32)[0]
	// log.Printf("\t%v", policy)
	return policy[:m.d.ActionSpace], value, nil
}

// ExecLog returns the execution log. If Infer was called with toLog = false, then it will return an empty string
func (m *Inferencer) ExecLog() string { return m.buf.String() }

// Close implements a closer, because well, a gorgonia VM is a resource.
func (m *Inferencer) Close() error { return m.m.Close() }
