package gtp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gorgonia/agogo/game"
	"github.com/pkg/errors"
)

var known_commands = []string{
	"protocol_version",
	"name",
	"version",
	"known_command",
	"list_commands",
	"quit",

	// setup
	"boardsize",
	"clear_board",
	"komi",
	"fixed_handicap",
	"place_free_handicap",
	"set_free_handicap",

	// play
	"play",
	"genmove",
	"undo",
}

type Engine struct {
	g game.State

	known map[string]Command

	ch  chan string
	ret chan string

	Generate      func(g game.State) game.PlayerMove
	New           func(m, n int) game.State
	name, version string
}

func New(g game.State, name, version string, known map[string]Command) *Engine {
	if known == nil {
		known = StandardLib()
	}
	return &Engine{
		g:       g,
		known:   known,
		name:    name,
		version: version,
	}
}

func (e *Engine) Start() (input, output chan string) {
	e.ch = make(chan string)
	e.ret = make(chan string)
	go e.start()
	return e.ch, e.ret
}

func (e *Engine) State() game.State { return e.g }

func (e *Engine) start() {
	for cmd := range e.ch {
		id, x, args, err := e.parse(cmd)
		if x == nil && err == nil {
			continue //
		}
		if err != nil {
			e.ret <- handleErr(id, err)
			continue
		}
		id, result, err := x.Do(id, args, e)
		e.ret <- handleResult(id, result, err)
	}
}

// refer to this
// https://www.lysator.liu.se/%7Egunnar/gtp/gtp2-spec-draft2/gtp2-spec.html#SECTION00030000000000000000
func (e *Engine) parse(cmd string) (id int, x Command, args []string, err error) {
	cmd = preprocess(cmd)
	tokens := strings.Fields(cmd)
	if id, err = strconv.Atoi(tokens[0]); err == nil {
		// we've consumed ID
		tokens = tokens[1:]
	} else {
		// set err to nil because ID is optional
		err = nil
		id = -1
	}

	if len(tokens) == 0 {
		return id, nil, nil, nil // GNUGo some how does nothing when there are no tokens left. An ID may be passed in but it'll be ignored
	}

	var ok bool
	if x, ok = e.known[tokens[0]]; !ok {
		return id, nil, nil, errors.Errorf("Unknown command %q", tokens[0])
	}
	if len(tokens) > 1 {
		args = tokens[1:]
	}
	return
}

func preprocess(a string) string {
	return strings.ToLower(strings.TrimSpace(a))
}

func sqrt(a int) int {
	if a == 0 || a == 1 {
		return a
	}
	start := 1
	end := a / 2
	var retVal int
	for start <= end {
		mid := (start + end) / 2
		sq := mid * mid
		if sq == a {
			return mid
		}
		if sq < a {
			start = mid + 1
			retVal = mid
		} else {
			end = mid - 1
		}
	}
	return retVal
}

func handleErr(id int, err error) string {
	if id != -1 {
		return fmt.Sprintf("? %d %v\n\n", id, err)
	}
	return fmt.Sprintf("? %v\n\n", err)
}

func handleResult(id int, result string, err error) string {
	if err != nil {
		return handleErr(id, err)
	}

	if id != -1 {
		return fmt.Sprintf("= %d %v\n\n", id, result)
	}
	return fmt.Sprintf("= %v\n\n", result)
}
