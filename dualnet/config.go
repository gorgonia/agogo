package dual

// Config configures the neural network
type Config struct {
	K            int     // number of filters
	SharedLayers int     // number of shared residual blocks
	FC           int     // fc layer width
	L2           float64 // L2 regularization

	BatchSize     int // batch size
	Width, Height int // board size
	Features      int // feature counts

	ActionSpace int
	FwdOnly     bool // is this a fwd only graph?
}

func DefaultConf(m, n, actionSpace int) Config {
	k := round((m * n) / 3)
	return Config{
		K:            k,
		SharedLayers: m,
		FC:           2 * k,

		BatchSize:   256,
		Width:       n,
		Height:      m,
		Features:    18,
		ActionSpace: actionSpace,
	}
}

func (conf Config) IsValid() bool {
	return conf.K >= 1 &&
		conf.ActionSpace >= 3 &&
		// conf.SharedLayers >= conf.BoardSize &&
		conf.SharedLayers >= 0 &&
		conf.FC > 1 &&
		conf.BatchSize >= 1 &&
		// conf.ActionSpace >= conf.Width*conf.Height &&
		conf.Features > 0
}

func round(a int) int {
	n := a - 1
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++

	lt := n / 2
	if (a - lt) < (n - a) {
		return lt
	}
	return n
}
