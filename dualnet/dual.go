package dual

import (
	"bytes"
	"encoding/gob"

	G "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

var Float = G.Float32

// Dual is the whole neural network architecture of the dual network.
//
// The policy and value outputs are shared
type Dual struct {
	Config
	ops []batchNormOp

	g    *G.ExprGraph
	Π, V *G.Node // pi and value labels. Pi is a matrix of 1s and 0s

	planes       *G.Node
	policyOutput *G.Node
	valueOutput  *G.Node

	policyValue G.Value // policy predicted
	value       G.Value // the actual value predicted
	cost        G.Value // cost, for training recoring
}

// New returns a new, uninitialized *Dual.
func New(conf Config) *Dual {
	retVal := &Dual{
		Config: conf,
	}

	return retVal
}

func (d *Dual) Init() error {
	d.reset()
	d.g = G.NewGraph()
	actionSpace := d.ActionSpace
	logits, valueOutput := d.fwd(actionSpace)
	return d.bwd(actionSpace, logits, valueOutput)

}

func (d *Dual) fwd(actionSpace int) (logits, valueOutput *G.Node) {
	boardSize := d.Width * d.Height

	// note, the data should be arranged like so:
	//	BatchSize, Features, Height, Width
	// because Gorgonia only supports doing convolutions on BCHW format
	d.planes = G.NewTensor(d.g, Float, 4, G.WithShape(d.BatchSize, d.Features, d.Height, d.Width), G.WithName("Planes"))

	var m maebe
	initialOut, initalOp := m.res(d.planes, d.K, "Init")
	d.ops = append(d.ops, initalOp)

	// shared stack
	sharedOut := initialOut
	for i := 0; i < d.SharedLayers; i++ {
		var op1, op2 batchNormOp
		sharedOut, op1, op2 = m.share(sharedOut, d.K, i)
		d.ops = append(d.ops, op1, op2)
	}

	// policy head
	var batches int
	policy, pop := m.batchnorm(m.conv(sharedOut, 2, 1, "PolicyHead"))
	policy = m.rectify(policy)
	if batches = policy.Shape().TotalSize() / (boardSize * 2); batches == 0 {
		batches = 1
	}
	policy = m.reshape(policy, tensor.Shape{batches, boardSize * 2})
	logits = m.linear(policy, actionSpace, "Policy")

	// Read to output which can be used for deciding the policy
	d.policyOutput = m.do(func() (*G.Node, error) { return G.SoftMax(logits) })
	G.Read(d.policyOutput, &d.policyValue)

	// value head
	value, vop := m.batchnorm(m.conv(sharedOut, 1, 1, "ValueHead"))
	value = m.rectify(value)
	batches = value.Shape().TotalSize() / boardSize
	value = m.reshape(value, tensor.Shape{batches, boardSize})
	value = m.linear(value, d.FC, "Value") // value hidden
	value = m.rectify(value)

	valueOutput = m.linear(value, 1, "ValueOutput")
	valueOutput = m.reshape(valueOutput, tensor.Shape{valueOutput.Shape().TotalSize()})

	// Read the output to a value
	d.valueOutput = m.do(func() (*G.Node, error) { return G.Tanh(valueOutput) })
	G.Read(d.valueOutput, &d.value)

	// add ops
	d.ops = append(d.ops, pop, vop)

	return logits, valueOutput
}

func (d *Dual) bwd(actionSpace int, logits, valueOutput *G.Node) error {
	if d.FwdOnly {
		return nil
	}
	d.Π = G.NewMatrix(d.g, Float, G.WithShape(d.BatchSize, actionSpace))
	d.V = G.NewVector(d.g, Float, G.WithShape(d.BatchSize))

	var m maebe
	// policy, value and combined costs
	var pcost, vcost, ccost *G.Node
	pcost = m.xent(logits, d.Π) // cross entropy, averaged.
	vcost = m.do(func() (*G.Node, error) { return G.Sub(valueOutput, d.V) })
	vcost = m.do(func() (*G.Node, error) { return G.Square(vcost) })
	vcost = m.do(func() (*G.Node, error) { return G.Mean(vcost) })

	// combined costs
	ccost = m.do(func() (*G.Node, error) { return G.Add(pcost, vcost) })
	if m.err != nil {
		return m.err
	}
	G.Read(ccost, &d.cost)

	if _, err := G.Grad(ccost, d.Model()...); err != nil {
		return err

	}
	return nil
}

func (d *Dual) Model() G.Nodes {
	retVal := make(G.Nodes, 0, d.g.Nodes().Len())
	for _, n := range d.g.AllNodes() {
		if n.IsVar() && n != d.planes && n != d.Π && n != d.V {
			retVal = append(retVal, n)
		}
	}
	return retVal
}

func (d *Dual) SetTesting() {
	for _, op := range d.ops {
		op.SetTesting()
	}
}

func (d *Dual) Clone() (*Dual, error) {
	d2 := New(d.Config)
	if err := d2.Init(); err != nil {
		return nil, err
	}

	model := d.Model()
	model2 := d2.Model()
	for i, n := range model {
		if err := G.Let(model2[i], n.Value()); err != nil {
			return nil, err
		}
	}

	return d2, nil
}

// Dual implemented Dualer
func (d *Dual) Dual() *Dual { return d }

func (d *Dual) reset() {
	d.ops = nil
	d.g = nil
	d.Π = nil
	d.V = nil

	d.planes = nil
	d.policyOutput = nil
}

func (d *Dual) GobEncode() (retVal []byte, err error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for _, n := range d.Model() {
		v := n.Value()
		if err = enc.Encode(&v); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (d *Dual) GobDecode(p []byte) error {
	d.reset()
	d.Init()

	buf := bytes.NewBuffer(p)
	dec := gob.NewDecoder(buf)
	for _, n := range d.Model() {
		var v G.Value
		if err := dec.Decode(&v); err != nil {
			return err
		}
		G.Let(n, v)
	}
	return nil
}
