package mcts

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"

	"github.com/awalterschulze/gographviz"
	"github.com/gorgonia/agogo/game"
)

type statefulNode struct {
	*Node
	Player game.Colour
	board  []game.Colour
	stride int
}

func (s *statefulNode) State() string {
	var buf bytes.Buffer
	for i, c := range s.board {
		if i%s.stride == 0 {
			fmt.Fprint(&buf, "⎢ ")
		}
		fmt.Fprintf(&buf, "%s ", c)
		if (i+1)%s.stride == 0 && i != 0 {
			fmt.Fprint(&buf, "⎥<BR />")
		}
	}
	return buf.String()
}

func (t *MCTS) ToDot() string {
	g := gographviz.NewGraph()
	if err := g.SetName("G"); err != nil {
		panic(err)
	}
	g.SetDir(true)

	var states []*statefulNode
	for i := range t.nodes {
		n := &t.nodes[i]
		s := &statefulNode{
			Node:   n,
			board:  make([]game.Colour, t.M*t.N),
			stride: t.M,
		}
		states = append(states, s)
	}

	var buf bytes.Buffer
	for i, kids := range t.children {
		n := states[i]
		if !n.IsActive() {
			continue
		}
		if n.Player == game.None {
			n.Player = game.Black
		}
		move := n.Move()
		if !move.IsPass() && !move.IsResignation() {
			n.board[move] = n.Player
		}

		tmpl.Execute(&buf, n)
		attrs := map[string]string{
			"fontname": "Monaco",
			"shape":    "none",
			"label":    buf.String(),
		}
		g.AddNode("G", fmt.Sprintf("%v", n.id), attrs)
		buf.Reset()
		sort.Sort(byMove{l: kids, t: t})

		for _, kid := range kids {
			child := t.nodeFromNaughty(kid)
			if !child.IsActive() {
				continue
			}
			s := states[child.id]
			copy(s.board, n.board)
			s.Player = game.Colour(opponent(game.Player(n.Player)))

			g.AddEdge(fmt.Sprintf("%v", n.id), fmt.Sprintf("%v", kid), true, nil)
		}

	}
	return g.String()
}

const tmplRaw = `<
<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0">
<TR><TD>Node ID</TD><TD>xx{{.ID}}</TD></TR>
<TR><TD>Move</TD><TD>{{.Move}}</TD></TR>
<TR><TD>Player</TD><TD>{{.Player}}</TD></TR>
<TR><TD>Visits</TD><TD>{{.Visits}}</TD></TR>
<TR><TD>Score</TD><TD>{{.Score}}</TD></TR>
<TR><TD>Value</TD><TD>{{.Value}}</TD></TR>
<TR><TD>State</TD><TD>{{.State}}</TD></TR>
</TABLE>
>
`

var tmpl *template.Template

func init() {
	tmpl = template.Must(template.New("name").Parse(tmplRaw))
}
