package komi

import (
	"reflect"
	"unsafe"
)

func (z *zobrist) makeIterator() {
	z.it = make([][]int32, z.size, z.size)
	rowStride := 2
	for i := range z.it {
		start := i * rowStride
		hdr := &reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&z.table[start])),
			Len:  rowStride,
			Cap:  rowStride,
		}
		z.it[i] = *(*[]int32)(unsafe.Pointer(hdr))
	}
}
