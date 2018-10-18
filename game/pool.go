package game

import (
	"sync"
)

var iterPool = make(map[int32]map[int32]*sync.Pool)

func borrowIterator(m, n int32) [][]Colour {
	if d, ok := iterPool[m]; ok {
		if d2, ok := d[n]; ok {
			return d2.Get().([][]Colour)
		}
	}
	retVal := make([][]Colour, m)
	for i := range retVal {
		retVal[i] = make([]Colour, n)
	}
	return retVal
}

func ReturnIterator(m, n int32, it [][]Colour) {
	if d, ok := iterPool[m]; ok {
		if _, ok := d[n]; ok {
			iterPool[m][n].Put(it)
		} else {
			iterPool[m][n] = &sync.Pool{
				New: func() interface{} {
					retVal := make([][]Colour, m)
					for i := range retVal {
						retVal[i] = make([]Colour, n)
					}
					return retVal
				},
			}
			iterPool[m][n].Put(it)
		}
	} else {
		iterPool[m] = make(map[int32]*sync.Pool)
		iterPool[m][n] = &sync.Pool{
			New: func() interface{} {
				retVal := make([][]Colour, m)
				for i := range retVal {
					retVal[i] = make([]Colour, n)
				}
				return retVal
			},
		}
		iterPool[m][n].Put(it)
	}
}
