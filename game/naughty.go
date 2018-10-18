package game

import (
	"reflect"
	"unsafe"
)

// MakeIterator makes a generic iterator of a board of colours
func MakeIterator(board []Colour, m, n int32) (retVal [][]Colour) {
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
