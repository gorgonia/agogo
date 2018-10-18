package mcts

import "testing"

func TestUnsafe(t *testing.T) {
	tree := &MCTS{}
	ptr := ptrFromTree(tree)
	if ptr == 0 {
		t.Fatal("Impossible to get 0x0 from a valid tree")
	}
	tree2 := treeFromUintptr(ptr)
	if tree2 != tree {
		t.Fatal("Expected the same pointer for trees")
	}

	ptr = ptrFromTree(nil)
	if ptr != 0x0 {
		t.Fatal("Must get 0x0 from a nil tree")
	}

	tree2 = treeFromUintptr(0x0)
	if tree2 != nil {
		t.Fatal("tree2 has to be nil")
	}
}
