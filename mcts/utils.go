package mcts

import (
	"github.com/chewxy/math32"
	"github.com/gorgonia/agogo/game"
)

// fancySort sorts the list of nodes under a certain condition of evaluation (i.e. which colour are we considering)
// it sorts in such a way that nils get put at the back
type fancySort struct {
	underEval game.Player
	l         []naughty
	t         *MCTS
}

func (l fancySort) Len() int      { return len(l.l) }
func (l fancySort) Swap(i, j int) { l.l[i], l.l[j] = l.l[j], l.l[i] }
func (l fancySort) Less(i, j int) bool {
	li := l.t.nodeFromNaughty(l.l[i])
	lj := l.t.nodeFromNaughty(l.l[j])

	// // push nils to the back
	// switch {
	// case li == nil && lj != nil:
	// 	return false
	// case li != nil && lj == nil:
	// 	return true
	// case li == nil && lj == nil:
	// 	return false
	// }

	// check if both have the same visits

	liVisits := li.Visits()
	ljVisits := lj.Visits()
	if liVisits != ljVisits {
		return liVisits > ljVisits
	}

	// no visits, we sort on score
	if liVisits == 0 {
		return li.Score() > lj.Score()
	}

	// same visit count. Evaluate
	return li.Evaluate(l.underEval) > lj.Evaluate(l.underEval)
}

// pair is a tuple of score and coordinate
type pair struct {
	Coord game.Single
	Score float32
}

// byScore is a sortable list of pairs It sorts the list with best score fist
type byScore []pair

func (l byScore) Len() int           { return len(l) }
func (l byScore) Less(i, j int) bool { return l[i].Score > l[j].Score }
func (l byScore) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

func combinedScore(state game.State) float32 {
	whiteScore := state.Score(White)
	blackScore := state.Score(Black)
	komi := state.AdditionalScore()
	return blackScore - whiteScore - komi
}

type byMove struct {
	t *MCTS
	l []naughty
}

func (l byMove) Len() int { return len(l.l) }
func (l byMove) Less(i, j int) bool {
	li := l.t.nodeFromNaughty(l.l[i])
	lj := l.t.nodeFromNaughty(l.l[j])
	return li.move < lj.move
}
func (l byMove) Swap(i, j int) {
	l.l[i], l.l[j] = l.l[j], l.l[i]
}

func argmax(a []float32) int {
	var retVal int
	var max float32 = math32.Inf(-1)
	for i := range a {
		if a[i] > max {
			max = a[i]
			retVal = i
		}
	}
	return retVal
}
