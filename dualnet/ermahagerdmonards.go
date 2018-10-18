package dual

import (
	"fmt"

	"github.com/pkg/errors"
	G "gorgonia.org/gorgonia"
	"gorgonia.org/gorgonia/ops/nn"
	"gorgonia.org/tensor"
)

type maebe struct {
	err error
}

type batchNormOp interface {
	SetTraining()
	SetTesting()
	Reset() error
}

// generic monad... may be useful
func (m *maebe) do(f func() (*G.Node, error)) (retVal *G.Node) {
	if m.err != nil {
		return nil
	}
	if retVal, m.err = f(); m.err != nil {
		m.err = errors.WithStack(m.err)
	}
	return
}

func (m *maebe) conv(input *G.Node, filterCount, size int, name string) (retVal *G.Node) {
	if m.err != nil {
		return nil
	}
	featureCount := input.Shape()[1]
	padding := findPadding(input.Shape()[2], input.Shape()[3], size, size)
	filter := G.NewTensor(input.Graph(), Float, 4, G.WithShape(filterCount, featureCount, size, size), G.WithName("Filter"+name), G.WithInit(G.GlorotU(1.0)))

	// assume well behaved images
	if retVal, m.err = nnops.Conv2d(input, filter, []int{size, size}, padding, []int{1, 1}, []int{1, 1}); m.err != nil {
		m.err = errors.WithStack(m.err)
	}
	return
}

func (m *maebe) batchnorm(input *G.Node) (retVal *G.Node, retOp batchNormOp) {
	if m.err != nil {
		return nil, nil
	}
	// note: the scale and biases will still be created
	// and they will still be backpropagated
	if retVal, _, _, retOp, m.err = nnops.BatchNorm(input, nil, nil, 0.997, 1e-5); m.err != nil {
		m.err = errors.WithStack(m.err)
	}
	return
}

func (m *maebe) res(input *G.Node, filterCount int, name string) (*G.Node, batchNormOp) {
	convolved := m.conv(input, filterCount, 3, name)
	normalized, op := m.batchnorm(convolved)
	retVal := m.rectify(normalized)
	return retVal, op
}

func (m *maebe) share(input *G.Node, filterCount, layer int) (*G.Node, batchNormOp, batchNormOp) {
	layer1, l1Op := m.res(input, filterCount, fmt.Sprintf("Layer1 of Shared Layer %d", layer))
	layer2, l2Op := m.res(input, filterCount, fmt.Sprintf("Layer2 of Shared Layer %d", layer))
	added := m.do(func() (*G.Node, error) { return G.Add(layer1, layer2) })
	retVal := m.rectify(added)
	return retVal, l1Op, l2Op
}

func (m *maebe) linear(input *G.Node, units int, name string) *G.Node {
	if m.err != nil {
		return nil
	}
	// figure out size
	w := G.NewTensor(input.Graph(), Float, 2, G.WithShape(input.Shape()[1], units), G.WithInit(G.GlorotN(1.0)), G.WithName(name+"_w"))
	xw := m.do(func() (*G.Node, error) { return G.Mul(input, w) })
	b := G.NewTensor(xw.Graph(), Float, xw.Shape().Dims(), G.WithShape(xw.Shape().Clone()...), G.WithName(name+"_b"), G.WithInit(G.Zeroes()))
	return m.do(func() (*G.Node, error) { return G.Add(xw, b) })
}

func (m *maebe) rectify(input *G.Node) (retVal *G.Node) {
	if m.err != nil {
		return nil
	}
	if retVal, m.err = nnops.Rectify(input); m.err != nil {
		m.err = errors.WithStack(m.err)
	}
	return
}

func (m *maebe) reshape(input *G.Node, to tensor.Shape) (retVal *G.Node) {
	if m.err != nil {
		return nil
	}
	if retVal, m.err = G.Reshape(input, to); m.err != nil {
		m.err = errors.WithStack(m.err)
	}
	return
}

func (m *maebe) xent(output, target *G.Node) (retVal *G.Node) {
	var one *G.Node
	switch Float {
	case G.Float32:
		one = G.NewConstant(float32(1))
	case G.Float64:
		one = G.NewConstant(float64(1))
	}
	var omy, omout *G.Node
	if omy, m.err = G.Sub(one, target); m.err != nil {
		m.err = errors.WithStack(m.err)
		return nil
	}

	if omout, m.err = G.Sub(one, output); m.err != nil {
		m.err = errors.WithStack(m.err)
		return nil
	}

	var fst, snd *G.Node
	if fst, m.err = G.HadamardProd(target, output); m.err != nil {
		m.err = errors.WithStack(m.err)
		return nil
	}
	if snd, m.err = G.HadamardProd(omy, omout); m.err != nil {
		m.err = errors.WithStack(m.err)
		return nil
	}

	if retVal, m.err = G.Add(fst, snd); m.err != nil {
		m.err = errors.WithStack(m.err)
		return nil
	}
	if retVal, m.err = G.Neg(retVal); m.err != nil {
		m.err = errors.WithStack(m.err)
		return nil
	}
	if retVal, m.err = G.Mean(retVal); m.err != nil {
		m.err = errors.WithStack(m.err)
	}
	return
}

func findPadding(inputX, inputY, kernelX, kernelY int) []int {
	return []int{
		(inputX - 1 - inputX + kernelX) / 2,
		(inputY - 1 - inputY + kernelY) / 2,
	}
}
