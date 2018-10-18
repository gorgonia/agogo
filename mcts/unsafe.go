package mcts

import "unsafe"

func treeFromUintptr(ptr uintptr) *MCTS { return (*MCTS)(unsafe.Pointer(ptr)) }

func ptrFromTree(t *MCTS) uintptr { return uintptr(unsafe.Pointer(t)) }
