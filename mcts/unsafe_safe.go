// +build !unsafe

package mcts

// nodeFromNaughty gets the node given the pointer.
func (t *MCTS) nodeFromNaughty(ptr naughty) *Node {
	t.RLock()
	nodes := t.nodes
	t.RUnlock()
	retVal := &nodes[int(ptr)]
	return retVal
}

// Children returns a list of children
func (t *MCTS) Children(of naughty) []naughty {
	t.RLock()
	retVal := t.children[of]
	t.RUnlock()
	return retVal
}
