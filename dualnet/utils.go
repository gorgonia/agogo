package dual

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"gorgonia.org/tensor"
)

type slicer struct {
	v   tensor.View
	err error
}

func (s *slicer) Slice(a *tensor.Dense, slices ...tensor.Slice) *tensor.Dense {
	if s.err != nil {
		return nil
	}
	if s.v, s.err = a.Slice(slices...); s.err != nil {
		s.err = errors.Wrapf(s.err, "Slicer failed") // get a stack trace
		return nil
	}
	return s.v.(*tensor.Dense)
}

type rs struct {
	start, end, step int
}

func (s rs) Start() int { return s.start }
func (s rs) End() int   { return s.end }
func (s rs) Step() int  { return s.step }

// s creates a ranged slice. It takes an optional step param.
func sli(start, end int, opts ...int) rs {
	step := 1
	if len(opts) > 0 {
		step = opts[0]
	}
	return rs{
		start: start,
		end:   end,
		step:  step,
	}
}

type manyErr []error

func (err manyErr) Error() string {
	var buf bytes.Buffer
	for _, e := range err {
		fmt.Fprintln(&buf, e.Error())
	}
	return buf.String()
}
