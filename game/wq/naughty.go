package 围碁

import (
	"reflect"
	"unsafe"

	"github.com/gorgonia/agogo/game"
)

// makeBoard makes a board of NxN. Additionally, it also returna s 2D iterator
func makeBoard(size int) (board []game.Colour, iterator [][]game.Colour) {
	board = make([]game.Colour, size*size, size*size)
	iterator = make([][]game.Colour, size)
	for i := range iterator {
		start := i * size
		hdr := &reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&board[start])),
			Len:  size,
			Cap:  size,
		}
		iterator[i] = *(*[]game.Colour)(unsafe.Pointer(hdr))
	}
	return
}

func makeZobristTable(size int) (table []int32, iterator [][]int32) {
	table = make([]int32, size*size*2, size*size*2)
	iterator = make([][]int32, size*size, size*size)
	rowStride := 2
	for i := range iterator {
		start := i * rowStride
		hdr := &reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&table[start])),
			Len:  rowStride,
			Cap:  rowStride,
		}
		iterator[i] = *(*[]int32)(unsafe.Pointer(hdr))
	}
	return
}
