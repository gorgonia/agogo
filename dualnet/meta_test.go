package dual

import (
	"testing"

	"gorgonia.org/tensor"
)

func TestTrain(t *testing.T) {
	features := 2
	height := 3
	width := 3
	actionSpace := 10
	batchSize := 100
	//batchesZero := 0
	batchesOne := 1

	conf := DefaultConf(3, 3, 10)
	conf.BatchSize = batchSize
	conf.Features = features
	conf.K = 3
	conf.SharedLayers = 3
	type args struct {
		d          *Dual
		Xs         *tensor.Dense
		policies   *tensor.Dense
		values     *tensor.Dense
		batches    int
		iterations int
	}
	d := &Dual{Config: conf}
	if err := d.Init(); err != nil {
		t.Fatalf("%+v", err)
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"issue #2 batch one (not working)",
			args{
				d:          d,
				Xs:         tensor.New(tensor.WithBacking(make([]float32, batchSize*batchesOne*features*height*width)), tensor.WithShape(batchSize*batchesOne, features, height, width)),
				policies:   tensor.New(tensor.WithBacking(make([]float32, batchSize*batchesOne*actionSpace)), tensor.WithShape(batchSize*batchesOne, actionSpace)),
				values:     tensor.New(tensor.WithBacking(make([]float32, batchSize*batchesOne)), tensor.WithShape(batchSize*batchesOne)),
				batches:    batchesOne,
				iterations: 100,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Train(tt.args.d, tt.args.Xs, tt.args.policies, tt.args.values, tt.args.batches, tt.args.iterations); (err != nil) != tt.wantErr {
				t.Errorf("Train() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
