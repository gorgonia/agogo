package gtp

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/gorgonia/agogo/game"
	"github.com/pkg/errors"
)

type Command interface {
	Do(id int, args []string, e *Engine) (int, string, error)
}

type stdlib func(e *Engine) string

type stdlib2 func(e *Engine, args []string) (string, error)

func (f stdlib) Do(id int, args []string, e *Engine) (int, string, error) {
	str := f(e)
	return id, str, nil
}

func (f stdlib2) Do(id int, args []string, e *Engine) (int, string, error) {
	str, err := f(e, args)
	return id, str, err
}

func protocolVersion(e *Engine) string { return "2" }
func name(e *Engine) string            { return e.name }
func version(e *Engine) string         { return e.version }

func listCommands(e *Engine) string {
	var buf bytes.Buffer
	for c := range e.known {
		fmt.Fprintf(&buf, "%v\n", c)
	}
	return buf.String()
}

func quit(e *Engine) string       { close(e.ch); return "QUIT" }
func clearBoard(e *Engine) string { e.g.Reset(); return "" }
func showboard(e *Engine) string  { return fmt.Sprintf("\n%v\n", e.g) }
func undo(e *Engine) string       { e.g.UndoLastMove(); return "" }

func knownCommand(e *Engine, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("Not enough arguments for \"known_command\"")
	}
	if _, ok := e.known[args[0]]; ok {
		return "true", nil
	}
	return "false", nil
}

func boardSize(e *Engine, args []string) (string, error) {
	// arg
	switch len(args) {
	case 0:
		return "", errors.New("Not enough arguments for \"boardsize\"")
	case 1:
		newsize, err := strconv.Atoi(args[0])
		if err != nil {
			return "", errors.WithMessage(err, "Unable to parse first argument of boardsize")
		}
		m := sqrt(newsize)
		e.g = e.New(m, m)
		return "", nil
	default:
		newM, err := strconv.Atoi(args[0])
		if err != nil {
			return "", errors.WithMessage(err, "Unable to parse first argument of boardsize")
		}
		newN, err := strconv.Atoi(args[1])
		if err != nil {
			return "", errors.WithMessage(err, "Unable to parse second argument of boardsize")
		}
		e.g = e.New(newM, newN)

		return "", nil
	}
}

func komi(e *Engine, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("Not enough arguments for \"komi\"")
	}

	komi, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return "", errors.WithMessage(err, "Unable to parse komi argument")
	}

	if ks, ok := e.g.(game.KomiSetter); ok {
		ks.SetKomi(komi) // ignore errors because GTP says so. Accept komi even if ridiculous
	}
	return "", nil
}

func play(e *Engine, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("Not enough arguments for \"play\"")
	}
	return "", errors.New("NYI")
}

func genmove(e *Engine, args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("Not enough arguments for \"genmove\"")
	}
	if e.Generate == nil {
		return "", errors.New("Unable to generate moves. No generator found")
	}
	return "", errors.New("NYI")
}

func StandardLib() map[string]Command {
	return map[string]Command{
		"protocol_version": stdlib(protocolVersion),
		"name":             stdlib(name),
		"version":          stdlib(version),
		"list_commands":    stdlib(listCommands),
		"quit":             stdlib(quit),
		"clear_board":      stdlib(clearBoard),
		"showboard":        stdlib(showboard),
		"undo":             stdlib(undo),

		"known_command": stdlib2(knownCommand),
		"boardsize":     stdlib2(boardSize),
		"komi":          stdlib2(komi),
		"play":          stdlib2(play),
		"genmove":       stdlib2(genmove),
	}
}
