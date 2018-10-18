// +build unsafe

package mcts

// nodeFromNaughty gets the node given the pointer.
func (t *MCTS) nodeFromNaughty(ptr naughty) *Node {
	retVal := &t.nodes[int(ptr)]
	return retVal
}

// Children returns a list of children
func (t *MCTS) Children(of naughty) []naughty {
	retVal := t.children[of]
	return retVal
}
