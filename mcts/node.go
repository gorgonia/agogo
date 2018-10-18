package mcts

import (
	"fmt"
	"sort"
	"sync/atomic"

	"github.com/chewxy/math32"
	"github.com/gorgonia/agogo/game"
)

type Status uint32

const (
	Invalid Status = iota
	Active
	Pruned
)

func (a Status) String() string {
	switch a {
	case Invalid:
		return "Invalid"
	case Active:
		return "Active"
	case Pruned:
		return "Pruned"
	}
	return "UNKNOWN STATUS"
}

type Node struct {

	// atomic access only pl0x
	move   int32  // should be game.Single
	visits uint32 // visits to this node - N(s, a) in the literature
	status uint32 // status

	// float32s
	blackScores         uint32 // actually float32.
	virtualLoss         uint32 // actually float32. a virtual loss number - 0 or 3
	minPSARatioChildren uint32 // actually float32. minimum P(s,a) ratio for the children. Default to 2
	score               uint32 // Policy estimate for taking the move above (from NN)
	value               uint32 // value from the neural network

	// naughty things
	id   naughty // index to the children allocation
	tree uintptr // pointer to the tree
}

func (n *Node) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "{NodeID: %v Move: %v, Score: %v, Value %v Visits %v minPSARatioChildren %v Status: %v}", n.id, n.Move(), n.Score(), n.value, n.Visits(), n.minPSARatioChildren, Status(n.status))
}

// AddChild adds a child to the node
func (n *Node) AddChild(child naughty) {
	tree := treeFromUintptr(n.tree)
	tree.Lock()
	tree.children[n.id] = append(tree.children[n.id], child)
	tree.Unlock()
}

// IsFirstVisit returns true if this node hasn't ever been visited
func (n *Node) IsNotVisited() bool {
	visits := atomic.LoadUint32(&n.visits)
	return visits == 0
}

// Update updates the accumulated score
func (n *Node) Update(score float32) {
	t := treeFromUintptr(n.tree)
	t.Lock()
	atomic.AddUint32(&n.visits, 1)
	n.accumulate(score)
	t.Unlock()
}

// BlackScores returns the scores for black
func (n *Node) BlackScores() float32 {
	blackScores := atomic.LoadUint32(&n.blackScores)
	return math32.Float32frombits(blackScores)
}

// Move gets the move associated with the node
func (n *Node) Move() game.Single { return game.Single(atomic.LoadInt32(&n.move)) }

// Score returns the score
func (n *Node) Score() float32 {
	v := atomic.LoadUint32(&n.score)
	return math32.Float32frombits(v)
}

// Value returns the predicted value (probability of winning from the NN) of the given node
func (n *Node) Value() float32 {
	v := atomic.LoadUint32(&n.value)
	return math32.Float32frombits(v)
}

func (n *Node) Visits() uint32 { return atomic.LoadUint32(&n.visits) }

// Activate activates the node
func (n *Node) Activate() { atomic.StoreUint32(&n.status, uint32(Active)) }

// Prune prunes the node
func (n *Node) Prune() { atomic.StoreUint32(&n.status, uint32(Pruned)) }

// Invalidate invalidates the node
func (n *Node) Invalidate() { atomic.StoreUint32(&n.status, uint32(Invalid)) }

// IsValid returns true if it's valid
func (n *Node) IsValid() bool {
	status := atomic.LoadUint32(&n.status)
	return Status(status) != Invalid
}

// IsActive returns true if the node is active
func (n *Node) IsActive() bool {
	status := atomic.LoadUint32(&n.status)
	return Status(status) == Active
}

// IsPruned returns true if the node has been pruned.
func (n *Node) IsPruned() bool {
	status := atomic.LoadUint32(&n.status)
	return Status(status) == Pruned
}

// HasChildren returns true if the node has children
func (n *Node) HasChildren() bool { return n.MinPsaRatio() <= 1 }

// IsExpandable returns true if the node is exandable. It may not be for memory reasons.
func (n *Node) IsExpandable(minPsaRatio float32) bool { return minPsaRatio < n.MinPsaRatio() }

func (n *Node) VirtualLoss() float32 {
	v := atomic.LoadUint32(&n.virtualLoss)
	return math32.Float32frombits(v)
}

func (n *Node) MinPsaRatio() float32 {
	v := atomic.LoadUint32(&n.minPSARatioChildren)
	return math32.Float32frombits(v)
}

func (n *Node) ID() int { return int(n.id) }

// Evaluate evaluates a move made by a player
func (n *Node) Evaluate(player game.Player) float32 {
	visits := n.Visits()
	blackScores := n.BlackScores()
	if player == White {
		blackScores += n.VirtualLoss()
	}

	score := blackScores / float32(visits)
	if player == White {
		score = 1 - score
	}
	return score
}

// NNEvaluate returns the result of the NN evaluation of the colour.
func (n *Node) NNEvaluate(player game.Player) float32 {
	if player == White {
		return 1.0 - n.Value()
	}
	return n.Value()
}

// Select selects the child of the given Colour
func (n *Node) Select(of game.Player) naughty {
	// sumScore is the sum of scores of the node that has been visited by the policy
	var sumScore float32
	var parentVisits uint32

	tree := treeFromUintptr(n.tree)

	children := tree.Children(n.id)
	for _, kid := range children {
		child := tree.nodeFromNaughty(kid)
		if child.IsValid() {
			visits := child.Visits()
			parentVisits += visits
			if visits > 0 {
				sumScore += child.Score()
			}
		}
	}

	// the upper bound formula is as such
	// U(s, a) = Q(s, a) + tree.PUCT * P(s, a) * ((sqrt(parent visits))/ (1+visits to this node))
	//
	// where
	// U(s, a) = upper confidence bound given state and action
	// Q(s, a) = reward of taking the action given the state
	// P(s, a) = iniital probability/estimate of taking an action from the state given according to the policy
	//
	// in the following code,
	// psa = P(s, a)
	// qsa = Q(s, a)
	//
	// Given the state and action is already known and encoded into Node itself,it doesn't have to be a function
	// like in most MCTS tutorials. This allows it to be slightly more performant (i.e. a AoS-ish data structure)

	var best naughty
	var bestValue float32 = math32.Inf(-1)
	fpu := n.NNEvaluate(of)                         // first play urgency is the value predicted by the NN
	numerator := math32.Sqrt(float32(parentVisits)) // in order to find the stochastic policy, we need to normalize the count

	for _, kid := range children {
		child := tree.nodeFromNaughty(kid)
		if !child.IsActive() {
			continue
		}

		qsa := fpu // the initial Q is what the NN predicts
		visits := child.Visits()
		if visits > 0 {
			qsa = child.Evaluate(of) // but if this node has been visited before, Q from the node is used.
		}
		psa := child.Score()
		denominator := 1.0 + float32(visits)
		lastTerm := (numerator / denominator)
		puct := tree.PUCT * psa * lastTerm
		usa := qsa + puct

		if usa > bestValue {
			bestValue = usa
			best = kid
		}
	}

	if best == nilNode {
		panic("Cannot return nil")
	}
	// log.Printf("SELECT %v. Best %v - %v", of, best, tree.nodeFromNaughty(best))
	return best
}

// BestChild returns the best scoring child. Note that fancySort has all sorts of heuristics
func (n *Node) BestChild(player game.Player) naughty {
	tree := treeFromUintptr(n.tree)
	children := tree.Children(n.id)

	sort.Sort(fancySort{player, children, tree})
	return children[len(children)-1]
}

func (n *Node) addVirtualLoss() {
	t := treeFromUintptr(n.tree)
	t.Lock()
	atomic.StoreUint32(&n.virtualLoss, virtualLoss1)
	t.Unlock()
}

func (n *Node) undoVirtualLoss() {
	t := treeFromUintptr(n.tree)
	t.Lock()
	atomic.StoreUint32(&n.virtualLoss, 0)
	t.Unlock()
}

// accumulate adds to the score atomically
func (n *Node) accumulate(score float32) {
	blackScores := atomic.LoadUint32(&n.blackScores)
	evals := math32.Float32frombits(blackScores)
	evals += score
	blackScores = math32.Float32bits(evals)
	atomic.StoreUint32(&n.blackScores, blackScores)

}

// countChildren counts the number of children node a node has and number of grandkids recursively
func (n *Node) countChildren() (retVal int) {
	tree := treeFromUintptr(n.tree)
	children := tree.Children(n.id)
	for _, kid := range children {
		child := tree.nodeFromNaughty(kid)
		if child.IsActive() {
			retVal += child.countChildren()
		}
		retVal++ // plus the child itself
	}
	return
}

// findChild finds the first child that has the wanted move
func (n *Node) findChild(move game.Single) naughty {
	tree := treeFromUintptr(n.tree)
	children := tree.Children(n.id)
	for _, kid := range children {
		child := tree.nodeFromNaughty(kid)
		if game.Single(child.Move()) == move {
			return kid
		}
	}
	return nilNode
}

func (n *Node) reset() {
	atomic.StoreInt32(&n.move, -1)
	atomic.StoreUint32(&n.visits, 0)
	atomic.StoreUint32(&n.status, 0)
	atomic.StoreUint32(&n.blackScores, 0)
	atomic.StoreUint32(&n.minPSARatioChildren, defaultMinPsaRatio)
	atomic.StoreUint32(&n.score, 0)
	atomic.StoreUint32(&n.value, 0)
	atomic.StoreUint32(&n.virtualLoss, 0)
}
