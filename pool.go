package agogo

import (
	"sync"
)

var iterPool = make(map[int]map[int]*sync.Pool)

func borrowIterator(m, n int) [][]float32 {
	if d, ok := iterPool[m]; ok {
		if d2, ok := d[n]; ok {
			return d2.Get().([][]float32)
		}
	}
	retVal := make([][]float32, m)
	for i := range retVal {
		retVal[i] = make([]float32, n)
	}
	return retVal
}

func ReturnIterator(m, n int, it [][]float32) {
	if d, ok := iterPool[m]; ok {
		if _, ok := d[n]; ok {
			iterPool[m][n].Put(it)
		} else {
			iterPool[m][n] = &sync.Pool{
				New: func() interface{} {
					retVal := make([][]float32, m)
					for i := range retVal {
						retVal[i] = make([]float32, n)
					}
					return retVal
				},
			}
			iterPool[m][n].Put(it)
		}
	} else {
		iterPool[m] = make(map[int]*sync.Pool)
		iterPool[m][n] = &sync.Pool{
			New: func() interface{} {
				retVal := make([][]float32, m)
				for i := range retVal {
					retVal[i] = make([]float32, n)
				}
				return retVal
			},
		}
		iterPool[m][n].Put(it)
	}
}
