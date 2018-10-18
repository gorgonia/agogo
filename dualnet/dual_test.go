package dual

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	G "gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func TestSanity(t *testing.T) {
	// defer leaktest.Check(t)()
	conf := DefaultConf(11, 11, 11*11+1)
	conf.BatchSize = 32

	d := &Dual{Config: conf}
	if err := d.Init(); err != nil {
		t.Fatalf("%+v", err)
	}
	t.Logf("Number of nodes: %d", len(d.g.AllNodes()))
	prog, _, err := G.Compile(d.g)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Requires %d bytes", prog.CPUMemReq())
	// log.Printf("%v", prog)

	//	ioutil.WriteFile("g.dot", []byte(d.g.ToDot()), 0644)
	var buf bytes.Buffer
	// logger := log.New(&buf, "", 0)

	//	m := G.NewTapeMachine(d.g, G.BindDualValues(d.Model()...), G.WithLogger(logger), G.WithWatchlist(), G.WithValueFmt("%+1.1f"))

	m := G.NewTapeMachine(d.g, G.BindDualValues(d.Model()...))
	f := tensor.New(tensor.WithShape(d.planes.Shape()...), tensor.WithBacking(tensor.Random(Float, d.planes.Shape().TotalSize())))
	π := tensor.New(tensor.WithShape(d.Π.Shape()...), tensor.WithBacking(tensor.Random(Float, d.Π.Shape().TotalSize())))
	v := tensor.New(tensor.WithShape(d.V.Shape()...), tensor.WithBacking(tensor.Random(Float, d.V.Shape().TotalSize())))
	G.Let(d.planes, f)
	G.Let(d.Π, π)
	G.Let(d.V, v)

	model := G.NodesToValueGrads(d.Model())
	solver := G.NewVanillaSolver(G.WithBatchSize(float64(conf.BatchSize)), G.WithLearnRate(0.1))
	costFile, _ := os.OpenFile("cost.csv", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)

	defer costFile.Close()
	for i := 0; i < 250; i++ {
		start := time.Now()
		if i > 0 {
			buf.Reset()
		}
		fmt.Fprintf(&buf, "ITERATION %d\n", i)
		if err := m.RunAll(); err != nil {
			t.Fatal(err)
		}
		fwd := time.Now()
		solver.Step(model)
		step := time.Now()
		fmt.Fprintf(costFile, "%v, %v, %v, %v\n", d.cost, fwd.Sub(start), step.Sub(fwd), time.Since(start))
		// fmt.Fprintf(costFile, "%v, %v, %v, %v\n", d.cost, 0.0, 0.0, 0.0)
		m.Reset()
		shuffleBatch(f, π, v)
		time.Sleep(1)
	}
	// costFile.WriteString("Cost, Fowward Time, SGD Time, Total Time")
	// costFile.Write(buf.Bytes())
	if err := m.Close(); err != nil {
		t.Fatalf("closing machine: %v", err)
	}

	if t.Failed() {
		fmt.Printf("%v", buf.String())
	}
	runtime.GC()
}

func TestInferenceSanity(t *testing.T) {
	// defer leaktest.Check(t)()
	boardSize := 3
	conf := DefaultConf(boardSize, boardSize, boardSize*boardSize+1)
	conf.BatchSize = 32
	d := &Dual{Config: conf}
	if err := d.Init(); err != nil {
		t.Fatalf("%+v", err)
	}
	inferer, err := Infer(d, boardSize*boardSize+1, false)
	if err != nil {
		t.Fatal(err)
	}
	defer inferer.Close()

	policy, value, err := inferer.Infer([]float32{
		-1, 0, 1,
		-1, 1, 0,
		0, 0, 0})

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("POLICY %v | %v", policy, value)
	runtime.GC()
}

func TestEncodeDecode(t *testing.T) {
	assert := assert.New(t)
	boardSize := 3
	conf := DefaultConf(boardSize, boardSize, boardSize*boardSize+1)
	conf.BatchSize = 32
	d := &Dual{Config: conf}
	if err := d.Init(); err != nil {
		t.Fatalf("%+v", err)
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(d); err != nil {
		t.Fatalf("Encoding Failure %v", err)
	}

	dec := gob.NewDecoder(&buf)
	d2 := &Dual{Config: conf}
	if err := dec.Decode(d2); err != nil {
		t.Fatalf("Decoding Failure %v", err)
	}

	dmodel := d.Model()
	d2model := d2.Model()

	for i, n := range dmodel {
		fstVal := n.Value()
		sndVal := d2model[i].Value()
		assert.Equal(fstVal.Data(), sndVal.Data(), "%d - %v vs %v should have the same data", i, dmodel[i], d2model[i])
	}
}

func TestInferencer_ExecLog(t *testing.T) {
	boardSize := 3
	conf := DefaultConf(boardSize, boardSize, boardSize*boardSize+1)
	conf.BatchSize = 32
	d := &Dual{Config: conf}
	if err := d.Init(); err != nil {
		t.Fatalf("%+v", err)
	}

	inferer, err := Infer(d, boardSize*boardSize+1, false)
	if err != nil {
		t.Fatal(err)
	}
	defer inferer.Close()

	if inferer.ExecLog() != "" {
		t.Error("Should not have any logs")
	}
}

func TestShuffleBatch(t *testing.T) {
	Xs := tensor.New(tensor.WithShape(5, 1, 3, 2), tensor.WithBacking(G.Uniform(150, 152)(tensor.Float32, 5, 1, 3, 2)))
	pis := tensor.New(tensor.WithShape(5, 6), tensor.WithBacking(G.Uniform(0, 1)(tensor.Float32, 5, 6)))
	vs := tensor.New(tensor.WithShape(5, 6), tensor.WithBacking(G.Uniform(0, 1)(tensor.Float32, 5, 6)))

	originalXs := Xs.Clone().(*tensor.Dense)
	originalPis := pis.Clone().(*tensor.Dense)
	originalVs := vs.Clone().(*tensor.Dense)

	if err := shuffleBatch(Xs, pis, vs); err != nil {
		t.Errorf("err")
	}
	assert := assert.New(t)
	assert.NotEqual(originalXs.Data(), Xs.Data(), "Xs should not be equal")
	assert.NotEqual(originalPis.Data(), pis.Data(), "Pis should not be equal")
	assert.NotEqual(originalVs.Data(), vs.Data(), "Vs should not be equal")

	if t.Failed() {
		t.Logf("Xs:\n%v\nPis:\n%v\nVs:\n%v", originalXs, originalPis, originalVs)
		t.Logf("Xs:\n%v\nPis:\n%v\nVs:\n%v", Xs, pis, vs)
	}
}
