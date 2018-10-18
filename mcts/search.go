package mcts

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chewxy/math32"
	"github.com/gorgonia/agogo/game"
)

/*
Here lies the majority of the MCTS search code, while node.go and tree.go handles the data structure stuff.

Right now the code is very specific to the game of Go. Ideally we'd be able to export the correct things and make it
so that a search can be written for any other games but uses the same data structures
*/

const (
	MAXTREESIZE = 25000000 // a tree is at max allowed this many nodes - at about 56 bytes per node that is 1.2GB of memory required
)

func opponent(p game.Player) game.Player {
	switch p {
	case Black:
		return White
	case White:
		return Black
	}
	panic("Unreachable")
}

// Result is a NaN tagged floating point, used to represent the reuslts.
type Result float32

const (
	noResultBits = 0x7FE00000
)

func noResult() Result {
	return Result(math32.Float32frombits(noResultBits))
}

// isNullResult returns true if the Result (a NaN tagged number) is noResult
func isNullResult(r Result) bool {
	b := math32.Float32bits(float32(r))
	return b == noResultBits
}

type searchState struct {
	tree          uintptr
	current, prev game.State
	root          naughty
	depth         int

	wg *sync.WaitGroup

	// config
	maxPlayouts, maxVisits, maxDepth int
}

func (s *searchState) nodeCount() int32 {
	t := treeFromUintptr(s.tree)
	return atomic.LoadInt32(&t.nc)
}

func (s *searchState) incrementPlayout() {
	t := treeFromUintptr(s.tree)
	atomic.AddInt32(&t.playouts, 1)
}

func (s *searchState) isRunning() bool {
	t := treeFromUintptr(s.tree)
	running := t.running.Load().(bool)
	return running && t.nodeCount() < MAXTREESIZE
}

func (s *searchState) minPsaRatio() float32 {
	ratio := float32(s.nodeCount()) / float32(MAXTREESIZE)
	switch {
	case ratio > 0.95:
		return 0.01
	case ratio > 0.5:
		return 0.001
	}
	return 0
}

func (t *MCTS) Search(player game.Player) (retVal game.Single) {
	t.log("SEARCH. Player %v\n%v", player, t.current)
	t.updateRoot()
	t.current.SetToMove(player)
	boardHash := t.current.Hash()

	// freeables
	// if t.current.MoveNumber() == 1 {

	// t.log("Acquiring lock ")
	t.Lock()
	for _, f := range t.freeables {
		t.free(f)
	}
	t.Unlock()
	// }

	t.prepareRoot(player, t.current)
	root := t.nodeFromNaughty(t.root)

	ch := make(chan *searchState, runtime.NumCPU())
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		ss := &searchState{
			tree:     ptrFromTree(t),
			current:  t.current,
			root:     t.root,
			maxDepth: t.M * t.N,
			wg:       &wg,
		}
		ch <- ss
	}

	var iter int32
	t.running.Store(true)
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go doSearch(t.root, &iter, ch, ctx, &wg)
	}
	<-time.After(t.Timeout)
	cancel()

	// TODO
	// reactivate all pruned children
	wg.Wait()
	close(ch)

	root = t.nodeFromNaughty(t.root)
	if !root.HasChildren() {
		policy, _ := t.nn.Infer(t.current)
		moveID := argmax(policy)
		if moveID > t.current.ActionSpace() {
			return Pass
		}
		t.log("Returning Early. Best %v", moveID)
		return game.Single(moveID)
	}

	retVal = t.bestMove()
	t.prev = t.current.Clone().(game.State)
	t.log("Move Number %d, Iterations %d Playouts: %v Nodes: %v. Best: %v", t.current.MoveNumber(), iter, t.playouts, len(t.nodes), retVal)
	t.log("DUMMY")
	// log.Printf("\n%v", t.prev)
	// log.Printf("\tIterations %d Playouts: %v Nodes: %v. Best move %v Player %v", iter, t.playouts, len(t.nodes), retVal, player)

	// update the cached policies.
	// Again, nothing like having side effects to what appears to be a straightforwards
	// pure function eh?
	t.cachedPolicies[sa{boardHash, retVal}]++

	return retVal
}

func doSearch(start naughty, iterBudget *int32, ch chan *searchState, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

loop:
	for {
		select {
		case s := <-ch:
			current := s.current.Clone().(game.State)
			root := start
			res := s.pipeline(current, root)
			if !isNullResult(res) {
				s.incrementPlayout()
			}

			t := treeFromUintptr(s.tree)
			val := atomic.AddInt32(iterBudget, 1)

			if val > t.Budget {
				t.running.Store(false)
			}
			// running := t.running.Load().(bool)
			// running = running && !s.stopThinking( /*TODO*/ )
			// running = running && s.hasAlternateMoves( /*TODO*/ )
			if s.depth == s.maxDepth {
				// reset s for another bout of playouts
				s.root = t.root
				s.current = t.current
				s.depth = 0
			}
			ch <- s
		case <-ctx.Done():
			break loop
		}
	}

	return
}

// pipeline is a recursive MCTS pipeline:
//	SELECT, EXPAND, SIMULATE, BACKPROPAGATE.
//
// Because of the recursive nature, the pipeline is altered a bit to be this:
//	EXPAND and SIMULATE, SELECT and RECURSE, BACKPROPAGATE.
func (s *searchState) pipeline(current game.State, start naughty) (retVal Result) {
	retVal = noResult()
	s.depth++
	if s.depth > s.maxDepth {
		s.depth--
		return
	}

	player := current.ToMove()
	nodeCount := s.nodeCount()

	t := treeFromUintptr(s.tree)
	n := t.nodeFromNaughty(start)
	n.addVirtualLoss()
	t.log("\t%p PIPELINE: %v", s, n)

	// EXPAND and SIMULATE
	isExpandable := n.IsExpandable(0)
	if isExpandable && current.Passes() >= 2 {
		retVal = Result(combinedScore(current))
	} else if isExpandable && nodeCount < MAXTREESIZE {
		hadChildren := n.HasChildren()
		value, ok := s.expandAndSimulate(start, current, s.minPsaRatio())
		if !hadChildren && ok {
			retVal = Result(value)
		}
	}

	// SELECT and RECURSE
	if n.HasChildren() && isNullResult(retVal) {
		next := t.nodeFromNaughty(n.Select(player))
		move := next.Move()
		pm := game.PlayerMove{player, move}

		// Check should check Superko. If it's superko, the node should be invalidated
		if current.Check(pm) {
			current = current.Apply(pm).(game.State)
			retVal = s.pipeline(current, next.id)
		}
	}

	// BACKPROPAGATE
	if !isNullResult(retVal) {
		n.Update(float32(retVal)) // nothing says non functional programs like side effects. Insert more functional programming circle jerk here.
	}
	n.undoVirtualLoss()
	s.depth--
	return retVal
}

func (s *searchState) expandAndSimulate(parent naughty, state game.State, minPsaRatio float32) (value float32, ok bool) {
	t := treeFromUintptr(s.tree)
	n := t.nodeFromNaughty(parent)

	t.log("\t\t%p Expand and Simulate. Parent Move: %v. Player: %v. Move number %d\n%v", s, n.Move(), state.ToMove(), state.MoveNumber(), state)
	if !n.IsExpandable(minPsaRatio) {
		t.log("\t\tNot expandable. MinPSA Ratio %v", minPsaRatio)
		return 0, false
	}

	if state.Passes() >= 2 {
		t.log("\t\t%p Passes >= 2", s)
		return 0, false
	}
	// get scored moves
	var policy []float32              // boardSize + 1
	policy, value = t.nn.Infer(state) // get policy probability, value from neural network
	passProb := policy[len(policy)-1] // probability of a pass is the last in the policy
	player := state.ToMove()
	if player == White {
		value = 1 - value
	}

	var nodelist []pair
	var legalSum float32

	for i := 0; i < s.current.ActionSpace(); i++ {
		if state.Check(game.PlayerMove{player, game.Single(i)}) {
			nodelist = append(nodelist, pair{Score: policy[i], Coord: game.Single(i)})
			legalSum += policy[i]
		}
	}
	t.log("\t\t%p Available Moves %d: %v", s, len(nodelist), nodelist)

	if state.Check(game.PlayerMove{player, Pass}) {
		nodelist = append(nodelist, pair{Score: passProb, Coord: Pass})
		legalSum += passProb
	}

	if legalSum > math32.SmallestNonzeroFloat32 {
		// re normalize
		for i := range nodelist {
			nodelist[i].Score /= legalSum
		}
	} else {
		prob := 1 / float32(len(nodelist))
		for i := range nodelist {
			nodelist[i].Score = prob
		}
	}

	if len(nodelist) == 0 {
		t.log("\t\tNodelist is empty")
		return value, true
	}
	sort.Sort(byScore(nodelist))
	maxPsa := nodelist[0].Score
	oldMinPsa := maxPsa * n.MinPsaRatio()
	newMinPsa := maxPsa * minPsaRatio

	var skippedChildren bool
	for _, p := range nodelist {
		if p.Score < newMinPsa {
			t.log("\t\tp.score %v <  %v", p.Score, newMinPsa)
			skippedChildren = true
		} else if p.Score < oldMinPsa {
			if nn := n.findChild(p.Coord); nn == nilNode {
				nn := t.New(p.Coord, p.Score, value)
				n.AddChild(nn)
			}
		}
	}
	t.log("\t\t%p skipped children? %v", s, skippedChildren)
	if skippedChildren {
		atomic.StoreUint32(&n.minPSARatioChildren, math32.Float32bits(minPsaRatio))
	} else {
		// if no children were skipped, then all that can be expanded has been expanded
		atomic.StoreUint32(&n.minPSARatioChildren, 0)
	}
	return value, true
}

func (t *MCTS) bestMove() game.Single {
	player := t.current.ToMove()
	moveNum := t.current.MoveNumber()

	children := t.children[t.root]
	t.log("%p Children: ", &t.searchState)
	for _, child := range children {
		nc := t.nodeFromNaughty(child)
		t.log("\t\t\t%v", nc)
	}
	t.log("%v", t.current)
	t.childLock[t.root].Lock()
	sort.Sort(fancySort{underEval: player, l: children, t: t})
	t.childLock[t.root].Unlock()

	if moveNum < t.Config.RandomCount {
		t.randomizeChildren(t.root)
	}
	if len(children) == 0 {
		t.log("Board\n%v |%v", t.current, t.nodeFromNaughty(t.root))
		return Pass
	}

	firstChild := t.nodeFromNaughty(children[0])
	bestMove := firstChild.Move()
	bestScore := firstChild.Evaluate(player)

	root := t.nodeFromNaughty(t.root)
	switch {
	case t.Config.PassPreference == DontPreferPass && bestMove.IsPass():
		bestMove, bestScore = t.noPassBestMove(bestMove, bestScore, t.root, t.current, player)
	case !t.Config.DumbPass && bestMove.IsPass():
		score := root.Score()
		if (score > 0 && player == White) || (score < 0 && player == Black) {
			// passing will cause a loss. Let's find an alternative
			bestMove, bestScore = t.noPassBestMove(bestMove, bestScore, t.root, t.current, player)
		}
	case !t.Config.DumbPass && t.current.LastMove().IsPass():
		score := root.Score()
		if (score > 0 && player == White) || (score < 0 && player == Black) {
			// passing loses. Play on.
		} else {
			bestMove = Pass
		}
	}
	if bestMove.IsPass() && t.shouldResign(bestScore, player) {
		bestMove = Resign
	}
	return bestMove
}

func (t *MCTS) prepareRoot(player game.Player, state game.State) {
	root := t.nodeFromNaughty(t.root)
	hadChildren := len(t.children[t.root]) > 0
	expandable := root.IsExpandable(0)
	var value float32
	if expandable {
		value, _ = t.expandAndSimulate(t.root, state, t.minPsaRatio())
	}

	if hadChildren {
		value = root.Evaluate(player)
	} else {
		root.Update(value)
		if player == White {
			// DO SOMETHING
		}
	}

	// disable any children that is not suitable to be used
	// children := t.children[t.root]
	// for _, child := range children {
	// 	c := t.nodeFromNaughty(child)
	// 	if !t.searchState.current.Check(game.PlayerMove{player, game.Single(c.move)}) {
	// 		log.Printf("Invalidating %v", c.move)
	// 		c.Invalidate()
	// 	}
	// }
}

// newRootState moves the search state to use a new root state. It returns true when a new root state was created.
//
// As a side effect, the freeables list is also updated.
func (t *MCTS) newRootState() bool {
	if t.root == nilNode || t.prev == nil {
		t.log("No root")
		return false // no current state. Cannot advance to new state
	}
	depth := t.current.MoveNumber() - t.prev.MoveNumber()
	if depth < 0 {
		t.log("depth < 0")
		return false // oops too far
	}

	tmp := t.current.Clone().(game.State)
	for i := 0; i < depth; i++ {
		tmp.UndoLastMove()
	}
	if !tmp.Eq(t.prev) {
		return false // they're not the same tree - a new root needs to be created
	}
	// try to replay tmp
	t.log("depth %v", depth)
	for i := 0; i < depth; i++ {
		tmp.Fwd()
		move := tmp.LastMove()

		oldRoot := t.root
		oldRootNode := t.nodeFromNaughty(oldRoot)
		newRoot := oldRootNode.findChild(move.Single)
		if newRoot == nilNode {
			return false
		}
		t.Lock()
		t.root = newRoot
		t.Unlock()
		t.cleanup(oldRoot, newRoot)

		t.prev = t.prev.Apply(move).(game.State)
	}

	if t.current.MoveNumber() != t.prev.MoveNumber() {
		return false
	}
	if !t.current.Eq(t.prev) {
		return false
	}
	return true
}

// updateRoot updates the root after searching for a new root state.
// If no new root state can be found, a new Node indicating a PASS move is made.
func (t *MCTS) updateRoot() {
	t.freeables = t.freeables[:0]
	player := t.searchState.current.ToMove()
	if !t.newRootState() || t.searchState.root == nilNode {
		// search for the first useful
		if ok := t.searchState.current.Check(game.PlayerMove{player, Pass}); ok {
			t.searchState.root = t.New(Pass, 0, 0)
		} else {
			for i := 0; i < t.searchState.current.ActionSpace(); i++ {
				if t.searchState.current.Check(game.PlayerMove{player, game.Single(i)}) {
					t.searchState.root = t.New(game.Single(i), 0, 0)
					break
				}
			}
		}
	}
	t.log("freables %d", len(t.freeables))
	t.searchState.prev = nil
	root := t.nodeFromNaughty(t.searchState.root)
	atomic.StoreInt32(&t.nc, int32(root.countChildren()))

	// if root has no children
	children := t.Children(t.searchState.root)
	if len(children) == 0 {
		atomic.StoreUint32(&root.minPSARatioChildren, defaultMinPsaRatio)
	}

}

func (t *MCTS) shouldResign(bestScore float32, player game.Player) bool {

	if t.Config.PassPreference == DontResign {
		return false
	}
	if t.Config.ResignPercentage == 0 {
		return false
	}
	squares := t.Config.M * t.Config.N
	threshold := squares / 4
	moveNumber := t.current.MoveNumber()
	if moveNumber <= threshold {
		// too early to resign
		return false
	}

	var resignThreshold float32
	if t.Config.ResignPercentage < 0 {
		resignThreshold = 0.1
	} else {
		resignThreshold = t.Config.ResignPercentage
	}

	if bestScore > resignThreshold {
		return false
	}
	// TODO handicap
	// handicap := t.current.Handicap()
	// if handicap > 0 && player == White && t.Config.ResignPercentage < 0 {

	// }
	return true
}

// noPass finds aa child that is NOT a pass move that is valid (i.e. not in eye states for example)
func (t *MCTS) noPass(of naughty, state game.State, player game.Player) naughty {
	children := t.children[of]
	for _, kid := range children {
		child := t.nodeFromNaughty(kid)
		move := child.Move()

		// in Go games, this also checks for eye-ish situations
		ok := state.Check(game.PlayerMove{player, move})
		if !move.IsPass() && ok {
			return kid
		}
	}
	return nilNode
}

func (t *MCTS) noPassBestMove(bestMove game.Single, bestScore float32, of naughty, state game.State, player game.Player) (game.Single, float32) {
	nopass := t.noPass(of, state, player)
	if nopass.isValid() {
		np := t.nodeFromNaughty(nopass)
		bestMove = np.Move()
		bestScore = 1
		if !np.IsNotVisited() {
			bestScore = np.Evaluate(player)
		}
	}
	return bestMove, bestScore
}
