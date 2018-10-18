package agogo

import (
	"reflect"
	"unsafe"
)

// MakeIterator makes a generic iterator of a board
func MakeIterator(board []float32, m, n int) (retVal [][]float32) {
	retVal = borrowIterator(m, n)
	for i := range retVal {
		start := i * int(m)
		hdr := (*reflect.SliceHeader)(unsafe.Pointer(&retVal[i]))
		hdr.Data = uintptr(unsafe.Pointer(&board[start]))
		hdr.Len = int(n)
		hdr.Cap = int(n)
	}
	return
}
