package mcts

// naughty is essentially *Node
type naughty int

func (n naughty) isValid() bool { return n >= 0 }

const (
	nilNode naughty = -1
)
