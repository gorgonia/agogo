package online

type FwdWorkItem struct {
	Layer            *Layer
	HiddenChunkIndex int
	// RNG??
}

func (item *FwdWorkItem) Do() error { return nil }

type Layer struct {
	Width, Height, Chunksize int

	HiddenStates     []int
	PrevHiddenStates []int

	FeedFwdWeights [][]float64
	FeedBwdWeights [][][]Pair

	ReconActivations     [][]Pair
	PrevReconActivations [][]Pair

	VisibleLayers []Desc

	Predictions [][]int
	Input       [][]int
	PrevInput   [][]int

	Feedback     []int
	PrevFeedback []int

	Alpha, Beta float64
}

type Pair struct {
	A, B float64
}

type Desc struct {
	W, H, Chunksize int

	FwdRadius, BwdRadius int

	Predict bool // should this description be predicted?
}
